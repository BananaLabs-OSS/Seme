package wasmtarget

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

type composedValue struct {
	kind                string
	typeID              wire.ID
	code, state, result []byte
}
type composedScope struct {
	values   map[wire.ID]composedValue
	receiver map[wire.ID]composedValue
}
type composedLowerer struct {
	graph   wire.Envelope
	members map[wire.ID]bool
	locals  map[wire.ID]byte
	types   map[wire.ID]string
	typeIDs map[wire.ID]wire.ID
	extra   []byte
	stack   map[wire.ID]bool
	budget  int
}

func certifyComposedPureFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	listed, _ := field(program, 0x9150)
	members := map[wire.ID]bool{}
	if listed.Tag != 7 {
		return nil, PureABI{}, fmt.Errorf("wasm.composed_program")
	}
	for _, item := range listed.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.composed_program")
		}
		members[item.Reference] = true
	}
	parameters, _ := field(function, 0x9111)
	body, _ := field(function, 0x9113)
	if parameters.Tag != 7 || body.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.composed_function")
	}
	l := &composedLowerer{graph: graph, members: members, locals: map[wire.ID]byte{}, types: map[wire.ID]string{}, typeIDs: map[wire.ID]wire.ID{}, stack: map[wire.ID]bool{function.ID: true}, budget: 32768}
	var parameterTypes []pureValueType
	abi := PureABI{Contract: "seme.pure-abi/v2", Provider: "seme.function-v2", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String()}
	for index, item := range parameters.List {
		parameter := graph.Entities[item.Reference]
		typeRef, a := field(parameter, 0x9121)
		position, b := field(parameter, 0x9122)
		if item.Tag != 6 || parameter.Schema != identity(0x9012) || a != nil || b != nil || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.composed_parameter")
		}
		valueType, err := composedPhysicalType(graph, typeRef.Reference)
		if err != nil {
			return nil, PureABI{}, err
		}
		parameterTypes = append(parameterTypes, valueType)
		l.locals[parameter.ID] = byte(index)
		l.types[parameter.ID] = valueType.name
		l.typeIDs[parameter.ID] = typeRef.Reference
		abi.Parameters = append(abi.Parameters, PureABIField{Index: uint64(index), Type: valueType.name, Offset: abi.RequestSize, Size: valueType.size, Encoding: pureEncoding(valueType.name)})
		abi.RequestSize += valueType.size
	}
	abi.FixedHeaderSize = abi.RequestSize
	abi.MaximumRequestSize = 7160
	abi.VariablePayload = true
	if err := validateComposedBlock(graph, body.Reference, members, map[wire.ID]bool{function.ID: true}, map[wire.ID]bool{}, 32768); err != nil {
		return nil, PureABI{}, err
	}
	value, err := l.block(body.Reference, composedScope{values: map[wire.ID]composedValue{}, receiver: map[wire.ID]composedValue{}})
	if err != nil {
		return nil, PureABI{}, err
	}
	abi.ResponseSize = 16
	resultName, encoding := "state-transition:record:i64,i64", "state-then-result-little-endian-i64"
	if value.kind == "result:i64,i64" {
		declared, _ := field(function, 0x9112)
		if declared.Tag != 6 || value.typeID != declared.Reference {
			return nil, PureABI{}, fmt.Errorf("wasm.composed_result_type")
		}
		resultName, encoding = "result:i64,i64", "variant-tag-u64-then-payload-i64"
	} else if value.kind != "transition" {
		return nil, PureABI{}, fmt.Errorf("wasm.composed_result_kind")
	}
	abi.Result = PureABIField{Index: 0, Type: resultName, Offset: 0, Size: 16, Encoding: encoding}
	helper := append(append([]byte{}, value.state...), value.result...)
	wasm, err := pureStringModule(parameterTypes, pureValueType{name: resultName, wasm: 0x7e, size: 16}, helper, abi, nil, l.extra)
	return wasm, abi, err
}

