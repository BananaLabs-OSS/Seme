package wasmtarget

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"seme.local/reference/wire"
)

// CompositeFunctionCertificate is deliberately distinct from the legacy pure
// certificate. Only the composite certifier can construct its executable plan.
type CompositeFunctionCertificate struct {
	wasm []byte
	abi  PureCompositeABI
}

func (c CompositeFunctionCertificate) ABI() PureCompositeABI {
	a := c.abi
	a.Parameters = make([]PureValueLayout, len(c.abi.Parameters))
	for i := range c.abi.Parameters {
		a.Parameters[i] = clonePureValueLayout(c.abi.Parameters[i])
	}
	a.Result = clonePureValueLayout(c.abi.Result)
	return a
}

func clonePureValueLayout(in PureValueLayout) PureValueLayout {
	out := in
	if in.Elements != nil {
		value := clonePureValueLayout(*in.Elements)
		out.Elements = &value
	}
	if in.Key != nil {
		value := clonePureValueLayout(*in.Key)
		out.Key = &value
	}
	if in.Value != nil {
		value := clonePureValueLayout(*in.Value)
		out.Value = &value
	}
	out.Fields = make([]PureValueFieldLayout, len(in.Fields))
	for i, field := range in.Fields {
		out.Fields[i] = field
		out.Fields[i].Value = clonePureValueLayout(field.Value)
	}
	out.Variants = make([]PureValueVariantLayout, len(in.Variants))
	for i, variant := range in.Variants {
		out.Variants[i] = variant
		if variant.Payload != nil {
			payload := clonePureValueLayout(*variant.Payload)
			out.Variants[i].Payload = &payload
		}
	}
	return out
}

func LowerCertifiedCompositeFunction(c CompositeFunctionCertificate) ([]byte, PureCompositeABI, error) {
	if len(c.wasm) == 0 {
		return nil, PureCompositeABI{}, fmt.Errorf("wasm.composite_certificate_invalid")
	}
	return append([]byte(nil), c.wasm...), c.ABI(), nil
}

// CertifyCompositeFunction certifies the v32 UAB execution proof without
// changing the accepted language or emitted bytes of CertifyPureFunction.
func CertifyCompositeFunction(graph wire.Envelope) (CompositeFunctionCertificate, error) {
	abi, err := CertifyPureCompositeABI(graph)
	if err != nil {
		return CompositeFunctionCertificate{}, err
	}
	if len(abi.Parameters) != 1 || abi.Parameters[0].Type != "option<result<bytes,text>>" || abi.Result.Type != "bool" {
		return CompositeFunctionCertificate{}, fmt.Errorf("wasm.composite_signature")
	}
	program := bySchema(graph, 0x9015)[0]
	entry, _ := field(program, 0x9151)
	fn := graph.Entities[entry.Reference]
	params, _ := field(fn, 0x9111)
	param := params.List[0].Reference
	body, e := field(fn, 0x9113)
	if e != nil || body.Tag != 6 {
		return CompositeFunctionCertificate{}, fmt.Errorf("wasm.composite_body")
	}
	proof, err := certifyCompositeProof(graph, body.Reference, param)
	if err != nil {
		return CompositeFunctionCertificate{}, err
	}
	wasm, err := compositeModule(proof)
	if err != nil {
		return CompositeFunctionCertificate{}, err
	}
	return CompositeFunctionCertificate{append([]byte(nil), wasm...), abi}, nil
}

type compositeProof struct {
	none        bool
	ok, failure []byte
}

