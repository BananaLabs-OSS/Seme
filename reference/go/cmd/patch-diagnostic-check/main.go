// Command patch-diagnostic-check independently verifies canonical Patch v1
// rejection reports. It is differential evidence, not part of the trusted path.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
)

type value struct {
	tag    byte
	u      uint64
	b      []byte
	ref    []byte
	list   []value
	record map[string]value
}
type entity struct {
	id, schema []byte
	version    uint64
	fields     map[string]value
}
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
func (r *reader) uleb() uint64 {
	var v uint64
	for shift := uint(0); shift < 64; shift += 7 {
		b := r.take(1)[0]
		v |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return v
		}
	}
	panic("overflowing uleb")
}
func (r *reader) val() value {
	t := r.take(1)[0]
	v := value{tag: t}
	switch t {
	case 0, 1, 2:
	case 3, 4:
		v.u = r.uleb()
	case 5:
		v.b = append([]byte(nil), r.take(int(r.uleb()))...)
	case 6:
		v.ref = append([]byte(nil), r.take(16)...)
	case 7:
		for n := r.uleb(); n > 0; n-- {
			v.list = append(v.list, r.val())
		}
	case 8:
		v.record = map[string]value{}
		for n := r.uleb(); n > 0; n-- {
			v.record[hex.EncodeToString(r.take(16))] = r.val()
		}
	case 9:
		r.take(16)
	default:
		panic(fmt.Sprintf("unknown value tag %d", t))
	}
	return v
}
func low(n uint16) []byte { b := make([]byte, 16); b[14], b[15] = byte(n>>8), byte(n); return b }
func key(n uint16) string { return hex.EncodeToString(low(n)) }
func must(ok bool, format string, args ...any) {
	if !ok {
		panic(fmt.Sprintf(format, args...))
	}
}
func field(e entity, n uint16) value {
	v, ok := e.fields[key(n)]
	must(ok, "missing field %04x", n)
	return v
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: patch-diagnostic-check REPORT MODE")
		os.Exit(64)
	}
	wants := map[string]struct {
		name             string
		rule             uint16
		index            uint64
		target, field    uint16
		expected, actual string
	}{
		"stale":             {"patch.stale_revision", 0x5601, ^uint64(0), 0, 0, "", ""},
		"unknown-target":    {"patch.unknown_target", 0x5602, 0, 0x5401, 0x5310, "", ""},
		"unknown-field":     {"patch.unknown_field", 0x5603, 0, 0x5400, 0x5311, "", ""},
		"precondition":      {"patch.precondition_failed", 0x5604, 0, 0x5400, 0x5310, "Wrong", "Greeting"},
		"duplicate":         {"patch.duplicate_write", 0x5605, 1, 0x5400, 0x5310, "", ""},
		"invalid-candidate": {"patch.invalid_candidate", 0x5606, 0, 0x5400, 0x5310, "", ""},
	}
	w, ok := wants[os.Args[2]]
	must(ok, "unknown mode %q", os.Args[2])
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	r := reader{b: b}
	must(bytes.Equal(r.take(8), []byte("SEMEK1\r\n")), "bad magic")
	must(r.uleb() == 1, "bad wire version")
	must(bytes.Equal(r.take(16), low(0x5701)), "bad module")
	revision := append([]byte(nil), r.take(16)...)
	must(bytes.Equal(revision, low(0x5500)), "bad report revision")
	must(r.uleb() == 0 && r.uleb() == 3, "bad envelope counts")
	entities := map[string]entity{}
	for i := 0; i < 3; i++ {
		e := entity{id: append([]byte(nil), r.take(16)...), schema: append([]byte(nil), r.take(16)...), version: r.uleb(), fields: map[string]value{}}
		for n := r.uleb(); n > 0; n-- {
			e.fields[hex.EncodeToString(r.take(16))] = r.val()
		}
		entities[hex.EncodeToString(e.id)] = e
	}
	must(r.p == len(r.b), "trailing bytes: parsed %d of %d", r.p, len(r.b))
	rule, ok := entities[key(w.rule)]
	must(ok, "missing rule")
	must(bytes.Equal(rule.schema, low(0x19)) && rule.version == 1, "bad rule header")
	must(string(field(rule, 0x190).b) == w.name, "bad rule name")
	must(len(field(rule, 0x191).list) == 2, "bad rule argument shapes")
	diagnostic, ok := entities[key(0x5700)]
	must(ok, "missing diagnostic")
	must(bytes.Equal(field(diagnostic, 0x1a0).ref, low(w.rule)), "bad diagnostic rule")
	must(bytes.Equal(field(diagnostic, 0x1a1).b, revision), "bad diagnostic revision")
	entityID := field(diagnostic, 0x1a2).b
	if w.target == 0 {
		must(len(entityID) == 0, "unexpected target")
	} else {
		must(bytes.Equal(entityID, low(w.target)), "bad target")
	}
	path := field(diagnostic, 0x1a3).list
	wantPath := 0
	if w.index != ^uint64(0) {
		wantPath++
	}
	if w.target != 0 {
		wantPath++
	}
	if w.field != 0 {
		wantPath++
	}
	must(len(path) == wantPath, "path count got %d want %d", len(path), wantPath)
	p := 0
	if w.index != ^uint64(0) {
		must(path[p].record[key(0x2100)].u == 2 && path[p].record[key(0x2102)].u == w.index, "bad operation path")
		p++
	}
	if w.target != 0 {
		must(path[p].record[key(0x2100)].u == 0 && bytes.Equal(path[p].record[key(0x2101)].b, low(w.target)), "bad target path")
		p++
	}
	if w.field != 0 {
		must(path[p].record[key(0x2100)].u == 1 && bytes.Equal(path[p].record[key(0x2101)].b, low(w.field)), "bad field path")
	}
	arguments := field(diagnostic, 0x1a4).list
	must(len(arguments) == 2 && string(arguments[0].b) == w.expected && string(arguments[1].b) == w.actual, "bad expected/actual arguments")
	must(field(diagnostic, 0x1a5).u == 0, "bad severity")
	report, ok := entities[key(0x5701)]
	must(ok, "missing report")
	must(field(report, 0x1b0).u == 3, "bad report status")
	diagnostics := field(report, 0x1b1).list
	must(len(diagnostics) == 1 && bytes.Equal(diagnostics[0].ref, low(0x5700)), "bad diagnostic list")
}