func requiredComposedRef(entity wire.Entity, id uint64) (wire.ID, error) {
	value, err := field(entity, id)
	if err != nil || value.Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.composed_field:%x", id)
	}
	return value.Reference, nil
}
func requiredComposedList(entity wire.Entity, id uint64) ([]wire.Value, error) {
	value, err := field(entity, id)
	if err != nil || value.Tag != 7 {
		return nil, fmt.Errorf("wasm.composed_field:%x", id)
	}
	for _, item := range value.List {
		if item.Tag != 6 {
			return nil, fmt.Errorf("wasm.composed_list:%x", id)
		}
	}
	return value.List, nil
}
func validateComposedBlock(graph wire.Envelope, id wire.ID, members map[wire.ID]bool, stack, visiting map[wire.ID]bool, budget int) error {
	if budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.composed_structure_cycle")
	}
	visiting[id] = true
	defer delete(visiting, id)
	block, ok := graph.Entities[id]
	if !ok || block.Schema != identity(0x9080) {
		return fmt.Errorf("wasm.composed_structure_block")
	}
	items, err := requiredComposedList(block, 0x9800)
	if err != nil || len(items) == 0 {
		return fmt.Errorf("wasm.composed_structure_block")
	}
	for _, item := range items {
		statement := graph.Entities[item.Reference]
		switch statement.Schema {
		case identity(0x90d1):
			bindingID, e := requiredComposedRef(statement, 0x9d10)
			if e != nil {
				return e
			}
			binding := graph.Entities[bindingID]
			initializer, e := requiredComposedRef(binding, 0x9d02)
			if binding.Schema != identity(0x90d0) || e != nil {
				return fmt.Errorf("wasm.composed_structure_local")
			}
			if e = validateComposedExpr(graph, initializer, members, stack, visiting, budget-1); e != nil {
				return e
			}
		case identity(0x9081):
			values, e := requiredComposedList(statement, 0x9810)
			if e != nil || len(values) != 1 {
				return fmt.Errorf("wasm.composed_structure_return")
			}
			if e = validateComposedExpr(graph, values[0].Reference, members, stack, visiting, budget-1); e != nil {
				return e
			}
		case identity(0x90c0):
			condition, e := requiredComposedRef(statement, 0x9c00)
			if e != nil {
				return e
			}
			thenID, e := requiredComposedRef(statement, 0x9c01)
			if e != nil {
				return e
			}
			elseID, e := requiredComposedRef(statement, 0x9c02)
			if e != nil {
				return e
			}
			if e = validateComposedExpr(graph, condition, members, stack, visiting, budget-1); e != nil {
				return e
			}
			if e = validateComposedBlock(graph, thenID, members, stack, visiting, budget-1); e != nil {
				return e
			}
			if e = validateComposedBlock(graph, elseID, members, stack, visiting, budget-1); e != nil {
				return e
			}
		default:
			return fmt.Errorf("wasm.composed_structure_statement:%s", statement.Schema.String())
		}
	}
	return nil
}
func validateComposedExpr(graph wire.Envelope, id wire.ID, members map[wire.ID]bool, stack, visiting map[wire.ID]bool, budget int) error {
	if budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.composed_structure_cycle")
	}
	visiting[id] = true
	defer delete(visiting, id)
	e, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.composed_structure_expression")
	}
	refs := []uint64{}
	lists := []uint64{}
	switch e.Schema {
	case identity(0x9013):
		refs = []uint64{0x9130}
	case identity(0x90d2):
		refs = []uint64{0x9d20}
	case identity(0xa001):
		refs = []uint64{0xa0010}
	case identity(0xa061):
		refs = []uint64{0xa0610}
	case identity(0x9070):
		if value, x := field(e, 0x9700); x != nil || value.Tag != 3 {
			return fmt.Errorf("wasm.composed_literal")
		}
		return nil
	case identity(0x9014):
		refs = []uint64{0x9140, 0x9141}
	case identity(0x9021):
		refs = []uint64{0x9160, 0x9161}
	case identity(0x9032):
		refs = []uint64{0x9320, 0x9321}
	case identity(0x9033):
		refs = []uint64{0x9330}
		lists = []uint64{0x9331}
	case identity(0xa005):
		refs = []uint64{0xa0050, 0xa0051, 0xa0052}
	case identity(0xa006):
		refs = []uint64{0xa0060}
	case identity(0xa007):
		refs = []uint64{0xa0070}
	case identity(0x9043):
		refs = []uint64{0x9410, 0x9411}
	case identity(0x9044):
		refs = []uint64{0x9420, 0x9421}
	case identity(0xa062):
		value, x := requiredComposedRef(e, 0xa0620)
		if x != nil {
			return x
		}
		if x = validateComposedExpr(graph, value, members, stack, visiting, budget-1); x != nil {
			return x
		}
		for _, fieldID := range []uint64{0xa0621, 0xa0623} {
			binding, x := requiredComposedRef(e, fieldID)
			if x != nil || graph.Entities[binding].Schema != identity(0xa060) {
				return fmt.Errorf("wasm.composed_result_binding")
			}
		}
		for _, fieldID := range []uint64{0xa0622, 0xa0624} {
			body, x := requiredComposedRef(e, fieldID)
			if x != nil {
				return x
			}
			if x = validateComposedBlock(graph, body, members, stack, visiting, budget-1); x != nil {
				return x
			}
		}
		return nil
	case identity(0x90f6):
		refs = []uint64{0x9f60}
	case identity(0x90f7):
		refs = []uint64{0x9f70, 0x9f71, 0x9f72, 0x9f73, 0x9f74}
	case identity(0x9060):
		callee, x := requiredComposedRef(e, 0x9600)
		if x != nil || !members[callee] || stack[callee] {
			return fmt.Errorf("wasm.composed_call_target")
		}
		args, x := requiredComposedList(e, 0x9601)
		if x != nil {
			return x
		}
		for _, arg := range args {
			if x = validateComposedExpr(graph, arg.Reference, members, stack, visiting, budget-1); x != nil {
				return x
			}
		}
		function := graph.Entities[callee]
		body, x := requiredComposedRef(function, 0x9113)
		if function.Schema != identity(0x9011) || x != nil {
			return fmt.Errorf("wasm.composed_call_shape")
		}
		next := map[wire.ID]bool{}
		for key, value := range stack {
			next[key] = value
		}
		next[callee] = true
		return validateComposedBlock(graph, body, members, next, visiting, budget-1)
	case identity(0xa003):
		receiver, x := requiredComposedRef(e, 0xa0030)
		if x != nil {
			return x
		}
		methodID, x := requiredComposedRef(e, 0xa0031)
		if x != nil {
			return x
		}
		args, x := requiredComposedList(e, 0xa0032)
		if x != nil {
			return x
		}
		if x = validateComposedExpr(graph, receiver, members, stack, visiting, budget-1); x != nil {
			return x
		}
		for _, arg := range args {
			if x = validateComposedExpr(graph, arg.Reference, members, stack, visiting, budget-1); x != nil {
				return x
			}
		}
		method := graph.Entities[methodID]
		body, x := requiredComposedRef(method, 0xa0024)
		if method.Schema != identity(0xa002) || x != nil {
			return fmt.Errorf("wasm.composed_method_shape")
		}
		return validateComposedBlock(graph, body, members, stack, visiting, budget-1)
	default:
		return fmt.Errorf("wasm.composed_structure_expression:%s", e.Schema.String())
	}
	for _, fieldID := range refs {
		child, x := requiredComposedRef(e, fieldID)
		if x != nil {
			return x
		}
		childSchema := graph.Entities[child].Schema
		if childSchema == identity(0x9012) || childSchema == identity(0x90d0) || childSchema == identity(0xa000) || childSchema == identity(0x9030) || childSchema == identity(0x9031) || childSchema == identity(0xa004) || childSchema == identity(0x9042) || childSchema == identity(0xa060) || childSchema == identity(0x90f5) {
			continue
		}
		if x = validateComposedExpr(graph, child, members, stack, visiting, budget-1); x != nil {
			return x
		}
	}
	for _, fieldID := range lists {
		items, x := requiredComposedList(e, fieldID)
		if x != nil {
			return x
		}
		for _, item := range items {
			if x = validateComposedExpr(graph, item.Reference, members, stack, visiting, budget-1); x != nil {
				return x
			}
		}
	}
	return nil
}

