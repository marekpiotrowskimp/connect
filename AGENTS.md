# Connect Agent Rules

## Current Product State

- Connect is a LAN-only artifact sharing app for exactly paired computers.
- Current architecture is invitation-based paired workspaces, not the old peer list and not the old WebSocket session model.
- The current backend uses Go, embedded static HTML/CSS/JS, SQLite, and direct HTTP sync endpoints.
- Do not restore V1 `/peer/*`, SSE, peer lists, or old chat/message models.
- Do not restore the obsolete Host Session / Join Session / WebSocket transport model unless the user explicitly asks for a new architecture rewrite.

## Current Architecture

- Dashboard lists `paired_workspaces` through `GET /api/workspaces`.
- Pairing starts with `POST /api/invitations`, producing `/join/{code}`.
- The second computer pastes that invitation into local Connect, which calls `POST /api/workspaces/pair`.
- Peer pairing is accepted through `POST /sync/pair`.
- Workspace operations sync through `POST /sync/workspaces/{id}/operations`.
- Folders and artifacts are shared by common IDs inside one paired workspace.
- Text and file metadata are stored in SQLite; file bytes are stored under the app-data files directory.

## Non-Negotiable Working Rules

- Before editing, inspect the relevant current files and history if UX regression is involved.
- Change only the files required by the user's explicit request.
- Do not touch backend sync, schema, storage, or pairing while fixing UI unless the user explicitly asks for backend changes.
- Do not touch CSS if the existing CSS already supports the desired UI and the regression is in JS-generated markup.
- Do not remove working UX while fixing another feature.
- Prefer restoring the smallest missing behavior from the last known-good commit over inventing a new UI.
- Never rewrite architecture documents toward an obsolete model.
- Never commit unless the user explicitly asks.

## Current Product Direction (V3)

The project is migrating to V3 architecture:
- **Service (connectd):** separate Go daemon (launchd/Windows Service) with REST API on localhost.
- **Desktop UI (connect-ui):** Tauri native app communicating with connectd via HTTP.
- **Custom protocol:** `connect://join/{code}` instead of `http://host/join/{code}`.
- **No browser.** The UI runs in a native window, not a browser tab.
- Frontend code (HTML/CSS/JS) stays the same, just moves into Tauri `ui/` directory.
- Backend REST API endpoints stay identical to V2.
- Full spec in `PROJECT_V3.md`.

## Working Priority (V3 Migration)

1. **Phase 1:** Extract service binary (`cmd/connectd/`) — no UI changes.
2. **Phase 2:** Create Tauri project in `ui/` with existing web assets.
3. **Phase 3:** Custom protocol + deep linking.
4. **Phase 4:** Packaging, installer, CI/CD.

## Strict Rules

- Do NOT touch V2 frontend (`web/static/`) unless explicitly fixing a blocking bug for V3.
- Do NOT change backend REST API endpoints — V3 uses the identical API.
- Do NOT rewrite the frontend from scratch — reuse existing HTML/CSS/JS.
- Before any change, read `PROJECT_V3.md` and confirm the phase.
- Never commit unless the user explicitly asks.

## Required Checks

- For frontend-only changes (V2 bugfix): `node --check web/static/app.js`.
- For Go/backend changes: `go build ./...` and `go vet ./...`.
- For Tauri changes: `cd ui/connect-ui && cargo build`.
- Before reporting completion, inspect `git status --short` and confirm only intended files changed.
