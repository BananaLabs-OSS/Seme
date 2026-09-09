// Package canonicaleval independently evaluates the bounded pure v32 graph
// profile used by UAB value evidence. It does not call a language adapter or a
// target lowerer and never selects behavior by source, package, or function name.
package canonicaleval

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"

	"seme.local/reference/wire"
)

type Value struct {
	Kind      string           `json:"kind"`
	I64       string           `json:"i64,omitempty"`
	Bool      bool             `json:"bool,omitempty"`
	Text      string           `json:"text,omitempty"`
	Bytes     string           `json:"bytes_hex,omitempty"`
	Items     []Value          `json:"items,omitempty"`
	Fields    map[string]Value `json:"fields,omitempty"`
	Entries   []Entry          `json:"entries,omitempty"`
	ValueType string           `json:"value_type,omitempty"`
	Variant   string           `json:"variant,omitempty"`
	Payload   *Value           `json:"payload,omitempty"`
}
type Entry struct {
	Key   Value `json:"key"`
	Value Value `json:"value"`
}

func Evaluate(g wire.Envelope, arguments []Value) (Value, error) {
	programs := schemaEntities(g, 0x9015)
	if len(programs) != 1 {
		return Value{}, fmt.Errorf("canonicaleval.program")
	}
	entry, err := field(programs[0], 0x9151)
	if err != nil || entry.Tag != 6 {
		return Value{}, fmt.Errorf("canonicaleval.entry")
	}
	fn, ok := g.Entities[entry.Reference]
	if !ok || fn.Schema != id(0x9011) {
		return Value{}, fmt.Errorf("canonicaleval.entry")
	}
	params, pe := field(fn, 0x9111)
	body, be := field(fn, 0x9113)
	if pe != nil || be != nil || params.Tag != 7 || len(params.List) != len(arguments) || body.Tag != 6 {
		return Value{}, fmt.Errorf("canonicaleval.arguments")
	}
	env := map[wire.ID]Value{}
	for i, p := range params.List {
		if p.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.parameter")
		}
		parameter := g.Entities[p.Reference]
		if parameter.Schema != id(0x9012) {
			return Value{}, fmt.Errorf("canonicaleval.parameter_schema")
		}
		typ, er := field(parameter, 0x9121)
		if er != nil || typ.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.parameter_type")
		}
		if er := validateType(g, typ.Reference, map[wire.ID]bool{}, 32); er != nil {
			return Value{}, fmt.Errorf("canonicaleval.parameter_type:%w", er)
		}
		if er := validateValue(g, typ.Reference, arguments[i], 32); er != nil {
			return Value{}, fmt.Errorf("canonicaleval.argument.%d:%w", i, er)
		}
		env[p.Reference] = arguments[i]
	}
	return evalBlock(g, body.Reference, env, 64)
}

func validateType(g wire.Envelope, x wire.ID, visiting map[wire.ID]bool, budget int) error {
	if budget == 0 || visiting[x] {
		return fmt.Errorf("type_cycle_or_budget")
	}
	e, ok := g.Entities[x]
	if !ok {
		return fmt.Errorf("type_missing")
	}
	visiting[x] = true
	defer delete(visiting, x)
	ref := func(k uint64) (wire.ID, error) {
		v, er := field(e, k)
		if er != nil || v.Tag != 6 {
			return wire.ID{}, fmt.Errorf("type_reference")
		}
		return v.Reference, nil
	}
	recurse := func(k uint64) error {
		v, er := ref(k)
		if er != nil {
			return er
		}
		return validateType(g, v, visiting, budget-1)
	}
	switch e.Schema {
	case id(0x9010), id(0x9020), id(0x9040), id(0x9041):
		return nil
	case id(0x90f2):
		if er := recurse(0x9f20); er != nil {
			return er
		}
		v, er := field(e, 0x9f21)
		if er != nil || v.Tag != 3 || v.Unsigned == 0 {
			return fmt.Errorf("array_length")
		}
		return nil
	case id(0x90f8):
		return recurse(0x9f80)
	case id(0xa040):
		if er := recurse(0xa0400); er != nil {
			return er
		}
		return recurse(0xa0401)
	case id(0xa050):
		return recurse(0xa0500)
	case id(0x9042):
		if er := recurse(0x9400); er != nil {
			return er
		}
		return recurse(0x9401)
	case id(0x9030):
		v, er := field(e, 0x9301)
		if er != nil || v.Tag != 7 || len(v.List) == 0 {
			return fmt.Errorf("record_fields")
		}
		seen := map[wire.ID]bool{}
		for _, item := range v.List {
			if item.Tag != 6 || seen[item.Reference] {
				return fmt.Errorf("record_field_reference")
			}
			seen[item.Reference] = true
			f, ok := g.Entities[item.Reference]
			if !ok || f.Schema != id(0x9031) {
				return fmt.Errorf("record_field_schema")
			}
			ft, er := field(f, 0x9311)
			name, ne := field(f, 0x9310)
			if er != nil || ne != nil || ft.Tag != 6 || name.Tag != 5 {
				return fmt.Errorf("record_field")
			}
			if er := validateType(g, ft.Reference, visiting, budget-1); er != nil {
				return er
			}
		}
		return nil
	}
	return fmt.Errorf("unsupported_type")
}

