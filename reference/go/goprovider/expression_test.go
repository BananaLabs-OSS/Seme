package goprovider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestAnalyzeGoExpressionNormalizesPresentation(t *testing.T) {
	for _, source := range []string{
		"return a+b <= c",
		"return c >= b+a",
		"return (c) >= ((b) + (a))",
	} {
		expression, signature, info := checkedReturnExpression(t, source)
		analyzed, err := analyzeGoExpression(expression, signature, info)
		if err != nil {
			t.Fatalf("%q: %v", source, err)
		}
		profile, ok := matchAddLessEqualParameters(analyzed)
		if !ok {
			t.Fatalf("%q did not match", source)
		}
		if profile.limit != 2 {
			t.Fatalf("%q limit = %d", source, profile.limit)
		}
	}
}

func TestAnalyzeGoExpressionRejectsUnsupportedOperator(t *testing.T) {
	expression, signature, info := checkedReturnExpression(t, "return a/b <= c")
	if _, err := analyzeGoExpression(expression, signature, info); err == nil {
		t.Fatal("division was accepted before its semantic revision")
	}
}

func TestAnalyzeEvaluateOrderedNestedIntegerSubtract(t *testing.T) {
	expression, signature, info := checkedInt64ReturnExpression(t, "return (a - b) - (c * 2)")
	analyzed, err := analyzeGoExpression(expression, signature, info)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		parameters []int64
		want       int64
	}{
		{"left associativity", []int64{20, 3, 4}, 9},
		{"operand order", []int64{3, 20, 4}, -25},
		{"negative", []int64{-4, -9, 2}, 1},
		{"modular underflow", []int64{-9223372036854775808, 1, 0}, 9223372036854775807},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := evaluateIntegerExpression(analyzed, test.parameters)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("evaluate = %d, want %d", got, test.want)
			}
		})
	}
	entities, root, err := emitCanonicalExpression(analyzed, "subtract-function", []string{"p0", "p1", "p2"}, "i64")
	if err != nil {
		t.Fatal(err)
	}
	if root != expressionNodeID("subtract-function", "root", "subtract") {
		t.Fatal("root subtraction identity mismatch")
	}
	foundLeft := false
	for _, entity := range entities {
		if entity.id == expressionNodeID("subtract-function", "root.left", "subtract") {
			foundLeft = true
		}
	}
	if !foundLeft {
		t.Fatal("nested ordered subtraction was not emitted")
	}
}

func TestAnalyzeEvaluateBooleanAnd(t *testing.T) {
	expression, signature, info := checkedReturnExpression(t, "return true && a <= b && false")
	analyzed, err := analyzeGoExpression(expression, signature, info)
	if err != nil {
		t.Fatal(err)
	}
	got, err := evaluateBooleanExpression(analyzed, []int64{1, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("boolean expression evaluated true")
	}
	entities, root, err := emitCanonicalExpression(analyzed, "boolean-function", []string{"p0", "p1", "p2"}, "i64")
	if err != nil {
		t.Fatal(err)
	}
	if root != expressionNodeID("boolean-function", "root", "boolean-and") {
		t.Fatal("root BooleanAnd identity mismatch")
	}
	if len(entities) != 7 {
		t.Fatalf("emitted %d boolean expression entities, want 7", len(entities))
	}
}

func TestEvaluateBooleanAndShortCircuitsRightOperand(t *testing.T) {
	invalidRight := &goExpression{kind: goIntegerLessEqual,
		left:  &goExpression{kind: goParameterRead, parameter: 99},
		right: &goExpression{kind: goIntegerLiteral, integer: 0}}
	shortCircuited := &goExpression{kind: goBooleanAnd,
		left:  &goExpression{kind: goBooleanLiteral, boolean: false},
		right: invalidRight}
	got, err := evaluateBooleanExpression(shortCircuited, nil)
	if err != nil || got {
		t.Fatalf("false && invalid = %v, %v; want false, nil", got, err)
	}
	notShortCircuited := &goExpression{kind: goBooleanAnd,
		left:  &goExpression{kind: goBooleanLiteral, boolean: true},
		right: invalidRight}
	if _, err := evaluateBooleanExpression(notShortCircuited, nil); err == nil {
		t.Fatal("true left operand incorrectly skipped the invalid right operand")
	}
}

func TestAnalyzeEvaluateNestedIntegerMultiply(t *testing.T) {
	expression, signature, info := checkedInt64ReturnExpression(t, "return (a + 1) * (b * c)")
	analyzed, err := analyzeGoExpression(expression, signature, info)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		parameters []int64
		want       int64
	}{
		{"ordinary", []int64{2, 3, 4}, 36},
		{"negative", []int64{-4, 5, 2}, -30},
		{"zero", []int64{9, 0, 7}, 0},
		{"modular overflow", []int64{9223372036854775807, 1, 1}, -9223372036854775808},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := evaluateIntegerExpression(analyzed, test.parameters)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("evaluate = %d, want %d", got, test.want)
			}
		})
	}
	entities, root, err := emitCanonicalExpression(analyzed, "multiply-function", []string{"p0", "p1", "p2"}, "i64")
	if err != nil {
		t.Fatal(err)
	}
	if root != expressionNodeID("multiply-function", "root", "multiply") {
		t.Fatal("root multiplication identity mismatch")
	}
	multiplyCount := 0
	for _, entity := range entities {
		if entity.id == root || entity.id == expressionNodeID("multiply-function", "root.right", "multiply") {
			multiplyCount++
		}
	}
	if multiplyCount != 2 {
		t.Fatalf("emitted %d expected multiplication nodes", multiplyCount)
	}
}

