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
}

// RunHook dispatches hook subcommands: session-start, session-end, observe.
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
	default:
		writeJSON(map[string]any{"continue": true})
	}
}

// runSessionStart reads recent decisions and notes, returns them as additionalContext.
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

	mems, err := s.memSvc.List(ctx, s.cfg.HostID, "decision", 5)
	if err == nil && len(mems) > 0 {
		sb.WriteString("### Recent Decisions\n")
		for _, m := range mems {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", m.Key, m.Value))
		}
		sb.WriteString("\n")
	}

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

// runSessionEnd extracts user messages from the transcript and saves a session note.
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

// runObserve records file edits and build commands as observation memories.
func runObserve() {
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeJSON(map[string]any{"continue": true})
		return
	}

	switch input.ToolName {
	case "Edit", "Write":
		var ti struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(input.ToolInput, &ti); err != nil || ti.FilePath == "" {
			break
		}
		ctx := context.Background()
		s, err := initSvcs(ctx)
		if err != nil {
			break
		}
		project := detectProjectForHook(s.cfg)
		key := "edited:" + ti.FilePath
		val := fmt.Sprintf("%s @ %s", input.SessionID, time.Now().Format(time.RFC3339))
		if _, err := s.memSvc.Save(ctx, s.cfg.HostID, "observation", key, val, project); err != nil {
			log.Printf("mnemo hook observe: %v", err)
		}

	case "Bash":
		var ti struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
			break
		}
		buildPatterns := []string{"go build", "go test", "make ", "npm run build", "npm test", "pytest"}
		for _, p := range buildPatterns {
			if strings.Contains(ti.Command, p) {
				ctx := context.Background()
				s, err := initSvcs(ctx)
				if err != nil {
					break
				}
				project := detectProjectForHook(s.cfg)
				key := "build:" + project
				val := fmt.Sprintf("%s @ %s", ti.Command, time.Now().Format(time.RFC3339))
				if _, err := s.memSvc.Save(ctx, s.cfg.HostID, "observation", key, val, project); err != nil {
					log.Printf("mnemo hook observe bash: %v", err)
				}
				break
			}
		}
	}

	writeJSON(map[string]any{"continue": true})
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
