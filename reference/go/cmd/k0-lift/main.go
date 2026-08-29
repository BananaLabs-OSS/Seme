// Command k0-lift deterministically lifts a checked K0-P2 image into the
// canonical K0 semantic module's G1 construction projection.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

type instruction struct {
	opcode, operand, target, callee  uint64
	hasOperand, hasTarget, hasCallee bool
}
type function struct {
	parameters, locals uint64
	instructions       []instruction
}
type decoder struct {
	data []byte
	at   int
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: k0-lift IMAGE")
		os.Exit(64)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	d := decoder{data: data}
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{'S', 'E', 'M', 'E', '-', 'K', '0', 0}) {
		fail(fmt.Errorf("invalid K0 magic"))
	}
	d.at = 8
	version := d.uleb()
	if version != 3 {
		fail(fmt.Errorf("K0 profile %d is not P2", version))
	}
	capabilities := d.uleb()
	count := d.uleb()
	if count == 0 || count > 16 {
		fail(fmt.Errorf("invalid function count %d", count))
	}
	functions := make([]function, count)
	for i := range functions {
		fn := function{parameters: d.uleb(), locals: d.uleb()}
		instructionCount, codeSize := d.uleb(), d.uleb()
		end := d.at + int(codeSize)
		if end > len(data) {
			fail(fmt.Errorf("truncated function %d", i))
		}
		for j := uint64(0); j < instructionCount; j++ {
			fn.instructions = append(fn.instructions, d.instruction())
		}
		if d.at != end {
			fail(fmt.Errorf("function %d code size mismatch", i))
		}
		functions[i] = fn
	}
	entry := d.uleb()
	if entry >= count || d.at != len(data) {
		fail(fmt.Errorf("invalid entry or trailing bytes"))
	}
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	entityCount := 1 + len(functions)
	for _, fn := range functions {
		entityCount += len(fn.instructions)
	}
	fmt.Fprintf(w, "# Deterministically lifted K0-P2 semantic graph.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0xa000), id(0xc001), entityCount)
	fmt.Fprintf(w, "en %s %s 1 4\nfi %s uu 3\nfi %s uu %d\nfi %s li %d\n", id(0xd000), id(0xa010), id(0xa100), id(0xa101), capabilities, id(0xa102), len(functions))
	for i := range functions {
		fmt.Fprintf(w, "rf %s\n", functionID(i))
	}
	fmt.Fprintf(w, "fi %s rf %s\n\n", id(0xa103), functionID(int(entry)))
	for i, fn := range functions {
		fmt.Fprintf(w, "en %s %s 1 3\nfi %s uu %d\nfi %s uu %d\nfi %s li %d\n", functionID(i), id(0xa011), id(0xa110), fn.parameters, id(0xa111), fn.locals, id(0xa112), len(fn.instructions))
		for j := range fn.instructions {
			fmt.Fprintf(w, "rf %s\n", instructionID(i, j))
		}
		fmt.Fprintln(w)
		for j, in := range fn.instructions {
			fields := 1
			if in.hasOperand || in.hasTarget || in.hasCallee {
				fields = 2
			}
			fmt.Fprintf(w, "en %s %s 1 %d\nfi %s uu %d\n", instructionID(i, j), id(0xa012), fields, id(0xa120), in.opcode)
			if in.hasOperand {
				fmt.Fprintf(w, "fi %s uu %d\n", id(0xa121), in.operand)
			}
			if in.hasTarget {
				if in.target >= uint64(len(fn.instructions)) {
					fail(fmt.Errorf("invalid branch target"))
				}
				fmt.Fprintf(w, "fi %s rf %s\n", id(0xa122), instructionID(i, int(in.target)))
			}
			if in.hasCallee {
				if in.callee >= uint64(len(functions)) {
					fail(fmt.Errorf("invalid callee"))
				}
				fmt.Fprintf(w, "fi %s rf %s\n", id(0xa123), functionID(int(in.callee)))
			}
			fmt.Fprintln(w)
		}
	}
}

func (d *decoder) instruction() instruction {
	if d.at >= len(d.data) {
		fail(fmt.Errorf("truncated opcode"))
	}
	in := instruction{opcode: uint64(d.data[d.at])}
	d.at++
	switch in.opcode {
	case 1, 2, 3:
		in.operand = d.uleb()
		in.hasOperand = true
	case 32, 33:
		in.target = d.uleb()
		_ = d.uleb()
		in.hasTarget = true
	case 48:
		in.callee = d.uleb()
		in.hasCallee = true
	}
	return in
}
func (d *decoder) uleb() uint64 {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if d.at >= len(d.data) {
			fail(fmt.Errorf("truncated ULEB"))
		}
		b := d.data[d.at]
		d.at++
		if shift == 63 && b > 1 {
			fail(fmt.Errorf("ULEB overflow"))
		}
		value |= uint64(b&127) << shift
		if b&128 == 0 {
			return value
		}
	}
	fail(fmt.Errorf("ULEB overflow"))
	return 0
}
func functionID(index int) string { return id(0x100000 + uint64(index)*0x10000) }
func instructionID(function, index int) string {
	return id(0x100001 + uint64(function)*0x10000 + uint64(index))
}
func id(value uint64) string { return fmt.Sprintf("%032x", value) }
func fail(err error)         { fmt.Fprintln(os.Stderr, "k0-lift:", err); os.Exit(65) }
