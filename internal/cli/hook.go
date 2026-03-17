package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyungw00k/mnemo/internal/config"
)

// hookInput is the Claude Code hook stdin format.
type hookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
}

// RunHook dispatches hook subcommands: session-start, session-end, observe, pre-compact.
func RunHook(args []string) {
	if len(args) == 0 {
		writeJSON(map[string]any{"continue": true})
		return
	}
	switch args[0] {
	case "session-start":
		runSessionStart()
	case "session-end":
		runSessionEnd()
	case "observe":
		runObserve()
	case "pre-compact":
		runPreCompact()
	default:
		writeJSON(map[string]any{"continue": true})
	}
}

// runSessionStart reads recent decisions, notes, and compact snapshots, returns them as additionalContext.
func runSessionStart() {
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeJSON(map[string]any{})
		return
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		writeJSON(map[string]any{})
		return
	}

	project := detectProjectForHook(s.cfg)

	var sb strings.Builder
	sb.WriteString("## mnemo Memory\n\n")

	// Compact snapshot: inject the most recent pre-compact summary first.
	snapshots, err := s.memSvc.List(ctx, s.cfg.HostID, "compact_snapshot", 1)
	if err == nil && len(snapshots) > 0 {
		sb.WriteString("### Context Before Last Compact\n")
		sb.WriteString(snapshots[0].Value)
		sb.WriteString("\n\n")
	}

	// Recent decisions.
	mems, err := s.memSvc.List(ctx, s.cfg.HostID, "decision", 5)
	if err == nil && len(mems) > 0 {
		sb.WriteString("### Recent Decisions\n")
		for _, m := range mems {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", m.Key, m.Value))
		}
		sb.WriteString("\n")
	}

	// Recent session notes.
	notes, err := s.noteSvc.List(ctx, s.cfg.HostID, project, 3)
	if err == nil && len(notes) > 0 {
		sb.WriteString("### Recent Sessions\n")
		for _, n := range notes {
			sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", n.Title, n.CreatedAt.Format("2006-01-02")))
		}
		sb.WriteString("\n")
	}

	writeJSON(map[string]any{"additionalContext": sb.String()})
}

// runSessionEnd extracts user messages and modified files from the transcript and saves a session note.
// When ENABLE_AUTO_EXTRACT=true, also calls an LLM to extract key facts as memories.
func runSessionEnd() {
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeJSON(map[string]any{"continue": true})
		return
	}

	messages := extractMessages(input.TranscriptPath)
	if len(messages) == 0 {
		writeJSON(map[string]any{"continue": true})
		return
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		writeJSON(map[string]any{"continue": true})
		return
	}

	project := detectProjectForHook(s.cfg)
	title := fmt.Sprintf("Session %s: %s", time.Now().Format("2006-01-02"), project)

	displayed := messages
	if len(displayed) > 10 {
		displayed = displayed[len(displayed)-10:]
	}
	var lines []string
	for _, m := range displayed {
		lines = append(lines, "- "+m)
	}
	content := "## Requests\n\n" + strings.Join(lines, "\n")

	// Append modified files list.
	if modifiedFiles := extractModifiedFiles(input.TranscriptPath); len(modifiedFiles) > 0 {
		content += "\n\n## Modified Files\n\n"
		for _, f := range modifiedFiles {
			content += "- " + f + "\n"
		}
	}

	if _, err := s.noteSvc.Save(ctx, s.cfg.HostID, project, title, content, []string{"session", "auto"}); err != nil {
		log.Printf("mnemo hook session-end: %v", err)
	}

	// Auto-extract: call LLM to extract key facts as memories (opt-in via ENABLE_AUTO_EXTRACT=true).
	if s.extractSvc != nil {
		fullText := strings.Join(messages, "\n")
		extracted, extractErr := s.extractSvc.Extract(ctx, fullText)
		if extractErr != nil {
			log.Printf("mnemo hook session-end extract: %v", extractErr)
		}
		for _, m := range extracted {
			if m.Category == "" || m.Key == "" || m.Value == "" {
				continue
			}
			if _, saveErr := s.memSvc.Save(ctx, s.cfg.HostID, m.Category, m.Key, m.Value, project); saveErr != nil {
				log.Printf("mnemo hook session-end extract save: %v", saveErr)
			}
		}
	}

	writeJSON(map[string]any{"continue": true})
}

// runObserve records only meaningful build/test results from Bash tool use.
// Edit and Write observations are intentionally omitted — file history is already in git.
func runObserve() {
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeJSON(map[string]any{"continue": true})
		return
	}

	if input.ToolName != "Bash" {
		writeJSON(map[string]any{"continue": true})
		return
	}

	var ti struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input.ToolInput, &ti); err != nil || ti.Command == "" {
		writeJSON(map[string]any{"continue": true})
		return
	}

	buildPatterns := []string{"go build", "go test", "make ", "npm run build", "npm test", "pytest"}
	matched := false
	for _, p := range buildPatterns {
		if strings.Contains(ti.Command, p) {
			matched = true
			break
		}
	}
	if !matched {
		writeJSON(map[string]any{"continue": true})
		return
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		writeJSON(map[string]any{"continue": true})
		return
	}

	project := detectProjectForHook(s.cfg)
	status := bashResultStatus(input.ToolResponse)
	cmd := ti.Command
	if len(cmd) > 120 {
		cmd = cmd[:120] + "..."
	}
	key := "build:" + project
	val := fmt.Sprintf("[%s] %s @ %s", status, cmd, time.Now().Format("2006-01-02 15:04"))

	if _, err := s.memSvc.Save(ctx, s.cfg.HostID, "observation", key, val, project); err != nil {
		log.Printf("mnemo hook observe bash: %v", err)
	}

	writeJSON(map[string]any{"continue": true})
}

