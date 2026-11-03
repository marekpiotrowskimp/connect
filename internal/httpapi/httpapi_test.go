package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"connect/internal/httpapi"
	"connect/internal/storage"
)

func newServer(t *testing.T) (string, *storage.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(filepath.Join(dir, "connect.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	server := httpapi.New(fstest.MapFS{}, db, func() httpapi.Device {
		return httpapi.Device{ID: "self-id", Name: "Self", IP: "127.0.0.1", Port: 47810}
	}, dir)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go http.Serve(listener, server.Handler())
	t.Cleanup(func() { listener.Close() })
	return "http://" + listener.Addr().String(), db
}

func TestSelf(t *testing.T) {
	base, _ := newServer(t)
	resp, err := http.Get(base + "/api/self")
	if err != nil {
		t.Fatalf("GET /api/self: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dev httpapi.Device
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dev.ID != "self-id" {
		t.Fatalf("id = %q, want self-id", dev.ID)
	}
	if dev.Name != "Self" {
		t.Fatalf("name = %q, want Self", dev.Name)
	}
}

func TestListPairsEmpty(t *testing.T) {
	base, _ := newServer(t)
	resp, err := http.Get(base + "/api/workspaces")
	if err != nil {
		t.Fatalf("GET /api/workspaces: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var pairs []any
	if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pairs) != 0 {
		t.Fatalf("got %d pairs, want 0", len(pairs))
	}
}

func TestCreatePairViaSyncAccept(t *testing.T) {
	base, _ := newServer(t)

	// Create invitation
	inviteResp, err := http.Post(base+"/api/invitations", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/invitations: %v", err)
	}
	defer inviteResp.Body.Close()
	if inviteResp.StatusCode != http.StatusOK {
		t.Fatalf("invitation status = %d, want 200", inviteResp.StatusCode)
	}
	var invite struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(inviteResp.Body).Decode(&invite); err != nil {
		t.Fatalf("decode invitation: %v", err)
	}

	// Accept the invitation via POST /sync/pair (simulates the remote side)
	pairReq := map[string]any{
		"workspace_id":    "ws-test-1",
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote-id", Name: "Remote", IP: "192.168.1.2", Port: 47810},
		"url":             "http://192.168.1.2:47810",
	}
	body, _ := json.Marshal(pairReq)
	acceptResp, err := http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /sync/pair: %v", err)
	}
	defer acceptResp.Body.Close()
	if acceptResp.StatusCode != http.StatusOK {
		t.Fatalf("pair accept status = %d, want 200", acceptResp.StatusCode)
	}

	// List pairs — should include the new workspace
	listResp, err := http.Get(base + "/api/workspaces")
	if err != nil {
		t.Fatalf("GET /api/workspaces: %v", err)
	}
	defer listResp.Body.Close()
	var pairs []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&pairs); err != nil {
		t.Fatalf("decode pairs: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("got %d pairs, want 1", len(pairs))
	}
	if pairs[0].ID != "ws-test-1" {
		t.Fatalf("pair id = %q, want ws-test-1", pairs[0].ID)
	}
	if pairs[0].Name != "Remote" {
		t.Fatalf("pair name = %q, want Remote", pairs[0].Name)
	}
	if pairs[0].Status != "connected" {
		t.Fatalf("pair status = %q, want connected", pairs[0].Status)
	}
}

func TestRemoveWorkspace(t *testing.T) {
	base, _ := newServer(t)
	workspaceID := "ws-removable"

	// Create invitation + accept it
	inviteResp, err := http.Post(base+"/api/invitations", "application/json", nil)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	defer inviteResp.Body.Close()
	var invite struct{ Code string `json:"code"` }
	json.NewDecoder(inviteResp.Body).Decode(&invite)
	pairReq := map[string]any{
		"workspace_id":    workspaceID,
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote", Name: "R", IP: "10.0.0.1", Port: 47810},
		"url":             "http://10.0.0.1:47810",
	}
	body, _ := json.Marshal(pairReq)
	http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))

	// Delete the workspace
	delReq, err := http.NewRequest(http.MethodDelete, base+"/api/workspaces/"+workspaceID, nil)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /api/workspaces/%s: %v", workspaceID, err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", delResp.StatusCode)
	}

	// List pairs — should be empty
	listResp, err := http.Get(base + "/api/workspaces")
	if err != nil {
		t.Fatalf("GET /api/workspaces: %v", err)
	}
	defer listResp.Body.Close()
	var pairs []any
	if err := json.NewDecoder(listResp.Body).Decode(&pairs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pairs) != 0 {
		t.Fatalf("got %d pairs after delete, want 0", len(pairs))
	}
}