func composedPhysicalType(graph wire.Envelope, id wire.ID) (pureValueType, error) {
	entity := graph.Entities[id]
	if entity.Schema == identity(0x9030) {
		if validateSingleI64Record(graph, id) != nil {
			return pureValueType{}, fmt.Errorf("wasm.composed_record_type")
		}
		return pureValueType{name: "record:i64", wasm: 0x7e, size: 8}, nil
	}
	return pureType(graph, id)
}
func (l *composedLowerer) newLocal(t byte) byte {
	index := byte(len(l.locals) + len(l.extra))
	l.extra = append(l.extra, t)
	return index
}

func (l *composedLowerer) block(id wire.ID, scope composedScope) (composedValue, error) {
	if l.budget == 0 {
		return composedValue{}, fmt.Errorf("wasm.composed_budget")
	}
	l.budget--
	block := l.graph.Entities[id]
	statements, _ := field(block, 0x9800)
	if block.Schema != identity(0x9080) || statements.Tag != 7 || len(statements.List) == 0 {
		return composedValue{}, fmt.Errorf("wasm.composed_block")
	}
	visible := map[wire.ID]composedValue{}
	for key, value := range scope.values {
		visible[key] = value
	}
	for index, item := range statements.List {
		statement := l.graph.Entities[item.Reference]
		switch statement.Schema {
		case identity(0x90d1):
			bindingRef, _ := field(statement, 0x9d10)
			binding := l.graph.Entities[bindingRef.Reference]
			initializer, _ := field(binding, 0x9d02)
			value, err := l.expr(initializer.Reference, composedScope{values: visible, receiver: scope.receiver})
			if err != nil {
				return composedValue{}, err
			}
			visible[binding.ID] = value
		case identity(0x9081):
			if index != len(statements.List)-1 {
				return composedValue{}, fmt.Errorf("wasm.composed_terminal")
			}
			values, _ := field(statement, 0x9810)
			if values.Tag != 7 || len(values.List) != 1 {
				return composedValue{}, fmt.Errorf("wasm.composed_return")
			}
			return l.expr(values.List[0].Reference, composedScope{values: visible, receiver: scope.receiver})
		case identity(0x90c0):
			if index != len(statements.List)-1 {
				return composedValue{}, fmt.Errorf("wasm.composed_terminal")
			}
			condition, _ := field(statement, 0x9c00)
			thenRef, _ := field(statement, 0x9c01)
			elseRef, _ := field(statement, 0x9c02)
			test, err := l.expr(condition.Reference, composedScope{values: visible, receiver: scope.receiver})
			if err != nil || test.kind != "bool" {
				return composedValue{}, fmt.Errorf("wasm.composed_condition")
			}
			a, err := l.block(thenRef.Reference, composedScope{values: visible, receiver: scope.receiver})
			if err != nil {
				return composedValue{}, err
			}
			b, err := l.block(elseRef.Reference, composedScope{values: visible, receiver: scope.receiver})
			if err != nil || a.kind != b.kind {
				return composedValue{}, fmt.Errorf("wasm.composed_branch_type")
			}
			choose := func(x, y []byte) []byte {
				out := append(append(append([]byte{}, test.code...), 0x04, 0x7e), x...)
				out = append(out, 0x05)
				out = append(out, y...)
				return append(out, 0x0b)
			}
			if a.kind == "transition" {
				return composedValue{kind: "transition", state: choose(a.state, b.state), result: choose(a.result, b.result)}, nil
			}
			if a.kind == "result:i64,i64" {
				return composedValue{kind: a.kind, typeID: a.typeID, state: choose(a.state, b.state), result: choose(a.result, b.result)}, nil
			}
			return composedValue{kind: a.kind, code: choose(a.code, b.code)}, nil
		default:
			return composedValue{}, fmt.Errorf("wasm.composed_statement:%s", statement.Schema.String())
		}
	}
	return composedValue{}, fmt.Errorf("wasm.composed_missing_terminal")
}

