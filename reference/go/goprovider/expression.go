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
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind      goExpressionKind
	parameter int
	integer   uint64
	boolean   bool
	left      *goExpression
	right     *goExpression
}

func emitCanonicalExpression(expression *goExpression, owner string, parameterIDs []string, integerID string) ([]graphEntity, string, error) {
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
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		for index := 0; index < signature.Params().Len(); index++ {
			if info.Uses[expression] == signature.Params().At(index) {
				return &goExpression{kind: goParameterRead, parameter: index}, nil
			}
		}
		if object, ok := info.Uses[expression].(*types.Const); ok && object.Type() == types.Typ[types.UntypedBool] {
			return &goExpression{kind: goBooleanLiteral, boolean: constant.BoolVal(object.Val())}, nil
		}
		return nil, fmt.Errorf("expression.unresolved_parameter")
	case *ast.BasicLit:
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
			kind = goIntegerAdd
		case token.MUL:
			kind = goIntegerMultiply
		case token.SUB:
			kind = goIntegerSubtract
		case token.LAND:
			kind = goBooleanAnd
		case token.LEQ:
			kind = goIntegerLessEqual
		case token.GEQ:
			kind = goIntegerLessEqual
			left, right = right, left
		default:
			return nil, fmt.Errorf("expression.unsupported_operator:%s", expression.Op)
		}
		analyzedLeft, err := analyzeGoExpression(left, signature, info)
		if err != nil {
			return nil, err
		}
		analyzedRight, err := analyzeGoExpression(right, signature, info)
		if err != nil {
			return nil, err
		}
		return &goExpression{kind: kind, left: analyzedLeft, right: analyzedRight}, nil
	default:
		return nil, fmt.Errorf("expression.unsupported_node")
	}
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
