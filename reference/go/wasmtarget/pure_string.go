package wasmtarget

import (
	"bytes"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/wire"
)

const (
	stringTableOffset = 64
	stringArenaStart  = 4096
	stringArenaEnd    = 24576
	stringScratch     = 32768
	stringLimit       = 4096
)

type stringLowering struct {
	graph       wire.Envelope
	locals      map[wire.ID]byte
	used        map[byte]bool
	data        []byte
	literalBase int
	placeTypes  map[wire.ID]string
	extraLocals []byte
	loopCounter byte
}

func certifyPureStringFunction(graph wire.Envelope, body wire.ID, parameters []pureValueType, result pureValueType, parameterTypes map[wire.ID]string, locals map[wire.ID]byte, abi PureABI) ([]byte, PureABI, error) {
	context := &stringLowering{graph: graph, locals: locals, used: map[byte]bool{}, data: utf8TransitionTable(), placeTypes: map[wire.ID]string{}}
	context.literalBase = stringTableOffset + len(context.data)
	budget := 4096
	var instructions []byte
	var err error
	if len(bySchema(graph, 0x90e0))+len(bySchema(graph, 0x90e1))+len(bySchema(graph, 0x90e2))+len(bySchema(graph, 0x90e3))+len(bySchema(graph, 0x90e4))+len(bySchema(graph, 0x90f0)) > 0 {
		err = context.planPlaces(parameters)
		if err == nil {
			declared := map[wire.ID]bool{}
			err = validateStateScopes(graph, body, map[wire.ID]bool{}, declared, map[wire.ID]bool{}, &budget)
			if err == nil && len(declared) != len(bySchema(graph, 0x90e0)) {
				err = fmt.Errorf("wasm.pure_undeclared_place")
			}
		}
		if err == nil {
			budget = 4096
			instructions, err = context.lowerStateBlock(body, result.name, true, map[wire.ID]bool{}, &budget)
		}
	} else {
		instructions, err = context.lowerBlock(body, result.name, parameterTypes, map[wire.ID]bool{}, &budget)
	}
	if err != nil {
		return nil, PureABI{}, err
	}
	wasm, err := pureStringModule(parameters, result, instructions, abi, context.data, context.extraLocals)
	return wasm, abi, err
}

func (context *stringLowering) planPlaces(parameters []pureValueType) error {
	places := bySchema(context.graph, 0x90e0)
	sort.Slice(places, func(left, right int) bool {
		return bytes.Compare(places[left].ID[:], places[right].ID[:]) < 0
	})
	if len(places) > 32 {
		return fmt.Errorf("wasm.pure_place_count")
	}
	for _, place := range places {
		typeValue, err := field(place, 0x9e01)
		if err != nil || typeValue.Tag != 6 {
			return fmt.Errorf("wasm.pure_place_type")
		}
		valueType, err := pureType(context.graph, typeValue.Reference)
		if err != nil || (valueType.name != "string" && valueType.name != "bool" && valueType.name != "i64") {
			return fmt.Errorf("wasm.pure_place_type")
		}
		index := byte(len(parameters) + len(context.extraLocals))
		context.locals[place.ID] = index
		context.placeTypes[place.ID] = valueType.name
		context.extraLocals = append(context.extraLocals, valueType.wasm)
	}
	context.loopCounter = byte(len(parameters) + len(context.extraLocals))
	context.extraLocals = append(context.extraLocals, 0x7f)
	return nil
}