func TestAnalyzeAndEmitNestedIntegerLiteral(t *testing.T) {
	expression, signature, info := checkedReturnExpression(t, "return c >= (b+a)+0")
	analyzed, err := analyzeGoExpression(expression, signature, info)
	if err != nil {
		t.Fatal(err)
	}
	if !matchDecisionExpression(analyzed, 3) {
		t.Fatal("nested literal decision did not match")
	}
	entities, _, err := emitCanonicalExpression(analyzed, "function", []string{"p0", "p1", "p2"}, "i64")
	if err != nil {
		t.Fatal(err)
	}
	foundLiteral := false
	for _, item := range entities {
		if item.text != "" && item.id == expressionNodeID("function", "root.left.right", "integer-literal") {
			foundLiteral = true
		}
	}
	if !foundLiteral {
		t.Fatal("canonical integer literal was not emitted")
	}
}

func TestEmitCanonicalExpressionPreservesFrozenDecisionIdentities(t *testing.T) {
	expression := &goExpression{kind: goIntegerLessEqual,
		left: &goExpression{kind: goIntegerAdd,
			left:  &goExpression{kind: goParameterRead, parameter: 1},
			right: &goExpression{kind: goParameterRead, parameter: 0}},
		right: &goExpression{kind: goParameterRead, parameter: 2}}
	parameters := []string{"p0", "p1", "p2"}
	entities, root, err := emitCanonicalExpression(expression, "function", parameters, "i64")
	if err != nil {
		t.Fatal(err)
	}
	if root != stableID("execution", "function", "less-equal") {
		t.Fatal("root identity changed")
	}
	wantAdd := stableID("execution", "function", "add")
	foundAdd := false
	for _, item := range entities {
		if item.id == wantAdd {
			foundAdd = true
		}
	}
	if !foundAdd || len(entities) != 5 {
		t.Fatalf("emitted %d entities; frozen add found=%v", len(entities), foundAdd)
	}
}

func checkedReturnExpression(t *testing.T, statement string) (ast.Expr, *types.Signature, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "expression.go", "package p\nfunc f(a, b, c int64) bool { "+statement+" }", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	if _, err := (&types.Config{}).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	signature := info.Defs[function.Name].Type().(*types.Signature)
	returned := function.Body.List[0].(*ast.ReturnStmt)
	return returned.Results[0], signature, info
}

func checkedInt64ReturnExpression(t *testing.T, statement string) (ast.Expr, *types.Signature, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "expression.go", "package p\nfunc f(a, b, c int64) int64 { "+statement+" }", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	if _, err := (&types.Config{}).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	signature := info.Defs[function.Name].Type().(*types.Signature)
	returned := function.Body.List[0].(*ast.ReturnStmt)
	return returned.Results[0], signature, info
}
