package dashboard

import (
	"io/fs"
	"net/http"

	"github.com/kyungw00k/mnemo/internal/db"
	"github.com/kyungw00k/mnemo/internal/service"
)

// Server holds the dashboard dependencies and exposes HTTP handlers.
type Server struct {
	memSvc  *service.MemoryService
	noteSvc *service.NoteService
	db      *db.DBConn
}

// NewServer creates a new dashboard Server.
func NewServer(memSvc *service.MemoryService, noteSvc *service.NoteService, conn *db.DBConn) *Server {
	return &Server{
		memSvc:  memSvc,
		noteSvc: noteSvc,
		db:      conn,
	}
}

// FileServer returns an http.Handler that serves the embedded frontend /assets/ files.
func (s *Server) FileServer() http.Handler {
	dist, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(dist))
}

// HandleIndex serves the SPA index.html for the dashboard root.
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	dist, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		http.Error(w, "dashboard not built — run: make frontend-build", http.StatusNotFound)
		return
	}
	f, err := dist.Open("index.html")
	if err != nil {
		http.Error(w, "dashboard index not found — run: make frontend-build", http.StatusNotFound)
		return
	}
	f.Close()
	http.ServeFileFS(w, r, dist, "index.html")
}
