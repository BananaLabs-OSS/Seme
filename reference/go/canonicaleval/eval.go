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
	Kind          string           `json:"kind"`
	I64           string           `json:"i64,omitempty"`
	Bool          bool             `json:"bool,omitempty"`
	Text          string           `json:"text,omitempty"`
	Bytes         string           `json:"bytes_hex,omitempty"`
	Items         []Value          `json:"items,omitempty"`
	Fields        map[string]Value `json:"fields,omitempty"`
	Entries       []Entry          `json:"entries,omitempty"`
	ValueType     string           `json:"value_type,omitempty"`
	Variant       string           `json:"variant,omitempty"`
	Payload       *Value           `json:"payload,omitempty"`
	State         *Value           `json:"state,omitempty"`
	Result        *Value           `json:"result,omitempty"`
	interfaceType wire.ID
	witness       wire.ID
	closure       *closureValue
}
type closureValue struct {
	typeID     wire.ID
	parameters []wire.ID
	paramTypes []wire.ID
	resultType wire.ID
	captures   map[wire.ID]Value
	body       wire.ID
	mutable    bool
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
	case id(0xa004):
		if er := recurse(0xa0040); er != nil {
			return er
		}
		return recurse(0xa0041)
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
	case id(0xa004):
		stateType, se := field(t, 0xa0040)
		resultType, re := field(t, 0xa0041)
		if se != nil || re != nil || stateType.Tag != 6 || resultType.Tag != 6 || v.Kind != "transition" || v.State == nil || v.Result == nil {
			return fmt.Errorf("transition")
		}
		if e := validateValue(g, stateType.Reference, *v.State, budget-1); e != nil {
			return fmt.Errorf("transition_state:%w", e)
		}
		if e := validateValue(g, resultType.Reference, *v.Result, budget-1); e != nil {
			return fmt.Errorf("transition_result:%w", e)
		}
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
	if e.Schema == id(0x90d2) || e.Schema == id(0x90e2) {
		fieldID, bindingSchema, typeField := uint64(0x9d20), id(0x90d0), uint64(0x9d01)
		if e.Schema == id(0x90e2) {
			fieldID, bindingSchema, typeField = 0x9e20, id(0x90e0), 0x9e01
		}
		binding, err := field(e, fieldID)
		if err != nil || binding.Tag != 6 {
			return wire.ID{}, false
		}
		b, ok := g.Entities[binding.Reference]
		if !ok || b.Schema != bindingSchema {
			return wire.ID{}, false
		}
		t, err := field(b, typeField)
		return t.Reference, err == nil && t.Tag == 6
	}
	if e.Schema == id(0x9033) {
		t, err := field(e, 0x9330)
		return t.Reference, err == nil && t.Tag == 6
	}
	if e.Schema == id(0xa013) {
		t, err := field(e, 0xa0130)
		return t.Reference, err == nil && t.Tag == 6
	}
	if e.Schema == id(0xa001) {
		binding, err := field(e, 0xa0010)
		if err != nil || binding.Tag != 6 {
			return wire.ID{}, false
		}
		b, ok := g.Entities[binding.Reference]
		if !ok || b.Schema != id(0xa000) {
			return wire.ID{}, false
		}
		t, err := field(b, 0xa0001)
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
	if e.Schema == id(0xa005) {
		t, err := field(e, 0xa0050)
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
	case id(0x9043), id(0x9044):
		typeField, valueField, variant := uint64(0x9410), uint64(0x9411), "ok"
		if e.Schema == id(0x9044) {
			typeField, valueField, variant = 0x9420, 0x9421, "error"
		}
		typeID, te := refField(typeField)
		valueID, ve := refField(valueField)
		resultType, exists := g.Entities[typeID]
		okType, oe := field(resultType, 0x9400)
		errorType, ee := field(resultType, 0x9401)
		if te != nil || ve != nil || !exists || resultType.Schema != id(0x9042) || oe != nil || ee != nil || okType.Tag != 6 || errorType.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.result_construct")
		}
		value, err := eval(g, valueID, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		expected := okType.Reference
		if variant == "error" {
			expected = errorType.Reference
		}
		if validateValue(g, expected, value, 16) != nil {
			return Value{}, fmt.Errorf("canonicaleval.result_construct_type")
		}
		return Value{Kind: "result", Variant: variant, Payload: &value}, nil
	case id(0xa022), id(0xa031):
		key := uint64(0xa0220)
		if e.Schema == id(0xa031) {
			key = 0xa0310
		}
		binding, er := refField(key)
		value, ok := env[binding]
		if er != nil || !ok {
			return Value{}, fmt.Errorf("canonicaleval.capture_read")
		}
		return value, nil
	case id(0xa023), id(0xa034):
		tf, pf, cf, bf := uint64(0xa0230), uint64(0xa0231), uint64(0xa0232), uint64(0xa0233)
		cs, initial, mutable := id(0xa021), uint64(0xa0212), false
		if e.Schema == id(0xa034) {
			tf, pf, cf, bf, cs, initial, mutable = 0xa0340, 0xa0341, 0xa0342, 0xa0343, id(0xa030), 0xa0302, true
		}
		typeID, te := refField(tf)
		parameters, pe := field(e, pf)
		captures, ce := field(e, cf)
		body, be := refField(bf)
		if te != nil || pe != nil || ce != nil || be != nil || parameters.Tag != 7 || captures.Tag != 7 || g.Entities[typeID].Schema != id(0xa020) {
			return Value{}, fmt.Errorf("canonicaleval.closure_construct")
		}
		closure := &closureValue{typeID: typeID, captures: map[wire.ID]Value{}, body: body, mutable: mutable}
		functionType := g.Entities[typeID]
		typeParameters, tpe := field(functionType, 0xa0200)
		typeResult, tre := field(functionType, 0xa0201)
		if tpe != nil || tre != nil || typeParameters.Tag != 7 || typeResult.Tag != 6 || len(typeParameters.List) != len(parameters.List) {
			return Value{}, fmt.Errorf("canonicaleval.closure_type")
		}
		closure.resultType = typeResult.Reference
		for _, item := range parameters.List {
			if item.Tag != 6 || g.Entities[item.Reference].Schema != id(0x9012) {
				return Value{}, fmt.Errorf("canonicaleval.closure_parameter")
			}
			position := len(closure.parameters)
			parameterType, err := field(g.Entities[item.Reference], 0x9121)
			if err != nil || parameterType.Tag != 6 || typeParameters.List[position].Tag != 6 || parameterType.Reference != typeParameters.List[position].Reference {
				return Value{}, fmt.Errorf("canonicaleval.closure_parameter_type")
			}
			closure.parameters = append(closure.parameters, item.Reference)
			closure.paramTypes = append(closure.paramTypes, parameterType.Reference)
		}
		for _, item := range captures.List {
			if item.Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.closure_capture")
			}
			capture, ok := g.Entities[item.Reference]
			source, err := field(capture, initial)
			if !ok || capture.Schema != cs || err != nil || source.Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.closure_capture")
			}
			value, err := eval(g, source.Reference, env, budget-1)
			if err != nil {
				return Value{}, err
			}
			captureType, typeErr := field(capture, initial-1)
			if typeErr != nil || captureType.Tag != 6 || validateValue(g, captureType.Reference, value, 16) != nil {
				return Value{}, fmt.Errorf("canonicaleval.closure_capture_type")
			}
			closure.captures[capture.ID] = value
		}
		return Value{Kind: "closure", closure: closure}, nil
	case id(0xa024), id(0xa035):
		calleeField, argumentsField, stateful := uint64(0xa0240), uint64(0xa0241), false
		if e.Schema == id(0xa035) {
			calleeField, argumentsField, stateful = 0xa0350, 0xa0351, true
		}
		calleeID, ce := refField(calleeField)
		arguments, ae := field(e, argumentsField)
		callee, err := eval(g, calleeID, env, budget-1)
		if ce != nil || ae != nil || err != nil || arguments.Tag != 7 || callee.Kind != "closure" || callee.closure == nil || callee.closure.mutable != stateful {
			return Value{}, fmt.Errorf("canonicaleval.indirect_call")
		}
		result, next, err := invokeClosure(g, callee.closure, arguments, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		if !stateful {
			return result, nil
		}
		state := Value{Kind: "closure", closure: next}
		return Value{Kind: "transition", State: &state, Result: &result}, nil
	case id(0xa005):
		typeID, te := refField(0xa0050)
		stateID, se := refField(0xa0051)
		resultID, re := refField(0xa0052)
		transitionType, exists := g.Entities[typeID]
		stateType, ste := field(transitionType, 0xa0040)
		resultType, rte := field(transitionType, 0xa0041)
		if te != nil || se != nil || re != nil || !exists || transitionType.Schema != id(0xa004) || ste != nil || rte != nil || stateType.Tag != 6 || resultType.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.transition")
		}
		state, err := eval(g, stateID, env, budget-1)
		if err != nil || validateValue(g, stateType.Reference, state, budget-1) != nil {
			return Value{}, fmt.Errorf("canonicaleval.transition_state")
		}
		result, err := eval(g, resultID, env, budget-1)
		if err != nil || validateValue(g, resultType.Reference, result, budget-1) != nil {
			return Value{}, fmt.Errorf("canonicaleval.transition_result")
		}
		return Value{Kind: "transition", State: &state, Result: &result}, nil
	case id(0xa032):
		capture, ce := refField(0xa0320)
		valueID, ve := refField(0xa0321)
		if ce != nil || ve != nil {
			return Value{}, fmt.Errorf("canonicaleval.capture_update")
		}
		if _, ok := env[capture]; !ok {
			return Value{}, fmt.Errorf("canonicaleval.capture_ownership")
		}
		value, err := eval(g, valueID, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		binding := g.Entities[capture]
		captureType, typeErr := field(binding, 0xa0301)
		if binding.Schema != id(0xa030) || typeErr != nil || captureType.Tag != 6 || validateValue(g, captureType.Reference, value, 16) != nil {
			return Value{}, fmt.Errorf("canonicaleval.capture_update_type")
		}
		env[capture] = value
		return value, nil
	case id(0xa033):
		steps, se := field(e, 0xa0330)
		resultID, re := refField(0xa0331)
		if se != nil || re != nil || steps.Tag != 7 {
			return Value{}, fmt.Errorf("canonicaleval.sequence")
		}
		for _, step := range steps.List {
			if step.Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.sequence")
			}
			if _, err := eval(g, step.Reference, env, budget-1); err != nil {
				return Value{}, err
			}
		}
		return eval(g, resultID, env, budget-1)
	case id(0xa006), id(0xa007):
		key := uint64(0xa0060)
		if e.Schema == id(0xa007) {
			key = 0xa0070
		}
		transitionID, er := refField(key)
		transition, err := eval(g, transitionID, env, budget-1)
		if er != nil || err != nil || transition.Kind != "transition" || transition.State == nil || transition.Result == nil {
			return Value{}, fmt.Errorf("canonicaleval.transition_projection")
		}
		if e.Schema == id(0xa006) {
			return *transition.State, nil
		}
		return *transition.Result, nil
	case id(0xa001):
		receiver, er := refField(0xa0010)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.receiver_read")
		}
		value, ok := env[receiver]
		if !ok {
			return Value{}, fmt.Errorf("canonicaleval.receiver_read")
		}
		return value, nil
	case id(0xa003):
		receiver, re := refField(0xa0030)
		method, me := refField(0xa0031)
		arguments, ae := field(e, 0xa0032)
		if re != nil || me != nil || ae != nil || arguments.Tag != 7 {
			return Value{}, fmt.Errorf("canonicaleval.method_call")
		}
		rv, err := eval(g, receiver, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		concreteType, ok := expressionType(g, receiver)
		if !ok {
			return Value{}, fmt.Errorf("canonicaleval.method_receiver")
		}
		return evalMethod(g, method, concreteType, rv, arguments, env, budget-1)
	case id(0xa013):
		contract, ce := refField(0xa0130)
		concrete, ve := refField(0xa0131)
		witness, we := refField(0xa0132)
		w, exists := g.Entities[witness]
		if ce != nil || ve != nil || we != nil || !exists || w.Schema != id(0xa012) {
			return Value{}, fmt.Errorf("canonicaleval.interface_value")
		}
		wContract, err := field(w, 0xa0121)
		if err != nil || wContract.Tag != 6 || wContract.Reference != contract {
			return Value{}, fmt.Errorf("canonicaleval.interface_value")
		}
		wConcrete, err := field(w, 0xa0120)
		concreteType, typed := expressionType(g, concrete)
		if err != nil || wConcrete.Tag != 6 || !typed || wConcrete.Reference != concreteType {
			return Value{}, fmt.Errorf("canonicaleval.interface_value")
		}
		value, err := eval(g, concrete, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		value.interfaceType, value.witness = contract, witness
		return value, nil
	case id(0xa014):
		receiver, re := refField(0xa0140)
		requirement, qe := refField(0xa0141)
		arguments, ae := field(e, 0xa0142)
		if re != nil || qe != nil || ae != nil || arguments.Tag != 7 {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_call")
		}
		rv, err := eval(g, receiver, env, budget-1)
		if err != nil || rv.interfaceType == (wire.ID{}) || rv.witness == (wire.ID{}) {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_call")
		}
		contract := g.Entities[rv.interfaceType]
		requirements, err := field(contract, 0xa0101)
		witness := g.Entities[rv.witness]
		methods, me := field(witness, 0xa0122)
		witnessContract, wce := field(witness, 0xa0121)
		witnessConcrete, wte := field(witness, 0xa0120)
		if err != nil || me != nil || contract.Schema != id(0xa010) || witness.Schema != id(0xa012) || requirements.Tag != 7 || methods.Tag != 7 || len(requirements.List) != len(methods.List) {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_call")
		}
		if wce != nil || wte != nil || witnessContract.Tag != 6 || witnessContract.Reference != rv.interfaceType || witnessConcrete.Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_witness")
		}
		method, found := wire.ID{}, false
		for i, item := range requirements.List {
			if item.Tag == 6 && item.Reference == requirement && methods.List[i].Tag == 6 {
				method, found = methods.List[i].Reference, true
			}
		}
		if !found {
			return Value{}, fmt.Errorf("canonicaleval.dynamic_requirement")
		}
		if err := validateMethodRequirement(g, method, requirement, witnessConcrete.Reference); err != nil {
			return Value{}, err
		}
		return evalMethod(g, method, witnessConcrete.Reference, rv, arguments, env, budget-1)
	case id(0x9060):
		callee, ce := refField(0x9600)
		arguments, ae := field(e, 0x9601)
		fn, exists := g.Entities[callee]
		if ce != nil || ae != nil || !exists || fn.Schema != id(0x9011) || arguments.Tag != 7 {
			return Value{}, fmt.Errorf("canonicaleval.call")
		}
		parameters, pe := field(fn, 0x9111)
		body, be := field(fn, 0x9113)
		if pe != nil || be != nil || parameters.Tag != 7 || body.Tag != 6 || len(parameters.List) != len(arguments.List) {
			return Value{}, fmt.Errorf("canonicaleval.call")
		}
		callEnv := cloneEnv(env)
		for index := range arguments.List {
			if arguments.List[index].Tag != 6 || parameters.List[index].Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.call")
			}
			value, err := eval(g, arguments.List[index].Reference, env, budget-1)
			if err != nil {
				return Value{}, err
			}
			callEnv[parameters.List[index].Reference] = value
		}
		return evalBlock(g, body.Reference, callEnv, budget-1)
	case id(0x9033):
		typeID, te := refField(0x9330)
		values, ve := field(e, 0x9331)
		record, exists := g.Entities[typeID]
		fields, fe := field(record, 0x9301)
		if te != nil || ve != nil || fe != nil || !exists || record.Schema != id(0x9030) || values.Tag != 7 || fields.Tag != 7 || len(values.List) != len(fields.List) {
			return Value{}, fmt.Errorf("canonicaleval.record_construct")
		}
		result := Value{Kind: "record", Fields: map[string]Value{}}
		for i := range values.List {
			if values.List[i].Tag != 6 || fields.List[i].Tag != 6 {
				return Value{}, fmt.Errorf("canonicaleval.record_construct")
			}
			fieldEntity, ok := g.Entities[fields.List[i].Reference]
			name, ne := field(fieldEntity, 0x9310)
			if !ok || fieldEntity.Schema != id(0x9031) || ne != nil || name.Tag != 5 {
				return Value{}, fmt.Errorf("canonicaleval.record_construct")
			}
			value, err := eval(g, values.List[i].Reference, env, budget-1)
			if err != nil {
				return Value{}, err
			}
			result.Fields[string(name.Bytes)] = value
		}
		return result, nil
	case id(0x9032):
		valueID, ve := refField(0x9320)
		fieldID, fe := refField(0x9321)
		if ve != nil || fe != nil {
			return Value{}, fmt.Errorf("canonicaleval.field_read")
		}
		record, err := eval(g, valueID, env, budget-1)
		if err != nil || record.Kind != "record" {
			return Value{}, fmt.Errorf("canonicaleval.field_read")
		}
		typeID, typed := expressionType(g, valueID)
		recordType, exists := g.Entities[typeID]
		members, me := field(recordType, 0x9301)
		member := false
		if typed && exists && recordType.Schema == id(0x9030) && me == nil && members.Tag == 7 {
			for _, item := range members.List {
				member = member || item.Tag == 6 && item.Reference == fieldID
			}
		}
		fieldEntity, exists := g.Entities[fieldID]
		name, ne := field(fieldEntity, 0x9310)
		if !member || !exists || fieldEntity.Schema != id(0x9031) || ne != nil || name.Tag != 5 {
			return Value{}, fmt.Errorf("canonicaleval.record_member")
		}
		value, ok := record.Fields[string(name.Bytes)]
		if !ok {
			return Value{}, fmt.Errorf("canonicaleval.field_read")
		}
		return value, nil
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
	case id(0x90f6):
		binding, er := refField(0x9f60)
		value, exists := env[binding]
		if er != nil || !exists || g.Entities[binding].Schema != id(0x90f5) {
			return Value{}, fmt.Errorf("canonicaleval.iteration_read")
		}
		return value, nil
	case id(0x90f7):
		collectionID, ce := refField(0x9f70)
		initialID, ie := refField(0x9f71)
		accumulatorID, ae := refField(0x9f72)
		elementID, ee := refField(0x9f73)
		bodyID, be := refField(0x9f74)
		if ce != nil || ie != nil || ae != nil || ee != nil || be != nil || g.Entities[accumulatorID].Schema != id(0x90f5) || g.Entities[elementID].Schema != id(0x90f5) {
			return Value{}, fmt.Errorf("canonicaleval.fold_fields")
		}
		collection, er := eval(g, collectionID, env, budget-1)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.fold_collection:%w", er)
		}
		if (collection.Kind != "slice" && collection.Kind != "array") || len(collection.Items) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.fold_collection")
		}
		accumulator, er := eval(g, initialID, env, budget-1)
		if er != nil {
			return Value{}, fmt.Errorf("canonicaleval.fold_initial:%w", er)
		}
		for _, element := range collection.Items {
			scoped := cloneEnv(env)
			scoped[accumulatorID] = accumulator
			scoped[elementID] = element
			accumulator, er = eval(g, bodyID, scoped, budget-1)
			if er != nil {
				return Value{}, fmt.Errorf("canonicaleval.fold_body:%w", er)
			}
		}
		return accumulator, nil
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
	case id(0x90f7):
		collectionID, ce := refField(0x9f70)
		initialID, ie := refField(0x9f71)
		accumulatorID, ae := refField(0x9f72)
		elementID, ee := refField(0x9f73)
		bodyID, be := refField(0x9f74)
		accumulatorEntity, accumulatorOK := g.Entities[accumulatorID]
		elementEntity, elementOK := g.Entities[elementID]
		if ce != nil || ie != nil || ae != nil || ee != nil || be != nil || !accumulatorOK || !elementOK || accumulatorEntity.Schema != id(0x90f5) || elementEntity.Schema != id(0x90f5) || accumulatorID == elementID {
			return Value{}, fmt.Errorf("canonicaleval.fold_fields")
		}
		collection, err := eval(g, collectionID, env, budget-1)
		if err != nil || (collection.Kind != "slice" && collection.Kind != "array") || len(collection.Items) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.fold_collection")
		}
		accumulator, err := eval(g, initialID, env, budget-1)
		if err != nil {
			return Value{}, err
		}
		for _, element := range collection.Items {
			next := cloneEnv(env)
			next[accumulatorID], next[elementID] = accumulator, element
			accumulator, err = eval(g, bodyID, next, budget-1)
			if err != nil {
				return Value{}, err
			}
		}
		return accumulator, nil
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
	case id(0xa044):
		mapID, me := refField(0xa0440)
		keyID, ke := refField(0xa0441)
		optionID, oe := refField(0xa0442)
		mapTypeID, mapTypeKnown := expressionType(g, mapID)
		mapType, mapTypeExists := g.Entities[mapTypeID]
		optionType, optionExists := g.Entities[optionID]
		mapValueType, mve := field(mapType, 0xa0401)
		optionValueType, ove := field(optionType, 0xa0500)
		if me != nil || ke != nil || oe != nil || !mapTypeKnown || !mapTypeExists || mapType.Schema != id(0xa040) || !optionExists || optionType.Schema != id(0xa050) || mve != nil || ove != nil || mapValueType.Tag != 6 || optionValueType.Tag != 6 || mapValueType.Reference != optionValueType.Reference {
			return Value{}, fmt.Errorf("canonicaleval.map_option_type")
		}
		m, er := eval(g, mapID, env, budget-1)
		if er != nil || m.Kind != "map" || len(m.Entries) > 512 {
			return Value{}, fmt.Errorf("canonicaleval.map_option")
		}
		key, er := eval(g, keyID, env, budget-1)
		if er != nil {
			return Value{}, er
		}
		for _, item := range m.Entries {
			if equal(item.Key, key) {
				payload := item.Value
				return Value{Kind: "option", Variant: "some", Payload: &payload}, nil
			}
		}
		return Value{Kind: "option", Variant: "none"}, nil
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

