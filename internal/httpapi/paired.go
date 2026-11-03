package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connect/internal/idgen"
	"connect/internal/storage"
)

type pairView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Items  int    `json:"items"`
	Stale  int    `json:"stale_items"`
}
type folderView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ItemCount  int    `json:"item_count"`
	StaleCount int    `json:"stale_count"`
}
type artifactView struct {
	ID        string `json:"id"`
	FolderID  string `json:"folder_id,omitempty"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
type pairRequest struct {
	WorkspaceID    string `json:"workspace_id"`
	InvitationCode string `json:"invitation_code"`
	Device         Device `json:"device"`
	URL            string `json:"url"`
}

func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	code := idgen.New()
	if err := s.db.CreateInvitation(r.Context(), code); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	self := s.self()
	url := fmt.Sprintf("http://%s:%d/join/%s", self.IP, self.Port, code)
	writeJSON(w, http.StatusOK, map[string]string{"code": code, "url": url})
}

func parseInvitation(raw string) (*url.URL, string, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Trim(trimmed, "<>")
	trimmed = strings.ReplaceAll(trimmed, "\u00a0", " ")
	if trimmed == "" {
		return nil, "", errors.New("valid Connect invitation link is required")
	}
	hasScheme := strings.Contains(trimmed, "://")
	if !hasScheme {
		trimmed = "http://" + trimmed
	}
	trimmed = strings.TrimRight(trimmed, "/ \t\n")
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, "", errors.New("valid Connect invitation link is required")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", errors.New("valid Connect invitation link is required")
	}
	if parsed.Host == "" {
		return nil, "", errors.New("valid Connect invitation link is required")
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	code := ""
	for i, segment := range segments {
		if strings.EqualFold(segment, "join") {
			if i+1 < len(segments) {
				code = strings.TrimSpace(segments[i+1])
			}
			break
		}
	}
	if code == "" {
		return nil, "", errors.New("this is not a Connect invitation link")
	}
	return parsed, code, nil
}

func pairToView(p storage.PairedWorkspace) pairView {
	return pairView{p.ID, p.RemoteName, p.Status, p.ItemCount, p.StaleCount}
}
func folderToView(f storage.Folder) folderView {
	return folderView{f.ID, f.Name, f.ItemCount, f.StaleCount}
}
func artifactToView(a storage.Artifact) artifactView {
	return artifactView{a.ID, a.FolderID, a.Type, a.Title, a.Content, a.Filename, a.Size, a.MimeType, a.Status, a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339)}
}

func (s *Server) handleListPairs(w http.ResponseWriter, r *http.Request) {
	ps, e := s.db.ListPairs(r.Context())
	if e != nil {
		writeError(w, 500, e)
		return
	}
	out := make([]pairView, 0, len(ps))
	for _, p := range ps {
		out = append(out, pairToView(p))
	}
	writeJSON(w, 200, out)
}

func (s *Server) handlePairWorkspace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	remoteURL, invitationCode, perr := parseInvitation(req.URL)
	if perr != nil {
		writeError(w, http.StatusBadRequest, perr)
		return
	}
	remoteURL.Path = ""
	remoteURL.RawQuery = ""
	workspaceID := idgen.New()
	self := s.self()
	localURL := fmt.Sprintf("http://%s:%d", self.IP, self.Port)
	body, _ := json.Marshal(pairRequest{WorkspaceID: workspaceID, InvitationCode: invitationCode, Device: self, URL: localURL})
	response, err := http.Post(remoteURL.String()+"/sync/pair", "application/json", strings.NewReader(string(body)))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		writeError(w, http.StatusBadGateway, errors.New("remote Connect service rejected pairing"))
		return
	}
	var remote Device
	if err := json.NewDecoder(response.Body).Decode(&remote); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	pair := storage.PairedWorkspace{ID: workspaceID, LocalDeviceID: self.ID, RemoteDeviceID: remote.ID, RemoteName: remote.Name, RemoteURL: remoteURL.String(), Status: "connected"}
	if err := s.db.CreatePair(r.Context(), pair); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, pairToView(pair))
}

func (s *Server) handlePairAccept(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WorkspaceID == "" || req.InvitationCode == "" || req.Device.ID == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, errors.New("invalid pair request"))
		return
	}
	if err := s.db.UseInvitation(r.Context(), req.InvitationCode); err != nil {
		writeError(w, http.StatusForbidden, errors.New("invitation is invalid or already used"))
		return
	}
	self := s.self()
	pair := storage.PairedWorkspace{ID: req.WorkspaceID, LocalDeviceID: self.ID, RemoteDeviceID: req.Device.ID, RemoteName: req.Device.Name, RemoteURL: req.URL, Status: "connected"}
	if err := s.db.CreatePair(r.Context(), pair); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, self)
}

func (s *Server) handleSyncOperations(w http.ResponseWriter, r *http.Request) {
	var operations []storage.Operation
	if err := json.NewDecoder(r.Body).Decode(&operations); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspaceID := r.PathValue("id")
	for _, operation := range operations {
		if operation.WorkspaceID != workspaceID {
			writeError(w, http.StatusBadRequest, errors.New("workspace mismatch"))
			return
		}
		seen, err := s.db.HasOperation(r.Context(), operation.ID)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		if seen {
			continue
		}
		if err := s.applyOperation(r, operation); err != nil {
			writeError(w, 400, err)
			return
		}
		if err := s.db.SaveRemoteOperation(r.Context(), operation); err != nil {
			writeError(w, 500, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) applyOperation(r *http.Request, operation storage.Operation) error {
	switch operation.Kind {
	case "folder.create":
		var folder storage.Folder
		if err := json.Unmarshal(operation.Payload, &folder); err != nil {
			return err
		}
		return s.db.InsertFolder(r.Context(), folder)
	case "artifact.create":
		var artifact storage.Artifact
		if err := json.Unmarshal(operation.Payload, &artifact); err != nil {
			return err
		}
		if artifact.Type == "file" {
			artifact.Status = "awaiting-file"
			artifact.BlobPath = ""
		}
		return s.db.InsertArtifact(r.Context(), artifact)
	case "artifact.move":
		var move struct {
			ID       string `json:"id"`
			FolderID string `json:"folder_id"`
		}
		if err := json.Unmarshal(operation.Payload, &move); err != nil {
			return err
		}
		return s.db.MoveArtifact(r.Context(), operation.WorkspaceID, move.ID, move.FolderID)
	case "artifact.delete":
		var deletion struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(operation.Payload, &deletion); err != nil {
			return err
		}
		return s.db.DeleteArtifact(r.Context(), operation.WorkspaceID, deletion.ID)
	default:
		return errors.New("unsupported operation")
	}
}

func (s *Server) syncWorkspace(workspaceID string) {
	pair, err := s.db.GetPair(context.Background(), workspaceID)
	if err != nil || pair.RemoteURL == "" {
		return
	}
	operations, err := s.db.ListOperations(context.Background(), workspaceID)
	if err != nil || len(operations) == 0 {
		return
	}
	body, err := json.Marshal(operations)
	if err != nil {
		return
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(pair.RemoteURL, "/")+"/sync/workspaces/"+workspaceID+"/operations", strings.NewReader(string(body)))
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		_ = s.db.SetPairStatus(context.Background(), workspaceID, "offline")
		return
	}
	response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		_ = s.db.SetPairStatus(context.Background(), workspaceID, "connected")
	}
}
func (s *Server) handleGetPair(w http.ResponseWriter, r *http.Request) {
	p, e := s.db.GetPair(r.Context(), r.PathValue("id"))
	if e != nil {
		writeError(w, 404, e)
		return
	}
	writeJSON(w, 200, pairToView(p))
}
func (s *Server) handleRemovePair(w http.ResponseWriter, r *http.Request) {
	if e := s.db.DeletePair(r.Context(), r.PathValue("id")); e != nil {
		writeError(w, 500, e)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	fs, e := s.db.ListFolders(r.Context(), r.PathValue("id"))
	if e != nil {
		writeError(w, 500, e)
		return
	}
	out := make([]folderView, 0, len(fs))
	for _, f := range fs {
		out = append(out, folderToView(f))
	}
	writeJSON(w, 200, out)
}
func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Name string `json:"name"`
	}
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil {
		writeError(w, 400, e)
		return
	}
	f := storage.Folder{ID: idgen.New(), WorkspaceID: r.PathValue("id"), Name: strings.TrimSpace(x.Name)}
	if f.Name == "" {
		f.Name = "New folder"
	}
	if e := s.db.InsertFolder(r.Context(), f); e != nil {
		writeError(w, 500, e)
		return
	}
	s.record(r, f.WorkspaceID, "folder.create", f)
	writeJSON(w, 200, folderToView(f))
}
func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "all"
	}
	as, e := s.db.ListArtifacts(r.Context(), r.PathValue("id"), folder)
	if e != nil {
		writeError(w, 500, e)
		return
	}
	out := make([]artifactView, 0, len(as))
	for _, a := range as {
		out = append(out, artifactToView(a))
	}
	writeJSON(w, 200, out)
}
func (s *Server) handleCreateText(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		FolderID string `json:"folder_id"`
	}
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil {
		writeError(w, 400, e)
		return
	}
	x.Content = strings.TrimSpace(x.Content)
	if x.Content == "" {
		writeError(w, 400, errors.New("content is required"))
		return
	}
	if x.Title == "" {
		x.Title = strings.Split(x.Content, "\n")[0]
	}
	a := storage.Artifact{ID: idgen.New(), WorkspaceID: r.PathValue("id"), FolderID: x.FolderID, Type: "text", Title: x.Title, Content: x.Content, Size: int64(len(x.Content))}
	if e := s.db.InsertArtifact(r.Context(), a); e != nil {
		writeError(w, 500, e)
		return
	}
	s.record(r, a.WorkspaceID, "artifact.create", a)
	writeJSON(w, 200, artifactToView(a))
}
func (s *Server) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseMultipartForm(32 << 20); e != nil {
		writeError(w, 400, e)
		return
	}
	in, h, e := r.FormFile("file")
	if e != nil {
		writeError(w, 400, e)
		return
	}
	defer in.Close()
	id := idgen.New()
	name := filepath.Base(h.Filename)
	dir := filepath.Join(s.filesDir, "blobs", r.PathValue("id"))
	if e = os.MkdirAll(dir, 0755); e != nil {
		writeError(w, 500, e)
		return
	}
	path := filepath.Join(dir, id)
	out, e := os.Create(path)
	if e != nil {
		writeError(w, 500, e)
		return
	}
	hash := sha256.New()
	size, e := io.Copy(io.MultiWriter(out, hash), in)
	closeErr := out.Close()
	if e != nil || closeErr != nil {
		_ = os.Remove(path)
		writeError(w, 500, errors.Join(e, closeErr))
		return
	}
	a := storage.Artifact{ID: id, WorkspaceID: r.PathValue("id"), FolderID: r.FormValue("folder_id"), Type: "file", Title: name, Filename: name, BlobPath: path, Checksum: hex.EncodeToString(hash.Sum(nil)), Size: size, MimeType: mime.TypeByExtension(filepath.Ext(name))}
	if e := s.db.InsertArtifact(r.Context(), a); e != nil {
		_ = os.Remove(path)
		writeError(w, 500, e)
		return
	}
	s.record(r, a.WorkspaceID, "artifact.create", a)
	writeJSON(w, 200, artifactToView(a))
}
func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	a, e := s.db.GetArtifact(r.Context(), r.PathValue("id"), r.PathValue("itemID"))
	if e != nil || a.Type != "file" {
		writeError(w, 404, errors.New("file not found"))
		return
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Filename}))
	http.ServeFile(w, r, a.BlobPath)
}
func (s *Server) handleMoveArtifact(w http.ResponseWriter, r *http.Request) {
	var x struct {
		FolderID string `json:"folder_id"`
	}
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil {
		writeError(w, 400, e)
		return
	}
	id := r.PathValue("id")
	if e := s.db.MoveArtifact(r.Context(), id, r.PathValue("itemID"), x.FolderID); e != nil {
		writeError(w, 404, e)
		return
	}
	a, _ := s.db.GetArtifact(r.Context(), id, r.PathValue("itemID"))
	s.record(r, id, "artifact.move", map[string]string{"id": a.ID, "folder_id": x.FolderID})
	writeJSON(w, 200, artifactToView(a))
}
func (s *Server) handleDeleteArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item := r.PathValue("itemID")
	if e := s.db.DeleteArtifact(r.Context(), id, item); e != nil {
		writeError(w, 404, e)
		return
	}
	s.record(r, id, "artifact.delete", map[string]string{"id": item})
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) record(r *http.Request, wid, kind string, payload any) {
	b, e := json.Marshal(payload)
	if e != nil {
		return
	}
	_ = s.db.SaveOperation(r.Context(), storage.Operation{ID: idgen.New(), WorkspaceID: wid, ActorID: s.self().ID, Sequence: s.sequence.Add(1), Kind: kind, Payload: b, CreatedAt: time.Now().UTC()})
	go s.syncWorkspace(wid)
}
