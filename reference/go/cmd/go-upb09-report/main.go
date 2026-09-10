// Command go-upb09-report emits authenticated source-free effects authority.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"seme.local/reference/goupb09cmdload"
	"seme.local/reference/goupb09report"
	"time"
)

func main() {
	if e := run(context.Background(), os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "go-upb09-report:", e)
		os.Exit(65)
	}
}
func run(parent context.Context, args []string, out io.Writer) error {
	f := flag.NewFlagSet("go-upb09-report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var p goupb09cmdload.Paths
	p.Bind(f)
	if e := f.Parse(args); e != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	loaded, e := goupb09cmdload.Load(ctx, p)
	if e != nil {
		return e
	}
	r, e := goupb09report.Inspect(loaded.Bundle)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}
