package wasmtarget

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

// These tests deliberately cross the Wasm boundary. Instruction-shape tests can
// miss invalid local declarations, bad packed descriptors, and memory.copy bugs.
func TestUAB04GenericCollectionChainsExecuteInNode(t *testing.T) {
	t.Run("slice construct update append remove index length fold", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		one, two, three, twenty, four, zero := b.literal(1), b.literal(2), b.literal(3), b.literal(20), b.literal(4), b.literal(0)
		constructed := b.entity(0xa068, map[uint64]wire.Value{0xa0680: ref(b.sliceType), 0xa0681: refs(one, two, three)})
		updated := b.entity(0x90fc, map[uint64]wire.Value{0x9fc0: ref(constructed), 0x9fc1: ref(one), 0x9fc2: ref(twenty)})
		appended := b.entity(0x90fb, map[uint64]wire.Value{0x9fb0: ref(updated), 0x9fb1: ref(four)})
		removed := b.entity(0xa066, map[uint64]wire.Value{0xa0660: ref(appended), 0xa0661: ref(zero)})
		indexed := b.entity(0x90fa, map[uint64]wire.Value{0x9fa0: ref(removed), 0x9fa1: ref(zero)})
		length := b.entity(0x90f9, map[uint64]wire.Value{0x9f90: ref(removed)})
		acc := b.binding("acc")
		element := b.binding("element")
		accRead := b.entity(0x90f6, map[uint64]wire.Value{0x9f60: ref(acc)})
		elementRead := b.entity(0x90f6, map[uint64]wire.Value{0x9f60: ref(element)})
		foldBody := b.add(accRead, elementRead)
		fold := b.entity(0x90f7, map[uint64]wire.Value{0x9f70: ref(removed), 0x9f71: ref(zero), 0x9f72: ref(acc), 0x9f73: ref(element), 0x9f74: ref(foldBody)})
		b.finish(g, b.add(b.add(indexed, length), fold)) // 20 + 3 + (20+3+4) = 50
		runNodeI64(t, g, 50, true)
	})

	t.Run("empty slice is a valid zero descriptor", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		empty := b.entity(0xa068, map[uint64]wire.Value{0xa0680: ref(b.sliceType), 0xa0681: refs()})
		length := b.entity(0x90f9, map[uint64]wire.Value{0x9f90: ref(empty)})
		b.finish(g, length)
		runNodeI64(t, g, 0, true)
	})

	t.Run("symbolic map shadow remove reinsert missing and signed keys", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		minusOne, seven, nine, eleven, missing := b.literal(-1), b.literal(7), b.literal(9), b.literal(11), b.literal(-2)
		mapType := b.entity(0xa040, map[uint64]wire.Value{0xa0400: ref(b.i64Type), 0xa0401: ref(b.i64Type)})
		empty := b.entity(0xa041, map[uint64]wire.Value{0xa0410: ref(mapType)})
		first := b.entity(0xa043, map[uint64]wire.Value{0xa0430: ref(empty), 0xa0431: ref(minusOne), 0xa0432: ref(seven)})
		shadow := b.entity(0xa043, map[uint64]wire.Value{0xa0430: ref(first), 0xa0431: ref(minusOne), 0xa0432: ref(nine)})
		removed := b.entity(0xa067, map[uint64]wire.Value{0xa0670: ref(shadow), 0xa0671: ref(minusOne)})
		removeMissing := b.entity(0xa067, map[uint64]wire.Value{0xa0670: ref(removed), 0xa0671: ref(missing)})
		reinsert := b.entity(0xa043, map[uint64]wire.Value{0xa0430: ref(removeMissing), 0xa0431: ref(minusOne), 0xa0432: ref(eleven)})
		lookupSigned := b.entity(0xa042, map[uint64]wire.Value{0xa0420: ref(reinsert), 0xa0421: ref(minusOne)})
		lookupMissing := b.entity(0xa042, map[uint64]wire.Value{0xa0420: ref(reinsert), 0xa0421: ref(missing)})
		b.finish(g, b.add(lookupSigned, lookupMissing))
		runNodeI64(t, g, 11, true)
	})
}