func evalMethod(g wire.Envelope, methodID, concreteType wire.ID, receiver Value, arguments wire.Value, outer map[wire.ID]Value, budget int) (Value, error) {
	if budget <= 0 {
		return Value{}, fmt.Errorf("canonicaleval.budget")
	}
	method, ok := g.Entities[methodID]
	if !ok || method.Schema != id(0xa002) || arguments.Tag != 7 {
		return Value{}, fmt.Errorf("canonicaleval.method")
	}
	receiverBinding, re := field(method, 0xa0021)
	parameters, pe := field(method, 0xa0022)
	body, be := field(method, 0xa0024)
	if re != nil || pe != nil || be != nil || receiverBinding.Tag != 6 || parameters.Tag != 7 || body.Tag != 6 || len(parameters.List) != len(arguments.List) {
		return Value{}, fmt.Errorf("canonicaleval.method")
	}
	binding, exists := g.Entities[receiverBinding.Reference]
	bindingType, te := field(binding, 0xa0001)
	if !exists || binding.Schema != id(0xa000) || te != nil || bindingType.Tag != 6 || bindingType.Reference != concreteType {
		return Value{}, fmt.Errorf("canonicaleval.method_receiver")
	}
	env := cloneEnv(outer)
	env[receiverBinding.Reference] = receiver
	for i := range parameters.List {
		if parameters.List[i].Tag != 6 || arguments.List[i].Tag != 6 {
			return Value{}, fmt.Errorf("canonicaleval.method")
		}
		value, err := eval(g, arguments.List[i].Reference, outer, budget-1)
		if err != nil {
			return Value{}, err
		}
		env[parameters.List[i].Reference] = value
	}
	return evalBlock(g, body.Reference, env, budget-1)
}