func (l *composedLowerer) expr(id wire.ID, scope composedScope) (composedValue, error) {
	if l.budget == 0 {
		return composedValue{}, fmt.Errorf("wasm.composed_budget")
	}
	l.budget--
	e := l.graph.Entities[id]
	switch e.Schema {
	case identity(0x9013):
		binding, _ := field(e, 0x9130)
		if value, ok := scope.values[binding.Reference]; ok {
			return value, nil
		}
		local, ok := l.locals[binding.Reference]
		if !ok {
			return composedValue{}, fmt.Errorf("wasm.composed_parameter_read")
		}
		return composedValue{kind: l.types[binding.Reference], typeID: l.typeIDs[binding.Reference], code: []byte{0x20, local}}, nil
	case identity(0x90d2):
		binding, _ := field(e, 0x9d20)
		value, ok := scope.values[binding.Reference]
		if !ok {
			return composedValue{}, fmt.Errorf("wasm.composed_local_scope")
		}
		return value, nil
	case identity(0xa061):
		binding, _ := field(e, 0xa0610)
		value, ok := scope.values[binding.Reference]
		if !ok {
			return composedValue{}, fmt.Errorf("wasm.composed_variant_scope")
		}
		return value, nil
	case identity(0x90f6):
		binding, _ := field(e, 0x9f60)
		value, ok := scope.values[binding.Reference]
		if !ok {
			return composedValue{}, fmt.Errorf("wasm.composed_iteration_scope")
		}
		return value, nil
	case identity(0xa001):
		binding, _ := field(e, 0xa0010)
		value, ok := scope.receiver[binding.Reference]
		if !ok {
			return composedValue{}, fmt.Errorf("wasm.composed_receiver_scope")
		}
		return value, nil
	case identity(0x9070):
		value, _ := field(e, 0x9700)
		var code bytes.Buffer
		code.WriteByte(0x42)
		sleb(&code, int64(value.Unsigned))
		return composedValue{kind: "i64", code: code.Bytes()}, nil
	case identity(0x9014):
		return l.binary(e, 0x9140, 0x9141, 0x7c, scope)
	case identity(0x9021):
		left, _ := field(e, 0x9160)
		right, _ := field(e, 0x9161)
		a, err := l.expr(left.Reference, scope)
		if err != nil {
			return composedValue{}, err
		}
		b, err := l.expr(right.Reference, scope)
		if err != nil || a.kind != "i64" || b.kind != "i64" {
			return composedValue{}, fmt.Errorf("wasm.composed_compare")
		}
		return composedValue{kind: "bool", code: append(append(a.code, b.code...), 0x57)}, nil
	case identity(0x9033):
		typeRef, _ := field(e, 0x9330)
		values, _ := field(e, 0x9331)
		recordType := l.graph.Entities[typeRef.Reference]
		members, _ := field(recordType, 0x9301)
		if recordType.Schema != identity(0x9030) || members.Tag != 7 || len(members.List) != 1 || values.Tag != 7 || len(values.List) != 1 {
			return composedValue{}, fmt.Errorf("wasm.composed_record")
		}
		value, err := l.expr(values.List[0].Reference, scope)
		value.kind = "record:i64"
		value.typeID = typeRef.Reference
		return value, err
	case identity(0x9032):
		record, _ := field(e, 0x9320)
		member, _ := field(e, 0x9321)
		fieldEntity := l.graph.Entities[member.Reference]
		position, _ := field(fieldEntity, 0x9312)
		value, err := l.expr(record.Reference, scope)
		recordType := l.graph.Entities[value.typeID]
		members, _ := field(recordType, 0x9301)
		if err != nil || fieldEntity.Schema != identity(0x9031) || position.Unsigned != 0 || members.Tag != 7 || len(members.List) != 1 || members.List[0].Reference != fieldEntity.ID {
			return composedValue{}, fmt.Errorf("wasm.composed_field")
		}
		value.kind = "i64"
		value.typeID = wire.ID{}
		return value, err
	case identity(0x9060):
		return l.call(e, scope)
	case identity(0xa003):
		return l.methodCall(e, scope)
	case identity(0xa005):
		state, _ := field(e, 0xa0051)
		result, _ := field(e, 0xa0052)
		a, err := l.expr(state.Reference, scope)
		if err != nil {
			return composedValue{}, err
		}
		b, err := l.expr(result.Reference, scope)
		if err != nil {
			return composedValue{}, err
		}
		return composedValue{kind: "transition", state: a.code, result: b.code}, nil
	case identity(0x9043), identity(0x9044):
		typeField, valueField, tag := uint64(0x9410), uint64(0x9411), int64(0)
		if e.Schema == identity(0x9044) {
			typeField, valueField, tag = 0x9420, 0x9421, 1
		}
		typeRef, _ := field(e, typeField)
		valueRef, _ := field(e, valueField)
		resultType := l.graph.Entities[typeRef.Reference]
		okType, a := field(resultType, 0x9400)
		errorType, b := field(resultType, 0x9401)
		value, err := l.expr(valueRef.Reference, scope)
		okPhysical, okErr := pureType(l.graph, okType.Reference)
		errorPhysical, errorErr := pureType(l.graph, errorType.Reference)
		if a != nil || b != nil || resultType.Schema != identity(0x9042) || value.kind != "i64" || err != nil || okErr != nil || errorErr != nil || okPhysical.name != "i64" || errorPhysical.name != "i64" {
			return composedValue{}, fmt.Errorf("wasm.composed_result_construct")
		}
		var tagCode bytes.Buffer
		tagCode.WriteByte(0x42)
		sleb(&tagCode, tag)
		return composedValue{kind: "result:i64,i64", typeID: typeRef.Reference, state: tagCode.Bytes(), result: value.code}, nil
	case identity(0xa062):
		valueRef, _ := field(e, 0xa0620)
		value, err := l.expr(valueRef.Reference, scope)
		if err != nil || value.kind != "result:i64,i64" {
			return composedValue{}, fmt.Errorf("wasm.composed_result_match")
		}
		okBinding, _ := field(e, 0xa0621)
		okBody, _ := field(e, 0xa0622)
		errorBinding, _ := field(e, 0xa0623)
		errorBody, _ := field(e, 0xa0624)
		resultType := l.graph.Entities[value.typeID]
		okExpected, _ := field(resultType, 0x9400)
		errorExpected, _ := field(resultType, 0x9401)
		okEntity, okExists := l.graph.Entities[okBinding.Reference]
		errorEntity, errorExists := l.graph.Entities[errorBinding.Reference]
		okActual, okTypeErr := field(okEntity, 0xa0601)
		errorActual, errorTypeErr := field(errorEntity, 0xa0601)
		if !okExists || !errorExists || okEntity.Schema != identity(0xa060) || errorEntity.Schema != identity(0xa060) || okTypeErr != nil || errorTypeErr != nil || okActual.Reference != okExpected.Reference || errorActual.Reference != errorExpected.Reference {
			return composedValue{}, fmt.Errorf("wasm.composed_result_binding_type")
		}
		okValues := map[wire.ID]composedValue{}
		errorValues := map[wire.ID]composedValue{}
		for k, v := range scope.values {
			okValues[k] = v
			errorValues[k] = v
		}
		payload := composedValue{kind: "i64", code: value.result}
		okValues[okBinding.Reference] = payload
		errorValues[errorBinding.Reference] = payload
		ok, err := l.block(okBody.Reference, composedScope{values: okValues, receiver: scope.receiver})
		if err != nil {
			return composedValue{}, err
		}
		bad, err := l.block(errorBody.Reference, composedScope{values: errorValues, receiver: scope.receiver})
		if err != nil || ok.kind != bad.kind || ok.kind != "result:i64,i64" || ok.typeID != bad.typeID {
			return composedValue{}, fmt.Errorf("wasm.composed_result_branches")
		}
		test := append(append([]byte{}, value.state...), 0x50)
		choose := func(a, b []byte) []byte {
			out := append(append(append([]byte{}, test...), 0x04, 0x7e), a...)
			out = append(out, 0x05)
			out = append(out, b...)
			return append(out, 0x0b)
		}
		return composedValue{kind: ok.kind, typeID: ok.typeID, state: choose(ok.state, bad.state), result: choose(ok.result, bad.result)}, nil
	case identity(0xa006), identity(0xa007):
		var fieldID uint64 = 0xa0060
		if e.Schema == identity(0xa007) {
			fieldID = 0xa0070
		}
		valueRef, _ := field(e, fieldID)
		value, err := l.expr(valueRef.Reference, scope)
		if err != nil || value.kind != "transition" {
			return composedValue{}, fmt.Errorf("wasm.composed_transition_projection")
		}
		if fieldID == 0xa0060 {
			return composedValue{kind: "record:i64", code: value.state}, nil
		}
		return composedValue{kind: "i64", code: value.result}, nil
	case identity(0x90f7):
		return l.fold(e, scope)
	default:
		return composedValue{}, fmt.Errorf("wasm.composed_expression:%s", e.Schema.String())
	}
}

