package goprovider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestInjectBeforeNativeContinueSkipsNestedLoops(t *testing.T) {
	outerContinue := &goStatement{nativeBranch: "continue"}
	nestedContinue := &goStatement{nativeBranch: "continue"}
	block := &goBlock{statements: []*goStatement{
		{thenBlock: &goBlock{statements: []*goStatement{outerContinue}}},
		{loopBlock: &goBlock{statements: []*goStatement{nestedContinue}}},
	}}
	injectBeforeNativeContinue(block, func() *goStatement { return &goStatement{localName: "post"} })
	if got := block.statements[0].thenBlock.statements; len(got) != 2 || got[0].localName != "post" || got[1] != outerContinue {
		t.Fatalf("outer continue post injection = %#v", got)
	}
	if got := block.statements[1].loopBlock.statements; len(got) != 1 || got[0] != nestedContinue {
		t.Fatalf("nested loop was rewritten = %#v", got)
	}
}

func TestAnalyzeGoBlockRejectsMissingIndexTypeWithoutPanic(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "invalid.go", "package p\nfunc Read() int64 { value, ok := missing[0]; _ = ok; return value }", 0)
	if err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	signature := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int64])), false)
	if _, err := analyzeGoBlock(function.Body.List, signature, info); err == nil {
		t.Fatal("missing map type accepted")
	}
}

func TestAnalyzeGoBlockNormalizesNestedTotalReturns(t *testing.T) {
	source := `package p
func Decide(enabled bool, value, limit int64) bool {
	if enabled || ("λ" + "!" == "never") {
		if value <= limit { return true }
		return false
	}
	return false
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "control.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	if _, err := (&types.Config{}).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	signature := info.Defs[function.Name].Type().(*types.Signature)
	block, err := analyzeGoBlock(function.Body.List, signature, info)
	if err != nil {
		t.Fatal(err)
	}
	if block.statements[0].condition.kind != goBooleanOr || block.statements[0].thenBlock.statements[0].condition.kind != goIntegerLessEqual {
		t.Fatal("nested if structure was not preserved")
	}
	if block.statements[0].elseBlock.statements[0].returned.kind != goBooleanLiteral {
		t.Fatal("following return was not normalized into the else block")
	}
	instances := []graphEntity{}
	root, err := emitCanonicalBlock(block, "function", "body", []string{"p0", "p1", "p2"}, "i64", &instances)
	if err != nil {
		t.Fatal(err)
	}
	if root == "" || len(instances) < 10 {
		t.Fatalf("emitted root %q with %d entities", root, len(instances))
	}
}

func TestAnalyzeGoBlockRejectsNonTotalIf(t *testing.T) {
	source := `package p
func Decide(enabled bool) bool { if enabled { return true } }`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "control.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	_, _ = (&types.Config{}).Check("p", fset, []*ast.File{file}, info)
	function := file.Decls[0].(*ast.FuncDecl)
	signature := info.Defs[function.Name].Type().(*types.Signature)
	if _, err := analyzeGoBlock(function.Body.List, signature, info); err == nil {
		t.Fatal("non-total control flow was accepted")
	}
}
