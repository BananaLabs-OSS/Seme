package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"seme.local/reference/upb12authority"
	"seme.local/reference/upb12entry"
)

func main() {
	authority := flag.String("authority", "", "source-free UPB12 authority")
	graph := flag.String("graph", "", "compiled graph derived from the authority")
	from := flag.String("from", "Run", "current entry")
	to := flag.String("to", "DispatchControlled", "selected entry")
	out := flag.String("out", "", "new executable G1 view")
	flag.Parse()
	if flag.NArg() != 0 || *authority == "" || *graph == "" || *out == "" {
		fatal("arguments")
	}
	files, err := upb12authority.Load(*authority)
	if err != nil {
		fatal(err)
	}
	_ = files
	graphBytes, err := os.ReadFile(*graph)
	if err != nil {
		fatal(err)
	}
	value, err := upb12entry.Select(graphBytes, *from, *to)
	if err != nil {
		fatal(err)
	}
	if !filepath.IsAbs(*out) || filepath.Clean(*out) != *out {
		fatal("output")
	}
	parent := filepath.Dir(*out)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		fatal("parent")
	}
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fatal(err)
	}
	if _, err = file.Write(value); err != nil {
		file.Close()
		os.Remove(*out)
		fatal(err)
	}
	if err = file.Sync(); err != nil {
		file.Close()
		os.Remove(*out)
		fatal(err)
	}
	if err = file.Close(); err != nil {
		os.Remove(*out)
		fatal(err)
	}
}
func fatal(value any) { fmt.Fprintf(os.Stderr, "upb12-entry-view: %v\n", value); os.Exit(64) }
