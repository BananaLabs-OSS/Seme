package goprovider

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

type goExpressionKind uint8

const (
	goParameterRead goExpressionKind = iota + 1
	goIntegerAdd
	goIntegerLessEqual
)

// goExpression is the provider's small typed source-expression tree. It keeps
// Go AST details out of canonical emission and normalizes equivalent source
// spellings before a target profile decides which tree shapes it supports.
type goExpression struct {
	kind      goExpressionKind
	parameter int
	left      *goExpression
	right     *goExpression
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
	return decisionExpressionProfile{addLeft: left, addRight: right, limit: limit}, true
}