func validateValue(g wire.Envelope, typeID wire.ID, v Value, budget int) error {
	if budget == 0 {
		return fmt.Errorf("type_budget")
	}
	t, ok := g.Entities[typeID]
	if !ok {
		return fmt.Errorf("type_missing")
	}
	switch t.Schema {
	case id(0x9010):
		n, ok := new(big.Int).SetString(v.I64, 10)
		if v.Kind != "i64" || !ok || n.Cmp(new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 63))) < 0 || n.Cmp(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 63), big.NewInt(1))) > 0 {
			return fmt.Errorf("i64")
		}
	case id(0x9020):
		if v.Kind != "bool" {
			return fmt.Errorf("bool")
		}
	case id(0x9040):
		if v.Kind != "text" || !utf8.ValidString(v.Text) {
			return fmt.Errorf("text")
		}
	case id(0x9041):
		if v.Kind != "bytes" {
			return fmt.Errorf("bytes")
		}
		if _, e := hex.DecodeString(v.Bytes); e != nil {
			return fmt.Errorf("bytes")
		}
	case id(0x9030):
		if v.Kind != "record" {
			return fmt.Errorf("record")
		}
		fs, ferr := field(t, 0x9301)
		if ferr != nil || fs.Tag != 7 || len(v.Fields) != len(fs.List) {
			return fmt.Errorf("record_shape")
		}
		for _, f := range fs.List {
			if f.Tag != 6 {
				return fmt.Errorf("record_field")
			}
			fe, exists := g.Entities[f.Reference]
			name, ne := field(fe, 0x9310)
			ft, te := field(fe, 0x9311)
			value, yes := v.Fields[string(name.Bytes)]
			if !exists || fe.Schema != id(0x9031) || ne != nil || te != nil || name.Tag != 5 || !yes || ft.Tag != 6 {
				return fmt.Errorf("record_field")
			}
			if e := validateValue(g, ft.Reference, value, budget-1); e != nil {
				return e
			}
		}
	case id(0x90f2):
		element, ee := field(t, 0x9f20)
		length, le := field(t, 0x9f21)
		if ee != nil || le != nil || element.Tag != 6 || v.Kind != "array" || length.Tag != 3 || uint64(len(v.Items)) != length.Unsigned {
			return fmt.Errorf("array")
		}
		for _, item := range v.Items {
			if e := validateValue(g, element.Reference, item, budget-1); e != nil {
				return e
			}
		}
	case id(0x90f8):
		element, ee := field(t, 0x9f80)
		if ee != nil || element.Tag != 6 || v.Kind != "slice" || len(v.Items) > 512 {
			return fmt.Errorf("slice")
		}
		for _, item := range v.Items {
			if e := validateValue(g, element.Reference, item, budget-1); e != nil {
				return e
			}
		}
	case id(0xa040):
		keyT, ke := field(t, 0xa0400)
		valueT, ve := field(t, 0xa0401)
		valueEntity, exists := g.Entities[valueT.Reference]
		valueKind, known := scalarTypeKind(valueEntity.Schema)
		if ke != nil || ve != nil || keyT.Tag != 6 || valueT.Tag != 6 || !exists || v.Kind != "map" || len(v.Entries) > 512 || !known || v.ValueType != valueKind {
			return fmt.Errorf("map")
		}
		var previous *big.Int
		for _, entry := range v.Entries {
			if e := validateValue(g, keyT.Reference, entry.Key, budget-1); e != nil {
				return e
			}
			if e := validateValue(g, valueT.Reference, entry.Value, budget-1); e != nil {
				return e
			}
			if entry.Key.Kind == "i64" {
				current, _ := new(big.Int).SetString(entry.Key.I64, 10)
				if previous != nil && previous.Cmp(current) >= 0 {
					return fmt.Errorf("map_order")
				}
				previous = current
			}
		}
	case id(0xa050):
		inner, ie := field(t, 0xa0500)
		if ie != nil || inner.Tag != 6 || v.Kind != "option" || (v.Variant != "none" && v.Variant != "some") {
			return fmt.Errorf("option")
		}
		if v.Variant == "none" {
			if v.Payload != nil {
				return fmt.Errorf("option_payload")
			}
		} else if v.Payload == nil {
			return fmt.Errorf("option_payload")
		} else {
			return validateValue(g, inner.Reference, *v.Payload, budget-1)
		}
	case id(0x9042):
		var inner wire.Value
		var innerErr error
		if v.Kind != "result" || v.Payload == nil {
			return fmt.Errorf("result")
		}
		if v.Variant == "ok" {
			inner, innerErr = field(t, 0x9400)
		} else if v.Variant == "error" {
			inner, innerErr = field(t, 0x9401)
		} else {
			return fmt.Errorf("result_variant")
		}
		if innerErr != nil || inner.Tag != 6 {
			return fmt.Errorf("result_type")
		}
		return validateValue(g, inner.Reference, *v.Payload, budget-1)
	default:
		return fmt.Errorf("unsupported_type")
	}
	return nil
}