func invokeClosure(g wire.Envelope, closure *closureValue, arguments wire.Value, outer map[wire.ID]Value, budget int) (Value, *closureValue, error) {
	if budget <= 0 || closure == nil || arguments.Tag != 7 || len(arguments.List) != len(closure.parameters) {
		return Value{}, nil, fmt.Errorf("canonicaleval.closure_call")
	}
	next := &closureValue{typeID: closure.typeID, parameters: append([]wire.ID(nil), closure.parameters...), paramTypes: append([]wire.ID(nil), closure.paramTypes...), resultType: closure.resultType, captures: cloneEnv(closure.captures), body: closure.body, mutable: closure.mutable}
	env := cloneEnv(outer)
	for id, value := range next.captures {
		env[id] = value
	}
	for i, item := range arguments.List {
		if item.Tag != 6 {
			return Value{}, nil, fmt.Errorf("canonicaleval.closure_argument")
		}
		value, err := eval(g, item.Reference, outer, budget-1)
		if err != nil {
			return Value{}, nil, err
		}
		if i >= len(next.paramTypes) || validateValue(g, next.paramTypes[i], value, 16) != nil {
			return Value{}, nil, fmt.Errorf("canonicaleval.closure_argument_type")
		}
		env[next.parameters[i]] = value
	}
	var result Value
	var err error
	if next.mutable {
		result, err = eval(g, next.body, env, budget-1)
	} else {
		result, err = eval(g, next.body, env, budget-1)
	}
	if err != nil {
		return Value{}, nil, err
	}
	if validateValue(g, next.resultType, result, 16) != nil {
		return Value{}, nil, fmt.Errorf("canonicaleval.closure_result_type")
	}
	for id := range next.captures {
		next.captures[id] = env[id]
	}
	return result, next, nil
}

