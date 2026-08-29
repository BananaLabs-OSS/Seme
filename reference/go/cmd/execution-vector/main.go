// Command execution-vector bridges human-readable test vectors and the frozen
// Core Execution v1 binary invocation contract. It is test tooling, not part of
// the Seme runtime.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 5 || (os.Args[1] != "write" && os.Args[1] != "check") {
		fmt.Fprintln(os.Stderr, "usage: execution-vector write|check A B FILE")
		os.Exit(64)
	}
	a := parse(os.Args[2])
	b := parse(os.Args[3])
	if os.Args[1] == "write" {
		data := make([]byte, 16)
		binary.LittleEndian.PutUint64(data[0:8], uint64(a))
		binary.LittleEndian.PutUint64(data[8:16], uint64(b))
		check(os.WriteFile(os.Args[4], data, 0o644))
		return
	}
	data, err := os.ReadFile(os.Args[4])
	check(err)
	if len(data) != 8 {
		fatal("result has %d bytes, want 8", len(data))
	}
	got := int64(binary.LittleEndian.Uint64(data))
	// Conversion back to int64 defines the same modular two's-complement result
	// that ordinary Go's fixed-width signed addition produces.
	want := int64(uint64(a) + uint64(b))
	if got != want {
		fatal("Seme result for %d + %d is %d, Go result is %d", a, b, got, want)
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
	fmt.Fprintf(os.Stderr, "execution-vector: "+format+"\n", values...)
	os.Exit(65)
}
