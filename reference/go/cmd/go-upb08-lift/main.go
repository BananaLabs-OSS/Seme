// Command go-upb08-lift is a fixture-only evidence adapter for the cumulative
// ordered transport service. It is not a general filesystem project loader.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"seme.local/reference/goprovider"
)

func main() {
	project := flag.String("project", "", "materialized UPB-08 project")
	execution := flag.String("execution-g1", "", "Execution v36 module G1")
	out := flag.String("out", "", "new canonical graph path")
	flag.Parse()
	if flag.NArg() != 0 || *project == "" || *execution == "" || *out == "" {
		fatal("arguments")
	}
	module, err := os.ReadFile(*execution)
	if err != nil {
		fatal(err.Error())
	}
	files := map[string]string{}
	err = filepath.WalkDir(*project, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink:%s", path)
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative, relErr := filepath.Rel(*project, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		fatal(err.Error())
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		fatal(err.Error())
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/streamservice", Entry: "Dispatch", Files: files})
	if !result.Valid {
		fatal(fmt.Sprintf("lift:%v", result.Diagnostics))
	}
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fatal(err.Error())
	}
	if _, err = file.WriteString(result.CanonicalG1); err != nil {
		_ = file.Close()
		_ = os.Remove(*out)
		fatal(err.Error())
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(*out)
		fatal(err.Error())
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "go-upb08-lift:", message)
	os.Exit(1)
}