func TestGetPair(t *testing.T) {
	base, _ := newServer(t)
	workspaceID := "ws-test-get"

	// Create pair via sync/accept
	inviteResp, _ := http.Post(base+"/api/invitations", "application/json", nil)
	var invite struct{ Code string `json:"code"` }
	json.NewDecoder(inviteResp.Body).Decode(&invite)
	inviteResp.Body.Close()

	pairReq := map[string]any{
		"workspace_id":    workspaceID,
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote", Name: "Remote", IP: "10.0.0.2", Port: 47810},
		"url":             "http://10.0.0.2:47810",
	}
	body, _ := json.Marshal(pairReq)
	http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))

	// GET the pair by ID
	getResp, err := http.Get(base + "/api/workspaces/" + workspaceID)
	if err != nil {
		t.Fatalf("GET /api/workspaces/%s: %v", workspaceID, err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", getResp.StatusCode)
	}
	var view struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.ID != workspaceID {
		t.Fatalf("id = %q, want %q", view.ID, workspaceID)
	}
	if view.Name != "Remote" {
		t.Fatalf("name = %q, want Remote", view.Name)
	}
	if view.Status != "connected" {
		t.Fatalf("status = %q, want connected", view.Status)
	}
}

func TestDeleteNonExistentIsNoOp(t *testing.T) {
	base, _ := newServer(t)
	delReq, _ := http.NewRequest(http.MethodDelete, base+"/api/workspaces/non-existent", nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE non-existent: %v", err)
	}
	defer delResp.Body.Close()
	// SQLite DELETE on missing row succeeds (0 rows affected)
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", delResp.StatusCode)
	}
}

func TestCorsHeaders(t *testing.T) {
	base, _ := newServer(t)

	// OPTIONS preflight for /api/self
	req, err := http.NewRequest(http.MethodOptions, base+"/api/self", nil)
	if err != nil {
		t.Fatalf("OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", "tauri://localhost")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /api/self: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing Access-Control-Allow-Origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("missing Access-Control-Allow-Methods header")
	}
	if resp.Header.Get("Access-Control-Allow-Headers") == "" {
		t.Fatalf("missing Access-Control-Allow-Headers header")
	}

	// Regular GET also has CORS headers
	getResp, err := http.Get(base + "/api/workspaces")
	if err != nil {
		t.Fatalf("GET /api/workspaces: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("GET response missing Access-Control-Allow-Origin")
	}
}

func TestSyncOperationsCreateFolder(t *testing.T) {
	base, _ := newServer(t)

	// Create pair first
	inviteResp, _ := http.Post(base+"/api/invitations", "application/json", nil)
	var invite struct{ Code string `json:"code"` }
	json.NewDecoder(inviteResp.Body).Decode(&invite)
	inviteResp.Body.Close()

	pairReq := map[string]any{
		"workspace_id":    "ws-ops-test",
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote", Name: "R", IP: "10.0.0.3", Port: 47810},
		"url":             "http://10.0.0.3:47810",
	}
	body, _ := json.Marshal(pairReq)
	http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))

		// Create folder via sync operation
		// Keys match Go struct field names (no json tags on storage types)
		folderOp := map[string]any{
			"id":           "op-1",
			"WorkspaceID":  "ws-ops-test",
			"kind":         "folder.create",
			"payload": map[string]any{
				"ID":          "folder-1",
				"WorkspaceID": "ws-ops-test",
				"Name":        "Test Folder",
			},
		}
	opsBody, _ := json.Marshal([]any{folderOp})
	syncResp, err := http.Post(base+"/sync/workspaces/ws-ops-test/operations", "application/json", bytes.NewReader(opsBody))
	if err != nil {
		t.Fatalf("POST sync operations: %v", err)
	}
	defer syncResp.Body.Close()
	if syncResp.StatusCode != http.StatusOK {
		t.Fatalf("sync operations status = %d, want 200", syncResp.StatusCode)
	}

	// List folders — should include Test Folder
	foldResp, err := http.Get(base + "/api/workspaces/ws-ops-test/folders")
	if err != nil {
		t.Fatalf("GET folders: %v", err)
	}
	defer foldResp.Body.Close()
	if foldResp.StatusCode != http.StatusOK {
		t.Fatalf("folders status = %d, want 200", foldResp.StatusCode)
	}
	var folders []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(foldResp.Body).Decode(&folders); err != nil {
		t.Fatalf("decode folders: %v", err)
	}
	found := false
	for _, f := range folders {
		if f.Name == "Test Folder" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Test Folder not found in folders list: %+v", folders)
	}
}

