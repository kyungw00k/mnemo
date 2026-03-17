package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// qp returns a query parameter value, falling back to fallback if empty.
func qp(r *http.Request, key, fallback string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return fallback
}

// qpi returns an integer query parameter, falling back to fallback if invalid.
func qpi(r *http.Request, key string, fallback int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// --- /api/stats ---

type statsResponse struct {
	Memories   int64    `json:"memories"`
	Notes      int64    `json:"notes"`
	Categories []string `json:"categories"`
	Hosts      []string `json:"hosts"`
}

// HandleStats handles GET /api/stats.
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	memCount := s.countTable(ctx, "memories")
	noteCount := s.countTable(ctx, "notes")
	categories := s.distinctValues(ctx, "memories", "category")
	if categories == nil {
		categories = []string{}
	}
	hosts := s.distinctValuesUnion(ctx, "host")
	if hosts == nil {
		hosts = []string{}
	}

	writeJSON(w, http.StatusOK, statsResponse{
		Memories:   memCount,
		Notes:      noteCount,
		Categories: categories,
		Hosts:      hosts,
	})
}

func (s *Server) countTable(ctx context.Context, table string) int64 {
	var n int64
	var q string
	if s.db.IsSQLite() {
		q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > datetime('now'))`, table)
	} else {
		q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > NOW())`, table)
	}
	_ = s.db.QueryRow(ctx, q).Scan(&n)
	return n
}

func (s *Server) distinctValues(ctx context.Context, table, column string) []string {
	var q string
	if s.db.IsSQLite() {
		q = fmt.Sprintf(`SELECT DISTINCT %s FROM %s WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > datetime('now')) ORDER BY %s`, column, table, column)
	} else {
		q = fmt.Sprintf(`SELECT DISTINCT %s FROM %s WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > NOW()) ORDER BY %s`, column, table, column)
	}
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err == nil {
			vals = append(vals, v)
		}
	}
	return vals
}

// distinctValuesUnion returns distinct host values from both memories and notes.
func (s *Server) distinctValuesUnion(ctx context.Context, column string) []string {
	var q string
	if s.db.IsSQLite() {
		q = fmt.Sprintf(
			`SELECT DISTINCT %s FROM memories WHERE del_yn='N'
			 UNION
			 SELECT DISTINCT %s FROM notes WHERE del_yn='N'
			 ORDER BY %s`, column, column, column)
	} else {
		q = fmt.Sprintf(
			`SELECT DISTINCT %s FROM memories WHERE del_yn='N'
			 UNION
			 SELECT DISTINCT %s FROM notes WHERE del_yn='N'
			 ORDER BY %s`, column, column, column)
	}
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err == nil {
			vals = append(vals, v)
		}
	}
	return vals
}

// --- /api/memories ---

type memoryItem struct {
	ID        int64  `json:"id"`
	Host      string `json:"host"`
	Category  string `json:"category"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type memoriesResponse struct {
	Items []memoryItem `json:"items"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

// HandleMemories handles GET /api/memories?host=&category=&page=&limit=.
func (s *Server) HandleMemories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	host := qp(r, "host", "")
	category := qp(r, "category", "")
	page := qpi(r, "page", 1)
	limit := qpi(r, "limit", 50)
	offset := (page - 1) * limit

	items, total, err := s.queryMemories(ctx, host, category, limit, offset)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []memoryItem{}
	}
	writeJSON(w, http.StatusOK, memoriesResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (s *Server) queryMemories(ctx context.Context, host, category string, limit, offset int) ([]memoryItem, int64, error) {
	var where []string
	var args []any
	ph := placeholderFunc(s.db.IsSQLite())

	where = append(where, "del_yn='N'")
	if s.db.IsSQLite() {
		where = append(where, "(expires_at IS NULL OR expires_at > datetime('now'))")
	} else {
		where = append(where, "(expires_at IS NULL OR expires_at > NOW())")
	}

	idx := 1
	if host != "" {
		where = append(where, fmt.Sprintf("host=%s", ph(idx)))
		args = append(args, host)
		idx++
	}
	if category != "" {
		where = append(where, fmt.Sprintf("category=%s", ph(idx)))
		args = append(args, category)
		idx++
	}

	cond := strings.Join(where, " AND ")

	// Count total
	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM memories WHERE %s", cond)
	_ = s.db.QueryRow(ctx, countQ, args...).Scan(&total)

	// Fetch page
	listQ := fmt.Sprintf(
		`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
		 FROM memories WHERE %s ORDER BY updated_at DESC LIMIT %s OFFSET %s`,
		cond, ph(idx), ph(idx+1),
	)
	listArgs := append(args, limit, offset)

	rows, err := s.db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	var items []memoryItem
	for rows.Next() {
		var m memoryItem
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&m.ID, &m.Host, &m.Category, &m.Key, &m.Value, &m.Metadata, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan memory: %w", err)
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		m.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, m)
	}
	return items, total, rows.Err()
}

