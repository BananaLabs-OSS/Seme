// Command go-upb10-report independently reopens and reports one authenticated
// source-free UPB-10 mixed placement.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/goupb10report"
	"seme.local/reference/targetplaninstance"
)

func main() {
	set := flag.NewFlagSet("go-upb10-report", flag.ExitOnError)
	var paths goupb10cmdload.Paths
	paths.Bind(set)
	name := set.String("target-name", "wasm32-pulp-go-host-v1", "target identity")
	revision := set.Uint64("target-revision", 1, "target revision")
	set.Parse(os.Args[1:])
	if set.NArg() != 0 || *name == "" || *revision == 0 {
		fmt.Fprintln(os.Stderr, "go-upb10-report: arguments")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	loaded, err := goupb10cmdload.Load(ctx, paths, goprojectplacementadapter.Policy{Name: *name, Revision: *revision, AllowedFidelity: []targetplaninstance.Fidelity{targetplaninstance.Exact, targetplaninstance.NativeIsland}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-upb10-report:", err)
		os.Exit(1)
	}
	report, err := goupb10report.Inspect(loaded.Bundle)
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-upb10-report:", err)
		os.Exit(1)
	}
	encoded, err := goupb10report.Marshal(report)
	if err != nil || !bytes.Equal(encoded, loaded.Placement.Report) {
		fmt.Fprintln(os.Stderr, "go-upb10-report: stale_report")
		os.Exit(1)
	}
	os.Stdout.Write(encoded)
	os.Stdout.Write([]byte("\n"))
}