func certifyCompositeProof(g wire.Envelope, blockID, parameter wire.ID) (compositeProof, error) {
	retExpr := func(block wire.ID) (wire.ID, error) {
		b, ok := g.Entities[block]
		if !ok || b.Schema != identity(0x9080) {
			return wire.ID{}, fmt.Errorf("wasm.composite_block")
		}
		s, e := field(b, 0x9800)
		if e != nil || s.Tag != 7 || len(s.List) != 1 || s.List[0].Tag != 6 {
			return wire.ID{}, fmt.Errorf("wasm.composite_block")
		}
		r, ok := g.Entities[s.List[0].Reference]
		if !ok || r.Schema != identity(0x9081) {
			return wire.ID{}, fmt.Errorf("wasm.composite_return")
		}
		v, e := field(r, 0x9810)
		if e != nil || v.Tag != 7 || len(v.List) != 1 || v.List[0].Tag != 6 {
			return wire.ID{}, fmt.Errorf("wasm.composite_return")
		}
		return v.List[0].Reference, nil
	}
	root, err := retExpr(blockID)
	if err != nil {
		return compositeProof{}, err
	}
	om, ok := g.Entities[root]
	if !ok || om.Schema != identity(0xa063) {
		return compositeProof{}, fmt.Errorf("wasm.composite_option_match")
	}
	value, a := field(om, 0xa0630)
	noneBody, b := field(om, 0xa0631)
	someBinding, c := field(om, 0xa0632)
	someBody, d := field(om, 0xa0633)
	if a != nil || b != nil || c != nil || d != nil || value.Tag != 6 || noneBody.Tag != 6 || someBinding.Tag != 6 || someBody.Tag != 6 {
		return compositeProof{}, fmt.Errorf("wasm.composite_option_match")
	}
	read := g.Entities[value.Reference]
	rv, re := field(read, 0x9130)
	if read.Schema != identity(0x9013) || re != nil || rv.Tag != 6 || rv.Reference != parameter {
		return compositeProof{}, fmt.Errorf("wasm.composite_option_value")
	}
	return certifyCompositeBranches(g, retExpr, noneBody.Reference, someBody.Reference, someBinding.Reference)
}

func certifyCompositeBranches(g wire.Envelope, retExpr func(wire.ID) (wire.ID, error), noneBlock, someBlock, someBinding wire.ID) (compositeProof, error) {
	bind := g.Entities[someBinding]
	bt, e := field(bind, 0xa0601)
	if bind.Schema != identity(0xa060) || e != nil || bt.Tag != 6 || g.Entities[bt.Reference].Schema != identity(0x9042) {
		return compositeProof{}, fmt.Errorf("wasm.composite_some_binding")
	}
	n, err := retExpr(noneBlock)
	if err != nil {
		return compositeProof{}, err
	}
	nl := g.Entities[n]
	nv, e := field(nl, 0x9b00)
	if nl.Schema != identity(0x90b0) || e != nil || nv.Tag != 1 || nv.Unsigned > 1 {
		return compositeProof{}, fmt.Errorf("wasm.composite_none_branch")
	}
	rid, err := retExpr(someBlock)
	if err != nil {
		return compositeProof{}, err
	}
	rm := g.Entities[rid]
	if rm.Schema != identity(0xa062) {
		return compositeProof{}, fmt.Errorf("wasm.composite_result_match")
	}
	v, a := field(rm, 0xa0620)
	ob, b := field(rm, 0xa0621)
	oby, c := field(rm, 0xa0622)
	eb, d := field(rm, 0xa0623)
	eby, f := field(rm, 0xa0624)
	if a != nil || b != nil || c != nil || d != nil || f != nil || v.Tag != 6 || ob.Tag != 6 || oby.Tag != 6 || eb.Tag != 6 || eby.Tag != 6 {
		return compositeProof{}, fmt.Errorf("wasm.composite_result_match")
	}
	vr := g.Entities[v.Reference]
	br, er := field(vr, 0xa0610)
	if vr.Schema != identity(0xa061) || er != nil || br.Tag != 6 || br.Reference != someBinding {
		return compositeProof{}, fmt.Errorf("wasm.composite_binding_scope")
	}
	okLit, err := certifyCompositeEqual(g, retExpr, oby.Reference, ob.Reference, identity(0x9041), identity(0xa065), 0xa0650, 0xa0651, identity(0xa064), 0xa0640)
	if err != nil {
		return compositeProof{}, err
	}
	errLit, err := certifyCompositeEqual(g, retExpr, eby.Reference, eb.Reference, identity(0x9040), identity(0x90c2), 0x9c20, 0x9c21, identity(0x9050), 0x9500)
	if err != nil {
		return compositeProof{}, err
	}
	return compositeProof{nv.Unsigned == 1, okLit, errLit}, nil
}

