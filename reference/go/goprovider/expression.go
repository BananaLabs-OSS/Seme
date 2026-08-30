package goprovider

import (
	"fmt"
	"go/ast"
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
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind      goExpressionKind
	parameter int
	integer   uint64
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
		case goIntegerAdd, goIntegerLessEqual:
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
		case goIntegerAdd:
			return integer(expression.left) && integer(expression.right)
		default:
			return false
		}
	}
	return integer(expression.left) && integer(expression.right) && len(seen) == parameterCount
}
