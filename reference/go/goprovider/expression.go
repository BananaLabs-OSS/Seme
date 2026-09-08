package goprovider

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
)

type goExpressionKind uint8

const (
	goParameterRead goExpressionKind = iota + 1
	goIntegerAdd
	goIntegerLessEqual
	goIntegerLiteral
	goIntegerMultiply
	goIntegerSubtract
	goBooleanLiteral
	goBooleanAnd
	goStringLiteral
	goBooleanOr
	goStringEqual
	goStringConcat
	goLocalRead
	goFunctionCall
	goRecordConstruct
	goFieldRead
	goPlaceRead
	goFixedArrayConstruct
	goIndexRead
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind       goExpressionKind
	parameter  int
	local      int
	integer    uint64
	boolean    bool
	text       string
	left       *goExpression
	right      *goExpression
	callee     string
	arguments  []*goExpression
	recordType string
	field      string
	values     []*goExpression
	mutable    bool
	arrayType  string
	arrayLen   uint64
}

func emitCanonicalExpression(expression *goExpression, owner string, parameterIDs []string, integerID string) ([]graphEntity, string, error) {
	return emitCanonicalExpressionWithLocals(expression, owner, parameterIDs, nil, integerID)
}

func emitCanonicalExpressionWithLocals(expression *goExpression, owner string, parameterIDs []string, localIDs map[int]string, integerID string) ([]graphEntity, string, error) {
	emitted := make(map[string]graphEntity)
	var emit func(*goExpression, string) (string, error)
	emit = func(expression *goExpression, path string) (string, error) {
		if expression == nil {
			return "", fmt.Errorf("expression.nil")
		}
		switch expression.kind {
		case goParameterRead:
			if expression.parameter < 0 || expression.parameter >= len(parameterIDs) {
				return "", fmt.Errorf("expression.parameter_out_of_range")
			}
			id := stableID("execution", owner, "read", strconv.Itoa(expression.parameter))
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[expression.parameter])})}
			return id, nil
		case goLocalRead:
			bindingID, ok := localIDs[expression.local]
			if !ok {
				return "", fmt.Errorf("expression.local_out_of_scope")
			}
			id := stableID("execution", owner, "local-read", bindingID)
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090d2", []graphField{refField(0x9d20, bindingID)})}
			return id, nil
		case goPlaceRead:
			placeID, ok := localIDs[expression.local]
			if !ok {
				return "", fmt.Errorf("expression.place_out_of_scope")
			}
			id := stableID("execution", owner, "place-read", placeID)
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090e2", []graphField{refField(0x9e20, placeID)})}
			return id, nil
		case goFunctionCall:
			if expression.callee == "" {
				return "", fmt.Errorf("expression.call_missing_callee")
			}
			arguments := make([]string, len(expression.arguments))
			for index, argument := range expression.arguments {
				id, err := emit(argument, path+".argument."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				arguments[index] = id
			}
			id := expressionNodeID(owner, path, "function-call")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009060", []graphField{refField(0x9600, expression.callee), refsField(0x9601, arguments)})}
			return id, nil
		case goRecordConstruct:
			values := make([]string, len(expression.values))
			for index, value := range expression.values {
				id, err := emit(value, path+".field."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				values[index] = id
			}
			id := expressionNodeID(owner, path, "record-construct")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009033", []graphField{refField(0x9330, expression.recordType), refsField(0x9331, values)})}
			return id, nil
		case goFieldRead:
			record, err := emit(expression.left, path+".record")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "field-read")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009032", []graphField{refField(0x9320, record), refField(0x9321, expression.field)})}
			return id, nil
		case goFixedArrayConstruct:
			values := make([]string, len(expression.values))
			for index, value := range expression.values {
				id, err := emit(value, path+".element."+strconv.Itoa(index))
				if err != nil {
					return "", err
				}
				values[index] = id
			}
			emitted[expression.arrayType] = graphEntity{expression.arrayType, entity(expression.arrayType, "000000000000000000000000000090f2", []graphField{refField(0x9f20, integerID), unsignedField(0x9f21, expression.arrayLen)})}
			id := expressionNodeID(owner, path, "fixed-array-construct")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f3", []graphField{refField(0x9f30, expression.arrayType), refsField(0x9f31, values)})}
			return id, nil
		case goIndexRead:
			collection, err := emit(expression.left, path+".collection")
			if err != nil {
				return "", err
			}
			index, err := emit(expression.right, path+".index")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "index-read")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090f4", []graphField{refField(0x9f40, collection), refField(0x9f41, index)})}
			return id, nil
		case goIntegerLiteral:
			id := expressionNodeID(owner, path, "integer-literal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009070", []graphField{unsignedField(0x9700, expression.integer), refField(0x9701, integerID)})}
			return id, nil
		case goBooleanLiteral:
			id := expressionNodeID(owner, path, "boolean-literal")
			value := "fa"
			if expression.boolean {
				value = "tr"
			}
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090b0", []graphField{{0x9b00, value}})}
			return id, nil
		case goBooleanAnd:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			id := expressionNodeID(owner, path, "boolean-and")
			emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090b1", []graphField{refField(0x9b10, left), refField(0x9b11, right)})}
			return id, nil
		case goBooleanOr, goStringEqual, goStringConcat:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			kind, schema, leftField, rightField := "boolean-or", "000000000000000000000000000090c1", uint64(0x9c10), uint64(0x9c11)
			if expression.kind == goStringEqual {
				kind, schema, leftField, rightField = "string-equal", "000000000000000000000000000090c2", 0x9c20, 0x9c21
			}
			if expression.kind == goStringConcat {
				kind, schema, leftField, rightField = "string-concat", "000000000000000000000000000090c3", 0x9c30, 0x9c31
			}
			id := expressionNodeID(owner, path, kind)
			emitted[id] = graphEntity{id, entity(id, schema, []graphField{refField(leftField, left), refField(rightField, right)})}
			return id, nil
		case goStringLiteral:
			id := expressionNodeID(owner, path, "string-literal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009050", []graphField{bytesField(0x9500, expression.text)})}
			return id, nil
		case goIntegerAdd, goIntegerMultiply, goIntegerSubtract, goIntegerLessEqual:
			left, err := emit(expression.left, path+".left")
			if err != nil {
				return "", err
			}
			right, err := emit(expression.right, path+".right")
			if err != nil {
				return "", err
			}
			if expression.kind == goIntegerAdd {
				id := expressionNodeID(owner, path, "add")
				emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009014", []graphField{refField(0x9140, left), refField(0x9141, right), refField(0x9142, integerID)})}
				return id, nil
			}
			if expression.kind == goIntegerMultiply {
				id := expressionNodeID(owner, path, "multiply")
				emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009090", []graphField{refField(0x9900, left), refField(0x9901, right), refField(0x9902, integerID)})}
				return id, nil
			}
			if expression.kind == goIntegerSubtract {
				id := expressionNodeID(owner, path, "subtract")
				emitted[id] = graphEntity{id, entity(id, "000000000000000000000000000090a0", []graphField{refField(0x9a00, left), refField(0x9a01, right), refField(0x9a02, integerID)})}
				return id, nil
			}
			id := expressionNodeID(owner, path, "less-equal")
			emitted[id] = graphEntity{id, entity(id, "00000000000000000000000000009021", []graphField{refField(0x9160, left), refField(0x9161, right), refField(0x9162, integerID)})}
			return id, nil
		default:
			return "", fmt.Errorf("expression.unsupported_kind")
		}
	}
	root, err := emit(expression, "root")
	if err != nil {
		return nil, "", err
	}
	entities := make([]graphEntity, 0, len(emitted))
	for _, item := range emitted {
		entities = append(entities, item)
	}
	return entities, root, nil
}