func (l *composedLowerer) binary(e wire.Entity, leftField, rightField uint64, opcode byte, scope composedScope) (composedValue, error) {
	left, _ := field(e, leftField)
	right, _ := field(e, rightField)
	a, err := l.expr(left.Reference, scope)
	if err != nil {
		return composedValue{}, err
	}
	b, err := l.expr(right.Reference, scope)
	if err != nil || a.kind != "i64" || b.kind != "i64" {
		return composedValue{}, fmt.Errorf("wasm.composed_binary")
	}
	return composedValue{kind: "i64", code: append(append(a.code, b.code...), opcode)}, nil
}

func (l *composedLowerer) call(e wire.Entity, scope composedScope) (composedValue, error) {
	calleeRef, _ := field(e, 0x9600)
	arguments, _ := field(e, 0x9601)
	callee := l.graph.Entities[calleeRef.Reference]
	if !l.members[callee.ID] || l.stack[callee.ID] || callee.Schema != identity(0x9011) {
		return composedValue{}, fmt.Errorf("wasm.composed_call_target")
	}
	parameters, _ := field(callee, 0x9111)
	body, _ := field(callee, 0x9113)
	if arguments.Tag != 7 || parameters.Tag != 7 || len(arguments.List) != len(parameters.List) {
		return composedValue{}, fmt.Errorf("wasm.composed_call_arity")
	}
	values := map[wire.ID]composedValue{}
	for index, item := range arguments.List {
		value, err := l.expr(item.Reference, scope)
		if err != nil {
			return composedValue{}, err
		}
		values[parameters.List[index].Reference] = value
	}
	l.stack[callee.ID] = true
	value, err := l.block(body.Reference, composedScope{values: values, receiver: map[wire.ID]composedValue{}})
	delete(l.stack, callee.ID)
	return value, err
}

