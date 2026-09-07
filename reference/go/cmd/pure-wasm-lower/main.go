// Command pure-wasm-lower lowers a canonical structured pure function to a
// runnable Pulp-compatible WebAssembly cell and emits its derived ABI.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 4 {
		fatal(fmt.Errorf("usage: pure-wasm-lower PROGRAM.seme OUTPUT.wasm ABI.json"))
	}
	graph, err := wire.Read(os.Args[1])
	fatal(err)
	wasm, abi, err := wasmtarget.LowerPureFunction(graph)
	fatal(err)
	fatal(os.WriteFile(os.Args[2], wasm, 0o644))
	encoded, err := json.MarshalIndent(abi, "", "  ")
	fatal(err)
	encoded = append(encoded, '\n')
	fatal(os.WriteFile(os.Args[3], encoded, 0o644))
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "pure-wasm-lower:", err)
		os.Exit(65)
	}
}
