package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojector"
	"seme.local/reference/upb12authority"
)

func main() {
	authority := flag.String("authority", "", "source-free UPB12 authority")
	language := flag.String("language", "", "javascript or lua")
	module := flag.String("module", "", "canonical project module")
	foundation := flag.String("foundation", "", "Foundation contract")
	execution := flag.String("execution", "", "Execution contract")
	packages := flag.String("packages", "", "Package contract")
	dependency := flag.String("dependency", "", "Dependency contract")
	configuration := flag.String("configuration", "", "Configuration contract")
	project := flag.String("project", "", "Project-v8 contract")
	out := flag.String("out", "", "new graph JSON")
	flag.Parse()
	if flag.NArg() != 0 || *authority == "" || *language == "" || *module == "" || *out == "" {
		fatal("arguments")
	}
	files, err := upb12authority.Load(*authority)
	if err != nil {
		fatal(err)
	}
	read := func(name string) []byte {
		real, e := filepath.EvalSymlinks(name)
		if e != nil || real != name {
			fatal("contract_symlink")
		}
		info, e := os.Lstat(name)
		if e != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<20 {
			fatal("contract")
		}
		value, e := os.ReadFile(name)
		if e != nil || int64(len(value)) != info.Size() {
			fatal("contract_changed")
		}
		return value
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(read(*foundation), read(*execution), read(*packages), read(*dependency), read(*configuration), read(*project))
	if err != nil {
		fatal(err)
	}
	graph, err := goprojector.ProjectionGraphV4(files["construction-v36.g1"], contracts, files["package-detail-v4.seme"], files["package-v4.seme"], *module, *language)
	if err != nil {
		fatal(err)
	}
	value, err := json.Marshal(graph)
	if err != nil {
		fatal(err)
	}
	value = append(value, '\n')
	if !filepath.IsAbs(*out) || filepath.Clean(*out) != *out {
		fatal("output")
	}
	parent := filepath.Dir(*out)
	real, e := filepath.EvalSymlinks(parent)
	if e != nil || real != parent {
		fatal("output_parent")
	}
	if _, e = os.Lstat(*out); !os.IsNotExist(e) {
		fatal("output_exists")
	}
	stage, e := os.CreateTemp(parent, ".seme-upb12-graph-")
	if e != nil {
		fatal(e)
	}
	stageName := stage.Name()
	keep := false
	defer func() {
		stage.Close()
		if !keep {
			os.Remove(stageName)
		}
	}()
	if _, e = stage.Write(value); e != nil {
		fatal(e)
	}
	if e = stage.Sync(); e != nil {
		fatal(e)
	}
	if e = stage.Close(); e != nil {
		fatal(e)
	}
	if e = os.Link(stageName, *out); e != nil {
		fatal(e)
	}
	if e = os.Remove(stageName); e != nil {
		_ = os.Remove(*out)
		fatal(e)
	}
	keep = true
}
func fatal(value any) { fmt.Fprintln(os.Stderr, "upb12-projection-graph:", value); os.Exit(1) }
