# Connect V3 — Service + Native Desktop App

## Cel Produktu

Connect V3 to aplikacja desktopowa do technicznej wymiany artefaktów między komputerami w LAN:
pliki, teksty, snippety, komendy, logi. Działa bez przeglądarki — jako natywna aplikacja
komunikująca się z lokalnym serwisem systemowym.

---

## Architektura

```
┌─────────────────────────────────────┐      localhost      ┌─────────────────────────────┐
│  connectd  (serwis systemowy)       │ ◄──── HTTP/1.1 ───► │  connect-ui  (desktop app)  │
│                                     │                     │                             │
│  Język: Go                          │                     │  Shell: Tauri (Rust)        │
│  Uruchamianie: launchd / Windows Sv.│                     │  UI: HTML/CSS/JS (istniejący)│
│  Zależności: sqlite, brak CGO       │                     │  Rozmiar: ~5 MB + sidecar   │
│  Binary: connectd (~12 MB)          │                     │                             │
│                                     │                     │  Custom protocol:           │
│  - storage/SQLite                   │                     │    connect://join/{code}    │
│  - pairing/invitations              │                     │                             │
│  - sync / operations                │                     │  - native window            │
│  - file storage                     │                     │  - system tray              │
│  - REST API (localhost:47810)       │                     │  - native file dialogs      │
└─────────────────────────────────────┘                     │  - OS notifications         │
                                                            │  - drag & drop z OS         │
                                                            └─────────────────────────────┘
```

### Zasady

1. **Pełna separacja.** Serwis i UI to osobne procesy, osobne binary, osobne repozytoria
   (lub osobne katalogi w monorepo). Jedyna komunikacja: REST API na localhost.

2. **Brak przeglądarki.** UI działa w natywnym oknie (WebView), nie w karcie przeglądarki.
   Użytkownik nigdy nie widzi paska adresu, zakładek ani DevTools.

3. **Custom protocol.** Linki parowania używają `connect://join/{code}` zamiast
   `http://host/join/{code}`. Kliknięcie w link otwiera aplikację, nie przeglądarkę.

4. **Backward compatibility.** Backend REST API z V2 pozostaje IDENTYCZNY.
   Nie zmieniamy endpointów, formatów odpowiedzi ani schematu SQLite.

---

## Stos Technologiczny

### Serwis (connectd)

| Obszar | Wybór | Uzasadnienie |
|---|---|---|
| Język | Go | Istniejący kod, sprawdzony |
| Baza | modernc.org/sqlite | Pure Go, bez CGO, działa |
| Tray | getlantern/systray (pozostaje) | Tylko indicator że serwis działa |
| Auto-start | launchd (macOS) / Windows Service | goreleaser + własny skrypt |
| Komunikacja | REST HTTP na 127.0.0.1 | Istniejący API, bez zmian |
| Build | CGO_ENABLED=0 | Windows service, prostszy build |

### Desktop UI (connect-ui)

| Obszar | Wybór | Uzasadnienie |
|---|---|---|
| Shell | Tauri 2.x | Mały binary, native, cross-platform |
| Język shell | Rust | Wymagany przez Tauri |
| Frontend | HTML/CSS/JS (istniejący) | Zero przepisywania, ten sam kod |
| Custom protocol | Tauri `deep-link` plugin | connect://join/{code} |
| File dialogs | Tauri `dialog` plugin | Native open/save |
| Clipboard | Tauri `clipboard` plugin | Działa bez HTTPS |
| Notifications | Tauri `notification` plugin | Systemowe notyfikacje |
| Sidecar | connectd jako zasób Tauri | Auto-start przy uruchomieniu UI |
| Rozmiar | ~3–5 MB + connectd ~12 MB | Razem ~17 MB |

### Frontend UI kod

**Nie przepisujemy UI.** Istniejące `web/static/`:
- `index.html` → lekko zmodyfikowany (usunąć header z adresem, niepotrzebny)
- `style.css` → ten sam, drobne poprawki dla natywnego okna
- `app.js` → ten sam, API fetch na `http://127.0.0.1:{PORT}`
- `join.html` → zastąpiony natywnym oknem parowania (custom protocol handler)

---

## Service connectd

### Uruchomienie

