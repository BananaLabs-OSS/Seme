// Command wasm-lower lowers the checked scoped Target Contract plan to Wasm.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 3 {
		fatal(fmt.Errorf("usage: wasm-lower PLAN.seme OUTPUT.wasm"))
	}
	plan, err := wire.Read(os.Args[1])
	fatal(err)
	wasm, err := wasmtarget.Lower(plan)
	fatal(err)
	fatal(os.WriteFile(os.Args[2], wasm, 0o644))
}
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "wasm-lower:", err)
		os.Exit(65)
	}
}