func validateMethodRequirement(g wire.Envelope, methodID, requirementID, concreteType wire.ID) error {
	method, mok := g.Entities[methodID]
	requirement, rok := g.Entities[requirementID]
	if !mok || !rok || method.Schema != id(0xa002) || requirement.Schema != id(0xa011) {
		return fmt.Errorf("canonicaleval.dynamic_method")
	}
	mName, mn := field(method, 0xa0020)
	rName, rn := field(requirement, 0xa0110)
	mParams, mp := field(method, 0xa0022)
	rTypes, rp := field(requirement, 0xa0111)
	mResult, mr := field(method, 0xa0023)
	rResult, rr := field(requirement, 0xa0112)
	receiver, re := field(method, 0xa0021)
	if mn != nil || rn != nil || mp != nil || rp != nil || mr != nil || rr != nil || re != nil || mName.Tag != 5 || rName.Tag != 5 || string(mName.Bytes) != string(rName.Bytes) || mParams.Tag != 7 || rTypes.Tag != 7 || len(mParams.List) != len(rTypes.List) || mResult.Tag != 6 || rResult.Tag != 6 || mResult.Reference != rResult.Reference || receiver.Tag != 6 {
		return fmt.Errorf("canonicaleval.dynamic_signature")
	}
	binding, ok := g.Entities[receiver.Reference]
	bt, be := field(binding, 0xa0001)
	if !ok || binding.Schema != id(0xa000) || be != nil || bt.Tag != 6 || bt.Reference != concreteType {
		return fmt.Errorf("canonicaleval.dynamic_receiver")
	}
	for i := range mParams.List {
		if mParams.List[i].Tag != 6 || rTypes.List[i].Tag != 6 {
			return fmt.Errorf("canonicaleval.dynamic_signature")
		}
		parameter, ok := g.Entities[mParams.List[i].Reference]
		pt, pe := field(parameter, 0x9121)
		if !ok || parameter.Schema != id(0x9012) || pe != nil || pt.Tag != 6 || pt.Reference != rTypes.List[i].Reference {
			return fmt.Errorf("canonicaleval.dynamic_signature")
		}
	}
	return nil
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