func (context *stringLowering) lowerStateBlock(id wire.ID, result string, requireReturn bool, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.pure_state_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	block, ok := context.graph.Entities[id]
	statements, err := field(block, 0x9800)
	if !ok || block.Schema != identity(0x9080) || err != nil || statements.Tag != 7 || len(statements.List) == 0 {
		return nil, fmt.Errorf("wasm.pure_state_block")
	}
	declared := map[wire.ID]bool{}
	var code []byte
	returned := false
	for index, item := range statements.List {
		if item.Tag != 6 {
			return nil, fmt.Errorf("wasm.pure_state_statement")
		}
		statement, exists := context.graph.Entities[item.Reference]
		if !exists {
			return nil, fmt.Errorf("wasm.pure_state_statement")
		}
		switch statement.Schema {
		case identity(0x90e1):
			placeValue, e := field(statement, 0x9e10)
			place := context.graph.Entities[placeValue.Reference]
			initializer, iErr := field(place, 0x9e02)
			if e != nil || iErr != nil || placeValue.Tag != 6 || place.Schema != identity(0x90e0) || declared[place.ID] {
				return nil, fmt.Errorf("wasm.pure_place_declaration")
			}
			value, lowerErr := context.lowerStateExpression(initializer.Reference, context.placeTypes[place.ID], budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			code = append(code, value...)
			code = append(code, 0x21, context.locals[place.ID])
			declared[place.ID] = true
		case identity(0x90e3):
			placeValue, e := field(statement, 0x9e30)
			valueValue, vErr := field(statement, 0x9e31)
			if e != nil || vErr != nil || placeValue.Tag != 6 || valueValue.Tag != 6 || context.placeTypes[placeValue.Reference] == "" {
				return nil, fmt.Errorf("wasm.pure_assignment")
			}
			value, lowerErr := context.lowerStateExpression(valueValue.Reference, context.placeTypes[placeValue.Reference], budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			code = append(code, value...)
			code = append(code, 0x21, context.locals[placeValue.Reference])
		case identity(0x90e4):
			condition, e := field(statement, 0x9e40)
			body, bErr := field(statement, 0x9e41)
			if e != nil || bErr != nil || condition.Tag != 6 || body.Tag != 6 {
				return nil, fmt.Errorf("wasm.pure_while")
			}
			conditionCode, lowerErr := context.lowerStateExpression(condition.Reference, "bool", budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			bodyCode, lowerErr := context.lowerStateBlock(body.Reference, result, false, visiting, budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			code = append(code, 0x41, 0x00, 0x21, context.loopCounter, 0x02, 0x40, 0x03, 0x40)
			code = append(code, conditionCode...)
			code = append(code, 0x45, 0x0d, 0x01)
			code = append(code, 0x20, context.loopCounter, 0x41)
			slebBytes := &bytes.Buffer{}
			sleb(slebBytes, 1)
			code = append(code, slebBytes.Bytes()...)
			code = append(code, 0x6a, 0x22, context.loopCounter, 0x41)
			limit := &bytes.Buffer{}
			sleb(limit, 10000)
			code = append(code, limit.Bytes()...)
			code = append(code, 0x4b, 0x04, 0x40, 0x00, 0x0b)
			code = append(code, bodyCode...)
			code = append(code, 0x0c, 0x00, 0x0b, 0x0b)
		case identity(0x90f0):
			condition, conditionErr := field(statement, 0x9f00)
			body, bodyErr := field(statement, 0x9f01)
			if conditionErr != nil || bodyErr != nil || condition.Tag != 6 || body.Tag != 6 {
				return nil, fmt.Errorf("wasm.pure_when")
			}
			conditionCode, lowerErr := context.lowerStateExpression(condition.Reference, "bool", budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			bodyCode, lowerErr := context.lowerStateBlock(body.Reference, result, false, visiting, budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			code = append(code, conditionCode...)
			code = append(code, 0x04, 0x40)
			code = append(code, bodyCode...)
			code = append(code, 0x0b)
		case identity(0x9081):
			values, e := field(statement, 0x9810)
			if e != nil || values.Tag != 7 || len(values.List) != 1 || index != len(statements.List)-1 {
				return nil, fmt.Errorf("wasm.pure_state_return")
			}
			value, lowerErr := context.lowerStateExpression(values.List[0].Reference, result, budget)
			if lowerErr != nil {
				return nil, lowerErr
			}
			code = append(code, value...)
			returned = true
		default:
			return nil, fmt.Errorf("wasm.pure_state_statement")
		}
	}
	if requireReturn && !returned {
		return nil, fmt.Errorf("wasm.pure_state_missing_return")
	}
	return code, nil
}

func (context *stringLowering) lowerStateExpression(id wire.ID, expected string, budget *int) ([]byte, error) {
	entity, ok := context.graph.Entities[id]
	if ok && entity.Schema == identity(0x90e2) {
		place, err := field(entity, 0x9e20)
		if err != nil || place.Tag != 6 || context.placeTypes[place.Reference] != expected {
			return nil, fmt.Errorf("wasm.pure_place_read_type")
		}
		return []byte{0x20, context.locals[place.Reference]}, nil
	}
	if expected == "string" {
		return context.lowerString(id, map[wire.ID]bool{}, budget)
	}
	if expected == "bool" {
		return context.lowerBoolean(id, map[wire.ID]bool{}, budget)
	}
	if expected == "i64" {
		return context.lowerInteger(id, budget)
	}
	return nil, fmt.Errorf("wasm.pure_state_type")
}

func (context *stringLowering) lowerBlock(id wire.ID, result string, parameterTypes map[wire.ID]string, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.pure_control_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	block, ok := context.graph.Entities[id]
	if !ok || block.Schema != identity(0x9080) {
		return nil, fmt.Errorf("wasm.pure_block")
	}
	statements, err := field(block, 0x9800)
	if err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return nil, fmt.Errorf("wasm.pure_block")
	}
	statement, ok := context.graph.Entities[statements.List[0].Reference]
	if !ok {
		return nil, fmt.Errorf("wasm.pure_statement_missing")
	}
	if statement.Schema == identity(0x9081) {
		values, err := field(statement, 0x9810)
		if err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
			return nil, fmt.Errorf("wasm.pure_return_values")
		}
		if err := validatePureExpression(context.graph, values.List[0].Reference, result, parameterTypes, map[wire.ID]bool{}, budget); err != nil {
			return nil, err
		}
		if result == "string" {
			return context.lowerString(values.List[0].Reference, map[wire.ID]bool{}, budget)
		}
		if result == "bool" {
			return context.lowerBoolean(values.List[0].Reference, map[wire.ID]bool{}, budget)
		}
		return context.lowerInteger(values.List[0].Reference, budget)
	}
	if statement.Schema != identity(0x90c0) {
		return nil, fmt.Errorf("wasm.pure_return")
	}
	condition, a := field(statement, 0x9c00)
	thenValue, b := field(statement, 0x9c01)
	elseValue, c := field(statement, 0x9c02)
	if a != nil || b != nil || c != nil || condition.Tag != 6 || thenValue.Tag != 6 || elseValue.Tag != 6 {
		return nil, fmt.Errorf("wasm.pure_if_fields")
	}
	conditionCode, err := context.lowerBoolean(condition.Reference, map[wire.ID]bool{}, budget)
	if err != nil {
		return nil, err
	}
	thenCode, err := context.lowerBlock(thenValue.Reference, result, parameterTypes, visiting, budget)
	if err != nil {
		return nil, err
	}
	elseCode, err := context.lowerBlock(elseValue.Reference, result, parameterTypes, visiting, budget)
	if err != nil {
		return nil, err
	}
	wasmType := byte(0x7e)
	if result == "bool" {
		wasmType = 0x7f
	}
	code := append(conditionCode, 0x04, wasmType)
	code = append(code, thenCode...)
	code = append(code, 0x05)
	code = append(code, elseCode...)
	return append(code, 0x0b), nil
}

func (context *stringLowering) lowerInteger(id wire.ID, budget *int) ([]byte, error) {
	expression, ok := context.graph.Entities[id]
	if !ok || expression.Schema != identity(0x90f7) {
		return lowerHelperInteger(context.graph, id, context.locals, context.used, map[wire.ID]bool{}, budget)
	}
	collection, initial, accumulator, element, body, err := foldFields(context.graph, expression)
	if err != nil {
		return nil, err
	}
	if err := validateI64AddFoldBody(context.graph, body, accumulator, element); err != nil {
		return nil, err
	}
	collectionEntity, ok := context.graph.Entities[collection]
	if !ok || collectionEntity.Schema != identity(0x9013) {
		return lowerHelperInteger(context.graph, id, context.locals, context.used, map[wire.ID]bool{}, budget)
	}
	parameter, err := field(collectionEntity, 0x9130)
	local, exists := context.locals[parameter.Reference]
	if err != nil || parameter.Tag != 6 || !exists {
		return nil, fmt.Errorf("wasm.fold_parameter")
	}
	parameterEntity := context.graph.Entities[parameter.Reference]
	typeValue, typeErr := field(parameterEntity, 0x9121)
	valueType, valueTypeErr := pureType(context.graph, typeValue.Reference)
	if typeErr != nil || valueTypeErr != nil || valueType.name != "slice:i64" {
		return lowerHelperInteger(context.graph, id, context.locals, context.used, map[wire.ID]bool{}, budget)
	}
	initialCode, err := lowerHelperInteger(context.graph, initial, context.locals, context.used, map[wire.ID]bool{}, budget)
	if err != nil {
		return nil, err
	}
	accumulatorLocal := byte(len(context.locals) + len(context.extraLocals))
	context.extraLocals = append(context.extraLocals, 0x7e)
	indexLocal := byte(len(context.locals) + len(context.extraLocals))
	context.extraLocals = append(context.extraLocals, 0x7f)
	context.used[local] = true
	code := append(initialCode, 0x21, accumulatorLocal, 0x41, 0x00, 0x21, indexLocal)
	code = append(code, 0x02, 0x40, 0x03, 0x40) // block; loop
	// Exit when index >= packed slice element count.
	code = append(code, 0x20, indexLocal, 0x20, local, 0x42, 0x20, 0x88, 0xa7, 0x4f, 0x0d, 0x01)
	// accumulator += load_i64(pointer + index*8)
	code = append(code, 0x20, accumulatorLocal, 0x20, local, 0xa7, 0x20, indexLocal, 0x41, 0x03, 0x74, 0x6a, 0x29, 0x03, 0x00, 0x7c, 0x21, accumulatorLocal)
	code = append(code, 0x20, indexLocal, 0x41, 0x01, 0x6a, 0x21, indexLocal, 0x0c, 0x00, 0x0b, 0x0b)
	return append(code, 0x20, accumulatorLocal), nil
}

func (context *stringLowering) lowerString(id wire.ID, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.helper_string_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := context.graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.helper_string_missing")
	}
	switch expression.Schema {
	case identity(0x90e2):
		place, err := field(expression, 0x9e20)
		if err != nil || context.placeTypes[place.Reference] != "string" {
			return nil, fmt.Errorf("wasm.pure_place_read_type")
		}
		return []byte{0x20, context.locals[place.Reference]}, nil
	case identity(0x9013):
		parameter, err := field(expression, 0x9130)
		local, ok := context.locals[parameter.Reference]
		if err != nil || !ok {
			return nil, fmt.Errorf("wasm.helper_parameter_reference")
		}
		context.used[local] = true
		return []byte{0x20, local}, nil
	case identity(0x9050):
		literal, err := field(expression, 0x9500)
		if err != nil || literal.Tag != 5 || !utf8.Valid(literal.Bytes) || len(literal.Bytes) > stringLimit {
			return nil, fmt.Errorf("wasm.helper_string_literal")
		}
		offset := context.literalBase + len(context.data) - (context.literalBase - stringTableOffset)
		context.data = append(context.data, literal.Bytes...)
		return packedStringConstant(uint64(offset), uint64(len(literal.Bytes))), nil
	case identity(0x90c3):
		left, leftErr := field(expression, 0x9c30)
		right, rightErr := field(expression, 0x9c31)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_string_concat")
		}
		leftCode, err := context.lowerString(left.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightCode, err := context.lowerString(right.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		return append(append(leftCode, rightCode...), 0x10, 0x08), nil
	default:
		return nil, fmt.Errorf("wasm.helper_string_expression")
	}
}

func (context *stringLowering) lowerBoolean(id wire.ID, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.helper_expression_cycle_or_size")
	}
	expression, ok := context.graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.helper_expression_missing")
	}
	if expression.Schema == identity(0x90e2) {
		place, err := field(expression, 0x9e20)
		if err != nil || context.placeTypes[place.Reference] != "bool" {
			return nil, fmt.Errorf("wasm.pure_place_read_type")
		}
		return []byte{0x20, context.locals[place.Reference]}, nil
	}
	if expression.Schema == identity(0x90c2) {
		*budget--
		left, a := field(expression, 0x9c20)
		right, b := field(expression, 0x9c21)
		if a != nil || b != nil {
			return nil, fmt.Errorf("wasm.helper_string_equal")
		}
		leftCode, err := context.lowerString(left.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightCode, err := context.lowerString(right.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		return append(append(leftCode, rightCode...), 0x10, 0x07), nil
	}
	if expression.Schema == identity(0x90b1) || expression.Schema == identity(0x90c1) {
		*budget--
		leftField, rightField := uint64(0x9b10), uint64(0x9b11)
		isOr := expression.Schema == identity(0x90c1)
		if isOr {
			leftField, rightField = 0x9c10, 0x9c11
		}
		left, a := field(expression, leftField)
		right, b := field(expression, rightField)
		if a != nil || b != nil {
			return nil, fmt.Errorf("wasm.helper_boolean_fields")
		}
		leftCode, err := context.lowerBoolean(left.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightCode, err := context.lowerBoolean(right.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		code := append(leftCode, 0x04, 0x7f)
		if isOr {
			code = append(code, 0x41, 0x01, 0x05)
			code = append(code, rightCode...)
		} else {
			code = append(code, rightCode...)
			code = append(code, 0x05, 0x41, 0x00)
		}
		return append(code, 0x0b), nil
	}
	return lowerHelperBoolean(context.graph, id, context.locals, context.used, visiting, budget)
}

func packedStringConstant(pointer, length uint64) []byte {
	var output bytes.Buffer
	output.WriteByte(0x42)
	sleb(&output, int64(pointer|(length<<32)))
	return output.Bytes()
}

func utf8TransitionTable() []byte {
	table := make([]byte, 9*256)
	for state := 0; state < 9; state++ {
		for value := 0; value < 256; value++ {
			next := byte(8)
			switch state {
			case 0:
				switch {
				case value <= 0x7f:
					next = 0
				case value >= 0xc2 && value <= 0xdf:
					next = 1
				case value == 0xe0:
					next = 4
				case value >= 0xe1 && value <= 0xec || value >= 0xee && value <= 0xef:
					next = 2
				case value == 0xed:
					next = 5
				case value == 0xf0:
					next = 6
				case value >= 0xf1 && value <= 0xf3:
					next = 3
				case value == 0xf4:
					next = 7
				}
			case 1:
				if value >= 0x80 && value <= 0xbf {
					next = 0
				}
			case 2:
				if value >= 0x80 && value <= 0xbf {
					next = 1
				}
			case 3:
				if value >= 0x80 && value <= 0xbf {
					next = 2
				}
			case 4:
				if value >= 0xa0 && value <= 0xbf {
					next = 1
				}
			case 5:
				if value >= 0x80 && value <= 0x9f {
					next = 1
				}
			case 6:
				if value >= 0x90 && value <= 0xbf {
					next = 2
				}
			case 7:
				if value >= 0x80 && value <= 0x8f {
					next = 2
				}
			}
			table[state*256+value] = next
		}
	}
	return table
}

func pureStringModule(parameters []pureValueType, result pureValueType, helper []byte, abi PureABI, data []byte, helperLocals []byte) ([]byte, error) {
	if len(helper) == 0 || abi.FixedHeaderSize > 7160 || len(data)+stringTableOffset >= stringArenaStart {
		return nil, fmt.Errorf("wasm.pure_string_layout")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 8)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	parameterWasm := make([]byte, len(parameters))
	for index, parameter := range parameters {
		parameterWasm[index] = parameter.wasm
	}
	functionType(&types, parameterWasm, []byte{result.wasm})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	functionType(&types, []byte{0x7e, 0x7e}, []byte{0x7f})
	functionType(&types, []byte{0x7e, 0x7e}, []byte{0x7e})
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{10, 0, 5, 3, 3, 2, 4, 3, 6, 7, 1})
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, stringArenaStart)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	var exports bytes.Buffer
	uleb(&exports, 7)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 0)
	export(&exports, "pulp_free", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, 9)
	section(&wasm, 7, exports.Bytes())
	localDecls := &bytes.Buffer{}
	uleb(localDecls, uint64(len(helperLocals)))
	for _, valueType := range helperLocals {
		uleb(localDecls, 1)
		localDecls.WriteByte(valueType)
	}
	helperBody := append(localDecls.Bytes(), helper...)
	helperBody = append(helperBody, 0x0b)
	bodies := [][]byte{
		stringAllocatorBody(), stringFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b},
		helperBody, utf8ValidatorBody(), stringEqualBody(), stringConcatBody(), stringProviderBody(parameters, result, abi),
	}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	var payload bytes.Buffer
	uleb(&payload, 1)
	payload.WriteByte(0)
	constI32(&payload, stringTableOffset)
	payload.WriteByte(0x0b)
	uleb(&payload, uint64(len(data)))
	payload.Write(data)
	section(&wasm, 11, payload.Bytes())
	return wasm.Bytes(), nil
}

func stringAllocatorBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 2, 0x7f})
	body.Write([]byte{0x20, 0, 0x45, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b})
	body.Write([]byte{0x20, 0})
	constI32(&body, stringArenaEnd-stringArenaStart-8)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b})
	body.Write([]byte{0x23, 0, 0x22, 1, 0x20, 0, 0x6a, 0x41, 8, 0x6a, 0x22, 2})
	constI32(&body, stringArenaEnd)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b, 0x20, 2, 0x24, 0, 0x20, 1, 0x0b})
	return body.Bytes()
}

