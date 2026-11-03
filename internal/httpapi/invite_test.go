package httpapi_test

import (
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

func newInviteServer(t *testing.T) (string, *storage.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(filepath.Join(dir, "connect.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	server := httpapi.New(fstest.MapFS{}, db, func() httpapi.Device { return httpapi.Device{ID: "self", Name: "Self", IP: "127.0.0.1", Port: 47810} }, dir)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go http.Serve(listener, server.Handler())
	t.Cleanup(func() { listener.Close() })
	return "http://" + listener.Addr().String(), db
}

func TestCreateInvitationReturnsURL(t *testing.T) {
	base, _ := newInviteServer(t)
	resp, err := http.Post(base+"/api/invitations", "application/json", nil)
	if err != nil {
		t.Fatalf("post invitation: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code == "" || body.URL == "" {
		t.Fatalf("empty invitation: %+v", body)
	}
	if !strings.HasSuffix(body.URL, "/join/"+body.Code) {
		t.Fatalf("url %q does not embed code %q", body.URL, body.Code)
	}
}
