// Command patch-result-check independently verifies Patch v1 revision
// derivation and stamping. It is a differential oracle, not authority.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: patch-result-check BASE CANDIDATE FINAL")
		os.Exit(64)
	}
	base := read(os.Args[1])
	candidate := read(os.Args[2])
	final := read(os.Args[3])
	if len(base) < 41 || len(candidate) < 41 || len(final) != len(candidate) {
		panic("invalid envelope lengths")
	}
	if !bytes.Equal(candidate[25:41], make([]byte, 16)) {
		panic("candidate revision is not the zero placeholder")
	}
	transcript := append([]byte("seme.patch.revision.v1\x00"), base[25:41]...)
	transcript = append(transcript, candidate...)
	digest := sha256.Sum256(transcript)
	if !bytes.Equal(final[25:41], digest[:16]) {
		panic(fmt.Sprintf("revision: got %x want %x", final[25:41], digest[:16]))
	}
	want := append([]byte(nil), candidate...)
	copy(want[25:41], digest[:16])
	if !bytes.Equal(final, want) {
		panic("final differs from candidate outside revision identity")
	}
}

func read(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