func expressionNodeID(owner, path, kind string) string {
	// Preserve the frozen v6 identities for the original decision shape. New
	// nested nodes use their normalized semantic path and do not renumber peers.
	if path == "root" && kind == "less-equal" {
		return stableID("execution", owner, "less-equal")
	}
	if path == "root.left" && kind == "add" {
		return stableID("execution", owner, "add")
	}
	return stableID("execution", owner, "expression", path, kind)
}

func analyzeGoExpression(expression ast.Expr, signature *types.Signature, info *types.Info) (*goExpression, error) {
	return analyzeGoExpressionWithLocals(expression, signature, info, nil)
}

func analyzeGoExpressionWithLocals(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int) (*goExpression, error) {
	return analyzeGoExpressionWithContext(expression, signature, info, locals, nil)
}

func analyzeGoExpressionWithContext(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string) (*goExpression, error) {
	return analyzeGoExpressionWithProgram(expression, signature, info, locals, functions, nil, nil)
}

type goRecordInfo struct {
	id      string
	fields  map[*types.Var]string
	ordered []*types.Var
}

func analyzeGoExpressionWithProgram(expression ast.Expr, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutableLocals map[types.Object]bool) (*goExpression, error) {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[expression] == signature.Params().At(index) {
				return &goExpression{kind: goParameterRead, parameter: index}, nil
			}
		}
		if local, ok := locals[info.Uses[expression]]; ok {
			kind := goLocalRead
			if mutableLocals[info.Uses[expression]] {
				kind = goPlaceRead
			}
			return &goExpression{kind: kind, local: local}, nil
		}
		if object, ok := info.Uses[expression].(*types.Const); ok && object.Type() == types.Typ[types.UntypedBool] {
			return &goExpression{kind: goBooleanLiteral, boolean: constant.BoolVal(object.Val())}, nil
		}
		return nil, fmt.Errorf("expression.unresolved_parameter")
	case *ast.BasicLit:
		if expression.Kind == token.STRING {
			value, err := strconv.Unquote(expression.Value)
			if err != nil {
				return nil, fmt.Errorf("expression.invalid_string_literal")
			}
			return &goExpression{kind: goStringLiteral, text: value}, nil
		}
		if expression.Kind != token.INT {
			return nil, fmt.Errorf("expression.unsupported_literal")
		}
		value, err := strconv.ParseInt(expression.Value, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("expression.invalid_integer_literal")
		}
		return &goExpression{kind: goIntegerLiteral, integer: uint64(value)}, nil
	case *ast.BinaryExpr:
		left, right := expression.X, expression.Y
		kind := goExpressionKind(0)
		switch expression.Op {
		case token.ADD:
			if isGoStringExpression(expression, info) {
				kind = goStringConcat
			} else {
				kind = goIntegerAdd
			}
		case token.MUL:
			kind = goIntegerMultiply
		case token.SUB:
			kind = goIntegerSubtract
		case token.LAND:
			kind = goBooleanAnd
		case token.LOR:
			kind = goBooleanOr
		case token.EQL:
			if isGoStringExpression(expression.X, info) {
				kind = goStringEqual
			} else {
				return nil, fmt.Errorf("expression.unsupported_operator:%s", expression.Op)
			}
		case token.LEQ:
			kind = goIntegerLessEqual
		case token.GEQ:
			kind = goIntegerLessEqual
			left, right = right, left
		default:
			return nil, fmt.Errorf("expression.unsupported_operator:%s", expression.Op)
		}
		analyzedLeft, err := analyzeGoExpressionWithProgram(left, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		analyzedRight, err := analyzeGoExpressionWithProgram(right, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: kind, left: analyzedLeft, right: analyzedRight}, nil
	case *ast.CallExpr:
		identifier, ok := ast.Unparen(expression.Fun).(*ast.Ident)
		callee, exists := functions[info.Uses[identifier]]
		if !ok || !exists || expression.Ellipsis.IsValid() {
			return nil, fmt.Errorf("expression.unsupported_call")
		}
		arguments := make([]*goExpression, len(expression.Args))
		for index, argument := range expression.Args {
			analyzed, err := analyzeGoExpressionWithProgram(argument, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			arguments[index] = analyzed
		}
		return &goExpression{kind: goFunctionCall, callee: callee, arguments: arguments}, nil
	case *ast.CompositeLit:
		if array, ok := info.TypeOf(expression).Underlying().(*types.Array); ok {
			if array.Len() <= 0 || array.Len() > 32 || !isInt64(array.Elem()) || int64(len(expression.Elts)) != array.Len() {
				return nil, fmt.Errorf("expression.unsupported_fixed_array")
			}
			values := make([]*goExpression, len(expression.Elts))
			for index, element := range expression.Elts {
				if _, keyed := element.(*ast.KeyValueExpr); keyed {
					return nil, fmt.Errorf("expression.keyed_array_unsupported")
				}
				value, err := analyzeGoExpressionWithProgram(element, signature, info, locals, functions, records, mutableLocals)
				if err != nil {
					return nil, err
				}
				values[index] = value
			}
			length := uint64(array.Len())
			return &goExpression{kind: goFixedArrayConstruct, arrayType: stableID("execution", "type", "fixed-array", "i64", strconv.FormatUint(length, 10)), arrayLen: length, values: values}, nil
		}
		named, ok := info.TypeOf(expression).(*types.Named)
		record, exists := records[named]
		if !ok || !exists || len(expression.Elts) != len(record.ordered) {
			return nil, fmt.Errorf("expression.unsupported_record_construct")
		}
		values := make([]*goExpression, len(record.ordered))
		seen := make(map[*types.Var]bool, len(values))
		for sourceIndex, element := range expression.Elts {
			fieldIndex := sourceIndex
			valueExpression := element
			if keyed, keyedOK := element.(*ast.KeyValueExpr); keyedOK {
				identifier, identifierOK := keyed.Key.(*ast.Ident)
				fieldIndex = -1
				for index, field := range record.ordered {
					if identifierOK && field.Name() == identifier.Name {
						fieldIndex = index
						break
					}
				}
				if fieldIndex < 0 {
					return nil, fmt.Errorf("expression.unknown_record_field")
				}
				valueExpression = keyed.Value
			}
			field := record.ordered[fieldIndex]
			if seen[field] {
				return nil, fmt.Errorf("expression.duplicate_record_field")
			}
			seen[field] = true
			value, err := analyzeGoExpressionWithProgram(valueExpression, signature, info, locals, functions, records, mutableLocals)
			if err != nil {
				return nil, err
			}
			values[fieldIndex] = value
		}
		for _, value := range values {
			if value == nil {
				return nil, fmt.Errorf("expression.missing_record_field")
			}
		}
		return &goExpression{kind: goRecordConstruct, recordType: record.id, values: values}, nil
	case *ast.SelectorExpr:
		selection := info.Selections[expression]
		if selection == nil {
			return nil, fmt.Errorf("expression.unsupported_selector")
		}
		field, ok := selection.Obj().(*types.Var)
		if !ok {
			return nil, fmt.Errorf("expression.unsupported_selector")
		}
		var fieldID string
		for _, record := range records {
			if id, exists := record.fields[field]; exists {
				fieldID = id
				break
			}
		}
		if fieldID == "" {
			return nil, fmt.Errorf("expression.unknown_record_field")
		}
		record, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: goFieldRead, left: record, field: fieldID}, nil
	case *ast.IndexExpr:
		array, ok := info.TypeOf(expression.X).Underlying().(*types.Array)
		if !ok || !isInt64(array.Elem()) || !isInt64(info.TypeOf(expression.Index)) {
			return nil, fmt.Errorf("expression.unsupported_index_read")
		}
		collection, err := analyzeGoExpressionWithProgram(expression.X, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		index, err := analyzeGoExpressionWithProgram(expression.Index, signature, info, locals, functions, records, mutableLocals)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: goIndexRead, left: collection, right: index}, nil
	default:
		return nil, fmt.Errorf("expression.unsupported_node")
	}
}

