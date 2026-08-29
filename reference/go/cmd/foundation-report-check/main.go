// Command foundation-report-check is a differential fixture checker for the
// canonical Foundation reporter. It is not part of Seme's trusted path.
package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

type reader struct {
	b []byte
	p int
}

func (r *reader) take(n int) []byte {
	if n < 0 || r.p+n > len(r.b) {
		panic("truncated report")
	}
	v := r.b[r.p : r.p+n]
	r.p += n
	return v
}
func (r *reader) byte(want byte) {
	if got := r.take(1)[0]; got != want {
		panic(fmt.Sprintf("byte: got %d want %d", got, want))
	}
}
func (r *reader) uleb() uint64 {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		b := r.take(1)[0]
		value |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return value
		}
	}
	panic("overflowing uleb")
}
func id(n uint16) []byte {
	b := make([]byte, 16)
	b[14], b[15] = byte(n>>8), byte(n)
	return b
}
func (r *reader) identity(want uint16) {
	if got := r.take(16); !bytes.Equal(got, id(want)) {
		panic(fmt.Sprintf("identity: got %x want %x", got, id(want)))
	}
}

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: foundation-report-check REPORT TOTAL CERTIFIED UNKNOWN UNAVAILABLE")
		os.Exit(64)
	}
	want := make([]uint64, 4)
	for i := 1; i < len(want); i++ {
		parsed, err := strconv.ParseUint(os.Args[i+2], 10, 64)
		if err != nil {
			panic(err)
		}
		want[i] = parsed
	}
	total, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		panic(err)
	}

	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	r := reader{b: b}
	if !bytes.Equal(r.take(8), []byte("SEMEK1\r\n")) || r.uleb() != 1 {
		panic("invalid envelope")
	}
	r.identity(0x4000)
	r.take(16) // projection revision
	if r.uleb() != 0 || r.uleb() != 1 {
		panic("unexpected envelope shape")
	}
	r.identity(0x4000)
	r.identity(0x001f)
	if r.uleb() != 1 || r.uleb() != 2 {
		panic("unexpected report entity")
	}
	r.identity(0x2210)
	r.byte(5)
	if r.uleb() != 16 {
		panic("invalid revision value")
	}
	r.take(16)
	r.identity(0x2211)
	r.byte(7)
	count := r.uleb()
	if count != total {
		panic(fmt.Sprintf("results: got %d want %d", count, total))
	}
	got := make([]uint64, 4)
	for i := uint64(0); i < count; i++ {
		r.byte(8)
		if r.uleb() != 2 {
			panic("invalid result record")
		}
		r.identity(0x2200)
		r.byte(5)
		if r.uleb() != 16 {
			panic("invalid result identity")
		}
		r.take(16)
		r.identity(0x2201)
		r.byte(3)
		disposition := r.uleb()
		if disposition > 2 {
			panic("invalid successful disposition")
		}
		got[disposition+1]++
	}
	if r.p != len(r.b) {
		panic("trailing report bytes")
	}
	for i := 1; i < len(got); i++ {
		if got[i] != want[i] {
			panic(fmt.Sprintf("disposition %d: got %d want %d", i-1, got[i], want[i]))
		}
	}
}