func (l *composedLowerer) methodCall(e wire.Entity, scope composedScope) (composedValue, error) {
	receiverRef, _ := field(e, 0xa0030)
	methodRef, _ := field(e, 0xa0031)
	arguments, _ := field(e, 0xa0032)
	method := l.graph.Entities[methodRef.Reference]
	receiverBinding, _ := field(method, 0xa0021)
	receiverEntity := l.graph.Entities[receiverBinding.Reference]
	receiverType, _ := field(receiverEntity, 0xa0001)
	parameters, _ := field(method, 0xa0022)
	body, _ := field(method, 0xa0024)
	if method.Schema != identity(0xa002) || receiverEntity.Schema != identity(0xa000) || arguments.Tag != 7 || parameters.Tag != 7 || len(arguments.List) != len(parameters.List) {
		return composedValue{}, fmt.Errorf("wasm.composed_method")
	}
	receiver, err := l.expr(receiverRef.Reference, scope)
	if err != nil {
		return composedValue{}, err
	}
	if receiver.typeID != receiverType.Reference {
		return composedValue{}, fmt.Errorf("wasm.composed_method_receiver_type")
	}
	values := map[wire.ID]composedValue{}
	for index, item := range arguments.List {
		value, err := l.expr(item.Reference, scope)
		if err != nil {
			return composedValue{}, err
		}
		values[parameters.List[index].Reference] = value
	}
	return l.block(body.Reference, composedScope{values: values, receiver: map[wire.ID]composedValue{receiverBinding.Reference: receiver}})
}

