// Command wasm-error-check verifies the canonical ResultError response and
// confirms that the rejected request did not perform the logging effect.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type result struct {
	Error  string `json:"error"`
	Events []bool `json:"events"`
}

func main() {
	if len(os.Args) != 3 {
		fatal("usage: wasm-error-check RESULT.json EXPECTED-MESSAGE")
	}
	data, err := os.ReadFile(os.Args[1])
	check(err)
	var got result
	check(json.Unmarshal(data, &got))
	if got.Error != os.Args[2] || len(got.Events) != 0 {
		fatal("Wasm error/effects = %q/%v, want %q/[]", got.Error, got.Events, os.Args[2])
	}
}

func check(err error) {
	if err != nil {
		fatal("%v", err)
	}
}

func fatal(format string, values ...any) {
	fmt.Fprintf(os.Stderr, "wasm-error-check: "+format+"\n", values...)
	os.Exit(65)
}