func TestCreateTextArtifact(t *testing.T) {
	base, _ := newServer(t)

	// Create pair
	inviteResp, _ := http.Post(base+"/api/invitations", "application/json", nil)
	var invite struct{ Code string `json:"code"` }
	json.NewDecoder(inviteResp.Body).Decode(&invite)
	inviteResp.Body.Close()
	pairReq := map[string]any{
		"workspace_id":    "ws-text-test",
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote", Name: "R", IP: "10.0.0.4", Port: 47810},
		"url":             "http://10.0.0.4:47810",
	}
	body, _ := json.Marshal(pairReq)
	http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))

	// Create text artifact via sync operation
	textOp := map[string]any{
		"id":          "op-text-1",
		"WorkspaceID": "ws-text-test",
		"kind":        "artifact.create",
		"payload": map[string]any{
			"ID":          "text-1",
			"WorkspaceID": "ws-text-test",
			"Type":        "text",
			"Title":       "Test Note",
			"Content":     "Hello world",
			"Status":      "active",
		},
	}
	opsBody, _ := json.Marshal([]any{textOp})
	http.Post(base+"/sync/workspaces/ws-text-test/operations", "application/json", bytes.NewReader(opsBody))

	// List artifacts
	artResp, err := http.Get(base + "/api/workspaces/ws-text-test/artifacts?folder=all")
	if err != nil {
		t.Fatalf("GET artifacts: %v", err)
	}
	defer artResp.Body.Close()
	if artResp.StatusCode != http.StatusOK {
		t.Fatalf("artifacts status = %d, want 200", artResp.StatusCode)
	}
	var artifacts []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(artResp.Body).Decode(&artifacts); err != nil {
		t.Fatalf("decode artifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("got %d artifacts, want 1", len(artifacts))
	}
	if artifacts[0].Title != "Test Note" {
		t.Fatalf("title = %q, want Test Note", artifacts[0].Title)
	}
	if artifacts[0].Content != "Hello world" {
		t.Fatalf("content = %q, want Hello world", artifacts[0].Content)
	}
	if artifacts[0].Type != "text" {
		t.Fatalf("type = %q, want text", artifacts[0].Type)
	}
}

func TestMultiplePairs(t *testing.T) {
	base, _ := newServer(t)

	createPair := func(id, code, remoteName string) {
		inviteResp, _ := http.Post(base+"/api/invitations", "application/json", nil)
		var inv struct{ Code string `json:"code"` }
		json.NewDecoder(inviteResp.Body).Decode(&inv)
		inviteResp.Body.Close()
		pr := map[string]any{
			"workspace_id":    id,
			"invitation_code": inv.Code,
			"device":          httpapi.Device{ID: id, Name: remoteName, IP: "10.0.0.1", Port: 47810},
			"url":             "http://10.0.0.1:47810",
		}
		body, _ := json.Marshal(pr)
		http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))
	}

	createPair("ws-a", "code1", "Alpha")
	createPair("ws-b", "code2", "Beta")
	createPair("ws-c", "code3", "Gamma")

	listResp, err := http.Get(base + "/api/workspaces")
	if err != nil {
		t.Fatalf("GET /api/workspaces: %v", err)
	}
	defer listResp.Body.Close()
	var pairs []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&pairs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("got %d pairs, want 3", len(pairs))
	}
	names := map[string]bool{}
	for _, p := range pairs {
		names[p.Name] = true
	}
	for _, want := range []string{"Alpha", "Beta", "Gamma"} {
		if !names[want] {
			t.Fatalf("missing pair %q in list", want)
		}
	}
}

func TestInviteMismatchRejected(t *testing.T) {
	base, _ := newServer(t)

	// Try to accept with a made-up code that doesn't exist
	pairReq := map[string]any{
		"workspace_id":    "ws-fake",
		"invitation_code": "nonexistent-code",
		"device":          httpapi.Device{ID: "remote", Name: "R", IP: "10.0.0.1", Port: 47810},
		"url":             "http://10.0.0.1:47810",
	}
	body, _ := json.Marshal(pairReq)
	resp, err := http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /sync/pair: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestCorsOnSyncEndpoint(t *testing.T) {
	base, _ := newServer(t)
	req, _ := http.NewRequest(http.MethodOptions, base+"/sync/pair", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /sync/pair: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS origin header")
	}
}

func TestListFoldersAfterPair(t *testing.T) {
	base, _ := newServer(t)

	// Create pair
	inviteResp, _ := http.Post(base+"/api/invitations", "application/json", nil)
	var invite struct{ Code string `json:"code"` }
	json.NewDecoder(inviteResp.Body).Decode(&invite)
	inviteResp.Body.Close()
	pairReq := map[string]any{
		"workspace_id":    "ws-folders",
		"invitation_code": invite.Code,
		"device":          httpapi.Device{ID: "remote", Name: "R", IP: "10.0.0.5", Port: 47810},
		"url":             "http://10.0.0.5:47810",
	}
	body, _ := json.Marshal(pairReq)
	http.Post(base+"/sync/pair", "application/json", bytes.NewReader(body))

	// List folders — should be empty (no auto-created folders)
	foldResp, err := http.Get(base + "/api/workspaces/ws-folders/folders")
	if err != nil {
		t.Fatalf("GET /api/workspaces/ws-folders/folders: %v", err)
	}
	defer foldResp.Body.Close()
	if foldResp.StatusCode != http.StatusOK {
		t.Fatalf("folders status = %d, want 200", foldResp.StatusCode)
	}
	var folders []any
	if err := json.NewDecoder(foldResp.Body).Decode(&folders); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(folders) != 0 {
		t.Fatalf("got %d folders, want 0 (empty initially)", len(folders))
	}
}

func TestBadRequestOnPair(t *testing.T) {
	base, _ := newServer(t)

	// POST with missing fields
	resp, err := http.Post(base+"/api/workspaces/pair", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST pair empty: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("pair empty status = %d, want 400", resp.StatusCode)
	}
}