func (l *composedLowerer) fold(e wire.Entity, scope composedScope) (composedValue, error) {
	collectionRef, _ := field(e, 0x9f70)
	initialRef, _ := field(e, 0x9f71)
	accRef, _ := field(e, 0x9f72)
	elementRef, _ := field(e, 0x9f73)
	bodyRef, _ := field(e, 0x9f74)
	collection, err := l.expr(collectionRef.Reference, scope)
	if err != nil || collection.kind != "slice:i64" {
		return composedValue{}, fmt.Errorf("wasm.composed_fold_collection")
	}
	initial, err := l.expr(initialRef.Reference, scope)
	if err != nil || initial.kind != "i64" {
		return composedValue{}, fmt.Errorf("wasm.composed_fold_initial")
	}
	packed := l.newLocal(0x7e)
	pointer := l.newLocal(0x7f)
	length := l.newLocal(0x7f)
	index := l.newLocal(0x7f)
	acc := l.newLocal(0x7e)
	element := l.newLocal(0x7e)
	inner := map[wire.ID]composedValue{}
	for key, value := range scope.values {
		inner[key] = value
	}
	inner[accRef.Reference] = composedValue{kind: "i64", code: []byte{0x20, acc}}
	inner[elementRef.Reference] = composedValue{kind: "i64", code: []byte{0x20, element}}
	body, err := l.expr(bodyRef.Reference, composedScope{values: inner, receiver: scope.receiver})
	if err != nil || body.kind != "i64" {
		return composedValue{}, fmt.Errorf("wasm.composed_fold_body")
	}
	code := append(collection.code, 0x21, packed, 0x20, packed, 0xa7, 0x21, pointer, 0x20, packed, 0x42, 32, 0x88, 0xa7, 0x21, length)
	code = append(code, initial.code...)
	code = append(code, 0x21, acc, 0x41, 0, 0x21, index, 0x02, 0x40, 0x03, 0x40, 0x20, index, 0x20, length, 0x4f, 0x0d, 1, 0x20, pointer, 0x20, index, 0x41, 3, 0x74, 0x6a, 0x29, 3, 0, 0x21, element)
	code = append(code, body.code...)
	code = append(code, 0x21, acc, 0x20, index, 0x41, 1, 0x6a, 0x21, index, 0x0c, 0, 0x0b, 0x0b, 0x20, acc)
	return composedValue{kind: "i64", code: code}, nil
}