// runPreCompact saves a snapshot of the current session context before compaction.
// This allows session-start to restore meaningful context after compaction.
func runPreCompact() {
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeJSON(map[string]any{})
		return
	}

	messages := extractMessages(input.TranscriptPath)
	if len(messages) == 0 {
		writeJSON(map[string]any{})
		return
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		writeJSON(map[string]any{})
		return
	}

	project := detectProjectForHook(s.cfg)

	// If LLM extraction is available, use it for a high-quality summary.
	if s.extractSvc != nil {
		fullText := strings.Join(messages, "\n")
		extracted, extractErr := s.extractSvc.Extract(ctx, fullText)
		if extractErr == nil && len(extracted) > 0 {
			var parts []string
			for _, m := range extracted {
				if m.Key != "" && m.Value != "" {
					parts = append(parts, fmt.Sprintf("**%s**: %s", m.Key, m.Value))
				}
			}
			if len(parts) > 0 {
				snapshot := fmt.Sprintf("Compact on %s:\n%s",
					time.Now().Format("2006-01-02 15:04"),
					strings.Join(parts, "\n"))
				if _, err := s.memSvc.Save(ctx, s.cfg.HostID, "compact_snapshot", "latest_compact", snapshot, project); err != nil {
					log.Printf("mnemo hook pre-compact save: %v", err)
				}
				writeJSON(map[string]any{})
				return
			}
		}
	}

	// Fallback: save last 10 user messages + modified files as snapshot.
	recent := messages
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Compact on %s (project: %s)\n\n", time.Now().Format("2006-01-02 15:04"), project))
	sb.WriteString("Recent requests:\n")
	for _, m := range recent {
		sb.WriteString("- " + m + "\n")
	}

	if modifiedFiles := extractModifiedFiles(input.TranscriptPath); len(modifiedFiles) > 0 {
		sb.WriteString("\nModified files:\n")
		for _, f := range modifiedFiles {
			sb.WriteString("- " + f + "\n")
		}
	}

	if _, err := s.memSvc.Save(ctx, s.cfg.HostID, "compact_snapshot", "latest_compact", sb.String(), project); err != nil {
		log.Printf("mnemo hook pre-compact save: %v", err)
	}

	writeJSON(map[string]any{})
}

// extractMessages reads user messages from a Claude Code JSONL transcript.
func extractMessages(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var messages []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var entry struct {
			Type    string `json:"type"`
			Message any    `json:"message"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil || entry.Type != "user" {
			continue
		}
		switch m := entry.Message.(type) {
		case string:
			if t := strings.TrimSpace(m); t != "" {
				messages = append(messages, t)
			}
		case []any:
			for _, part := range m {
				if p, ok := part.(map[string]any); ok && p["type"] == "text" {
					if t, ok := p["text"].(string); ok {
						if t = strings.TrimSpace(t); t != "" {
							messages = append(messages, t)
						}
					}
				}
			}
		}
	}
	return messages
}

// extractModifiedFiles parses the transcript JSONL and returns unique file paths
// from Edit and Write tool use entries in assistant messages.
func extractModifiedFiles(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	seen := map[string]bool{}
	var files []string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		var entry struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil || entry.Type != "assistant" {
			continue
		}
		for _, part := range entry.Message.Content {
			if part.Type != "tool_use" {
				continue
			}
			if part.Name != "Edit" && part.Name != "Write" {
				continue
			}
			var inp struct {
				FilePath string `json:"file_path"`
			}
			if err := json.Unmarshal(part.Input, &inp); err != nil || inp.FilePath == "" {
				continue
			}
			if !seen[inp.FilePath] {
				seen[inp.FilePath] = true
				files = append(files, inp.FilePath)
			}
		}
	}
	return files
}

// bashResultStatus extracts a SUCCESS/FAILED status from a Bash tool_response.
func bashResultStatus(response json.RawMessage) string {
	if len(response) == 0 {
		return "UNKNOWN"
	}
	// Try plain string response (stdout).
	var s string
	if json.Unmarshal(response, &s) == nil {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "error:") || strings.Contains(lower, "failed") ||
			strings.Contains(lower, "panic:") || strings.Contains(lower, "fatal") {
			return "FAILED"
		}
		return "SUCCESS"
	}
	// Try structured response.
	var obj struct {
		Error  string `json:"error"`
		Stderr string `json:"stderr"`
	}
	if json.Unmarshal(response, &obj) == nil && (obj.Error != "" || obj.Stderr != "") {
		return "FAILED"
	}
	return "SUCCESS"
}

// detectProject returns the git repo name, or working directory name as fallback.
func detectProject() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return filepath.Base(strings.TrimSpace(string(out)))
	}
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(wd)
}

// projectFromGitRemote extracts the repo name from the git remote origin URL.
// Supports both SSH (git@github.com:org/repo.git) and HTTPS (https://github.com/org/repo.git) formats.
func projectFromGitRemote() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(out))
	raw = strings.TrimSuffix(raw, ".git")
	return filepath.Base(raw)
}

// detectProjectForHook returns the project name.
// When ENABLE_GIT_CONTEXT=true, uses the git remote origin URL for a more stable name.
// Falls back to detectProject (git toplevel or cwd) otherwise.
func detectProjectForHook(cfg *config.Config) string {
	if cfg.EnableGitContext {
		if name := projectFromGitRemote(); name != "" {
			return name
		}
	}
	return detectProject()
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
