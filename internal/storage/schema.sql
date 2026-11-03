CREATE TABLE IF NOT EXISTS paired_workspaces (
    id TEXT PRIMARY KEY,
    local_device_id TEXT NOT NULL,
    remote_device_id TEXT NOT NULL UNIQUE,
    remote_name TEXT NOT NULL,
    remote_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS invitations (
    code TEXT PRIMARY KEY,
    created_at TEXT NOT NULL,
    used_at TEXT
);

CREATE TABLE IF NOT EXISTS folders (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES paired_workspaces(id),
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES paired_workspaces(id),
    folder_id TEXT REFERENCES folders(id),
    type TEXT NOT NULL CHECK (type IN ('text', 'file')),
    title TEXT NOT NULL,
    content TEXT,
    filename TEXT,
    blob_path TEXT,
    checksum TEXT,
    size INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT,
    status TEXT NOT NULL DEFAULT 'ready',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS operations (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES paired_workspaces(id),
    actor_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    kind TEXT NOT NULL,
    payload TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(workspace_id, actor_id, sequence)
);

CREATE TABLE IF NOT EXISTS outbox (
    operation_id TEXT PRIMARY KEY REFERENCES operations(id),
    delivered_at TEXT,
    attempts INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sync_cursors (
    workspace_id TEXT NOT NULL REFERENCES paired_workspaces(id),
    actor_id TEXT NOT NULL,
    sequence INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(workspace_id, actor_id)
);

CREATE INDEX IF NOT EXISTS idx_folders_workspace ON folders(workspace_id, deleted_at, sort_order);
CREATE INDEX IF NOT EXISTS idx_artifacts_workspace ON artifacts(workspace_id, folder_id, deleted_at, updated_at);
CREATE INDEX IF NOT EXISTS idx_operations_workspace ON operations(workspace_id, created_at);
