package wasmtarget

import (
	"bytes"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestCompositeCertificateIsDeterministicAndSeparate(t *testing.T) {
	g := compositeTestGraph()
	first, err := CertifyCompositeFunction(g)
	if err != nil {
		t.Fatal(err)
	}
	a, abi, err := LowerCertifiedCompositeFunction(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CertifyCompositeFunction(g)
	if err != nil {
		t.Fatal(err)
	}
	b, abi2, err := LowerCertifiedCompositeFunction(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) || abi.CanonicalProgram != abi2.CanonicalProgram || len(a) < 8 || string(a[:4]) != "\x00asm" {
		t.Fatal("nondeterministic composite certificate")
	}
	if abi.RequestFixedSize != 10 || abi.ResponseFixedSize != 1 || abi.Target != "wasm32-pulp-reactor-v1" {
		t.Fatalf("ABI = %#v", abi)
	}
	if _, _, err := LowerCertifiedCompositeFunction(CompositeFunctionCertificate{}); err == nil {
		t.Fatal("zero certificate lowered")
	}
}

func TestCompositeCertificateRejectsMalformedGraphAndScope(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		mutate     func(*wire.Envelope)
	}{
		{"wrong root", "wasm.composite_option_match", func(g *wire.Envelope) {
			e := g.Entities[identity(0x4108)]
			e.Schema = identity(0x90b0)
			g.Entities[e.ID] = e
		}},
		{"escaped binding", "wasm.composite_binding_scope", func(g *wire.Envelope) {
			e := g.Entities[identity(0x4115)]
			e.Fields[identity(0xa0610)] = ref(identity(0x4110))
			g.Entities[e.ID] = e
		}},
		{"missing total branch", "wasm.composite_block", func(g *wire.Envelope) { delete(g.Entities, identity(0x410b)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := compositeTestGraph()
			tc.mutate(&g)
			_, err := CertifyCompositeFunction(g)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want %s", err, tc.want)
			}
		})
	}
}

func compositeTestGraph() wire.Envelope {
	id := identity
	program, fn, param := id(0x4101), id(0x4102), id(0x4103)
	boolT, textT, bytesT, resultT, optionT := id(0x4104), id(0x4105), id(0x4106), id(0x4107), id(0x4109)
	rootMatch, rootBlock, rootRet := id(0x4108), id(0x410a), id(0x411a)
	noneBlock, noneRet, noneLit := id(0x410b), id(0x410c), id(0x410d)
	someBind, someBlock, someRet, resultMatch := id(0x410e), id(0x410f), id(0x4111), id(0x4112)
	okBind, errBind, okBlock, errBlock := id(0x4113), id(0x4114), id(0x4116), id(0x4117)
	okRet, errRet, okEq, errEq := id(0x4118), id(0x4119), id(0x411b), id(0x411c)
	paramRead, someRead, okRead, errRead := id(0x411d), id(0x411e), id(0x4115), id(0x411f)
	bytesLit, textLit := id(0x4120), id(0x4121)
	bv := func(v bool) wire.Value {
		u := uint64(0)
		if v {
			u = 1
		}
		return wire.Value{Tag: 1, Unsigned: u}
	}
	raw := func(v string) wire.Value { return wire.Value{Tag: 5, Bytes: []byte(v)} }
	es := map[wire.ID]wire.Entity{}
	add := func(x, s wire.ID, f map[wire.ID]wire.Value) { es[x] = wire.Entity{ID: x, Schema: s, Fields: f} }
	add(program, id(0x9015), map[wire.ID]wire.Value{id(0x9150): refs(fn), id(0x9151): ref(fn)})
	add(fn, id(0x9011), map[wire.ID]wire.Value{id(0x9111): refs(param), id(0x9112): ref(boolT), id(0x9113): ref(rootBlock)})
	add(param, id(0x9012), map[wire.ID]wire.Value{id(0x9121): ref(optionT), id(0x9122): unsigned(0)})
	add(boolT, id(0x9020), nil)
	add(textT, id(0x9040), nil)
	add(bytesT, id(0x9041), nil)
	add(resultT, id(0x9042), map[wire.ID]wire.Value{id(0x9400): ref(bytesT), id(0x9401): ref(textT)})
	add(optionT, id(0xa050), map[wire.ID]wire.Value{id(0xa0500): ref(resultT)})
	add(rootBlock, id(0x9080), map[wire.ID]wire.Value{id(0x9800): refs(rootRet)})
	add(rootRet, id(0x9081), map[wire.ID]wire.Value{id(0x9810): refs(rootMatch)})
	add(paramRead, id(0x9013), map[wire.ID]wire.Value{id(0x9130): ref(param)})
	add(rootMatch, id(0xa063), map[wire.ID]wire.Value{id(0xa0630): ref(paramRead), id(0xa0631): ref(noneBlock), id(0xa0632): ref(someBind), id(0xa0633): ref(someBlock)})
	add(noneBlock, id(0x9080), map[wire.ID]wire.Value{id(0x9800): refs(noneRet)})
	add(noneRet, id(0x9081), map[wire.ID]wire.Value{id(0x9810): refs(noneLit)})
	add(noneLit, id(0x90b0), map[wire.ID]wire.Value{id(0x9b00): bv(false)})
	add(someBind, id(0xa060), map[wire.ID]wire.Value{id(0xa0600): raw("some"), id(0xa0601): ref(resultT)})
	add(someRead, id(0xa061), map[wire.ID]wire.Value{id(0xa0610): ref(someBind)})
	add(someBlock, id(0x9080), map[wire.ID]wire.Value{id(0x9800): refs(someRet)})
	add(someRet, id(0x9081), map[wire.ID]wire.Value{id(0x9810): refs(resultMatch)})
	add(resultMatch, id(0xa062), map[wire.ID]wire.Value{id(0xa0620): ref(someRead), id(0xa0621): ref(okBind), id(0xa0622): ref(okBlock), id(0xa0623): ref(errBind), id(0xa0624): ref(errBlock)})
	add(okBind, id(0xa060), map[wire.ID]wire.Value{id(0xa0600): raw("ok"), id(0xa0601): ref(bytesT)})
	add(errBind, id(0xa060), map[wire.ID]wire.Value{id(0xa0600): raw("err"), id(0xa0601): ref(textT)})
	add(okRead, id(0xa061), map[wire.ID]wire.Value{id(0xa0610): ref(okBind)})
	add(errRead, id(0xa061), map[wire.ID]wire.Value{id(0xa0610): ref(errBind)})
	add(bytesLit, id(0xa064), map[wire.ID]wire.Value{id(0xa0640): raw("ok")})
	add(textLit, id(0x9050), map[wire.ID]wire.Value{id(0x9500): raw("bad")})
	add(okEq, id(0xa065), map[wire.ID]wire.Value{id(0xa0650): ref(okRead), id(0xa0651): ref(bytesLit)})
	add(errEq, id(0x90c2), map[wire.ID]wire.Value{id(0x9c20): ref(errRead), id(0x9c21): ref(textLit)})
	add(okBlock, id(0x9080), map[wire.ID]wire.Value{id(0x9800): refs(okRet)})
	add(errBlock, id(0x9080), map[wire.ID]wire.Value{id(0x9800): refs(errRet)})
	add(okRet, id(0x9081), map[wire.ID]wire.Value{id(0x9810): refs(okEq)})
	add(errRet, id(0x9081), map[wire.ID]wire.Value{id(0x9810): refs(errEq)})
	return wire.Envelope{Module: id(0x4190), Revision: id(0x4191), Entities: es}
}