// HandleMemoryDetail handles GET /api/memories/:id.
func (s *Server) HandleMemoryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /api/memories/123
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/memories/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	idStr := strings.TrimPrefix(path, "/api/memories/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	memory, err := s.getMemoryByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "memory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, memory)
}

func (s *Server) getMemoryByID(ctx context.Context, id int64) (*memoryItem, error) {
	var q string
	if s.db.IsSQLite() {
		q = `SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
		     FROM memories WHERE id=? AND del_yn='N'`
	} else {
		q = `SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
		     FROM memories WHERE id=$1 AND del_yn='N'`
	}

	var m memoryItem
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, q, id).Scan(&m.ID, &m.Host, &m.Category, &m.Key, &m.Value, &m.Metadata, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("get memory by id: %w", err)
	}

	m.CreatedAt = createdAt.Format(time.RFC3339)
	m.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &m, nil
}

// --- /api/notes ---

type noteItem struct {
	ID        int64  `json:"id"`
	Host      string `json:"host"`
	Project   string `json:"project"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type notesResponse struct {
	Items []noteItem `json:"items"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

// HandleNoteDetail handles GET /api/notes/:id.
func (s *Server) HandleNoteDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /api/notes/123
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/notes/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	idStr := strings.TrimPrefix(path, "/api/notes/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	note, err := s.getNoteByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, note)
}

func (s *Server) getNoteByID(ctx context.Context, id int64) (*noteItem, error) {
	var q string
	if s.db.IsSQLite() {
		q = `SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
		     FROM notes WHERE id=? AND del_yn='N'`
	} else {
		q = `SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
		     FROM notes WHERE id=$1 AND del_yn='N'`
	}

	var n noteItem
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, q, id).Scan(&n.ID, &n.Host, &n.Project, &n.Title, &n.Content, &n.Tags, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("get note by id: %w", err)
	}

	n.CreatedAt = createdAt.Format(time.RFC3339)
	n.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &n, nil
}

// HandleNotes handles GET /api/notes?host=&project=&page=&limit=.
func (s *Server) HandleNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	host := qp(r, "host", "")
	project := qp(r, "project", "")
	page := qpi(r, "page", 1)
	limit := qpi(r, "limit", 50)
	offset := (page - 1) * limit

	items, total, err := s.queryNotes(ctx, host, project, limit, offset)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []noteItem{}
	}
	writeJSON(w, http.StatusOK, notesResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (s *Server) queryNotes(ctx context.Context, host, project string, limit, offset int) ([]noteItem, int64, error) {
	var where []string
	var args []any
	ph := placeholderFunc(s.db.IsSQLite())

	where = append(where, "del_yn='N'")
	if s.db.IsSQLite() {
		where = append(where, "(expires_at IS NULL OR expires_at > datetime('now'))")
	} else {
		where = append(where, "(expires_at IS NULL OR expires_at > NOW())")
	}

	idx := 1
	if host != "" {
		where = append(where, fmt.Sprintf("host=%s", ph(idx)))
		args = append(args, host)
		idx++
	}
	if project != "" {
		where = append(where, fmt.Sprintf("project=%s", ph(idx)))
		args = append(args, project)
		idx++
	}

	cond := strings.Join(where, " AND ")

	var total int64
	_ = s.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM notes WHERE %s", cond), args...).Scan(&total)

	listQ := fmt.Sprintf(
		`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
		 FROM notes WHERE %s ORDER BY updated_at DESC LIMIT %s OFFSET %s`,
		cond, ph(idx), ph(idx+1),
	)
	rows, err := s.db.Query(ctx, listQ, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var items []noteItem
	for rows.Next() {
		var n noteItem
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&n.ID, &n.Host, &n.Project, &n.Title, &n.Content, &n.Tags, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan note: %w", err)
		}
		n.CreatedAt = createdAt.Format(time.RFC3339)
		n.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, n)
	}
	return items, total, rows.Err()
}

// --- /api/search ---

type memorySearchItem struct {
	ID         int64   `json:"id"`
	Host       string  `json:"host"`
	Category   string  `json:"category"`
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Similarity float64 `json:"similarity"`
}

