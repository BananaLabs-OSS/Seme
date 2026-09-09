package wasmtarget

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestDynamicIndexTargetIsStructuralAndShortCircuiting(t *testing.T) {
	for _, tc := range []struct {
		name, operator                string
		schema, leftField, rightField uint64
		ifOpcode                      byte
	}{
		{"and", "and", 0x90b1, 0x9b10, 0x9b11, 0x04},
		{"or", "or", 0x90c1, 0x9c10, 0x9c11, 0x04},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, root, parameter := targetDynamicBooleanGraph(tc.schema, tc.leftField, tc.rightField)
			if err := validateIntegerTypeReference(g, g.Entities[identity(0xe109)], 0x9162); err != nil {
				t.Fatalf("comparison profile: %v", err)
			}
			parameterTypes := map[wire.ID]string{parameter: "slice:i64"}
			budget := 128
			if err := validatePureExpression(g, root, "bool", parameterTypes, map[wire.ID]bool{}, &budget); err != nil {
				t.Fatal(err)
			}
			budget = 128
			code, err := lowerHelperBoolean(g, root, map[wire.ID]byte{parameter: 0}, map[byte]bool{}, map[wire.ID]bool{}, &budget)
			if err != nil {
				t.Fatal(err)
			}
			// Both operators must lower with Wasm structured control flow. The
			// dynamic load must occur inside the conditional region, never before it.
			branch := bytes.IndexByte(code, tc.ifOpcode)
			load := bytes.Index(code, []byte{0x29, 0x03, 0x00})
			if branch < 0 || load <= branch {
				t.Fatalf("%s instructions are not short-circuit ordered: %x", tc.operator, code)
			}
		})
	}
}

func TestDynamicIndexTargetRejectsMalformedTypeCycleAndOwnership(t *testing.T) {
	base, root, parameter := targetDynamicBooleanGraph(0x90b1, 0x9b10, 0x9b11)
	tests := []struct {
		name, want string
		edit       func(map[wire.ID]wire.Entity)
	}{
		{"missing index", "wasm.dynamic_index_fields", func(e map[wire.ID]wire.Entity) {
			x := e[identity(0xe107)]
			delete(x.Fields, identity(0x9fa1))
			e[x.ID] = x
		}},
		{"wrong collection type", "wasm.", func(e map[wire.ID]wire.Entity) {
			x := e[identity(0xe107)]
			x.Fields[identity(0x9fa0)] = ref(identity(0xe106))
			e[x.ID] = x
		}},
		{"index cycle", "cycle", func(e map[wire.ID]wire.Entity) {
			x := e[identity(0xe107)]
			x.Fields[identity(0x9fa1)] = ref(x.ID)
			e[x.ID] = x
		}},
		{"foreign parameter", "wasm.", func(e map[wire.ID]wire.Entity) {
			x := e[identity(0xe104)]
			x.Fields[identity(0x9130)] = ref(identity(0xefff))
			e[x.ID] = x
		}},
		{"slice element type", "wasm.", func(e map[wire.ID]wire.Entity) {
			x := e[identity(0xe101)]
			x.Fields[identity(0x9f80)] = ref(identity(0xe103))
			e[x.ID] = x
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := cloneTargetGraph(base)
			tc.edit(g.Entities)
			budget := 64
			err := validatePureExpression(g, root, "bool", map[wire.ID]string{parameter: "slice:i64"}, map[wire.ID]bool{}, &budget)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%s", err, tc.want)
			}
		})
	}
}

func TestDynamicIndexSliceDescriptorValidation(t *testing.T) {
	g, _, _ := targetDynamicBooleanGraph(0x90b1, 0x9b10, 0x9b11)
	layout, err := CertifyPureValueLayout(g, identity(0xe101))
	if err != nil {
		t.Fatal(err)
	}
	valid := make([]byte, 24)
	binary.LittleEndian.PutUint32(valid, 8)
	binary.LittleEndian.PutUint32(valid[4:], 2)
	if err := ValidatePureValueBytes(layout, valid); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{
		valid[:7],
		func() []byte { x := append([]byte(nil), valid...); binary.LittleEndian.PutUint32(x[4:], 513); return x }(),
		func() []byte { x := append([]byte(nil), valid...); binary.LittleEndian.PutUint32(x, 7); return x }(),
		func() []byte { x := append([]byte(nil), valid...); binary.LittleEndian.PutUint32(x, 16); return x }(),
	} {
		if ValidatePureValueBytes(layout, bad) == nil {
			t.Fatalf("malformed descriptor accepted: %x", bad)
		}
	}
}

func targetDynamicBooleanGraph(schema, leftField, rightField uint64) (wire.Envelope, wire.ID, wire.ID) {
	i64, sliceType, parameter, boolType := identity(0xe100), identity(0xe101), identity(0xe102), identity(0xe103)
	read, indexLiteral, dynamic, zero, compare, left, root := identity(0xe104), identity(0xe106), identity(0xe107), identity(0xe108), identity(0xe109), identity(0xe10a), identity(0xe10b)
	e := map[wire.ID]wire.Entity{
		i64: {ID: i64, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{identity(0x9100): unsigned(64), identity(0x9101): {Tag: 2}, identity(0x9102): unsigned(0)}}, sliceType: {ID: sliceType, Schema: identity(0x90f8), Fields: map[wire.ID]wire.Value{identity(0x9f80): ref(i64)}},
		parameter: {ID: parameter, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9121): ref(sliceType)}}, boolType: {ID: boolType, Schema: identity(0x9020)},
		read:         {ID: read, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{identity(0x9130): ref(parameter)}},
		indexLiteral: {ID: indexLiteral, Schema: identity(0x9070), Fields: map[wire.ID]wire.Value{identity(0x9700): unsigned(2), identity(0x9701): ref(i64)}},
		dynamic:      {ID: dynamic, Schema: identity(0x90fa), Fields: map[wire.ID]wire.Value{identity(0x9fa0): ref(read), identity(0x9fa1): ref(indexLiteral)}},
		zero:         {ID: zero, Schema: identity(0x9070), Fields: map[wire.ID]wire.Value{identity(0x9700): unsigned(0), identity(0x9701): ref(i64)}},
		compare:      {ID: compare, Schema: identity(0x9021), Fields: map[wire.ID]wire.Value{identity(0x9160): ref(dynamic), identity(0x9161): ref(zero), identity(0x9162): ref(i64)}},
		left:         {ID: left, Schema: identity(0x90b0), Fields: map[wire.ID]wire.Value{identity(0x9b00): {Tag: 1}}},
		root:         {ID: root, Schema: identity(schema), Fields: map[wire.ID]wire.Value{identity(leftField): ref(left), identity(rightField): ref(compare)}},
	}
	return wire.Envelope{Entities: e}, root, parameter
}

func cloneTargetGraph(g wire.Envelope) wire.Envelope {
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for k, e := range g.Entities {
		f := map[wire.ID]wire.Value{}
		for fk, v := range e.Fields {
			f[fk] = v
		}
		e.Fields = f
		out.Entities[k] = e
	}
	return out
}
