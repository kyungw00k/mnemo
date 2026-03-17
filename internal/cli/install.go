package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func claudeSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// MaybeAutoInstallHooks installs mnemo hooks into ~/.claude/settings.json when:
//   - AUTO_INSTALL_HOOKS env var is not "false"
//   - ~/.claude/ directory exists (Claude Code is present)
//   - hooks are not already installed
//
// All output goes to stderr. Errors are logged but do not abort startup.
func MaybeAutoInstallHooks() {
	if os.Getenv("AUTO_INSTALL_HOOKS") == "false" {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	claudeDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return // Claude Code not detected
	}

	path := claudeSettingsPath()

	sf := settingsFile{}
	if data, err := os.ReadFile(path); err == nil {
		if jsonErr := json.Unmarshal(data, &sf); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: auto-install hooks: parse %s: %v\n", path, jsonErr)
			return
		}
	}

	hs := hooksSection{}
	if raw, ok := sf["hooks"]; ok {
		if jsonErr := json.Unmarshal(raw, &hs); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "mnemo: auto-install hooks: parse hooks: %v\n", jsonErr)
			return
		}
	}

	if !installAllMnemoHooks(hs) {
		return // already installed
	}

	if len(hs) == 0 {
		delete(sf, "hooks")
	} else {
		raw, _ := json.Marshal(hs)
		sf["hooks"] = raw
	}

	out, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: auto-install hooks: marshal: %v\n", err)
		return
	}
	out = append(out, '\n')

	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		fmt.Fprintf(os.Stderr, "mnemo: auto-install hooks: mkdir: %v\n", mkErr)
		return
	}
	if writeErr := os.WriteFile(path, out, 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "mnemo: auto-install hooks: write: %v\n", writeErr)
		return
	}
	fmt.Fprintln(os.Stderr, "mnemo: auto-installed Claude Code hooks (restart Claude to activate)")
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
