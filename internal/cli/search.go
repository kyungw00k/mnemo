package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// RunSearch handles: mnemo search <query> [--limit N] [--category C]
func RunSearch(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mnemo search <query> [--limit N] [--category C]")
		os.Exit(1)
	}

	query := args[0]
	limit := 10
	category := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--limit", "-n":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					limit = n
				}
				i++
			}
		case "--category", "-c":
			if i+1 < len(args) {
				category = args[i+1]
				i++
			}
		}
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo search: %v\n", err)
		os.Exit(1)
	}

	results, err := s.memSvc.Search(ctx, s.cfg.HostID, category, query, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo search: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
}

// RunSave handles: mnemo save <category> <key> <value> [--project P]
func RunSave(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mnemo save <category> <key> <value> [--project P]")
		os.Exit(1)
	}

	category, key, value := args[0], args[1], args[2]
	project := detectProject()

	for i := 3; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			project = args[i+1]
			i++
		}
	}

	ctx := context.Background()
	s, err := initSvcs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo save: %v\n", err)
		os.Exit(1)
	}

	mem, err := s.memSvc.Save(ctx, s.cfg.HostID, category, key, value, project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo save: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]any{"id": mem.ID, "ok": true})
}
