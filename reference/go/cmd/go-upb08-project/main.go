// Command go-upb08-project creates ordinary Go projections from authenticated UPB08 authority.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"seme.local/reference/goupb08bundle"
	"seme.local/reference/goupb08cmdload"
	"strings"
	"time"
)

func main() {
	if e := run(context.Background(), os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "go-upb08-project:", e)
		os.Exit(65)
	}
}
func run(parent context.Context, args []string) error {
	f := flag.NewFlagSet("go-upb08-project", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var p goupb08cmdload.Paths
	p.Bind(f)
	to, module := f.String("to", "", "destination"), f.String("module", "", "module path")
	if e := f.Parse(args); e != nil || f.NArg() != 0 || *to == "" || *module == "" || !filepath.IsAbs(*to) || filepath.Clean(*to) != *to {
		return fmt.Errorf("arguments")
	}
	if _, e := os.Lstat(*to); e == nil || !os.IsNotExist(e) {
		return fmt.Errorf("destination_exists")
	}
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	loaded, e := goupb08cmdload.Load(ctx, p)
	if e != nil {
		return e
	}
	packages, e := goupb08bundle.Project(loaded)
	if e != nil {
		return e
	}
	if e = os.Mkdir(*to, 0700); e != nil {
		return e
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(*to)
		}
	}()
	for pkg, data := range packages {
		rel := ""
		if pkg != *module {
			if !strings.HasPrefix(pkg, *module+"/") {
				return fmt.Errorf("package_outside_module")
			}
			rel = strings.TrimPrefix(pkg, *module+"/")
		}
		dir := filepath.Join(*to, filepath.FromSlash(rel))
		if e = os.MkdirAll(dir, 0700); e != nil {
			return e
		}
		path := filepath.Join(dir, "seme_projected.go")
		if e = os.WriteFile(path, data, 0600); e != nil {
			return e
		}
	}
	complete = true
	return nil
}