func isGoStringExpression(expression ast.Expr, info *types.Info) bool {
	if info == nil {
		return false
	}
	typeOf := info.TypeOf(expression)
	if typeOf == nil {
		return false
	}
	basic, ok := typeOf.Underlying().(*types.Basic)
	return ok && (basic.Kind() == types.String || basic.Kind() == types.UntypedString || basic.Info()&types.IsString != 0)
}

func evaluateBooleanExpression(expression *goExpression, parameters []int64) (bool, error) {
	if expression == nil {
		return false, fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goBooleanLiteral:
		return expression.boolean, nil
	case goBooleanAnd:
		left, err := evaluateBooleanExpression(expression.left, parameters)
		if err != nil || !left {
			return false, err
		}
		return evaluateBooleanExpression(expression.right, parameters)
	case goBooleanOr:
		left, err := evaluateBooleanExpression(expression.left, parameters)
		if err != nil || left {
			return left, err
		}
		return evaluateBooleanExpression(expression.right, parameters)
	case goStringEqual:
		left, err := evaluateStringExpression(expression.left)
		if err != nil {
			return false, err
		}
		right, err := evaluateStringExpression(expression.right)
		if err != nil {
			return false, err
		}
		return left == right, nil
	case goIntegerLessEqual:
		left, err := evaluateIntegerExpression(expression.left, parameters)
		if err != nil {
			return false, err
		}
		right, err := evaluateIntegerExpression(expression.right, parameters)
		if err != nil {
			return false, err
		}
		return left <= right, nil
	default:
		return false, fmt.Errorf("expression.not_boolean")
	}
}