func stringFreeBody() []byte {
	var body bytes.Buffer
	body.WriteByte(0)
	body.Write([]byte{0x20, 0})
	constI32(&body, stringArenaStart)
	body.Write([]byte{0x4f, 0x20, 0})
	constI32(&body, stringArenaEnd)
	body.Write([]byte{0x4d, 0x71, 0x20, 1, 0x45, 0x45, 0x71, 0x20, 1})
	constI32(&body, stringArenaEnd-stringArenaStart-8)
	body.Write([]byte{0x4d, 0x71, 0x04, 0x40, 0x20, 0, 0x20, 1, 0x6a, 0x41, 8, 0x6a, 0x23, 0, 0x46, 0x04, 0x40, 0x20, 0, 0x24, 0, 0x0b, 0x0b, 0x0b})
	return body.Bytes()
}

func utf8ValidatorBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 3, 0x7f}) // state, index, byte
	// block; loop; exit when index >= length.
	body.Write([]byte{0x02, 0x40, 0x03, 0x40, 0x20, 3, 0x20, 1, 0x4f, 0x0d, 1})
	constI32(&body, stringTableOffset)
	body.Write([]byte{0x20, 2, 0x41, 8, 0x74, 0x6a, 0x20, 0, 0x20, 3, 0x6a, 0x2d, 0, 0, 0x22, 4, 0x6a, 0x2d, 0, 0, 0x21, 2})
	body.Write([]byte{0x20, 2, 0x41, 8, 0x46, 0x0d, 1})
	body.Write([]byte{0x20, 3, 0x41, 1, 0x6a, 0x21, 3, 0x0c, 0, 0x0b, 0x0b})
	body.Write([]byte{0x20, 2, 0x45, 0x0b})
	return body.Bytes()
}

func stringEqualBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 1, 0x7f}) // index
	// Different lengths are unequal.
	body.Write([]byte{0x20, 0, 0x42, 32, 0x88, 0xa7, 0x20, 1, 0x42, 32, 0x88, 0xa7, 0x47, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b})
	body.Write([]byte{0x02, 0x40, 0x03, 0x40, 0x20, 2, 0x20, 0, 0x42, 32, 0x88, 0xa7, 0x4f, 0x0d, 1})
	body.Write([]byte{0x20, 0, 0xa7, 0x20, 2, 0x6a, 0x2d, 0, 0, 0x20, 1, 0xa7, 0x20, 2, 0x6a, 0x2d, 0, 0, 0x47, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b})
	body.Write([]byte{0x20, 2, 0x41, 1, 0x6a, 0x21, 2, 0x0c, 0, 0x0b, 0x0b, 0x41, 1, 0x0b})
	return body.Bytes()
}

func stringConcatBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 6, 0x7f}) // left ptr/len, right ptr/len, total, out
	// Unpack inputs.
	body.Write([]byte{0x20, 0, 0xa7, 0x21, 2, 0x20, 0, 0x42, 32, 0x88, 0xa7, 0x21, 3})
	body.Write([]byte{0x20, 1, 0xa7, 0x21, 4, 0x20, 1, 0x42, 32, 0x88, 0xa7, 0x21, 5})
	body.Write([]byte{0x20, 3, 0x20, 5, 0x6a, 0x21, 6})
	// Empty concatenation needs no allocation.
	body.Write([]byte{0x20, 6, 0x45, 0x04, 0x40, 0x42, 0, 0x0f, 0x0b})
	body.Write([]byte{0x20, 6, 0x10, 0, 0x22, 7, 0x45, 0x04, 0x40, 0x20, 6, 0xad, 0x42, 32, 0x86, 0x0f, 0x0b})
	// memory.copy(out,left,leftLen); memory.copy(out+leftLen,right,rightLen)
	body.Write([]byte{0x20, 7, 0x20, 2, 0x20, 3, 0xfc, 0x0a, 0, 0})
	body.Write([]byte{0x20, 7, 0x20, 3, 0x6a, 0x20, 4, 0x20, 5, 0xfc, 0x0a, 0, 0})
	body.Write([]byte{0x20, 7, 0xad, 0x20, 6, 0xad, 0x42, 32, 0x86, 0x84, 0x0b})
	return body.Bytes()
}

