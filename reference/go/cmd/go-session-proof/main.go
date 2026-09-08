// Command go-session-proof is a filesystem test adapter around the in-memory
// incremental Go session. Session state and lifting remain snapshot-only.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"seme.local/reference/goprovider"
)

func main() {
	modulePath := flag.String("module", "", "Core Execution module G1")
	project := flag.String("project", "", "generic Go fixture directory")
	packagePath := flag.String("package", "", "stable package path")
	revision := flag.Uint64("revision", 1, "client revision")
	entry := flag.String("entry", "", "entry function name (defaults to stable first declaration)")
	out := flag.String("out", "", "canonical G1 output")
	flag.Parse()
	if *modulePath == "" || *project == "" || *packagePath == "" || *out == "" {
		fatal(fmt.Errorf("requires --module, --project, --package, and --out"))
	}
	module, err := os.ReadFile(*modulePath)
	fatal(err)
	entries, err := os.ReadDir(*project)
	fatal(err)
	sort.Slice(entries, func(left, right int) bool { return entries[left].Name() < entries[right].Name() })
	files := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(*project, entry.Name()))
		fatal(err)
		files[entry.Name()] = string(content)
	}
	session, err := goprovider.NewIncrementalSession(module)
	fatal(err)
	result := session.Apply(goprovider.DocumentSnapshot{Revision: *revision, PackagePath: *packagePath, Entry: *entry, Files: files})
	if !result.Accepted || !result.Valid {
		fatal(fmt.Errorf("snapshot disposition %s: %#v", result.Disposition, result.Diagnostics))
	}
	fatal(os.WriteFile(*out, []byte(result.CanonicalG1), 0o644))
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-session-proof:", err)
		os.Exit(65)
	}
}
