package storage

import (
	"context"
	"encoding/json"
	"time"
)

type PairedWorkspace struct {
	ID, LocalDeviceID, RemoteDeviceID, RemoteName, RemoteURL, Status string
	CreatedAt, UpdatedAt                                             time.Time
	ItemCount, StaleCount                                            int
}
type Folder struct {
	ID, WorkspaceID, Name string
	SortOrder             int
	CreatedAt, UpdatedAt  time.Time
	ItemCount, StaleCount int
}
type Artifact struct {
	ID, WorkspaceID, FolderID, Type, Title, Content, Filename, BlobPath, Checksum, MimeType, Status string
	Size                                                                                            int64
	CreatedAt, UpdatedAt                                                                            time.Time
}
type Operation struct {
	ID, WorkspaceID, ActorID, Kind string
	Sequence                       int64
	Payload                        json.RawMessage
	CreatedAt                      time.Time
}

func (db *DB) CreateInvitation(ctx context.Context, code string) error {
	_, err := db.sql.ExecContext(ctx, `INSERT INTO invitations (code, created_at) VALUES (?, ?)`, code, formatTime(time.Now()))
	return err
}

func (db *DB) UseInvitation(ctx context.Context, code string) error {
	result, err := db.sql.ExecContext(ctx, `UPDATE invitations SET used_at=? WHERE code=? AND used_at IS NULL`, formatTime(time.Now()), code)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) CreatePair(ctx context.Context, p PairedWorkspace) error {
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = "connected"
	}
	_, err := db.sql.ExecContext(ctx, `INSERT INTO paired_workspaces (id, local_device_id, remote_device_id, remote_name, remote_url, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(remote_device_id) DO UPDATE SET remote_name=excluded.remote_name,remote_url=excluded.remote_url,status=excluded.status,updated_at=excluded.updated_at`, p.ID, p.LocalDeviceID, p.RemoteDeviceID, p.RemoteName, p.RemoteURL, p.Status, formatTime(p.CreatedAt), formatTime(p.UpdatedAt))
	return err
}
func (db *DB) ListPairs(ctx context.Context) ([]PairedWorkspace, error) {
	return db.listPairs(ctx, `SELECT p.id,p.local_device_id,p.remote_device_id,p.remote_name,p.remote_url,p.status,p.created_at,p.updated_at,(SELECT COUNT(*) FROM artifacts a WHERE a.workspace_id=p.id AND a.deleted_at IS NULL),(SELECT COUNT(*) FROM artifacts a WHERE a.workspace_id=p.id AND a.deleted_at IS NULL AND julianday('now')-julianday(a.updated_at)>=30) FROM paired_workspaces p ORDER BY p.remote_name`)
}
func (db *DB) GetPair(ctx context.Context, id string) (PairedWorkspace, error) {
	ps, e := db.listPairs(ctx, `SELECT p.id,p.local_device_id,p.remote_device_id,p.remote_name,p.remote_url,p.status,p.created_at,p.updated_at,(SELECT COUNT(*) FROM artifacts a WHERE a.workspace_id=p.id AND a.deleted_at IS NULL),(SELECT COUNT(*) FROM artifacts a WHERE a.workspace_id=p.id AND a.deleted_at IS NULL AND julianday('now')-julianday(a.updated_at)>=30) FROM paired_workspaces p WHERE p.id=?`, id)
	if e != nil {
		return PairedWorkspace{}, e
	}
	if len(ps) == 0 {
		return PairedWorkspace{}, ErrNotFound
	}
	return ps[0], nil
}
func (db *DB) listPairs(ctx context.Context, q string, args ...any) ([]PairedWorkspace, error) {
	rows, e := db.sql.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []PairedWorkspace
	for rows.Next() {
		var p PairedWorkspace
		var c, u string
		if e := rows.Scan(&p.ID, &p.LocalDeviceID, &p.RemoteDeviceID, &p.RemoteName, &p.RemoteURL, &p.Status, &c, &u, &p.ItemCount, &p.StaleCount); e != nil {
			return nil, e
		}
		p.CreatedAt = parseTime(c)
		p.UpdatedAt = parseTime(u)
		out = append(out, p)
	}
	return out, rows.Err()
}
func (db *DB) SetPairStatus(ctx context.Context, id, status string) error {
	_, e := db.sql.ExecContext(ctx, `UPDATE paired_workspaces SET status=?,updated_at=? WHERE id=?`, status, formatTime(time.Now()), id)
	return e
}
func (db *DB) DeletePair(ctx context.Context, id string) error {
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		`DELETE FROM outbox WHERE operation_id IN (SELECT id FROM operations WHERE workspace_id=?)`,
		`DELETE FROM operations WHERE workspace_id=?`,
		`DELETE FROM artifacts WHERE workspace_id=?`,
		`DELETE FROM folders WHERE workspace_id=?`,
		`DELETE FROM sync_cursors WHERE workspace_id=?`,
		`DELETE FROM paired_workspaces WHERE id=?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) InsertFolder(ctx context.Context, f Folder) error {
	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
	_, e := db.sql.ExecContext(ctx, `INSERT INTO folders (id,workspace_id,name,sort_order,created_at,updated_at) VALUES (?,?,?,?,?,?)`, f.ID, f.WorkspaceID, f.Name, f.SortOrder, formatTime(f.CreatedAt), formatTime(f.UpdatedAt))
	return e
}
func (db *DB) ListFolders(ctx context.Context, w string) ([]Folder, error) {
	rows, e := db.sql.QueryContext(ctx, `SELECT f.id,f.workspace_id,f.name,f.sort_order,f.created_at,f.updated_at,COUNT(a.id),SUM(CASE WHEN a.id IS NOT NULL AND julianday('now')-julianday(a.updated_at)>=30 THEN 1 ELSE 0 END) FROM folders f LEFT JOIN artifacts a ON a.folder_id=f.id AND a.deleted_at IS NULL WHERE f.workspace_id=? AND f.deleted_at IS NULL GROUP BY f.id ORDER BY f.sort_order,f.name`, w)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		var f Folder
		var c, u string
		if e := rows.Scan(&f.ID, &f.WorkspaceID, &f.Name, &f.SortOrder, &c, &u, &f.ItemCount, &f.StaleCount); e != nil {
			return nil, e
		}
		f.CreatedAt = parseTime(c)
		f.UpdatedAt = parseTime(u)
		out = append(out, f)
	}
	return out, rows.Err()
}
func (db *DB) InsertArtifact(ctx context.Context, a Artifact) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	if a.Status == "" {
		a.Status = "ready"
	}
	_, e := db.sql.ExecContext(ctx, `INSERT INTO artifacts (id,workspace_id,folder_id,type,title,content,filename,blob_path,checksum,size,mime_type,status,created_at,updated_at) VALUES (?,?,NULLIF(?,''),?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.WorkspaceID, a.FolderID, a.Type, a.Title, a.Content, a.Filename, a.BlobPath, a.Checksum, a.Size, a.MimeType, a.Status, formatTime(a.CreatedAt), formatTime(a.UpdatedAt))
	return e
}
func (db *DB) ListArtifacts(ctx context.Context, w, folder string) ([]Artifact, error) {
	q := `SELECT id,workspace_id,COALESCE(folder_id,''),type,title,COALESCE(content,''),COALESCE(filename,''),COALESCE(blob_path,''),COALESCE(checksum,''),size,COALESCE(mime_type,''),status,created_at,updated_at FROM artifacts WHERE workspace_id=? AND deleted_at IS NULL`
	args := []any{w}
	if folder != "all" {
		q += ` AND COALESCE(folder_id,'')=?`
		args = append(args, folder)
	}
	q += ` ORDER BY updated_at DESC`
	rows, e := db.sql.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		var c, u string
		if e := rows.Scan(&a.ID, &a.WorkspaceID, &a.FolderID, &a.Type, &a.Title, &a.Content, &a.Filename, &a.BlobPath, &a.Checksum, &a.Size, &a.MimeType, &a.Status, &c, &u); e != nil {
			return nil, e
		}
		a.CreatedAt = parseTime(c)
		a.UpdatedAt = parseTime(u)
		out = append(out, a)
	}
	return out, rows.Err()
}
func (db *DB) GetArtifact(ctx context.Context, w, id string) (Artifact, error) {
	as, e := db.ListArtifacts(ctx, w, "all")
	if e != nil {
		return Artifact{}, e
	}
	for _, a := range as {
		if a.ID == id {
			return a, nil
		}
	}
	return Artifact{}, ErrNotFound
}
func (db *DB) MoveArtifact(ctx context.Context, w, id, folder string) error {
	r, e := db.sql.ExecContext(ctx, `UPDATE artifacts SET folder_id=NULLIF(?,''),updated_at=? WHERE id=? AND workspace_id=? AND deleted_at IS NULL`, folder, formatTime(time.Now()), id, w)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (db *DB) DeleteArtifact(ctx context.Context, w, id string) error {
	r, e := db.sql.ExecContext(ctx, `UPDATE artifacts SET deleted_at=?,updated_at=? WHERE id=? AND workspace_id=? AND deleted_at IS NULL`, formatTime(time.Now()), formatTime(time.Now()), id, w)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (db *DB) SaveOperation(ctx context.Context, o Operation) error {
	_, e := db.sql.ExecContext(ctx, `INSERT INTO operations (id,workspace_id,actor_id,sequence,kind,payload,created_at) VALUES (?,?,?,?,?,?,?)`, o.ID, o.WorkspaceID, o.ActorID, o.Sequence, o.Kind, string(o.Payload), formatTime(o.CreatedAt))
	if e != nil {
		return e
	}
	_, e = db.sql.ExecContext(ctx, `INSERT INTO outbox (operation_id) VALUES (?)`, o.ID)
	return e
}
func (db *DB) ListOperations(ctx context.Context, workspaceID string) ([]Operation, error) {
	rows, err := db.sql.QueryContext(ctx, `SELECT id,workspace_id,actor_id,sequence,kind,payload,created_at FROM operations WHERE workspace_id=? ORDER BY created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Operation
	for rows.Next() {
		var o Operation
		var payload, created string
		if err := rows.Scan(&o.ID, &o.WorkspaceID, &o.ActorID, &o.Sequence, &o.Kind, &payload, &created); err != nil {
			return nil, err
		}
		o.Payload = json.RawMessage(payload)
		o.CreatedAt = parseTime(created)
		out = append(out, o)
	}
	return out, rows.Err()
}
func (db *DB) HasOperation(ctx context.Context, id string) (bool, error) {
	var found int
	err := db.sql.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM operations WHERE id=?)`, id).Scan(&found)
	return found != 0, err
}
func (db *DB) SaveRemoteOperation(ctx context.Context, o Operation) error {
	_, err := db.sql.ExecContext(ctx, `INSERT OR IGNORE INTO operations (id,workspace_id,actor_id,sequence,kind,payload,created_at) VALUES (?,?,?,?,?,?,?)`, o.ID, o.WorkspaceID, o.ActorID, o.Sequence, o.Kind, string(o.Payload), formatTime(o.CreatedAt))
	return err
}
func parseTime(s string) time.Time { t, _ := time.Parse(time.RFC3339Nano, s); return t }
