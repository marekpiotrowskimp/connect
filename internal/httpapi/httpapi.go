// Package httpapi serves the embedded web UI, the local UI-facing API
// ("/api/*"), and the API used by other Connect instances on the LAN
// ("/peer/*").
package httpapi

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"

	"connect/internal/storage"
)

// DefaultPort is the TCP port Connect tries to bind first, and the port
// assumed for manually-added peers that don't specify one. If it's already
// in use, Listen falls back to an OS-assigned port.
const DefaultPort = 47810

// defaultPeerPort is used internally by handlers; kept as an alias so the
// intent at each call site is clear.
const defaultPeerPort = DefaultPort

// Server holds the dependencies needed by the HTTP handlers.
type Server struct {
	mux      *http.ServeMux
	db       *storage.DB
	self     func() Device
	filesDir string
	sequence atomic.Int64
}

// New builds a Server. selfFn is called whenever a handler needs this
// instance's identity, so it always reflects the current port (only known
// once the listener is bound). filesDir is the directory received files
// are streamed to.
type Device struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func New(webFS fs.FS, db *storage.DB, selfFn func() Device, filesDir string) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		db:       db,
		self:     selfFn,
		filesDir: filesDir,
	}
	s.routes(webFS)
	return s
}

func (s *Server) routes(webFS fs.FS) {
	s.mux.Handle("GET /", http.FileServer(http.FS(webFS)))
	s.mux.HandleFunc("GET /join/{code}", s.handleJoin(webFS))

	s.mux.HandleFunc("GET /api/self", s.handleSelf)
	s.mux.HandleFunc("POST /api/invitations", s.handleCreateInvitation)
	s.mux.HandleFunc("GET /api/workspaces", s.handleListPairs)
	s.mux.HandleFunc("POST /api/workspaces/pair", s.handlePairWorkspace)
	s.mux.HandleFunc("GET /api/workspaces/{id}", s.handleGetPair)
	s.mux.HandleFunc("DELETE /api/workspaces/{id}", s.handleRemovePair)
	s.mux.HandleFunc("GET /api/workspaces/{id}/folders", s.handleListFolders)
	s.mux.HandleFunc("POST /api/workspaces/{id}/folders", s.handleCreateFolder)
	s.mux.HandleFunc("GET /api/workspaces/{id}/artifacts", s.handleListArtifacts)
	s.mux.HandleFunc("POST /api/workspaces/{id}/artifacts/text", s.handleCreateText)
	s.mux.HandleFunc("POST /api/workspaces/{id}/artifacts/files", s.handleCreateFile)
	s.mux.HandleFunc("GET /api/workspaces/{id}/artifacts/{itemID}/file", s.handleDownloadFile)
	s.mux.HandleFunc("PATCH /api/workspaces/{id}/artifacts/{itemID}", s.handleMoveArtifact)
	s.mux.HandleFunc("DELETE /api/workspaces/{id}/artifacts/{itemID}", s.handleDeleteArtifact)
	s.mux.HandleFunc("POST /sync/pair", s.handlePairAccept)
	s.mux.HandleFunc("POST /sync/workspaces/{id}/operations", s.handleSyncOperations)
}

func (s *Server) handleJoin(webFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := fs.ReadFile(webFS, "join.html")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
	}
}

func (s *Server) handleSelf(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.self())
}

// Handler returns the HTTP handler serving all routes ("/", "/api/*",
// "/peer/*"), for use with httptest or a custom *http.Server.
func (s *Server) Handler() http.Handler {
	return cors(s.mux)
}

// cors wraps an http.Handler with CORS headers needed by the Tauri webview
// (which runs on a non-file origin such as tauri://localhost).
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpapi: encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// Listen binds a TCP listener on the LAN-facing interface. It tries
// preferredPort first and falls back to an OS-assigned port if that one is
// already in use.
func Listen(preferredPort int) (net.Listener, error) {
	ln, err := net.Listen("tcp", portAddr(preferredPort))
	if err == nil {
		return ln, nil
	}
	return net.Listen("tcp", portAddr(0))
}

func portAddr(port int) string {
	return ":" + strconv.Itoa(port)
}

// Serve starts serving HTTP requests on ln until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	err := srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