type noteSearchItem struct {
	ID         int64   `json:"id"`
	Host       string  `json:"host"`
	Project    string  `json:"project"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

type searchResponse struct {
	Memories []memorySearchItem `json:"memories"`
	Notes    []noteSearchItem   `json:"notes"`
}

// HandleSearch handles GET /api/search?q=&type=memory|note|all&host=&limit=.
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := qp(r, "q", "")
	if q == "" {
		writeJSON(w, http.StatusOK, searchResponse{Memories: []memorySearchItem{}, Notes: []noteSearchItem{}})
		return
	}

	searchType := qp(r, "type", "all")
	host := qp(r, "host", "")
	limit := qpi(r, "limit", 20)
	ctx := r.Context()

	resp := searchResponse{
		Memories: []memorySearchItem{},
		Notes:    []noteSearchItem{},
	}

	if searchType == "all" || searchType == "memory" {
		mems := s.searchMemories(ctx, host, q, limit)
		if mems != nil {
			resp.Memories = mems
		}
	}
	if searchType == "all" || searchType == "note" {
		notes := s.searchNotes(ctx, host, q, limit)
		if notes != nil {
			resp.Notes = notes
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) searchMemories(ctx context.Context, host, query string, limit int) []memorySearchItem {
	like := "%" + query + "%"
	var rows interface{ Next() bool; Scan(...any) error; Close() error }
	var err error

	if s.db.IsSQLite() {
		var q string
		var args []any
		if host != "" {
			q = `SELECT id, host, category, memory_key, memory_value, 1.0 FROM memories
			     WHERE del_yn='N' AND host=? AND (memory_key LIKE ? OR memory_value LIKE ?)
			     AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY updated_at DESC LIMIT ?`
			args = []any{host, like, like, limit}
		} else {
			q = `SELECT id, host, category, memory_key, memory_value, 1.0 FROM memories
			     WHERE del_yn='N' AND (memory_key LIKE ? OR memory_value LIKE ?)
			     AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY updated_at DESC LIMIT ?`
			args = []any{like, like, limit}
		}
		rows, err = s.db.Query(ctx, q, args...)
	} else {
		var q string
		var args []any
		if host != "" {
			q = `SELECT id, host, category, memory_key, memory_value, 1.0 FROM memories
			     WHERE del_yn='N' AND host=$1 AND (memory_key ILIKE $2 OR memory_value ILIKE $2)
			     AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY updated_at DESC LIMIT $3`
			args = []any{host, like, limit}
		} else {
			q = `SELECT id, host, category, memory_key, memory_value, 1.0 FROM memories
			     WHERE del_yn='N' AND (memory_key ILIKE $1 OR memory_value ILIKE $1)
			     AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY updated_at DESC LIMIT $2`
			args = []any{like, limit}
		}
		rows, err = s.db.Query(ctx, q, args...)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []memorySearchItem
	for rows.Next() {
		var m memorySearchItem
		if err := rows.Scan(&m.ID, &m.Host, &m.Category, &m.Key, &m.Value, &m.Similarity); err == nil {
			items = append(items, m)
		}
	}
	return items
}

func (s *Server) searchNotes(ctx context.Context, host, query string, limit int) []noteSearchItem {
	like := "%" + query + "%"

	var q string
	var args []any

	if s.db.IsSQLite() {
		if host != "" {
			q = `SELECT id, host, COALESCE(project,''), title, content, 1.0 FROM notes
			     WHERE del_yn='N' AND host=? AND (title LIKE ? OR content LIKE ?)
			     AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY updated_at DESC LIMIT ?`
			args = []any{host, like, like, limit}
		} else {
			q = `SELECT id, host, COALESCE(project,''), title, content, 1.0 FROM notes
			     WHERE del_yn='N' AND (title LIKE ? OR content LIKE ?)
			     AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY updated_at DESC LIMIT ?`
			args = []any{like, like, limit}
		}
	} else {
		if host != "" {
			q = `SELECT id, host, COALESCE(project,''), title, content, 1.0 FROM notes
			     WHERE del_yn='N' AND host=$1 AND (title ILIKE $2 OR content ILIKE $2)
			     AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY updated_at DESC LIMIT $3`
			args = []any{host, like, limit}
		} else {
			q = `SELECT id, host, COALESCE(project,''), title, content, 1.0 FROM notes
			     WHERE del_yn='N' AND (title ILIKE $1 OR content ILIKE $1)
			     AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY updated_at DESC LIMIT $2`
			args = []any{like, limit}
		}
	}

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []noteSearchItem
	for rows.Next() {
		var n noteSearchItem
		if err := rows.Scan(&n.ID, &n.Host, &n.Project, &n.Title, &n.Content, &n.Similarity); err == nil {
			items = append(items, n)
		}
	}
	return items
}

// --- /api/graph ---

type graphNode struct {
	ID    string            `json:"id"`
	Type  string            `json:"type"`  // "category", "memory", "note"
	Label string            `json:"label"`
	Data  map[string]string `json:"data"`
}

type graphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "belongs"
}

type graphData struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

// HandleGraph handles GET /api/graph?host=.
func (s *Server) HandleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	host := qp(r, "host", "")

	var nodes []graphNode
	var edges []graphEdge

	// Collect category nodes and memory leaf nodes.
	catSet := map[string]bool{}
	memNodes, memEdges := s.graphMemories(ctx, host, catSet)
	noteNodes, noteEdges := s.graphNotes(ctx, host, catSet)

	// Build category nodes.
	for cat := range catSet {
		nodes = append(nodes, graphNode{
			ID:    "cat:" + cat,
			Type:  "category",
			Label: cat,
			Data:  map[string]string{"category": cat},
		})
	}

	nodes = append(nodes, memNodes...)
	nodes = append(nodes, noteNodes...)
	edges = append(edges, memEdges...)
	edges = append(edges, noteEdges...)

	if nodes == nil {
		nodes = []graphNode{}
	}
	if edges == nil {
		edges = []graphEdge{}
	}

	writeJSON(w, http.StatusOK, graphData{Nodes: nodes, Edges: edges})
}

func (s *Server) graphMemories(ctx context.Context, host string, catSet map[string]bool) ([]graphNode, []graphEdge) {
	var q string
	var args []any

	if s.db.IsSQLite() {
		if host != "" {
			q = `SELECT id, host, category, memory_key, memory_value FROM memories
			     WHERE del_yn='N' AND host=? AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY category, memory_key LIMIT 500`
			args = []any{host}
		} else {
			q = `SELECT id, host, category, memory_key, memory_value FROM memories
			     WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY category, memory_key LIMIT 500`
		}
	} else {
		if host != "" {
			q = `SELECT id, host, category, memory_key, memory_value FROM memories
			     WHERE del_yn='N' AND host=$1 AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY category, memory_key LIMIT 500`
			args = []any{host}
		} else {
			q = `SELECT id, host, category, memory_key, memory_value FROM memories
			     WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY category, memory_key LIMIT 500`
		}
	}

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var nodes []graphNode
	var edges []graphEdge
	for rows.Next() {
		var id int64
		var h, cat, key, val string
		if err := rows.Scan(&id, &h, &cat, &key, &val); err != nil {
			continue
		}
		catSet[cat] = true
		nodeID := fmt.Sprintf("mem:%d", id)
		nodes = append(nodes, graphNode{
			ID:    nodeID,
			Type:  "memory",
			Label: key,
			Data:  map[string]string{"host": h, "category": cat, "key": key, "value": val},
		})
		edges = append(edges, graphEdge{
			ID:     fmt.Sprintf("e:cat:%s->%s", cat, nodeID),
			Source: "cat:" + cat,
			Target: nodeID,
			Type:   "belongs",
		})
	}
	return nodes, edges
}

func (s *Server) graphNotes(ctx context.Context, host string, catSet map[string]bool) ([]graphNode, []graphEdge) {
	var q string
	var args []any

	if s.db.IsSQLite() {
		if host != "" {
			q = `SELECT id, host, COALESCE(project,''), title FROM notes
			     WHERE del_yn='N' AND host=? AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY project, title LIMIT 200`
			args = []any{host}
		} else {
			q = `SELECT id, host, COALESCE(project,''), title FROM notes
			     WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > datetime('now'))
			     ORDER BY project, title LIMIT 200`
		}
	} else {
		if host != "" {
			q = `SELECT id, host, COALESCE(project,''), title FROM notes
			     WHERE del_yn='N' AND host=$1 AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY project, title LIMIT 200`
			args = []any{host}
		} else {
			q = `SELECT id, host, COALESCE(project,''), title FROM notes
			     WHERE del_yn='N' AND (expires_at IS NULL OR expires_at > NOW())
			     ORDER BY project, title LIMIT 200`
		}
	}

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var nodes []graphNode
	var edges []graphEdge
	for rows.Next() {
		var id int64
		var h, proj, title string
		if err := rows.Scan(&id, &h, &proj, &title); err != nil {
			continue
		}
		// Notes use project as their category cluster.
		clusterKey := "notes"
		if proj != "" {
			clusterKey = "notes:" + proj
		}
		catSet[clusterKey] = true
		nodeID := fmt.Sprintf("note:%d", id)
		nodes = append(nodes, graphNode{
			ID:    nodeID,
			Type:  "note",
			Label: title,
			Data:  map[string]string{"host": h, "project": proj, "title": title},
		})
		edges = append(edges, graphEdge{
			ID:     fmt.Sprintf("e:cat:%s->%s", clusterKey, nodeID),
			Source: "cat:" + clusterKey,
			Target: nodeID,
			Type:   "belongs",
		})
	}
	return nodes, edges
}

// --- helpers ---

// placeholderFunc returns a function that generates SQL placeholders.
// For SQLite it always returns "?", for PostgreSQL it returns "$N".
func placeholderFunc(isSQLite bool) func(int) string {
	if isSQLite {
		return func(_ int) string { return "?" }
	}
	return func(n int) string { return fmt.Sprintf("$%d", n) }
}