func scalarTypeKind(schema wire.ID) (string, bool) {
	switch schema {
	case id(0x9010):
		return "i64", true
	case id(0x9040):
		return "text", true
	case id(0x9041):
		return "bytes", true
	}
	return "", false
}

func expressionType(g wire.Envelope, expressionID wire.ID) (wire.ID, bool) {
	e, ok := g.Entities[expressionID]
	if !ok {
		return wire.ID{}, false
	}
	if e.Schema == id(0x9013) {
		parameter, err := field(e, 0x9130)
		if err != nil || parameter.Tag != 6 {
			return wire.ID{}, false
		}
		p, ok := g.Entities[parameter.Reference]
		if !ok || p.Schema != id(0x9012) {
			return wire.ID{}, false
		}
		t, err := field(p, 0x9121)
		return t.Reference, err == nil && t.Tag == 6
	}
	fields := map[wire.ID]uint64{id(0x90fb): 0x9fb0, id(0x90fc): 0x9fc0, id(0xa066): 0xa0660, id(0xa043): 0xa0430, id(0xa067): 0xa0670}
	if key, yes := fields[e.Schema]; yes {
		base, err := field(e, key)
		if err == nil && base.Tag == 6 {
			return expressionType(g, base.Reference)
		}
	}
	if e.Schema == id(0xa041) {
		t, err := field(e, 0xa0410)
		return t.Reference, err == nil && t.Tag == 6
	}
	if e.Schema == id(0xa068) {
		t, err := field(e, 0xa0680)
		return t.Reference, err == nil && t.Tag == 6
	}
	return wire.ID{}, false
}

func evalBlock(g wire.Envelope, block wire.ID, env map[wire.ID]Value, budget int) (Value, error) {
	v, returned, err := execBlock(g, block, env, budget)
	if err != nil {
		return Value{}, err
	}
	if !returned {
		return Value{}, fmt.Errorf("canonicaleval.missing_return")
	}
	return v, nil
}

func execBlock(g wire.Envelope, block wire.ID, env map[wire.ID]Value, budget int) (Value, bool, error) {
	if budget == 0 {
		return Value{}, false, fmt.Errorf("canonicaleval.budget")
	}
	b, exists := g.Entities[block]
	statements, e := field(b, 0x9800)
	if !exists || b.Schema != id(0x9080) || e != nil || statements.Tag != 7 {
		return Value{}, false, fmt.Errorf("canonicaleval.block")
	}
	for _, item := range statements.List {
		if item.Tag != 6 {
			return Value{}, false, fmt.Errorf("canonicaleval.statement")
		}
		s, ok := g.Entities[item.Reference]
		if !ok {
			return Value{}, false, fmt.Errorf("canonicaleval.statement_missing")
		}
		ref := func(k uint64) (wire.ID, error) {
			v, er := field(s, k)
			if er != nil || v.Tag != 6 {
				return wire.ID{}, fmt.Errorf("canonicaleval.statement_field")
			}
			return v.Reference, nil
		}
		switch s.Schema {
		case id(0x9081):
			values, er := field(s, 0x9810)
			if er != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
				return Value{}, false, fmt.Errorf("canonicaleval.return")
			}
			v, er := eval(g, values.List[0].Reference, env, budget-1)
			return v, true, er
		case id(0x90d1), id(0x90e1):
			key := uint64(0x9d10)
			definitionSchema := id(0x90d0)
			initializerField := uint64(0x9d02)
			if s.Schema == id(0x90e1) {
				key = 0x9e10
				definitionSchema = id(0x90e0)
				initializerField = 0x9e02
			}
			definitionID, er := ref(key)
			definition, ok := g.Entities[definitionID]
			if er != nil || !ok || definition.Schema != definitionSchema {
				return Value{}, false, fmt.Errorf("canonicaleval.declaration")
			}
			initializer, er := field(definition, initializerField)
			if er != nil || initializer.Tag != 6 {
				return Value{}, false, fmt.Errorf("canonicaleval.initializer")
			}
			if _, duplicate := env[definitionID]; duplicate {
				return Value{}, false, fmt.Errorf("canonicaleval.duplicate_declaration")
			}
			v, er := eval(g, initializer.Reference, env, budget-1)
			if er != nil {
				return Value{}, false, er
			}
			env[definitionID] = v
		case id(0x90e3):
			place, er := ref(0x9e30)
			valueID, ve := ref(0x9e31)
			if er != nil || ve != nil {
				return Value{}, false, fmt.Errorf("canonicaleval.assign")
			}
			if _, declared := env[place]; !declared {
				return Value{}, false, fmt.Errorf("canonicaleval.assign_scope")
			}
			v, er := eval(g, valueID, env, budget-1)
			if er != nil {
				return Value{}, false, er
			}
			env[place] = v
		case id(0x90f0), id(0x90c0):
			conditionField, thenField := uint64(0x9f00), uint64(0x9f01)
			if s.Schema == id(0x90c0) {
				conditionField, thenField = 0x9c00, 0x9c01
			}
			conditionID, er := ref(conditionField)
			bodyID, be := ref(thenField)
			if er != nil || be != nil {
				return Value{}, false, fmt.Errorf("canonicaleval.condition")
			}
			condition, er := eval(g, conditionID, env, budget-1)
			if er != nil || condition.Kind != "bool" {
				return Value{}, false, fmt.Errorf("canonicaleval.condition_type")
			}
			if condition.Bool {
				child := cloneEnv(env)
				v, returned, er := execBlock(g, bodyID, child, budget-1)
				mergeExisting(env, child)
				if er != nil || returned {
					return v, returned, er
				}
			} else if s.Schema == id(0x90c0) {
				elseID, ee := ref(0x9c02)
				if ee != nil {
					return Value{}, false, fmt.Errorf("canonicaleval.if_else")
				}
				child := cloneEnv(env)
				v, returned, er := execBlock(g, elseID, child, budget-1)
				mergeExisting(env, child)
				if er != nil || returned {
					return v, returned, er
				}
			}
		case id(0x90e4):
			conditionID, er := ref(0x9e40)
			bodyID, be := ref(0x9e41)
			if er != nil || be != nil {
				return Value{}, false, fmt.Errorf("canonicaleval.while")
			}
			for iteration := 0; iteration < 10000; iteration++ {
				condition, er := eval(g, conditionID, env, budget-1)
				if er != nil || condition.Kind != "bool" {
					return Value{}, false, fmt.Errorf("canonicaleval.while_condition")
				}
				if !condition.Bool {
					break
				}
				child := cloneEnv(env)
				v, returned, er := execBlock(g, bodyID, child, budget-1)
				mergeExisting(env, child)
				if er != nil || returned {
					return v, returned, er
				}
				if iteration == 9999 {
					return Value{}, false, fmt.Errorf("canonicaleval.loop_budget")
				}
			}
		default:
			return Value{}, false, fmt.Errorf("canonicaleval.unsupported_statement:%s", s.Schema.String())
		}
	}
	return Value{}, false, nil
}

