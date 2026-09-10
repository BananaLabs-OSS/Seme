// Package packagecallinstance derives and validates the relationship between
// an Execution call graph and its Package v2 ownership/import graph.
package packagecallinstance

import (
	"fmt"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/wire"
)

const maxVisited = 1 << 16

func Validate(source []byte) error {
	if err := packagedetailinstance.Validate(source); err != nil {
		return fmt.Errorf("package_call.package:%w", err)
	}
	e, err := wire.Decode(source)
	if err != nil {
		return err
	}
	programs := schemaIDs(e, id("9015"))
	if len(programs) != 1 {
		return fmt.Errorf("package_call.program")
	}
	functions, err := refs(e.Entities[programs[0]], id("9150"))
	if err != nil {
		return err
	}
	members := set(functions)
	owners := map[wire.ID]wire.ID{}
	visible := map[wire.ID]bool{}
	imports := map[wire.ID]map[wire.ID]int{}
	for _, did := range schemaIDs(e, id("b021")) {
		d := e.Entities[did]
		pkg := d.Fields[id("b210")].Reference
		ms, _ := refs(d, id("b211"))
		for _, mid := range ms {
			m := e.Entities[mid]
			fn := m.Fields[id("b220")].Reference
			owners[fn] = pkg
			visibility := e.Entities[m.Fields[id("b222")].Reference].Fields[id("b230")].Unsigned
			_, exported := m.Fields[id("b223")]
			visible[fn] = visibility > 0 && exported
		}
		bs, _ := refs(d, id("b212"))
		for _, bid := range bs {
			b := e.Entities[bid]
			class := e.Entities[b.Fields[id("b242")].Reference].Fields[id("b250")].Unsigned
			if class == 0 {
				target := b.Fields[id("b243")].Reference
				if imports[pkg] == nil {
					imports[pkg] = map[wire.ID]int{}
				}
				imports[pkg][target]++
			}
		}
	}
	contained := map[wire.ID]wire.ID{}
	total := 0
	for _, fn := range functions {
		f := e.Entities[fn]
		body := f.Fields[id("9113")]
		if body.Tag != 6 {
			return fmt.Errorf("package_call.function_body:%s", fn)
		}
		active := map[wire.ID]bool{}
		var walk func(wire.ID) error
		walk = func(x wire.ID) error {
			q, ok := e.Entities[x]
			if !ok {
				return fmt.Errorf("package_call.missing:%s", x)
			}
			if x != body.Reference && stop(q.Schema) {
				return nil
			}
			if active[x] {
				return fmt.Errorf("package_call.cycle:%s", x)
			}
			if prior, yes := contained[x]; yes {
				if prior != fn {
					return fmt.Errorf("package_call.shared_executable:%s", x)
				}
				return nil
			}
			total++
			if total > maxVisited {
				return fmt.Errorf("package_call.limit")
			}
			contained[x] = fn
			active[x] = true
			defer delete(active, x)
			if q.Schema == id("9060") {
				callee := q.Fields[id("9600")]
				args := q.Fields[id("9601")]
				if callee.Tag != 6 || args.Tag != 7 || !members[callee.Reference] {
					return fmt.Errorf("package_call.target:%s", x)
				}
				parameters, _ := refs(e.Entities[callee.Reference], id("9111"))
				if len(args.List) != len(parameters) {
					return fmt.Errorf("package_call.arity:%s", x)
				}
				for i, a := range args.List {
					if a.Tag != 6 {
						return fmt.Errorf("package_call.argument:%s", x)
					}
					got, er := expressionType(e, a.Reference)
					if er != nil {
						return fmt.Errorf("package_call.argument_type:%s:%w", x, er)
					}
					want := e.Entities[parameters[i]].Fields[id("9121")]
					if want.Tag != 6 || got != want.Reference {
						return fmt.Errorf("package_call.signature:%s", x)
					}
					if er = walk(a.Reference); er != nil {
						return er
					}
				}
				callerPackage, calleePackage := owners[fn], owners[callee.Reference]
				if callerPackage == (wire.ID{}) || calleePackage == (wire.ID{}) {
					return fmt.Errorf("package_call.owner:%s", x)
				}
				if callerPackage != calleePackage {
					if !visible[callee.Reference] {
						return fmt.Errorf("package_call.inaccessible:%s", x)
					}
					if imports[callerPackage][calleePackage] != 1 {
						return fmt.Errorf("package_call.unauthorized:%s", x)
					}
				}
				return nil
			}
			for _, child := range children(q) {
				if er := walk(child); er != nil {
					return er
				}
			}
			return nil
		}
		if err = walk(body.Reference); err != nil {
			return err
		}
	}
	return nil
}

func expressionType(e wire.Envelope, x wire.ID) (wire.ID, error) {
	if t, ok := canonicaleval.ExpressionType(e, x); ok {
		return t, nil
	}
	q, ok := e.Entities[x]
	if !ok {
		return wire.ID{}, fmt.Errorf("missing")
	}
	switch q.Schema {
	case id("9013"):
		p := q.Fields[id("9130")]
		if p.Tag != 6 {
			return wire.ID{}, fmt.Errorf("parameter")
		}
		v := e.Entities[p.Reference].Fields[id("9121")]
		if v.Tag != 6 {
			return wire.ID{}, fmt.Errorf("parameter_type")
		}
		return v.Reference, nil
	case id("9070"):
		return typed(q, id("9701"))
	case id("9014"):
		return typed(q, id("9142"))
	case id("9090"):
		return typed(q, id("9902"))
	case id("9060"):
		c := q.Fields[id("9600")]
		if c.Tag != 6 {
			return wire.ID{}, fmt.Errorf("callee")
		}
		return typed(e.Entities[c.Reference], id("9112"))
	default:
		return wire.ID{}, fmt.Errorf("unsupported:%s", q.Schema)
	}
}
func typed(q wire.Entity, f wire.ID) (wire.ID, error) {
	v := q.Fields[f]
	if v.Tag != 6 {
		return wire.ID{}, fmt.Errorf("type")
	}
	return v.Reference, nil
}
func stop(s wire.ID) bool {
	switch s {
	case id("9010"), id("9011"), id("9012"), id("9015"), id("9020"), id("9030"), id("9031"), id("9040"), id("9041"), id("9042"), id("90f2"), id("90f8"), id("a004"), id("a010"), id("a011"), id("a012"), id("a020"), id("a040"), id("a050"):
		return true
	}
	return false
}
func children(q wire.Entity) []wire.ID {
	out := []wire.ID{}
	var add func(wire.Value)
	add = func(v wire.Value) {
		if v.Tag == 6 {
			out = append(out, v.Reference)
		}
		for _, x := range v.List {
			add(x)
		}
		for _, x := range v.Record {
			add(x)
		}
	}
	for _, v := range q.Fields {
		add(v)
	}
	return out
}
func refs(q wire.Entity, f wire.ID) ([]wire.ID, error) {
	v, ok := q.Fields[f]
	if !ok || v.Tag != 7 {
		return nil, fmt.Errorf("package_call.refs")
	}
	out := make([]wire.ID, len(v.List))
	for i, x := range v.List {
		if x.Tag != 6 {
			return nil, fmt.Errorf("package_call.refs")
		}
		out[i] = x.Reference
	}
	return out, nil
}
func schemaIDs(e wire.Envelope, s wire.ID) []wire.ID {
	out := []wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == s {
			out = append(out, x)
		}
	}
	return out
}
func set(xs []wire.ID) map[wire.ID]bool {
	m := map[wire.ID]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