func certifyCompositeEqual(g wire.Envelope, retExpr func(wire.ID) (wire.ID, error), block, binding, typ, eqSchema wire.ID, leftField, rightField uint64, literalSchema wire.ID, literalField uint64) ([]byte, error) {
	b := g.Entities[binding]
	tv, e := field(b, 0xa0601)
	if b.Schema != identity(0xa060) || e != nil || tv.Tag != 6 || g.Entities[tv.Reference].Schema != typ {
		return nil, fmt.Errorf("wasm.composite_binding_type")
	}
	id, err := retExpr(block)
	if err != nil {
		return nil, err
	}
	eq := g.Entities[id]
	l, a := field(eq, leftField)
	r, c := field(eq, rightField)
	if eq.Schema != eqSchema || a != nil || c != nil || l.Tag != 6 || r.Tag != 6 {
		return nil, fmt.Errorf("wasm.composite_equal")
	}
	readLiteral := func(readID, litID wire.ID) ([]byte, bool) {
		rd := g.Entities[readID]
		rv, e := field(rd, 0xa0610)
		lit := g.Entities[litID]
		lv, f := field(lit, literalField)
		return append([]byte(nil), lv.Bytes...), rd.Schema == identity(0xa061) && e == nil && rv.Tag == 6 && rv.Reference == binding && lit.Schema == literalSchema && f == nil && lv.Tag == 5
	}
	if x, ok := readLiteral(l.Reference, r.Reference); ok {
		if literalSchema == identity(0x9050) && !utf8.Valid(x) {
			return nil, fmt.Errorf("wasm.composite_text_literal")
		}
		return x, nil
	}
	if x, ok := readLiteral(r.Reference, l.Reference); ok {
		if literalSchema == identity(0x9050) && !utf8.Valid(x) {
			return nil, fmt.Errorf("wasm.composite_text_literal")
		}
		return x, nil
	}
	leftEntity := g.Entities[l.Reference]
	rightEntity := g.Entities[r.Reference]
	leftBinding, _ := field(leftEntity, 0xa0610)
	rightBinding, _ := field(rightEntity, 0xa0610)
	return nil, fmt.Errorf("wasm.composite_binding_scope:left=%s/%s right=%s/%s expected=%s", leftEntity.Schema.String(), leftBinding.Reference.String(), rightEntity.Schema.String(), rightBinding.Reference.String(), binding.String())
}

func compositeModule(p compositeProof) ([]byte, error) {
	const table = 16384
	okPtr := table + 9*256
	errPtr := okPtr + len(p.ok)
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
	section(&w, 3, []byte{8, 0, 5, 3, 3, 2, 3, 4, 1})
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
	export(&ex, "pulp_on_call", 0, 7)
	section(&w, 7, ex.Bytes())
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, compositeUTF8ValidatorBody(), compositeBytesEqualBody(), compositeProviderBody(p, uint64(okPtr), uint64(errPtr))}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, b := range bodies {
		uleb(&code, uint64(len(b)))
		code.Write(b)
	}
	section(&w, 10, code.Bytes())
	data := append(utf8TransitionTable(), p.ok...)
	data = append(data, p.failure...)
	var ds bytes.Buffer
	uleb(&ds, 1)
	ds.WriteByte(0)
	constI32(&ds, table)
	ds.WriteByte(0x0b)
	uleb(&ds, uint64(len(data)))
	ds.Write(data)
	section(&w, 11, ds.Bytes())
	return w.Bytes(), nil
}

