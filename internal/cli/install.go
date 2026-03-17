package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// settingsFile preserves all unknown fields in ~/.claude/settings.json.
type settingsFile map[string]json.RawMessage

// hooksSection maps event names (e.g. "SessionStart") to hook groups.
type hooksSection map[string][]hookGroup

type hookGroup struct {
	Matcher string    `json:"matcher,omitempty"`
	Hooks   []hookCmd `json:"hooks"`
}

type hookCmd struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout *int   `json:"timeout,omitempty"`
}

// mnemoHookDefs defines all hooks that mnemo installs.
var mnemoHookDefs = []struct {
	event   string
	matcher string
	command string
}{
	{"SessionStart", "startup|resume|compact", "mnemo hook session-start"},
	{"PreCompact", "", "mnemo hook pre-compact"},
	{"PostToolUse", "Bash", "mnemo hook observe"},
	{"Stop", "", "mnemo hook session-end"},
}

// mnemoMemoryInstructions is injected into AI tools that support agent prompt files
// but lack a shell-command hook system (opencode, Codex).
const mnemoMemoryInstructions = `
## mnemo Memory

You have access to a persistent memory server (mnemo) via MCP tools.
Use these tools to maintain context across sessions:

- ` + "`memory_save(category, key, value, project)`" + ` — save important facts, decisions, conventions
- ` + "`memory_search(query)`" + ` — retrieve relevant past context by semantic search
- ` + "`memory_list(category, limit)`" + ` — list recent memories by category
- ` + "`note_save(project, title, content, tags)`" + ` — save longer session summaries
- ` + "`note_search(query)`" + ` — search past session notes

At session start: call memory_search("project setup decisions conventions").
At session end: save key decisions with memory_save and a session summary with note_save.
`

// RunInstall handles 'mnemo hook install [--uninstall] [--dry-run]'.
func RunInstall(args []string) {
	uninstall, dryRun := false, false
	for _, arg := range args {
		switch arg {
		case "--uninstall":
			uninstall = true
		case "--dry-run":
			dryRun = true
		}
	}

	path := claudeSettingsPath()

	// Read existing file (missing file is treated as empty object).
	sf := settingsFile{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &sf); err != nil {
			fmt.Fprintf(os.Stderr, "mnemo hook install: parse %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	// Parse hooks section (may be absent).
	hs := hooksSection{}
	if raw, ok := sf["hooks"]; ok {
		if err := json.Unmarshal(raw, &hs); err != nil {
			fmt.Fprintf(os.Stderr, "mnemo hook install: parse hooks: %v\n", err)
			os.Exit(1)
		}
	}

	if uninstall {
		removeAllMnemoHooks(hs)
		printHookActions("Uninstalled")
	} else {
		if !installAllMnemoHooks(hs) {
			fmt.Println("mnemo hooks already installed — no changes made")
			return
		}
		printHookActions("Installed")
	}

	// Re-encode hooks section.
	if len(hs) == 0 {
		delete(sf, "hooks")
	} else {
		raw, _ := json.Marshal(hs)
		sf["hooks"] = raw
	}

	// Marshal full settings with indentation.
	out, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo hook install: marshal: %v\n", err)
		os.Exit(1)
	}
	out = append(out, '\n')

	if dryRun {
		fmt.Printf("--- %s (dry-run) ---\n%s", path, out)
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mnemo hook install: mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "mnemo hook install: write: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Written to", path)
}

// MaybeAutoInstall detects installed AI coding tools and registers mnemo in each:
//   - Claude Code (~/.claude/): installs shell-command hooks
//   - opencode (~/.config/opencode/): registers MCP server + memory instructions
//   - Codex CLI (~/.codex/): registers MCP server + memory instructions in AGENTS.md
//
// Skipped entirely when AUTO_INSTALL_HOOKS=false. Errors are logged to stderr
// but never abort server startup.
func MaybeAutoInstall() {
	if os.Getenv("AUTO_INSTALL_HOOKS") == "false" {
		return
	}
	maybeAutoInstallClaude()
	maybeAutoInstallOpenCode()
	maybeAutoInstallCodex()
}

// maybeAutoInstallClaude installs shell-command hooks into ~/.claude/settings.json.
func maybeAutoInstallClaude() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); os.IsNotExist(err) {
		return
	}

	path := claudeSettingsPath()
	sf := settingsFile{}
	if data, err := os.ReadFile(path); err == nil {
		if jsonErr := json.Unmarshal(data, &sf); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: claude auto-install: parse %s: %v\n", path, jsonErr)
			return
		}
	}

	hs := hooksSection{}
	if raw, ok := sf["hooks"]; ok {
		if jsonErr := json.Unmarshal(raw, &hs); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: claude auto-install: parse hooks: %v\n", jsonErr)
			return
		}
	}

	if !installAllMnemoHooks(hs) {
		return
	}

	if len(hs) == 0 {
		delete(sf, "hooks")
	} else {
		raw, _ := json.Marshal(hs)
		sf["hooks"] = raw
	}

	out, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: claude auto-install: marshal: %v\n", err)
		return
	}
	out = append(out, '\n')

	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		fmt.Fprintf(os.Stderr, "mnemo: claude auto-install: mkdir: %v\n", mkErr)
		return
	}
	if writeErr := os.WriteFile(path, out, 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "mnemo: claude auto-install: write: %v\n", writeErr)
	}
}

