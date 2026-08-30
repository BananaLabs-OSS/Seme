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
	expression, signature, info := checkedReturnExpression(t, "return a*b <= c")
	if _, err := analyzeGoExpression(expression, signature, info); err == nil {
		t.Fatal("multiplication was accepted")
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