func mergeExisting(target, source map[wire.ID]Value) {
	for key := range target {
		if value, ok := source[key]; ok {
			target[key] = value
		}
	}
}

func eval(g wire.Envelope, x wire.ID, env map[wire.ID]Value, budget int) (Value, error) {
	if budget == 0 {
		return Value{}, fmt.Errorf("canonicaleval.budget")
	}
	e, ok := g.Entities[x]
	if !ok {
		return Value{}, fmt.Errorf("canonicaleval.missing")
	}
	refField := func(k uint64) (wire.ID, error) {
		v, er := field(e, k)
		if er != nil || v.Tag != 6 {
			return wire.ID{}, fmt.Errorf("canonicaleval.field")
		}
		return v.Reference, nil
	}
	bin := func(a, b uint64) (Value, Value, error) {
		l, er := refField(a)
		if er != nil {
			return Value{}, Value{}, er
		}
		r, er := refField(b)
		if er != nil {
			return Value{}, Value{}, er
		}
		lv, er := eval(g, l, env, budget-1)
		if er != nil {
			return Value{}, Value{}, er
		}
		rv, er := eval(g, r, env, budget-1)
		return lv, rv, er
	}
	switch e.Schema {
	case id(0x9013):
		p, er := refField(0x9130)
		if er != nil {
			return Value{}, er
		}
		v, yes := env[p]
		if !yes {
			return Value{}, fmt.Errorf("canonicaleval.unbound")
		}
		return v, nil
	case id(0x90d2), id(0x90e2):
		key := uint64(0x9d20)
		definitionSchema := id(0x90d0)
		if e.Schema == id(0x90e2) {
			key = 0x9e20
			definitionSchema = id(0x90e0)
		}
		definition, er := refField(key)
		if er != nil {
			return Value{}, er
		}
		entity, exists := g.Entities[definition]
		v, bound := env[definition]
		if !exists || entity.Schema != definitionSchema || !bound {
			return Value{}, fmt.Errorf("canonicaleval.lexical_read")
		}
		return v, nil
	case id(0x9070):
		v, er := field(e, 0x9700)
		if er != nil || v.Tag != 3 {
			return Value{}, fmt.Errorf("canonicaleval.i64_literal")
		}
		return Value{Kind: "i64", I64: fmt.Sprint(int64(v.Unsigned))}, nil
	case id(0x9050):
		v, er := field(e, 0x9500)
		if er != nil || v.Tag != 5 {
			return Value{}, fmt.Errorf("canonicaleval.text_literal")
		}
		return Value{Kind: "text", Text: string(v.Bytes)}, nil
	case id(0xa064):
		v, er := field(e, 0xa0640)
		if er != nil || v.Tag != 5 {
			return Value{}, fmt.Errorf("canonicaleval.bytes_literal")
		}
		return Value{Kind: "bytes", Bytes: hex.EncodeToString(v.Bytes)}, nil
	case id(0x90b0):
		v, er := field(e, 0x9b00)
		if er != nil || v.Tag != 1 || v.Unsigned > 1 {
			return Value{}, fmt.Errorf("canonicaleval.bool_literal")
		}
		return Value{Kind: "bool", Bool: v.Unsigned == 1}, nil
	case id(0x9014):
		l, r, er := bin(0x9140, 0x9141)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.add:%w", er)
		}
		if l.Kind != "i64" || r.Kind != "i64" {
			return Value{}, fmt.Errorf("canonicaleval.add")
		}
		return add(l, r)
	case id(0x9090), id(0x90a0):
		leftField, rightField := uint64(0x9900), uint64(0x9901)
		if e.Schema == id(0x90a0) {
			leftField, rightField = 0x9a00, 0x9a01
		}
		l, r, er := bin(leftField, rightField)
		if er != nil || l.Kind != "i64" || r.Kind != "i64" {
			return Value{}, fmt.Errorf("canonicaleval.arithmetic")
		}
		a, aok := new(big.Int).SetString(l.I64, 10)
		b, bok := new(big.Int).SetString(r.I64, 10)
		if !aok || !bok {
			return Value{}, fmt.Errorf("canonicaleval.i64")
		}
		if e.Schema == id(0x9090) {
			a.Mul(a, b)
		} else {
			a.Sub(a, b)
		}
		mod := new(big.Int).Lsh(big.NewInt(1), 64)
		a.Mod(a, mod)
		sign := new(big.Int).Lsh(big.NewInt(1), 63)
		if a.Cmp(sign) >= 0 {
			a.Sub(a, mod)
		}
		return Value{Kind: "i64", I64: a.String()}, nil
	case id(0x9021):
		l, r, er := bin(0x9160, 0x9161)
		if er != nil || l.Kind != "i64" || r.Kind != "i64" {
			return Value{}, fmt.Errorf("canonicaleval.compare")
		}
		a, aok := new(big.Int).SetString(l.I64, 10)
		b, bok := new(big.Int).SetString(r.I64, 10)
		if !aok || !bok {
			return Value{}, fmt.Errorf("canonicaleval.i64")
		}
		return Value{Kind: "bool", Bool: a.Cmp(b) <= 0}, nil
	case id(0x90b1):
		leftID, er := refField(0x9b10)
		if er != nil {
			return Value{}, er
		}
		left, er := eval(g, leftID, env, budget-1)
		if er != nil || left.Kind != "bool" {
			return Value{}, fmt.Errorf("canonicaleval.and")
		}
		if !left.Bool {
			return left, nil
		}
		rightID, er := refField(0x9b11)
		if er != nil {
			return Value{}, er
		}
		right, er := eval(g, rightID, env, budget-1)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.and:%w", er)
		}
		if right.Kind != "bool" {
			return Value{}, fmt.Errorf("canonicaleval.and")
		}
		return right, nil
	case id(0x90c1):
		leftID, er := refField(0x9c10)
		if er != nil {
			return Value{}, er
		}
		left, er := eval(g, leftID, env, budget-1)
		if er != nil || left.Kind != "bool" {
			return Value{}, fmt.Errorf("canonicaleval.or")
		}
		if left.Bool {
			return left, nil
		}
		rightID, er := refField(0x9c11)
		if er != nil {
			return Value{}, er
		}
		right, er := eval(g, rightID, env, budget-1)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.or:%w", er)
		}
		if right.Kind != "bool" {
			return Value{}, fmt.Errorf("canonicaleval.or")
		}
		return right, nil
	case id(0x90c3):
		l, r, er := bin(0x9c30, 0x9c31)
		if er != nil || l.Kind != "text" || r.Kind != "text" {
			return Value{}, fmt.Errorf("canonicaleval.concat")
		}
		return Value{Kind: "text", Text: l.Text + r.Text}, nil
	case id(0x9032):
		baseID, er := refField(0x9320)
		if er != nil {
			return Value{}, er
		}
		memberID, er := refField(0x9321)
		if er != nil {
			return Value{}, er
		}
		baseExpression := g.Entities[baseID]
		parameterValue, pe := field(baseExpression, 0x9130)
		if baseExpression.Schema != id(0x9013) || pe != nil || parameterValue.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.record_base")
		}
		parameter := g.Entities[parameterValue.Reference]
		typeValue, te := field(parameter, 0x9121)
		if parameter.Schema != id(0x9012) || te != nil || typeValue.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.record_parameter")
		}
		recordType := g.Entities[typeValue.Reference]
		members, me := field(recordType, 0x9301)
		owned := false
		if te == nil && me == nil && members.Tag == 7 {
			for _, candidate := range members.List {
				if candidate.Tag == 6 && candidate.Reference == memberID {
					owned = true
				}
			}
		}
		if !owned {
			return Value{}, fmt.Errorf("canonicaleval.record_member")
		}
		memberEntity, exists := g.Entities[memberID]
		name, ne := field(memberEntity, 0x9310)
		if !exists || memberEntity.Schema != id(0x9031) || ne != nil || name.Tag != 5 {
			return Value{}, fmt.Errorf("canonicaleval.record_member")
		}
		base, er := eval(g, baseID, env, budget-1)
		v, yes := base.Fields[string(name.Bytes)]
		if er != nil || base.Kind != "record" || !yes {
			return Value{}, fmt.Errorf("canonicaleval.record")
		}
		return v, nil
	case id(0x90f3):
		typeValue, typeErr := field(e, 0x9f30)
		values, valuesErr := field(e, 0x9f31)
		if typeErr != nil || valuesErr != nil || typeValue.Tag != 6 || values.Tag != 7 {
			return Value{}, fmt.Errorf("canonicaleval.array_construct")
		}
		result := Value{Kind: "array", Items: make([]Value, len(values.List))}
		for index, item := range values.List {
			if item.Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.array_construct")
			}
			value, err := eval(g, item.Reference, env, budget-1)
			if err != nil {
				return Value{}, err
			}
			result.Items[index] = value
		}
		if err := validateValue(g, typeValue.Reference, result, 32); err != nil {
			return Value{}, fmt.Errorf("canonicaleval.array_construct:%w", err)
		}
		return result, nil
	case id(0x90f4):
		baseID, ber := refField(0x9f40)
		indexID, ier := refField(0x9f41)
		if ber != nil || ier != nil {
			return Value{}, fmt.Errorf("canonicaleval.index_fields")
		}
		base, er := eval(g, baseID, env, budget-1)
		index, indexErr := eval(g, indexID, env, budget-1)
		raw, parsed := new(big.Int).SetString(index.I64, 10)
		if er != nil || indexErr != nil || index.Kind != "i64" || !parsed || !raw.IsUint64() || base.Kind != "array" || raw.Uint64() >= uint64(len(base.Items)) {
			return Value{}, fmt.Errorf("canonicaleval.index")
		}
		return base.Items[raw.Uint64()], nil
	case id(0x90fa):
		baseID, ber := refField(0x9fa0)
		indexID, ier := refField(0x9fa1)
		if ber != nil || ier != nil {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_index_fields")
		}
		base, er := eval(g, baseID, env, budget-1)
		index, indexErr := eval(g, indexID, env, budget-1)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_index:%w", er)
		}
		if indexErr != nil {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_index:%w", indexErr)
		}
		raw, parsed := new(big.Int).SetString(index.I64, 10)
		if index.Kind != "i64" || !parsed || !raw.IsUint64() || base.Kind != "slice" || raw.Uint64() >= uint64(len(base.Items)) {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_index")
		}
		return base.Items[raw.Uint64()], nil
	case id(0x90f9):
		baseID, be := refField(0x9f90)
		if be != nil {
			return Value{}, fmt.Errorf("canonicaleval.length_field")
		}
		base, er := eval(g, baseID, env, budget-1)
		if er != nil || (base.Kind != "slice" && base.Kind != "array") || len(base.Items) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.length")
		}
		return Value{Kind: "i64", I64: fmt.Sprint(len(base.Items))}, nil
	case id(0xa068):
		typeID, te := refField(0xa0680)
		elements, ee := field(e, 0xa0681)
		if te != nil || ee != nil || elements.Tag != 7 || len(elements.List) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.slice_construct_fields")
		}
		items := make([]Value, len(elements.List))
		for index, element := range elements.List {
			if element.Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.slice_construct_element")
			}
			value, err := eval(g, element.Reference, env, budget-1)
			if err != nil {
				return Value{}, err
			}
			items[index] = value
		}
		result := Value{Kind: "slice", Items: items}
		if validateValue(g, typeID, result, budget-1) != nil {
			return Value{}, fmt.Errorf("canonicaleval.slice_construct_type")
		}
		return result, nil
	case id(0x90fb), id(0x90fc), id(0xa066):
		collectionField, indexField, valueField := uint64(0x9fb0), uint64(0), uint64(0x9fb1)
		if e.Schema == id(0x90fc) {
			collectionField, indexField, valueField = 0x9fc0, 0x9fc1, 0x9fc2
		}
		if e.Schema == id(0xa066) {
			collectionField, indexField, valueField = 0xa0660, 0xa0661, 0
		}
		collectionID, ce := refField(collectionField)
		if ce != nil {
			return Value{}, fmt.Errorf("canonicaleval.slice_operation_fields")
		}
		base, er := eval(g, collectionID, env, budget-1)
		if er != nil {
			return Value{}, er
		}
		if base.Kind != "slice" || len(base.Items) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.slice_operation")
		}
		items := append([]Value(nil), base.Items...)
		if indexField != 0 {
			indexID, ie := refField(indexField)
			if ie != nil {
				return Value{}, fmt.Errorf("canonicaleval.slice_operation_fields")
			}
			index, ie := eval(g, indexID, env, budget-1)
			raw, parsed := new(big.Int).SetString(index.I64, 10)
			if ie != nil || index.Kind != "i64" || !parsed || !raw.IsUint64() || raw.Uint64() >= uint64(len(items)) {
				return Value{}, fmt.Errorf("canonicaleval.slice_operation_index")
			}
			position := int(raw.Uint64())
			if valueField == 0 {
				items = append(append([]Value(nil), items[:position]...), items[position+1:]...)
			} else {
				valueID, ve := refField(valueField)
				if ve != nil {
					return Value{}, fmt.Errorf("canonicaleval.slice_operation_fields")
				}
				value, ve := eval(g, valueID, env, budget-1)
				if ve != nil {
					return Value{}, ve
				}
				items[position] = value
			}
		} else {
			valueID, ve := refField(valueField)
			if ve != nil {
				return Value{}, fmt.Errorf("canonicaleval.slice_operation_fields")
			}
			value, ve := eval(g, valueID, env, budget-1)
			if ve != nil || len(items) == 512 {
				return Value{}, fmt.Errorf("canonicaleval.slice_append")
			}
			items = append(items, value)
		}
		result := Value{Kind: "slice", Items: items}
		if typeID, ok := expressionType(g, collectionID); !ok || validateValue(g, typeID, result, budget-1) != nil {
			return Value{}, fmt.Errorf("canonicaleval.slice_operation_type")
		}
		return result, nil
	case id(0xa041):
		typeID, te := refField(0xa0410)
		if te != nil {
			return Value{}, fmt.Errorf("canonicaleval.empty_map")
		}
		t, ok := g.Entities[typeID]
		value, ve := field(t, 0xa0401)
		valueEntity, exists := g.Entities[value.Reference]
		kind, known := scalarTypeKind(valueEntity.Schema)
		if !ok || t.Schema != id(0xa040) || ve != nil || !exists || !known {
			return Value{}, fmt.Errorf("canonicaleval.empty_map_type")
		}
		return Value{Kind: "map", ValueType: kind}, nil
	case id(0xa043), id(0xa067):
		mapField, keyField, valueField := uint64(0xa0430), uint64(0xa0431), uint64(0xa0432)
		if e.Schema == id(0xa067) {
			mapField, keyField, valueField = 0xa0670, 0xa0671, 0
		}
		mapID, me := refField(mapField)
		keyID, ke := refField(keyField)
		if me != nil || ke != nil {
			return Value{}, fmt.Errorf("canonicaleval.map_operation_fields")
		}
		m, er := eval(g, mapID, env, budget-1)
		key, ker := eval(g, keyID, env, budget-1)
		if er != nil {
			return Value{}, er
		}
		if ker != nil {
			return Value{}, ker
		}
		if m.Kind != "map" {
			return Value{}, fmt.Errorf("canonicaleval.map_operation")
		}
		entries := append([]Entry(nil), m.Entries...)
		found := -1
		for i, item := range entries {
			if equal(item.Key, key) {
				found = i
				break
			}
		}
		if valueField == 0 {
			if found >= 0 {
				entries = append(entries[:found], entries[found+1:]...)
			}
		} else {
			valueID, ve := refField(valueField)
			if ve != nil {
				return Value{}, fmt.Errorf("canonicaleval.map_operation_fields")
			}
			value, ve := eval(g, valueID, env, budget-1)
			if ve != nil {
				return Value{}, ve
			}
			if found >= 0 {
				entries[found].Value = value
			} else {
				if len(entries) == 512 {
					return Value{}, fmt.Errorf("canonicaleval.map_size")
				}
				entries = append(entries, Entry{Key: key, Value: value})
			}
		}
		result := Value{Kind: "map", ValueType: m.ValueType, Entries: entries}
		if typeID, ok := expressionType(g, mapID); !ok || validateValue(g, typeID, result, budget-1) != nil {
			return Value{}, fmt.Errorf("canonicaleval.map_operation_type")
		}
		return result, nil
	case id(0xa042):
		mapID, me := refField(0xa0420)
		keyID, ke := refField(0xa0421)
		if me != nil || ke != nil {
			return Value{}, fmt.Errorf("canonicaleval.map_fields")
		}
		m, er := eval(g, mapID, env, budget-1)
		if er != nil || m.Kind != "map" || len(m.Entries) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.map")
		}
		key, er := eval(g, keyID, env, budget-1)
		if er != nil {
			return Value{}, er
		}
		for _, item := range m.Entries {
			if equal(item.Key, key) {
				return item.Value, nil
			}
		}
		return zero(m.ValueType)
	case id(0xa061):
		binding, er := refField(0xa0610)
		if er != nil {
			return Value{}, er
		}
		bindingEntity, exists := g.Entities[binding]
		v, yes := env[binding]
		if !exists || bindingEntity.Schema != id(0xa060) || !yes {
			return Value{}, fmt.Errorf("canonicaleval.unbound_variant")
		}
		return v, nil
	case id(0xa063):
		valueID, ve := refField(0xa0630)
		if ve != nil {
			return Value{}, fmt.Errorf("canonicaleval.option_fields")
		}
		v, er := eval(g, valueID, env, budget-1)
		if er != nil || v.Kind != "option" {
			return Value{}, fmt.Errorf("canonicaleval.option")
		}
		if v.Variant == "none" {
			b, be := refField(0xa0631)
			if be != nil {
				return Value{}, fmt.Errorf("canonicaleval.option_fields")
			}
			return evalBlock(g, b, env, budget-1)
		}
		binding, bie := refField(0xa0632)
		body, boe := refField(0xa0633)
		if bie != nil || boe != nil {
			return Value{}, fmt.Errorf("canonicaleval.option_fields")
		}
		bindEntity, exists := g.Entities[binding]
		if !exists || bindEntity.Schema != id(0xa060) {
			return Value{}, fmt.Errorf("canonicaleval.option_binding")
		}
		next := cloneEnv(env)
		next[binding] = *v.Payload
		return evalBlock(g, body, next, budget-1)
	case id(0xa062):
		valueID, ve := refField(0xa0620)
		if ve != nil {
			return Value{}, fmt.Errorf("canonicaleval.result_fields")
		}
		v, er := eval(g, valueID, env, budget-1)
		if er != nil || v.Kind != "result" || v.Payload == nil {
			return Value{}, fmt.Errorf("canonicaleval.result")
		}
		var binding, body wire.ID
		if v.Variant == "ok" {
			binding, ve = refField(0xa0621)
			if ve == nil {
				body, ve = refField(0xa0622)
			}
		} else if v.Variant == "error" {
			binding, ve = refField(0xa0623)
			if ve == nil {
				body, ve = refField(0xa0624)
			}
		} else {
			return Value{}, fmt.Errorf("canonicaleval.result_variant")
		}
		if ve != nil {
			return Value{}, fmt.Errorf("canonicaleval.result_fields")
		}
		bindEntity, exists := g.Entities[binding]
		if !exists || bindEntity.Schema != id(0xa060) {
			return Value{}, fmt.Errorf("canonicaleval.result_binding")
		}
		next := cloneEnv(env)
		next[binding] = *v.Payload
		return evalBlock(g, body, next, budget-1)
	case id(0xa065):
		l, r, er := bin(0xa0650, 0xa0651)
		if er != nil || l.Kind != "bytes" || r.Kind != "bytes" {
			return Value{}, fmt.Errorf("canonicaleval.bytes_equal")
		}
		equal, er := canonicalBytesEqual(l.Bytes, r.Bytes)
		if er != nil {
			return Value{}, er
		}
		return Value{Kind: "bool", Bool: equal}, nil
	case id(0x90c2):
		l, r, er := bin(0x9c20, 0x9c21)
		if er != nil || l.Kind != "text" || r.Kind != "text" {
			return Value{}, fmt.Errorf("canonicaleval.text_equal")
		}
		return Value{Kind: "bool", Bool: l.Text == r.Text}, nil
	}
	return Value{}, fmt.Errorf("canonicaleval.unsupported:%s", e.Schema.String())
}