func stringProviderBody(parameters []pureValueType, result pureValueType, abi PureABI) []byte {
	var body bytes.Buffer
	// cursor plus result temporary.
	body.Write([]byte{2, 1, 0x7f, 1, result.wasm})
	// Bounds for the complete request and initialize canonical payload cursor.
	body.Write([]byte{0x20, 3})
	constI32(&body, abi.FixedHeaderSize)
	body.Write([]byte{0x49, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b, 0x20, 3})
	constI32(&body, 7160)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	constI32(&body, abi.FixedHeaderSize)
	body.Write([]byte{0x21, 6})
	var offset uint64
	for _, parameter := range parameters {
		switch parameter.name {
		case "bool":
			body.Write([]byte{0x20, 2, 0x2d, 0})
			uleb(&body, offset)
			body.Write([]byte{0x41, 1, 0x4b, 0x04, 0x40, 0x41, 3, 0x0f, 0x0b})
		case "string":
			// Descriptor offset must equal the running cursor.
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset)
			body.Write([]byte{0x20, 6, 0x47, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
			// length <= 4096 and cursor+length <= exact request length.
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			constI32(&body, stringLimit)
			body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
			body.Write([]byte{0x20, 6, 0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			body.Write([]byte{0x6a, 0x22, 6, 0x20, 3, 0x4b, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
			// Strict UTF-8 validator over request pointer + descriptor offset.
			body.Write([]byte{0x20, 2, 0x20, 2, 0x28, 2})
			uleb(&body, offset)
			body.Write([]byte{0x6a, 0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			body.Write([]byte{0x10, 6, 0x45, 0x04, 0x40, 0x41, 5, 0x0f, 0x0b})
		case "slice:i64":
			// Payload offset must be canonical and element count is bounded.
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset)
			body.Write([]byte{0x20, 6, 0x47, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			constI32(&body, 512)
			body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
			// cursor += element_count * 8; it must remain within the request.
			body.Write([]byte{0x20, 6, 0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			body.Write([]byte{0x41, 0x03, 0x74, 0x6a, 0x22, 6, 0x20, 3, 0x4b, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
		}
		offset += parameter.size
	}
	body.Write([]byte{0x20, 6, 0x20, 3, 0x47, 0x04, 0x40, 0x41, 4, 0x0f, 0x0b})
	// Decode helper arguments.
	offset = 0
	for _, parameter := range parameters {
		body.Write([]byte{0x20, 2})
		switch parameter.name {
		case "i64":
			body.Write([]byte{0x29, 3})
			uleb(&body, offset)
		case "bool":
			body.Write([]byte{0x2d, 0})
			uleb(&body, offset)
		case "string":
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset)
			body.WriteByte(0x6a)
			body.WriteByte(0xad)
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			body.Write([]byte{0xad, 0x42, 32, 0x86, 0x84})
		case "slice:i64":
			body.Write([]byte{0x20, 2, 0x28, 2})
			uleb(&body, offset)
			body.Write([]byte{0x6a, 0xad, 0x20, 2, 0x28, 2})
			uleb(&body, offset+4)
			body.Write([]byte{0xad, 0x42, 32, 0x86, 0x84})
		}
		offset += parameter.size
	}
	body.Write([]byte{0x10, 5, 0x21, 7})
	if result.name == "string" {
		// Discard helper allocations on both success and result-validation
		// failure while retaining the host's output-header allocation.
		body.Write([]byte{0x20, 4, 0x41, 16, 0x6a, 0x24, 0})
		// Validate result length/pointer, copy to stable response scratch, discard temporaries.
		body.Write([]byte{0x20, 7, 0x42, 32, 0x88, 0xa7})
		constI32(&body, stringLimit)
		body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 6, 0x0f, 0x0b})
		body.Write([]byte{0x20, 7, 0x42, 32, 0x88, 0xa7, 0x45, 0x04, 0x40, 0x05, 0x20, 7, 0xa7, 0x41})
		sleb(&body, stringArenaStart)
		body.Write([]byte{0x49, 0x04, 0x40, 0x41, 6, 0x0f, 0x0b, 0x0b})
		body.Write([]byte{0x20, 7, 0xa7, 0x20, 7, 0x42, 32, 0x88, 0xa7, 0x6a})
		constI32(&body, stringArenaEnd)
		body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 6, 0x0f, 0x0b})
		constI32(&body, stringScratch)
		body.Write([]byte{0x20, 7, 0xa7, 0x20, 7, 0x42, 32, 0x88, 0xa7, 0xfc, 0x0a, 0, 0})
		body.Write([]byte{0x20, 4})
		constI32(&body, stringScratch)
		body.Write([]byte{0x36, 2, 0})
		body.Write([]byte{0x20, 5, 0x20, 7, 0x42, 32, 0x88, 0xa7, 0x36, 2, 0, 0x41, 0, 0x0b})
		return body.Bytes()
	}
	constI32(&body, stringScratch)
	body.Write([]byte{0x20, 7})
	if result.name == "i64" {
		body.Write([]byte{0x37, 3, 0})
	} else {
		body.Write([]byte{0x3a, 0, 0})
	}
	body.Write([]byte{0x20, 4})
	constI32(&body, stringScratch)
	body.Write([]byte{0x36, 2, 0})
	body.Write([]byte{0x20, 5, 0x41})
	sleb(&body, int64(result.size))
	body.Write([]byte{0x36, 2, 0, 0x41, 0, 0x0b})
	return body.Bytes()
}
