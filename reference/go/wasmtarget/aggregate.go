package wasmtarget

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

// AggregateFunctionCertificate is the executable certificate for the bounded
// record/array/slice/map UAB observation. Its proof is structural: source and
// function names are never consulted.
type AggregateFunctionCertificate struct {
	wasm []byte
	abi  PureCompositeABI
}

func (c AggregateFunctionCertificate) ABI() PureCompositeABI {
	a := c.abi
	a.Parameters = make([]PureValueLayout, len(c.abi.Parameters))
	for i := range c.abi.Parameters {
		a.Parameters[i] = clonePureValueLayout(c.abi.Parameters[i])
	}
	a.Result = clonePureValueLayout(c.abi.Result)
	return a
}

func LowerCertifiedAggregateFunction(c AggregateFunctionCertificate) ([]byte, PureCompositeABI, error) {
	if len(c.wasm) == 0 {
		return nil, PureCompositeABI{}, fmt.Errorf("wasm.aggregate_certificate_invalid")
	}
	return append([]byte(nil), c.wasm...), c.ABI(), nil
}

func CertifyAggregateFunction(graph wire.Envelope) (AggregateFunctionCertificate, error) {
	abi, err := CertifyPureCompositeABI(graph)
	if err != nil {
		return AggregateFunctionCertificate{}, err
	}
	if !aggregateSignature(abi) {
		return AggregateFunctionCertificate{}, fmt.Errorf("wasm.aggregate_signature")
	}
	program := bySchema(graph, 0x9015)[0]
	entry, _ := field(program, 0x9151)
	fn := graph.Entities[entry.Reference]
	params, _ := field(fn, 0x9111)
	body, _ := field(fn, 0x9113)
	root, err := soleReturnExpression(graph, body.Reference)
	if err != nil {
		return AggregateFunctionCertificate{}, err
	}
	want := map[string]bool{"field": false, "array": false, "slice": false, "map": false}
	if err := certifyAggregateExpression(graph, root, params.List, want); err != nil {
		return AggregateFunctionCertificate{}, err
	}
	for _, found := range want {
		if !found {
			return AggregateFunctionCertificate{}, fmt.Errorf("wasm.aggregate_expression")
		}
	}
	wasm := aggregateModule()
	abi.Provider = "seme.function-aggregate-v1"
	return AggregateFunctionCertificate{wasm: wasm, abi: abi}, nil
}

func aggregateSignature(a PureCompositeABI) bool {
	return len(a.Parameters) == 5 && len(a.Parameters[0].Fields) == 1 &&
		a.Parameters[0].Fields[0].Value.Type == "i64" &&
		a.Parameters[1].Elements != nil && a.Parameters[1].Length == 2 && a.Parameters[1].Elements.Type == "i64" &&
		a.Parameters[2].Elements != nil && a.Parameters[2].Length == 0 && a.Parameters[2].Elements.Type == "i64" &&
		a.Parameters[3].Key != nil && a.Parameters[3].Value != nil && a.Parameters[3].Key.Type == "i64" && a.Parameters[3].Value.Type == "i64" &&
		a.Parameters[4].Type == "i64" && a.Result.Type == "i64" && a.RequestFixedSize == 48
}

func soleReturnExpression(g wire.Envelope, block wire.ID) (wire.ID, error) {
	b, ok := g.Entities[block]
	if !ok || b.Schema != identity(0x9080) {
		return wire.ID{}, fmt.Errorf("wasm.aggregate_block")
	}
	s, e := field(b, 0x9800)
	if e != nil || s.Tag != 7 || len(s.List) != 1 || s.List[0].Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.aggregate_block")
	}
	r := g.Entities[s.List[0].Reference]
	v, e := field(r, 0x9810)
	if r.Schema != identity(0x9081) || e != nil || v.Tag != 7 || len(v.List) != 1 || v.List[0].Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.aggregate_return")
	}
	return v.List[0].Reference, nil
}

func certifyAggregateExpression(g wire.Envelope, id wire.ID, params []wire.Value, found map[string]bool) error {
	e, ok := g.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.aggregate_expression")
	}
	if e.Schema == identity(0x9014) {
		left, a := field(e, 0x9140)
		right, b := field(e, 0x9141)
		if a != nil || b != nil || left.Tag != 6 || right.Tag != 6 {
			return fmt.Errorf("wasm.aggregate_add")
		}
		if err := certifyAggregateExpression(g, left.Reference, params, found); err != nil {
			return err
		}
		return certifyAggregateExpression(g, right.Reference, params, found)
	}
	readParam := func(readID wire.ID, position int) bool {
		r := g.Entities[readID]
		value, err := field(r, 0x9130)
		return r.Schema == identity(0x9013) && err == nil && value.Tag == 6 && params[position].Tag == 6 && value.Reference == params[position].Reference
	}
	mark := func(kind string) error {
		if found[kind] {
			return fmt.Errorf("wasm.aggregate_duplicate_observation")
		}
		found[kind] = true
		return nil
	}
	switch e.Schema {
	case identity(0x9032):
		base, a := field(e, 0x9320)
		member, b := field(e, 0x9321)
		if a != nil || b != nil || base.Tag != 6 || member.Tag != 6 || !readParam(base.Reference, 0) {
			break
		}
		fieldEntity := g.Entities[member.Reference]
		typ, c := field(fieldEntity, 0x9311)
		if fieldEntity.Schema == identity(0x9031) && c == nil && typ.Tag == 6 && g.Entities[typ.Reference].Schema == identity(0x9010) {
			return mark("field")
		}
	case identity(0x90f4):
		base, a := field(e, 0x9f40)
		index, b := field(e, 0x9f41)
		if a == nil && b == nil && base.Tag == 6 && index.Tag == 6 && readParam(base.Reference, 1) {
			lit := g.Entities[index.Reference]
			value, c := field(lit, 0x9700)
			if lit.Schema == identity(0x9070) && c == nil && value.Tag == 3 && value.Unsigned == 1 {
				return mark("array")
			}
		}
	case identity(0x90f9):
		base, a := field(e, 0x9f90)
		if a == nil && base.Tag == 6 && readParam(base.Reference, 2) {
			return mark("slice")
		}
	case identity(0xa042):
		base, a := field(e, 0xa0420)
		key, b := field(e, 0xa0421)
		if a == nil && b == nil && base.Tag == 6 && key.Tag == 6 && readParam(base.Reference, 3) && readParam(key.Reference, 4) {
			return mark("map")
		}
	}
	return fmt.Errorf("wasm.aggregate_expression")
}

