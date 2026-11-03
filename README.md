# Connect V3

Connect is migrating to V3 — a native desktop app with a separate Go backend service.

- **V3 spec:** `PROJECT_V3.md`
- **Current state (V2):** browser-based web UI, single binary, invitation-based paired workspaces
- **Target (V3):** Tauri desktop app + connectd daemon, `connect://` protocol, native integrations

## Project Structure

```
cmd/connect/          current V2 entrypoint (will become a wrapper)
cmd/connectd/         V3 service binary (planned)
internal/             shared Go packages (storage, httpapi, identity, idgen)
ui/                   V3 Tauri desktop app (planned)
web/static/           current V2 frontend assets (will move into ui/)
PROJECT_V3.md         full V3 architecture specification
```

## Quick Start (V2)

```
make run
```

## V3 Migration Status

| Phase | Status |
|---|---|
| 1. Extract connectd service | 📋 Planned |
| 2. Create Tauri UI project | 📋 Planned |
| 3. Custom protocol + deep linking | 📋 Planned |
| 4. Packaging, installer, CI/CD | 📋 Planned |

## Build

- `make build` — builds all V2 binaries
- `make test` — runs Go tests
- `make package` — creates macOS .app bundle (V2 only)