func canonicalBytesEqual(left, right string) (bool, error) {
	a, e := hex.DecodeString(left)
	if e != nil {
		return false, fmt.Errorf("canonicaleval.bytes")
	}
	b, e := hex.DecodeString(right)
	if e != nil {
		return false, fmt.Errorf("canonicaleval.bytes")
	}
	return bytes.Equal(a, b), nil
}

func add(a, b Value) (Value, error) {
	x, ok := new(big.Int).SetString(a.I64, 10)
	if !ok {
		return Value{}, fmt.Errorf("canonicaleval.i64")
	}
	y, ok := new(big.Int).SetString(b.I64, 10)
	if !ok {
		return Value{}, fmt.Errorf("canonicaleval.i64")
	}
	x.Add(x, y)
	mod := new(big.Int).Lsh(big.NewInt(1), 64)
	x.Mod(x, mod)
	sign := new(big.Int).Lsh(big.NewInt(1), 63)
	if x.Cmp(sign) >= 0 {
		x.Sub(x, mod)
	}
	return Value{Kind: "i64", I64: x.String()}, nil
}
func equal(a, b Value) bool {
	return a.Kind == b.Kind && a.I64 == b.I64 && a.Text == b.Text && a.Bytes == b.Bytes && a.Bool == b.Bool
}
func zero(kind string) (Value, error) {
	switch kind {
	case "i64":
		return Value{Kind: "i64", I64: "0"}, nil
	case "text":
		return Value{Kind: "text"}, nil
	case "bytes":
		return Value{Kind: "bytes"}, nil
	}
	return Value{}, fmt.Errorf("canonicaleval.zero")
}
func cloneEnv(in map[wire.ID]Value) map[wire.ID]Value {
	out := map[wire.ID]Value{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func id(v uint64) wire.ID {
	var x wire.ID
	for i := 0; i < 8; i++ {
		x[15-i] = byte(v)
		v >>= 8
	}
	return x
}
func field(e wire.Entity, k uint64) (wire.Value, error) {
	v, ok := e.Fields[id(k)]
	if !ok {
		return wire.Value{}, fmt.Errorf("canonicaleval.missing_field")
	}
	return v, nil
}
func schemaEntities(g wire.Envelope, s uint64) []wire.Entity {
	out := []wire.Entity{}
	for _, e := range g.Entities {
		if e.Schema == id(s) {
			out = append(out, e)
		}
	}
	return out
}