func TestUAB04CollectionBoundsAndAdversarialGraphs(t *testing.T) {
	t.Run("512 elements execute and 513 are rejected", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		items := make([]wire.ID, 512)
		for i := range items {
			items[i] = b.literal(int64(i))
		}
		full := b.entity(0xa068, map[uint64]wire.Value{0xa0680: ref(b.sliceType), 0xa0681: refs(items...)})
		length := b.entity(0x90f9, map[uint64]wire.Value{0x9f90: ref(full)})
		b.finish(g, length)
		runNodeI64(t, g, 512, true)

		g, b = runtimeCollectionGraph()
		items = make([]wire.ID, 513)
		for i := range items {
			items[i] = b.literal(int64(i))
		}
		tooMany := b.entity(0xa068, map[uint64]wire.Value{0xa0680: ref(b.sliceType), 0xa0681: refs(items...)})
		tooManyLength := b.entity(0x90f9, map[uint64]wire.Value{0x9f90: ref(tooMany)})
		b.finish(g, tooManyLength)
		if _, err := CertifyPureFunction(g); err == nil || !strings.Contains(err.Error(), "collection_length_collection") {
			t.Fatalf("513-element construct error = %v", err)
		}
	})

	for _, index := range []int64{-1, 3} {
		t.Run(fmt.Sprintf("index_%d_traps", index), func(t *testing.T) {
			g, b := runtimeCollectionGraph()
			items := []wire.ID{b.literal(1), b.literal(2), b.literal(3)}
			slice := b.entity(0xa068, map[uint64]wire.Value{0xa0680: ref(b.sliceType), 0xa0681: refs(items...)})
			read := b.entity(0x90fa, map[uint64]wire.Value{0x9fa0: ref(slice), 0x9fa1: ref(b.literal(index))})
			b.finish(g, read)
			runNodeI64(t, g, 0, false)
		})
	}

	t.Run("slice cycle", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		cycle := b.entity(0x90fb, map[uint64]wire.Value{})
		e := g.Entities[cycle]
		e.Fields = map[wire.ID]wire.Value{identity(0x9fb0): ref(cycle), identity(0x9fb1): ref(b.literal(1))}
		g.Entities[cycle] = e
		length := b.entity(0x90f9, map[uint64]wire.Value{0x9f90: ref(cycle)})
		b.finish(g, length)
		if _, err := CertifyPureFunction(g); err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("cycle error = %v", err)
		}
	})

	t.Run("symbolic map cycle", func(t *testing.T) {
		g, b := runtimeCollectionGraph()
		key := b.literal(1)
		cycle := b.entity(0xa067, map[uint64]wire.Value{})
		e := g.Entities[cycle]
		e.Fields = map[wire.ID]wire.Value{identity(0xa0670): ref(cycle), identity(0xa0671): ref(key)}
		g.Entities[cycle] = e
		lookup := b.entity(0xa042, map[uint64]wire.Value{0xa0420: ref(cycle), 0xa0421: ref(key)})
		b.finish(g, lookup)
		if _, err := CertifyPureFunction(g); err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("cycle error = %v", err)
		}
	})
}

type collectionGraphBuilder struct {
	graph                                         *wire.Envelope
	next                                          uint64
	i64Type, sliceType, function, block, returned wire.ID
}

func runtimeCollectionGraph() (wire.Envelope, *collectionGraphBuilder) {
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	b := &collectionGraphBuilder{graph: &g, next: 0xd100}
	b.i64Type = b.entityIn(g, 0x9010, map[uint64]wire.Value{0x9100: unsigned(64), 0x9101: {Tag: 2}, 0x9102: unsigned(0)})
	b.sliceType = b.entityIn(g, 0x90f8, map[uint64]wire.Value{0x9f80: ref(b.i64Type)})
	parameter := b.entityIn(g, 0x9012, map[uint64]wire.Value{0x9120: byteValue("activation"), 0x9121: ref(b.sliceType), 0x9122: unsigned(0)})
	b.returned = b.entityIn(g, 0x9081, map[uint64]wire.Value{})
	b.block = b.entityIn(g, 0x9080, map[uint64]wire.Value{0x9800: refs(b.returned)})
	b.function = b.entityIn(g, 0x9011, map[uint64]wire.Value{0x9110: byteValue("Evaluate"), 0x9111: refs(parameter), 0x9112: ref(b.i64Type), 0x9113: ref(b.block)})
	program := b.entityIn(g, 0x9015, map[uint64]wire.Value{0x9150: refs(b.function), 0x9151: ref(b.function)})
	_ = program
	return g, b
}

func (b *collectionGraphBuilder) entity(schema uint64, fields map[uint64]wire.Value) wire.ID {
	return b.entityIn(*b.graph, schema, fields)
}

func (b *collectionGraphBuilder) entityIn(g wire.Envelope, schema uint64, fields map[uint64]wire.Value) wire.ID {
	id := identity(b.next)
	b.next++
	wf := map[wire.ID]wire.Value{}
	for k, v := range fields {
		wf[identity(k)] = v
	}
	g.Entities[id] = wire.Entity{ID: id, Schema: identity(schema), Fields: wf}
	return id
}

func (b *collectionGraphBuilder) literal(v int64) wire.ID {
	return b.entity(0x9070, map[uint64]wire.Value{0x9700: unsigned(uint64(v)), 0x9701: ref(b.i64Type)})
}
func (b *collectionGraphBuilder) binding(name string) wire.ID {
	return b.entity(0x90f5, map[uint64]wire.Value{0x9f50: byteValue(name), 0x9f51: ref(b.i64Type)})
}
func (b *collectionGraphBuilder) add(a, c wire.ID) wire.ID {
	return b.entity(0x9014, map[uint64]wire.Value{0x9140: ref(a), 0x9141: ref(c), 0x9142: ref(b.i64Type)})
}
func (b *collectionGraphBuilder) finish(g wire.Envelope, result wire.ID) {
	e := g.Entities[b.returned]
	e.Fields = map[wire.ID]wire.Value{identity(0x9810): refs(result)}
	g.Entities[b.returned] = e
}

func runNodeI64(t *testing.T, graph wire.Envelope, expected int64, success bool) {
	t.Helper()
	certificate, err := CertifyPureFunction(graph)
	if err != nil {
		t.Fatal(err)
	}
	wasm, _, err := LowerCertifiedPureFunction(certificate)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "program.wasm")
	if err := os.WriteFile(path, wasm, 0o600); err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 8)
	binary.LittleEndian.PutUint64(want, uint64(expected))
	cmd := exec.Command("node", filepath.Join("..", "..", "js", "pure-function-v2-runner.mjs"), path, "0800000000000000", fmt.Sprintf("%x", want))
	output, err := cmd.CombinedOutput()
	if success && err != nil {
		t.Fatalf("node runtime: %v\n%s", err, output)
	}
	if !success && err == nil {
		t.Fatalf("expected Wasm trap, got %s", output)
	}
}
