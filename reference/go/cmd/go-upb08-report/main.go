// Command go-upb08-report emits authenticated, source-free transport authority.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"seme.local/reference/goupb08cmdload"
	"seme.local/reference/goupb08report"
	"time"
)

func main() {
	if e := run(context.Background(), os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "go-upb08-report:", e)
		os.Exit(65)
	}
}
func run(parent context.Context, args []string, out io.Writer) error {
	f := flag.NewFlagSet("go-upb08-report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var p goupb08cmdload.Paths
	p.Bind(f)
	if e := f.Parse(args); e != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	loaded, e := goupb08cmdload.Load(ctx, p)
	if e != nil {
		return e
	}
	r, e := goupb08report.Inspect(loaded)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}