func compositeProviderBody(p compositeProof, okPtr, errPtr uint64) []byte {
	var b bytes.Buffer
	b.Write([]byte{1, 4, 0x7f}) // tag, variant, len, ptr
	fail := func() { b.Write([]byte{0x41, 3, 0x0f}) }
	b.Write([]byte{0x20, 3, 0x41, 10, 0x49, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 2, 0x2d, 0, 0, 0x22, 6, 0x41, 1, 0x4b, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	// none requires exactly the fixed header and a zero inactive payload.
	b.Write([]byte{0x20, 6, 0x45, 0x04, 0x40, 0x20, 3, 0x41, 10, 0x47, 0x04, 0x40})
	fail()
	b.Write([]byte{0x0b, 0x20, 2, 0x41, 1, 0x6a, 0x29, 0, 0, 0x50, 0x20, 2, 0x2d, 0, 9, 0x45, 0x71, 0x45, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	emitCompositeResponse(&b, p.none)
	b.WriteByte(0x0b)
	// some/result: validate total tag, absolute descriptor, bound, exact coverage.
	b.Write([]byte{0x20, 2, 0x2d, 0, 1, 0x22, 7, 0x41, 1, 0x4b, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 2, 0x28, 2, 2, 0x22, 9, 0x41, 10, 0x47, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 2, 0x28, 2, 6, 0x22, 8, 0x41})
	sleb(&b, 4096)
	b.Write([]byte{0x4b, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	b.Write([]byte{0x20, 8, 0x41, 10, 0x6a, 0x20, 3, 0x47, 0x04, 0x40})
	fail()
	b.WriteByte(0x0b)
	// Error text must be canonical UTF-8.
	b.Write([]byte{0x20, 7, 0x04, 0x40, 0x20, 2, 0x20, 9, 0x6a, 0x20, 8, 0x10, 5, 0x45, 0x04, 0x40})
	fail()
	b.Write([]byte{0x0b, 0x0b})
	// byte equality with the graph-derived selected literal.
	b.Write([]byte{0x20, 2, 0x20, 9, 0x6a, 0x20, 8})
	constI32(&b, okPtr)
	constI32(&b, uint64(len(p.ok)))
	b.Write([]byte{0x10, 6, 0x21, 6})
	b.Write([]byte{0x20, 7, 0x04, 0x40, 0x20, 2, 0x20, 9, 0x6a, 0x20, 8})
	constI32(&b, errPtr)
	constI32(&b, uint64(len(p.failure)))
	b.Write([]byte{0x10, 6, 0x21, 6, 0x0b})
	b.Write([]byte{0x20, 6})
	emitCompositeResponseStack(&b)
	b.Write([]byte{0x41, 3, 0x0b})
	return b.Bytes()
}

func compositeBytesEqualBody() []byte {
	var b bytes.Buffer
	b.Write([]byte{1, 1, 0x7f, 0x20, 1, 0x20, 3, 0x47, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b, 0x02, 0x40, 0x03, 0x40, 0x20, 4, 0x20, 1, 0x4f, 0x0d, 1, 0x20, 0, 0x20, 4, 0x6a, 0x2d, 0, 0, 0x20, 2, 0x20, 4, 0x6a, 0x2d, 0, 0, 0x47, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b, 0x20, 4, 0x41, 1, 0x6a, 0x21, 4, 0x0c, 0, 0x0b, 0x0b, 0x41, 1, 0x0b})
	return b.Bytes()
}

func compositeUTF8ValidatorBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 3, 0x7f, 0x02, 0x40, 0x03, 0x40, 0x20, 3, 0x20, 1, 0x4f, 0x0d, 1})
	constI32(&body, 16384)
	body.Write([]byte{0x20, 2, 0x41, 8, 0x74, 0x6a, 0x20, 0, 0x20, 3, 0x6a, 0x2d, 0, 0, 0x22, 4, 0x6a, 0x2d, 0, 0, 0x21, 2, 0x20, 2, 0x41, 8, 0x46, 0x0d, 1, 0x20, 3, 0x41, 1, 0x6a, 0x21, 3, 0x0c, 0, 0x0b, 0x0b, 0x20, 2, 0x45, 0x0b})
	return body.Bytes()
}

func emitCompositeResponse(b *bytes.Buffer, v bool) {
	if v {
		b.Write([]byte{0x41, 1})
	} else {
		b.Write([]byte{0x41, 0})
	}
	emitCompositeResponseStack(b)
}
func emitCompositeResponseStack(b *bytes.Buffer) {
	b.Write([]byte{0x21, 6})
	constI32(b, 8192)
	b.Write([]byte{0x20, 6, 0x3a, 0, 0, 0x20, 4})
	constI32(b, 8192)
	b.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 1, 0x36, 2, 0, 0x41, 0, 0x0f})
}