func evaluateStringExpression(expression *goExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goStringLiteral:
		return expression.text, nil
	case goStringConcat:
		left, err := evaluateStringExpression(expression.left)
		if err != nil {
			return "", err
		}
		right, err := evaluateStringExpression(expression.right)
		if err != nil {
			return "", err
		}
		return left + right, nil
	default:
		return "", fmt.Errorf("expression.not_string")
	}
}

// evaluateIntegerExpression is the provider-neutral differential oracle for
// the bounded integer expression vocabulary. uint64 arithmetic makes the
// declared i64 modular overflow behavior explicit and independent of host
// signed-overflow rules.
func evaluateIntegerExpression(expression *goExpression, parameters []int64) (int64, error) {
	if expression == nil {
		return 0, fmt.Errorf("expression.nil")
	}
	switch expression.kind {
	case goParameterRead:
		if expression.parameter < 0 || expression.parameter >= len(parameters) {
			return 0, fmt.Errorf("expression.parameter_out_of_range")
		}
		return parameters[expression.parameter], nil
	case goIntegerLiteral:
		return int64(expression.integer), nil
	case goIntegerAdd, goIntegerMultiply, goIntegerSubtract:
		left, err := evaluateIntegerExpression(expression.left, parameters)
		if err != nil {
			return 0, err
		}
		right, err := evaluateIntegerExpression(expression.right, parameters)
		if err != nil {
			return 0, err
		}
		if expression.kind == goIntegerAdd {
			return int64(uint64(left) + uint64(right)), nil
		}
		if expression.kind == goIntegerMultiply {
			return int64(uint64(left) * uint64(right)), nil
		}
		return int64(uint64(left) - uint64(right)), nil
	default:
		return 0, fmt.Errorf("expression.not_integer")
	}
}

