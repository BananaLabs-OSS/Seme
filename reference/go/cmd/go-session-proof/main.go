// Command go-session-proof is a filesystem test adapter around the in-memory
// incremental Go session. Session state and lifting remain snapshot-only.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"seme.local/reference/goprojector"
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
	files := map[string]string{}
	fatal(filepath.WalkDir(*project, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(*project, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(content)
		return nil
	}))
	moduleData, err := os.ReadFile(filepath.Join(*project, "go.mod"))
	fatal(err)
	moduleRoot := ""
	fields := strings.Fields(string(moduleData))
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "module" {
			moduleRoot = fields[i+1]
			break
		}
	}
	if moduleRoot == "" {
		fatal(fmt.Errorf("go.mod module path missing"))
	}
	session, err := goprovider.NewIncrementalSession(module)
	fatal(err)
	result := session.Apply(goprovider.DocumentSnapshot{Revision: *revision, ModulePath: moduleRoot, PackagePath: *packagePath, Entry: *entry, Files: files})
	if !result.Accepted || !result.Valid {
		fatal(fmt.Errorf("snapshot disposition %s: %#v", result.Disposition, result.Diagnostics))
	}
	canonical := []byte(result.CanonicalG1)
	var enveloped []byte
	for _, source := range files {
		graph, present, verifyErr := goprojector.VerifyProjectionEnvelope([]byte(source))
		fatal(verifyErr)
		if present {
			if enveloped != nil {
				fatal(fmt.Errorf("multiple projection envelopes"))
			}
			enveloped = graph
		}
	}
	if enveloped != nil {
		canonical = enveloped
	}
	fatal(os.WriteFile(*out, canonical, 0o644))
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-session-proof:", err)
		os.Exit(65)
	}
}