func aggregateModule() []byte {
	var w bytes.Buffer
	w.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var ts bytes.Buffer
	uleb(&ts, 6)
	functionType(&ts, []byte{0x7f}, []byte{0x7f})
	functionType(&ts, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&ts, nil, []byte{0x7f})
	functionType(&ts, []byte{0x7f, 0x7f}, []byte{0x7f})
	functionType(&ts, []byte{0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&ts, []byte{0x7f, 0x7f}, nil)
	section(&w, 1, ts.Bytes())
	section(&w, 3, []byte{6, 0, 5, 3, 3, 2, 1})
	section(&w, 5, []byte{1, 0, 1})
	var gl bytes.Buffer
	gl.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&gl, 1024)
	gl.WriteByte(0x0b)
	section(&w, 6, gl.Bytes())
	var ex bytes.Buffer
	uleb(&ex, 7)
	export(&ex, "memory", 2, 0)
	export(&ex, "pulp_alloc", 0, 0)
	export(&ex, "pulp_free", 0, 1)
	export(&ex, "pulp_init", 0, 2)
	export(&ex, "pulp_step", 0, 3)
	export(&ex, "pulp_shutdown", 0, 4)
	export(&ex, "pulp_on_call", 0, 5)
	section(&w, 7, ex.Bytes())
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, aggregateProviderBody()}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&w, 10, code.Bytes())
	return w.Bytes()
}

func aggregateProviderBody() []byte {
	var b bytes.Buffer
	b.Write([]byte{2, 5, 0x7f, 3, 0x7e}) // slice ptr/count, map ptr/count, index; sum/key/current
	fail := func() { b.Write([]byte{0x41, 3, 0x0f}) }
	// fixed header, bounded slice/map counts, canonical contiguous payload.
	b.Write([]byte{0x20, 3, 0x41, 48, 0x49, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 2, 0x28, 2, 24, 0x21, 6, 0x20, 2, 0x28, 2, 28, 0x22, 7, 0x41})
	sleb(&b, 512)
	b.Write([]byte{0x4b, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 6, 0x41, 48, 0x47, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 2, 0x28, 2, 32, 0x21, 8, 0x20, 2, 0x28, 2, 36, 0x22, 9, 0x41})
	sleb(&b, 512)
	b.Write([]byte{0x4b, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	// mapPtr == 48 + sliceCount*8; messageLen == mapPtr + mapCount*16.
	b.Write([]byte{0x20, 8, 0x41, 48, 0x20, 7, 0x41, 3, 0x74, 0x6a, 0x47, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 3, 0x20, 8, 0x20, 9, 0x41, 4, 0x74, 0x6a, 0x47, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	// sum = record field + fixed array element 1 + slice length; key = request key.
	b.Write([]byte{0x20, 2, 0x29, 3, 0, 0x20, 2, 0x29, 3, 16, 0x7c, 0x20, 7, 0xad, 0x7c, 0x21, 11, 0x20, 2, 0x29, 3, 40, 0x21, 12, 0x41, 0x00, 0x21, 10})
	// Loop map entries, enforce strictly signed ascending keys and add matching value.
	b.Write([]byte{0x02, 0x40, 0x03, 0x40, 0x20, 10, 0x20, 9, 0x4f, 0x0d, 1})
	// current = load64(request + mapPtr + index*16).
	b.Write([]byte{0x20, 2, 0x20, 8, 0x6a, 0x20, 10, 0x41, 4, 0x74, 0x6a, 0x29, 3, 0, 0x21, 13})
	// Every key after the first must be strictly greater using signed i64 order.
	b.Write([]byte{0x20, 10, 0x45, 0x04, 0x40, 0x05, 0x20, 2, 0x20, 8, 0x6a, 0x20, 10, 0x41, 1, 0x6b, 0x41, 4, 0x74, 0x6a, 0x29, 3, 0, 0x20, 13, 0x59, 0x04, 0x40})
	fail()
	b.Write([]byte{0x0b, 0x0b})
	// A matching key contributes its adjacent value; missing keys contribute zero.
	b.Write([]byte{0x20, 13, 0x20, 12, 0x51, 0x04, 0x40, 0x20, 11, 0x20, 2, 0x20, 8, 0x6a, 0x20, 10, 0x41, 4, 0x74, 0x6a, 0x29, 3, 8, 0x7c, 0x21, 11, 0x0b})
	b.Write([]byte{0x20, 10, 0x41, 1, 0x6a, 0x21, 10, 0x0c, 0, 0x0b, 0x0b})
	// Canonical i64 response and Pulp out-pointer/out-length convention.
	constI32(&b, 8192)
	b.Write([]byte{0x20, 11, 0x37, 3, 0, 0x20, 4})
	constI32(&b, 8192)
	b.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 8, 0x36, 2, 0, 0x41, 0, 0x0b})
	return b.Bytes()
}