// maybeAutoInstallOpenCode registers mnemo as an MCP server in
// ~/.config/opencode/opencode.json and injects memory instructions.
func maybeAutoInstallOpenCode() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".config", "opencode")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	configPath := filepath.Join(dir, "opencode.json")

	// Read existing config (treat missing as empty object).
	raw := settingsFile{}
	if data, err := os.ReadFile(configPath); err == nil {
		if jsonErr := json.Unmarshal(data, &raw); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: opencode auto-install: parse %s: %v\n", configPath, jsonErr)
			return
		}
	}

	changed := false

	// Register MCP server under "mcp.mnemo".
	mcpMap := map[string]any{}
	if existing, ok := raw["mcp"]; ok {
		_ = json.Unmarshal(existing, &mcpMap)
	}
	if _, alreadySet := mcpMap["mnemo"]; !alreadySet {
		mcpMap["mnemo"] = map[string]any{
			"type":    "local",
			"command": []string{"mnemo"},
			"enabled": true,
		}
		b, _ := json.Marshal(mcpMap)
		raw["mcp"] = b
		changed = true
	}

	// Register instructions file path.
	instrFile := filepath.Join(dir, "mnemo.md")
	instrPaths := []string{}
	if existing, ok := raw["instructions"]; ok {
		_ = json.Unmarshal(existing, &instrPaths)
	}
	hasInstr := false
	for _, p := range instrPaths {
		if p == instrFile {
			hasInstr = true
			break
		}
	}
	if !hasInstr {
		instrPaths = append(instrPaths, instrFile)
		b, _ := json.Marshal(instrPaths)
		raw["instructions"] = b
		changed = true
	}

	// Write mnemo.md if missing.
	if _, err := os.Stat(instrFile); os.IsNotExist(err) {
		if writeErr := os.WriteFile(instrFile, []byte(mnemoMemoryInstructions), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: opencode auto-install: write instructions: %v\n", writeErr)
		}
	}

	if !changed {
		return
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: opencode auto-install: marshal: %v\n", err)
		return
	}
	out = append(out, '\n')
	if writeErr := os.WriteFile(configPath, out, 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "mnemo: opencode auto-install: write config: %v\n", writeErr)
	}
}

// maybeAutoInstallCodex registers mnemo as an MCP server in ~/.codex/config.toml
// and appends memory instructions to ~/.codex/AGENTS.md.
func maybeAutoInstallCodex() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".codex")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	// Register MCP in config.toml (string-based append, no TOML library).
	tomlPath := filepath.Join(dir, "config.toml")
	tomlContent := ""
	if data, err := os.ReadFile(tomlPath); err == nil {
		tomlContent = string(data)
	}
	if !strings.Contains(tomlContent, "[mcp_servers.mnemo]") {
		entry := "\n[mcp_servers.mnemo]\ncommand = \"mnemo\"\nargs = []\nenabled = true\n"
		tomlContent += entry
		if writeErr := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: codex auto-install: write config.toml: %v\n", writeErr)
		}
	}

	// Inject memory instructions into AGENTS.md.
	agentsPath := filepath.Join(dir, "AGENTS.md")
	agentsContent := ""
	if data, err := os.ReadFile(agentsPath); err == nil {
		agentsContent = string(data)
	}
	if !strings.Contains(agentsContent, "mnemo") {
		agentsContent += mnemoMemoryInstructions
		if writeErr := os.WriteFile(agentsPath, []byte(agentsContent), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: codex auto-install: write AGENTS.md: %v\n", writeErr)
		}
	}
}

func claudeSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func printHookActions(verb string) {
	for _, def := range mnemoHookDefs {
		if def.matcher != "" {
			fmt.Printf("  %s [%s] matcher=%q → %q\n", verb, def.event, def.matcher, def.command)
		} else {
			fmt.Printf("  %s [%s] → %q\n", verb, def.event, def.command)
		}
	}
}

// installAllMnemoHooks adds missing hooks. Returns true if any change was made.
func installAllMnemoHooks(hs hooksSection) bool {
	changed := false
	for _, def := range mnemoHookDefs {
		if upsertHook(hs, def.event, def.matcher, def.command) {
			changed = true
		}
	}
	return changed
}

// upsertHook ensures command exists under event with the given matcher.
// Returns true if a change was made.
func upsertHook(hs hooksSection, event, matcher, command string) bool {
	groups := hs[event]

	// Already installed anywhere in this event → skip.
	for _, g := range groups {
		for _, cmd := range g.Hooks {
			if cmd.Command == command {
				return false
			}
		}
	}

	// Append to existing group with the same matcher.
	for i, g := range groups {
		if g.Matcher == matcher {
			groups[i].Hooks = append(groups[i].Hooks, hookCmd{Type: "command", Command: command})
			hs[event] = groups
			return true
		}
	}

	// No matching group — create a new one.
	hs[event] = append(groups, hookGroup{
		Matcher: matcher,
		Hooks:   []hookCmd{{Type: "command", Command: command}},
	})
	return true
}

// removeAllMnemoHooks removes all mnemo commands from hs.
func removeAllMnemoHooks(hs hooksSection) {
	for _, def := range mnemoHookDefs {
		removeHook(hs, def.event, def.command)
	}
}

// removeHook removes command from the event's groups and prunes empty groups/events.
func removeHook(hs hooksSection, event, command string) {
	var newGroups []hookGroup
	for _, g := range hs[event] {
		var cmds []hookCmd
		for _, cmd := range g.Hooks {
			if cmd.Command != command {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) > 0 {
			g.Hooks = cmds
			newGroups = append(newGroups, g)
		}
	}
	if len(newGroups) == 0 {
		delete(hs, event)
	} else {
		hs[event] = newGroups
	}
}