```
connectd [flags]

Flags:
  --port      TCP port (default: 47810)
  --data-dir  Ścieżka do danych (default: OS app-data)
  --daemon    Uruchom w tle (fork/detach, Unix-only)
  --install   Zainstaluj jako service systemowy
  --uninstall  Odinstaluj service
  --version   Wersja
```

### Auto-start i service management

#### macOS (launchd)

Plik `~/Library/LaunchAgents/local.connect.connectd.plist`:
- `KeepAlive` = true
- `RunAtLoad` = true
- `WatchPaths` na katalog danych (opcjonalnie)

#### Windows (Windows Service)

Użycie `golang.org/x/sys/windows/svc`:
- `connectd --install` → `sc create connectd …`
- `connectd --uninstall` → `sc delete connectd`
- Uruchomienie jako `SERVICE_WIN32_OWN_PROCESS`

### Backend API

**IDENTYCZNE Z V2.** Żadnych zmian w:
- `GET /api/workspaces`
- `POST /api/invitations`
- `POST /api/workspaces/pair`
- `POST /sync/pair`
- `POST /sync/workspaces/{id}/operations`
- CRUD folderów i artefaktów

### Tray indicator (opcjonalnie)

Serwis może mieć ikonę w tray:
- "Connect service is running"
- "Open connect-ui" → uruchamia aplikację desktopową
- "Quit service"
- Używa istniejącego `getlantern/systray`

---

## Desktop UI (Tauri)

### Struktura projektu

```
connect/
  cmd/connectd/               ← serwis Go
  internal/                   ← istniejący kod backendu
  ui/                         ← nowy katalog
    connect-ui/               ← projekt Tauri
      src/                    ← frontend (przeniesione web/static/)
        index.html
        style.css
        app.js
      src-tauri/              ← Rust + konfiguracja
        src/main.rs
        Cargo.toml
        tauri.conf.json
        icons/
        capabilities/
      package.json            ← npm deps (tylko dev)
```

### Custom protocol: `connect://`

Rejestracja w systemie:
- **macOS:** `Info.plist` → `CFBundleURLTypes` z scheme `connect`
- **Windows:** Rejestr `HKEY_CLASSES_ROOT\connect`

Obsługa w Tauri:
```rust
// main.rs — deep-link plugin
tauri::Builder::default()
    .plugin(tauri_plugin_deep_link::init())
    .invoke_handler(tauri::generate_handler![handle_deep_link])
```

Flow:
1. Użytkownik klika `connect://join/abc123` w dowolnej aplikacji
2. System otwiera connect-ui
3. Tauri interceptuje URL, parsuje kod `abc123`
4. Aplikacja pokazuje okno dialogowe "Dołączasz do workspace" z prefilled URL
5. Użytkownik potwierdza → użytkownik wkleja do pola parowania (albo auto-start pair)

### Komunikacja z serwisem

UI łączy się przez `fetch("http://127.0.0.1:{PORT}/api/...")`.

```javascript
// app.js — przykład
const API_BASE = "http://127.0.0.1:47810";

async function api(path, options) {
  const response = await fetch(API_BASE + path, options);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || response.statusText);
  }
  return response.json();
}
```

Port może być:
- Stały (default 47810)
- Odczytany z Tauri `invoke('get_service_port')` który czyści plik `.connectd.port` w app-data

### Natywne integracje

| Funkcja | V2 (browser) | V3 (Tauri) |
|---|---|---|
| Otwieranie pliku | `<input type="file">` | `dialog.open({ multiple: true })` |
| Zapis pliku | `<a download>` | `dialog.save()` + `fs.writeTextFile()` |
| Schowek | `navigator.clipboard` | `clipboard.writeText()` |
| Notyfikacje | `Notification API` | `notification.send()` |
| Drag & drop | HTML5 DnD (ograniczony) | Tauri `drag-drop` event (full OS) |
| Link parowania | Kopiuj/wklej HTTP URL | `connect://join/{code}` deep link |
| Otwarcie linku | `window.open` / `<a href>` | `shell.open` |

### Okno aplikacji

```json
// tauri.conf.json
{
  "app": {
    "windows": [
      {
        "title": "Connect",
        "width": 1200,
        "height": 800,
        "minWidth": 800,
        "minHeight": 600,
        "center": true,
        "resizable": true,
        "decorations": true
      }
    ]
  }
}
```