func matchAddParameters(expression *goExpression) (int, int, bool) {
	if expression == nil || expression.kind != goIntegerAdd || expression.left.kind != goParameterRead || expression.right.kind != goParameterRead {
		return 0, 0, false
	}
	return expression.left.parameter, expression.right.parameter, true
}

func matchAddLessEqualParameters(expression *goExpression) (decisionExpressionProfile, bool) {
	var profile decisionExpressionProfile
	if expression == nil || expression.kind != goIntegerLessEqual || expression.right.kind != goParameterRead {
		return profile, false
	}
	left, right, ok := matchAddParameters(expression.left)
	if !ok {
		return profile, false
	}
	limit := expression.right.parameter
	if left == right || left == limit || right == limit {
		return profile, false
	}
	return decisionExpressionProfile{addLeft: left, addRight: right, limit: limit, expression: expression}, true
}

func matchDecisionExpression(expression *goExpression, parameterCount int) bool {
	if expression == nil || expression.kind != goIntegerLessEqual {
		return false
	}
	seen := make(map[int]bool, parameterCount)
	var integer func(*goExpression) bool
	integer = func(expression *goExpression) bool {
		if expression == nil {
			return false
		}
		switch expression.kind {
		case goParameterRead:
			if expression.parameter < 0 || expression.parameter >= parameterCount {
				return false
			}
			seen[expression.parameter] = true
			return true
		case goIntegerLiteral:
			return true
		case goIntegerAdd, goIntegerMultiply, goIntegerSubtract:
			return integer(expression.left) && integer(expression.right)
		default:
			return false
		}
	}
	return integer(expression.left) && integer(expression.right) && len(seen) == parameterCount
}
