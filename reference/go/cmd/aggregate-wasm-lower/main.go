package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 4 { fatal(fmt.Errorf("usage: aggregate-wasm-lower PROGRAM.seme OUTPUT.wasm ABI.json")) }
	program, err := os.ReadFile(os.Args[1]); fatal(err)
	graph, err := wire.Decode(program); fatal(err)
	certificate, err := wasmtarget.CertifyAggregateFunction(graph); fatal(err)
	wasm, abi, err := wasmtarget.LowerCertifiedAggregateFunction(certificate); fatal(err)
	abi.ProgramSHA256 = fmt.Sprintf("%x", sha256.Sum256(program))
	abi.ArtifactSHA256 = fmt.Sprintf("%x", sha256.Sum256(wasm))
	fatal(os.WriteFile(os.Args[2], wasm, 0o644))
	encoded, err := json.MarshalIndent(abi, "", "  "); fatal(err)
	fatal(os.WriteFile(os.Args[3], append(encoded, '\n'), 0o644))
}

func fatal(err error) {
	if err != nil { fmt.Fprintln(os.Stderr, "aggregate-wasm-lower:", err); os.Exit(65) }
}
