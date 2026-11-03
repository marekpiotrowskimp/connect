# Connect Current Product Notes

This file replaces the obsolete V2 Host Session / Join Session plan. That WebSocket session architecture is no longer the active implementation.

## Product Model

- Connect is for LAN-only exchange of technical artifacts between exactly paired computers.
- A paired workspace is persistent and belongs to one pair of devices.
- The workspace contains folders and artifacts.
- Artifacts are text snippets or files, not chat messages.
- The UI is a workspace/desktop, not a conversation timeline.

## Pairing Flow

1. The inviting computer calls `POST /api/invitations`.
2. Connect creates a `/join/{code}` invitation URL.
3. The joining computer pastes that URL into its own local Connect UI.
4. The joining Connect calls `POST /api/workspaces/pair`.
5. The inviting Connect accepts through `POST /sync/pair`.
6. Both sides store the same workspace ID in `paired_workspaces`.

## Workspace Sync

- Local mutations create `operations` rows.
- Operations are sent to the paired computer through `POST /sync/workspaces/{id}/operations`.
- Received operations are applied idempotently by operation ID.
- Current known limitation: file metadata syncs, but file bytes are not yet transferred by the operation sync path.
- Current known limitation: sync is push-triggered by local mutations and does not yet perform robust periodic pull/retry reconciliation.

## Required UX

- Dashboard cards for paired workspaces.
- Detail screen with folders, toolbar, artifact cards, and inspector.
- Folder click navigation.
- Artifact drag/drop to folder.
- Inspector move control.
- Search, type filter, age filter, sort, grid/list toggle.
- Toasts for completed local UI actions.

## Do Not Reintroduce

- V1 peer list architecture.
- `/peer/*` APIs.
- SSE event stream.
- Old chat/message model.
- Obsolete Host Session / Join Session WebSocket transport.