- Standardowe natywne dekoracje okna (title bar, przyciski min/max/close)
- Zapamiętanie rozmiaru i pozycji między uruchomieniami
- Tryb ciemny: `@media (prefers-color-scheme: dark)`

---

## Migracja V2 → V3

### Faza 1: Wydzielenie serwisu (connectd)

**Pliki:**
- `cmd/connectd/main.go` — nowy, fork z `cmd/connect/main.go` bez UI
- `internal/service/service.go` — zarządzanie service (install/uninstall/run)
- `internal/service/launchd_darwin.go` — template launchd plist
- `internal/service/windows.go` — Windows Service wrapper

**Zmiany w istniejącym kodzie:**
- `internal/httpapi/httpapi.go` — brak zmian (API pozostaje)
- `internal/storage/` — brak zmian
- `cmd/connect/main.go` — pozostaje jako wrapper (deprecated)

**Testy:**
- `go test ./internal/service/...`
- Ręczne: `connectd --daemon`, sprawdzenie API

### Faza 2: Projekt Tauri

**Pliki:**
- `ui/connect-ui/src-tauri/` — konfiguracja Rust + Tauri
- `ui/connect-ui/src/` — kopi istniejących web/static/*
- `ui/connect-ui/package.json` — dev dependencies
- `ui/connect-ui/src-tauri/tauri.conf.json` — konfiguracja

**Kroki:**
1. `npm create tauri-app@latest` w `ui/`
2. Kopiować `web/static/*` do `ui/connect-ui/src/`
3. Skonfigurować sidecar: `connectd` jako external binary
4. Dodać pluginy: `deep-link`, `dialog`, `clipboard`, `notification`, `shell`
5. Zmodyfikować `app.js`: API base na `http://127.0.0.1:47810`
6. Zmodyfikować `index.html`: marker `<meta>` dla Tauri, usunąć niepotrzebny header

### Faza 3: Custom protocol + deep linking

**Pliki:**
- `ui/connect-ui/src-tauri/src/main.rs` — deep-link handler
- `ui/connect-ui/src/app.js` — obsługa `connect://join/{code}`
- `ui/connect-ui/src-tauri/tauri.conf.json` — rejestracja scheme

**Działanie:**
1. System rejestruje `connect://` scheme (Tauri deep-link plugin)
2. Kliknięcie w link → aplikacja uruchomiona/aktywowana
3. Rust handler parsuje URL, wysyła event do frontendu
4. Frontend pokazuje dialog parowania z prefilled invitation URL

### Faza 4: Packaging i instalator

**macOS:**
- DMG z `create-dmg` lub `appdmg`
- Kod podpisu: `codesign --deep`
- Notary: `xcrun notarytool`

**Windows:**
- NSIS lub WiX installer
- Podpis: `signtool`

**CI/CD:**
- GitHub Actions build + sign + notarize
- `goreleaser` dla serwisu

---

## Niezmienne (względem backendu V2)

- REST API endpointy i formaty odpowiedzi
- SQLite schema i zapytania
- Pairing flow (invitation → join → accept)
- Operation sync model (push operations)
- File storage (bytes w filesDir, metadata w DB)
- Identity management (UUID + hostname)
- UUID generation (`idgen`)

---

## Zmienne (nowe w V3)

- **Custom protocol** `connect://` — rejestracja w OS
- **Service auto-start** — launchd / Windows Service
- **Tauri shell** — native window zamiast przeglądarki
- **Native integrations** — file dialogs, clipboard, notifications
- **Sidecar model** — serwis uruchamiany przez Tauri
- **Build pipeline** — Tauri CLI + Github Actions

---

## Kryteria Sukcesu V3

- Aplikacja działa w natywnym oknie, nie w przeglądarce
- Link `connect://join/{code}` otwiera aplikację, stronę w przeglądarce
- Parowanie działa w obie strony (host → guest, guest → host)
- Synchronizacja operacji push działa
- Pliki można wybrać przez natywne okno dialogowe
- UI można przeciągnąć z systemu (drag & drop plików)
- Service uruchamia się automatycznie przy starcie systemu
- Binary UI + service łącznie < 25 MB
- Aplikacja działa na macOS Sequoia+ i Windows 11+
