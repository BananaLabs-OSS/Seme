// Command wasm-result-check compares Wasm result/effect behavior with Go's
// fixed-width semantics and the ordinary fixture's logging contract.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type result struct {
	Result bool   `json:"result"`
	Events []bool `json:"events"`
}

func main() {
	if len(os.Args) != 5 {
		fatal("usage: wasm-result-check CURRENT DELTA LIMIT RESULT.json")
	}
	current, delta, limit := parse(os.Args[1]), parse(os.Args[2]), parse(os.Args[3])
	data, err := os.ReadFile(os.Args[4])
	check(err)
	var got result
	check(json.Unmarshal(data, &got))
	want := current+delta <= limit
	if got.Result != want || len(got.Events) != 1 || got.Events[0] != want {
		fatal("Wasm result/effect = %v/%v, Go result/log decision = %v", got.Result, got.Events, want)
	}
}
func parse(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	check(err)
	return parsed
}
func check(err error) {
	if err != nil {
		fatal("%v", err)
	}
}
func fatal(format string, values ...any) {
	fmt.Fprintf(os.Stderr, "wasm-result-check: "+format+"\n", values...)
	os.Exit(65)
}
