package main

import (
	"encoding/json"
	"fmt"
	"os"

	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 2 {
		fatal(fmt.Errorf("usage: pure-application-boundary PROGRAM.seme"))
	}
	data, err := os.ReadFile(os.Args[1])
	fatal(err)
	graph, err := wire.Decode(data)
	fatal(err)
	boundary, err := wasmtarget.CertifyPureApplicationBoundary(graph)
	fatal(err)
	encoded, err := json.MarshalIndent(boundary, "", "  ")
	fatal(err)
	_, err = os.Stdout.Write(append(encoded, '\n'))
	fatal(err)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "pure-application-boundary:", err)
		os.Exit(65)
	}
}
