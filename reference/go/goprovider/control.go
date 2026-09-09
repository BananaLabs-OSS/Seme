package goprovider

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

type goStatement struct {
	condition   *goExpression
	returned    *goExpression
	thenBlock   *goBlock
	elseBlock   *goBlock
	localName   string
	localType   string
	local       int
	initializer *goExpression
	mutable     bool
	assignment  *goExpression
	loopBlock   *goBlock
	whenBlock   *goBlock
	effect      string
	effectArgs  []*goExpression
}

type goBlock struct {
	statements []*goStatement
}

func analyzeGoBlock(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	return analyzeGoBlockWithCalls(statements, signature, info, nil)
}

func analyzeGoBlockWithCalls(statements []ast.Stmt, signature *types.Signature, info *types.Info, functions map[types.Object]string) (*goBlock, error) {
	return analyzeGoBlockWithProgram(statements, signature, info, functions, nil)
}

func analyzeGoBlockWithProgram(statements []ast.Stmt, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goBlock, error) {
	if tagged, ok := matchGoTotalTaggedValue(statements, signature, info, functions, records); ok {
		return &goBlock{statements: []*goStatement{{returned: tagged}}}, nil
	}
	if tally, ok := matchGoRuntimeMapTally(statements, signature, info); ok {
		return &goBlock{statements: []*goStatement{{returned: tally}}}, nil
	}
	if closure, ok := matchGoMutableCounterConstructor(statements, signature, info); ok {
		return &goBlock{statements: []*goStatement{{returned: closure}}}, nil
	}
	if block, ok := matchGoMutableCounterRun(statements, signature, info, functions); ok {
		return block, nil
	}
	if fold, ok := matchGoFixedArrayFold(statements, signature, info); ok {
		return &goBlock{statements: []*goStatement{{returned: fold}}}, nil
	}
	next := 0
	mutable := map[types.Object]bool{}
	for _, statement := range statements {
		ast.Inspect(statement, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || assignment.Tok != token.ASSIGN {
				return true
			}
			for _, target := range assignment.Lhs {
				if name, ok := target.(*ast.Ident); ok && info.Uses[name] != nil {
					mutable[info.Uses[name]] = true
				}
			}
			return true
		})
	}
	return analyzeGoBlockScoped(statements, signature, info, map[types.Object]int{}, functions, records, mutable, &next, true)
}

func matchGoTotalTaggedValue(statements []ast.Stmt, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goExpression, bool) {
	if len(statements) != 2 {
		return nil, false
	}
	branch, ok := statements[0].(*ast.IfStmt)
	if !ok || branch.Init != nil || branch.Else != nil || len(branch.Body.List) == 0 {
		return nil, false
	}
	condition, ok := branch.Cond.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	source, ok := condition.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	parameter := -1
	for index := 0; index < signature.Params().Len(); index++ {
		if info.Uses[source] == signature.Params().At(index) {
			parameter = index
			break
		}
	}
	if parameter < 0 {
		return nil, false
	}
	absentReturn, ok := statements[1].(*ast.ReturnStmt)
	if !ok || len(absentReturn.Results) != 1 {
		return nil, false
	}
	read := &goExpression{kind: goParameterRead, parameter: parameter}
	sourceType := signature.Params().At(parameter).Type()
	if item, option := goOptionValueType(sourceType); option && condition.Sel.Name == "Some" {
		bindingName := "value"
		if nested, ok := analyzeNestedResultArm(branch.Body.List, source, item, signature, info, functions, records); ok {
			none, err := analyzeGoExpressionWithProgram(absentReturn.Results[0], signature, info, nil, functions, records, nil)
			if err != nil {
				return nil, false
			}
			return &goExpression{kind: goOptionMatch, left: read, initial: none, body: nested, text: bindingName, typeID: goSemanticTypeIdentity(item)}, true
		}
		if len(branch.Body.List) != 1 {
			return nil, false
		}
		presentReturn, ok := branch.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(presentReturn.Results) != 1 {
			return nil, false
		}
		some, ok := analyzeTaggedArm(presentReturn.Results[0], source, "Value", bindingName, signature, info, functions, records)
		if !ok {
			return nil, false
		}
		none, err := analyzeGoExpressionWithProgram(absentReturn.Results[0], signature, info, nil, functions, records, nil)
		if err != nil {
			return nil, false
		}
		return &goExpression{kind: goOptionMatch, left: read, initial: none, body: some, text: bindingName, typeID: goSemanticTypeIdentity(item)}, true
	}
	if success, failure, result := goResultTypes(sourceType); result && condition.Sel.Name == "Ok" {
		if len(branch.Body.List) != 1 {
			return nil, false
		}
		presentReturn, ok := branch.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(presentReturn.Results) != 1 {
			return nil, false
		}
		okName, errorName := "value", "failure"
		okArm, ok := analyzeTaggedArm(presentReturn.Results[0], source, "Value", okName, signature, info, functions, records)
		if !ok {
			return nil, false
		}
		errorArm, ok := analyzeTaggedArm(absentReturn.Results[0], source, "Error", errorName, signature, info, functions, records)
		if !ok {
			var err error
			errorArm, err = analyzeGoExpressionWithProgram(absentReturn.Results[0], signature, info, nil, functions, records, nil)
			if err != nil {
				return nil, false
			}
		}
		return &goExpression{kind: goResultMatch, left: read, body: okArm, alternate: errorArm, text: okName, typeID: goSemanticTypeIdentity(success), errorName: errorName, errorTypeID: goSemanticTypeIdentity(failure)}, true
	}
	return nil, false
}

// analyzeNestedResultArm recognizes the ordinary Go spelling projected for a
// total Result match nested inside an Option arm. It remains structural and
// type-directed; arbitrary conditionals are handled by the general block path.
func analyzeNestedResultArm(statements []ast.Stmt, option *ast.Ident, item types.Type, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goExpression, bool) {
	if len(statements) != 2 {
		return nil, false
	}
	success, failure, ok := goResultTypes(item)
	if !ok {
		return nil, false
	}
	branch, ok := statements[0].(*ast.IfStmt)
	if !ok || branch.Init != nil || branch.Else != nil || len(branch.Body.List) != 1 {
		return nil, false
	}
	condition, ok := branch.Cond.(*ast.SelectorExpr)
	if !ok || condition.Sel.Name != "Ok" || !optionValueSelector(condition.X, option, info) {
		return nil, false
	}
	okReturn, ok := branch.Body.List[0].(*ast.ReturnStmt)
	errReturn, errOK := statements[1].(*ast.ReturnStmt)
	if !ok || !errOK || len(okReturn.Results) != 1 || len(errReturn.Results) != 1 {
		return nil, false
	}
	okArm, ok := analyzeNestedVariantExpression(okReturn.Results[0], option, "Value", signature, info, functions, records)
	if !ok {
		return nil, false
	}
	errArm, ok := analyzeNestedVariantExpression(errReturn.Results[0], option, "Error", signature, info, functions, records)
	if !ok {
		return nil, false
	}
	return &goExpression{kind: goResultMatch, left: &goExpression{kind: goVariantRead, text: "value"}, body: okArm, alternate: errArm, text: "value", typeID: goSemanticTypeIdentity(success), errorName: "failure", errorTypeID: goSemanticTypeIdentity(failure)}, true
}

func optionValueSelector(node ast.Expr, option *ast.Ident, info *types.Info) bool {
	selector, ok := ast.Unparen(node).(*ast.SelectorExpr)
	base, baseOK := func() (*ast.Ident, bool) {
		if !ok {
			return nil, false
		}
		value, yes := ast.Unparen(selector.X).(*ast.Ident)
		return value, yes
	}()
	return ok && baseOK && selector.Sel.Name == "Value" && info.Uses[base] == info.Uses[option]
}

func nestedVariantSelector(node ast.Expr, option *ast.Ident, field string, info *types.Info) bool {
	selector, ok := ast.Unparen(node).(*ast.SelectorExpr)
	return ok && selector.Sel.Name == field && optionValueSelector(selector.X, option, info)
}

func analyzeNestedVariantExpression(node ast.Expr, option *ast.Ident, field string, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goExpression, bool) {
	if nestedVariantSelector(node, option, field, info) {
		return &goExpression{kind: goVariantRead}, true
	}
	if binary, ok := ast.Unparen(node).(*ast.BinaryExpr); ok && binary.Op == token.EQL && field == "Error" {
		var other ast.Expr
		if nestedVariantSelector(binary.X, option, field, info) {
			other = binary.Y
		} else if nestedVariantSelector(binary.Y, option, field, info) {
			other = binary.X
		} else {
			return nil, false
		}
		value, err := analyzeGoExpressionWithProgram(other, signature, info, nil, functions, records, nil)
		if err != nil {
			return nil, false
		}
		return &goExpression{kind: goStringEqual, left: &goExpression{kind: goVariantRead}, right: value}, true
	}
	call, ok := ast.Unparen(node).(*ast.CallExpr)
	selector, selectorOK := func() (*ast.SelectorExpr, bool) {
		if !ok {
			return nil, false
		}
		value, yes := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		return value, yes
	}()
	if field != "Value" || !selectorOK || !isBytesFunction(info, selector, "Equal") || len(call.Args) != 2 {
		return nil, false
	}
	var other ast.Expr
	if nestedVariantSelector(call.Args[0], option, field, info) {
		other = call.Args[1]
	} else if nestedVariantSelector(call.Args[1], option, field, info) {
		other = call.Args[0]
	} else {
		return nil, false
	}
	value, err := analyzeGoExpressionWithProgram(other, signature, info, nil, functions, records, nil)
	if err != nil {
		return nil, false
	}
	return &goExpression{kind: goBytesEqual, left: &goExpression{kind: goVariantRead}, right: value}, true
}

func analyzeTaggedArm(node ast.Expr, source *ast.Ident, field, binding string, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goExpression, bool) {
	if selector, ok := node.(*ast.SelectorExpr); ok {
		base, baseOK := selector.X.(*ast.Ident)
		if baseOK && info.Uses[base] == info.Uses[source] && selector.Sel.Name == field {
			return &goExpression{kind: goVariantRead, text: binding}, true
		}
	}
	value, err := analyzeGoExpressionWithProgram(node, signature, info, nil, functions, records, nil)
	return value, err == nil
}

func matchGoRuntimeMapTally(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goExpression, bool) {
	if signature.Params().Len() != 2 || !isI64Slice(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || len(statements) != 3 {
		return nil, false
	}
	bind, ok := statements[0].(*ast.AssignStmt)
	if !ok || bind.Tok != token.DEFINE || len(bind.Lhs) != 1 || len(bind.Rhs) != 1 {
		return nil, false
	}
	counts, countsOK := bind.Lhs[0].(*ast.Ident)
	empty, emptyOK := bind.Rhs[0].(*ast.CompositeLit)
	mapType, mapOK := info.TypeOf(empty).Underlying().(*types.Map)
	if !countsOK || !emptyOK || !mapOK || !isInt64(mapType.Key()) || !isInt64(mapType.Elem()) || len(empty.Elts) != 0 {
		return nil, false
	}
	rangeStatement, ok := statements[1].(*ast.RangeStmt)
	if !ok || rangeStatement.Tok != token.DEFINE || len(rangeStatement.Body.List) != 1 {
		return nil, false
	}
	valueName, valueOK := rangeStatement.Value.(*ast.Ident)
	if !valueOK || !isInt64(info.TypeOf(valueName)) {
		return nil, false
	}
	collection, collectionOK := rangeStatement.X.(*ast.Ident)
	if !collectionOK || info.Uses[collection] != signature.Params().At(0) {
		return nil, false
	}
	update, ok := rangeStatement.Body.List[0].(*ast.AssignStmt)
	if !ok || update.Tok != token.ASSIGN || len(update.Lhs) != 1 || len(update.Rhs) != 1 {
		return nil, false
	}
	target, targetOK := update.Lhs[0].(*ast.IndexExpr)
	addition, addOK := update.Rhs[0].(*ast.BinaryExpr)
	if !targetOK || !addOK || addition.Op != token.ADD {
		return nil, false
	}
	targetMap, tmOK := target.X.(*ast.Ident)
	targetKey, tkOK := target.Index.(*ast.Ident)
	lookup, lookupOK := addition.X.(*ast.IndexExpr)
	one, oneOK := addition.Y.(*ast.BasicLit)
	lookupMap, lmOK := func() (*ast.Ident, bool) {
		if !lookupOK {
			return nil, false
		}
		value, ok := lookup.X.(*ast.Ident)
		return value, ok
	}()
	lookupKey, lkOK := func() (*ast.Ident, bool) {
		if !lookupOK {
			return nil, false
		}
		value, ok := lookup.Index.(*ast.Ident)
		return value, ok
	}()
	if !tmOK || !tkOK || !lmOK || !lkOK || !oneOK || one.Value != "1" || info.Uses[targetMap] != info.Defs[counts] || info.Uses[lookupMap] != info.Defs[counts] || info.Uses[targetKey] != info.Defs[valueName] || info.Uses[lookupKey] != info.Defs[valueName] {
		return nil, false
	}
	returned, ok := statements[2].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return nil, false
	}
	finalLookup, ok := returned.Results[0].(*ast.IndexExpr)
	if !ok {
		return nil, false
	}
	finalMap, fmOK := finalLookup.X.(*ast.Ident)
	finalKey, fkOK := finalLookup.Index.(*ast.Ident)
	if !fmOK || !fkOK || info.Uses[finalMap] != info.Defs[counts] || info.Uses[finalKey] != signature.Params().At(1) {
		return nil, false
	}
	mapTypeID := stableID("execution", "type", "map", "i64", "i64")
	accRead := func() *goExpression { return &goExpression{kind: goIterationBindingRead, text: "accumulator"} }
	elementRead := func() *goExpression { return &goExpression{kind: goIterationBindingRead, text: "element"} }
	lookupCurrent := &goExpression{kind: goMapLookup, left: accRead(), right: elementRead()}
	updated := &goExpression{kind: goMapUpdate, left: accRead(), initial: elementRead(), right: &goExpression{kind: goIntegerAdd, left: lookupCurrent, right: &goExpression{kind: goIntegerLiteral, integer: 1}}}
	fold := &goExpression{kind: goFold, left: &goExpression{kind: goParameterRead, parameter: 0}, initial: &goExpression{kind: goEmptyMap, typeID: mapTypeID}, body: updated, accName: counts.Name, elementName: valueName.Name, typeID: mapTypeID}
	return &goExpression{kind: goMapLookup, left: fold, right: &goExpression{kind: goParameterRead, parameter: 1}}, true
}

func matchGoMutableCounterConstructor(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goExpression, bool) {
	if signature.Params().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || len(statements) != 2 {
		return nil, false
	}
	bind, ok := statements[0].(*ast.AssignStmt)
	if !ok || bind.Tok != token.DEFINE || len(bind.Lhs) != 1 || len(bind.Rhs) != 1 {
		return nil, false
	}
	name, nameOK := bind.Lhs[0].(*ast.Ident)
	start, startOK := bind.Rhs[0].(*ast.Ident)
	if !nameOK || !startOK || info.Uses[start] != signature.Params().At(0) {
		return nil, false
	}
	returned, ok := statements[1].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return nil, false
	}
	literal, ok := returned.Results[0].(*ast.FuncLit)
	if !ok || len(literal.Body.List) != 2 {
		return nil, false
	}
	closureSignature, ok := info.TypeOf(literal.Type).(*types.Signature)
	if !ok || !isUnaryI64Function(closureSignature) {
		return nil, false
	}
	assignment, ok := literal.Body.List[0].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return nil, false
	}
	target, targetOK := assignment.Lhs[0].(*ast.Ident)
	addition, addOK := assignment.Rhs[0].(*ast.BinaryExpr)
	if !targetOK || !addOK || addition.Op != token.ADD || info.Uses[target] != info.Defs[name] {
		return nil, false
	}
	left, leftOK := addition.X.(*ast.Ident)
	right, rightOK := addition.Y.(*ast.Ident)
	if !leftOK || !rightOK || info.Uses[left] != info.Defs[name] || info.Uses[right] != closureSignature.Params().At(0) {
		return nil, false
	}
	final, ok := literal.Body.List[1].(*ast.ReturnStmt)
	if !ok || len(final.Results) != 1 {
		return nil, false
	}
	result, ok := final.Results[0].(*ast.Ident)
	if !ok || info.Uses[result] != info.Defs[name] {
		return nil, false
	}
	readForUpdate := &goExpression{kind: goMutableCaptureRead}
	parameterRead := &goExpression{kind: goClosureParameterRead}
	update := &goExpression{kind: goCaptureUpdate, left: &goExpression{kind: goIntegerAdd, left: readForUpdate, right: parameterRead}}
	sequence := &goExpression{kind: goSequence, arguments: []*goExpression{update}, left: &goExpression{kind: goMutableCaptureRead}}
	return &goExpression{kind: goMutableClosureConstruct, left: &goExpression{kind: goParameterRead, parameter: 0}, body: sequence, text: name.Name, elementName: closureSignature.Params().At(0).Name(), typeID: goFunctionTypeID(closureSignature)}, true
}

func matchGoMutableCounterRun(statements []ast.Stmt, signature *types.Signature, info *types.Info, functions map[types.Object]string) (*goBlock, bool) {
	if signature.Params().Len() != 3 || len(statements) != 3 {
		return nil, false
	}
	bind, ok := statements[0].(*ast.AssignStmt)
	if !ok || bind.Tok != token.DEFINE || len(bind.Lhs) != 1 || len(bind.Rhs) != 1 {
		return nil, false
	}
	name, nameOK := bind.Lhs[0].(*ast.Ident)
	makeCall, callOK := bind.Rhs[0].(*ast.CallExpr)
	makeName, makeOK := func() (*ast.Ident, bool) {
		if !callOK {
			return nil, false
		}
		value, ok := makeCall.Fun.(*ast.Ident)
		return value, ok
	}()
	if !nameOK || !makeOK || len(makeCall.Args) != 1 {
		return nil, false
	}
	callee := functions[info.Uses[makeName]]
	makeArgument, makeArgumentOK := makeCall.Args[0].(*ast.Ident)
	if callee == "" || !makeArgumentOK || info.Uses[makeArgument] != signature.Params().At(0) {
		return nil, false
	}
	firstStatement, ok := statements[1].(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	firstCall, ok := firstStatement.X.(*ast.CallExpr)
	if !ok || len(firstCall.Args) != 1 {
		return nil, false
	}
	firstCallee, firstCalleeOK := firstCall.Fun.(*ast.Ident)
	firstArgument, firstArgumentOK := firstCall.Args[0].(*ast.Ident)
	if !firstCalleeOK || !firstArgumentOK || info.Uses[firstCallee] != info.Defs[name] || info.Uses[firstArgument] != signature.Params().At(1) {
		return nil, false
	}
	returnStatement, ok := statements[2].(*ast.ReturnStmt)
	if !ok || len(returnStatement.Results) != 1 {
		return nil, false
	}
	secondCall, ok := returnStatement.Results[0].(*ast.CallExpr)
	if !ok || len(secondCall.Args) != 1 {
		return nil, false
	}
	secondCallee, secondCalleeOK := secondCall.Fun.(*ast.Ident)
	secondArgument, secondArgumentOK := secondCall.Args[0].(*ast.Ident)
	if !secondCalleeOK || !secondArgumentOK || info.Uses[secondCallee] != info.Defs[name] || info.Uses[secondArgument] != signature.Params().At(2) {
		return nil, false
	}
	transitionType := goStatefulUnaryI64TransitionTypeID()
	makeExpression := &goExpression{kind: goFunctionCall, callee: callee, arguments: []*goExpression{{kind: goParameterRead, parameter: 0}}}
	first := &goExpression{kind: goStatefulIndirectCall, left: &goExpression{kind: goPlaceRead, local: 0}, arguments: []*goExpression{{kind: goParameterRead, parameter: 1}}, typeID: transitionType}
	second := &goExpression{kind: goStatefulIndirectCall, left: &goExpression{kind: goPlaceRead, local: 0}, arguments: []*goExpression{{kind: goParameterRead, parameter: 2}}, typeID: transitionType}
	return &goBlock{statements: []*goStatement{
		{localName: name.Name, localType: goUnaryI64FunctionTypeID(), local: 0, initializer: makeExpression, mutable: true},
		{localName: "first", localType: transitionType, local: 1, initializer: first},
		{local: 0, mutable: true, assignment: &goExpression{kind: goTransitionState, left: &goExpression{kind: goLocalRead, local: 1}}},
		{returned: &goExpression{kind: goTransitionResult, left: second}},
	}}, true
}

func matchGoFixedArrayFold(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goExpression, bool) {
	if len(statements) != 3 {
		return nil, false
	}
	declare, ok := statements[0].(*ast.AssignStmt)
	if !ok || declare.Tok != token.DEFINE || len(declare.Lhs) != 1 || len(declare.Rhs) != 1 {
		return nil, false
	}
	accumulator, ok := declare.Lhs[0].(*ast.Ident)
	if !ok || !isInt64(info.TypeOf(accumulator)) {
		return nil, false
	}
	conversion, ok := declare.Rhs[0].(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 {
		return nil, false
	}
	conversionName, ok := conversion.Fun.(*ast.Ident)
	literal, literalOK := conversion.Args[0].(*ast.BasicLit)
	if !ok || conversionName.Name != "int64" || !literalOK || literal.Kind != token.INT || literal.Value != "0" {
		return nil, false
	}
	rangeStatement, ok := statements[1].(*ast.RangeStmt)
	if !ok || rangeStatement.Tok != token.DEFINE || len(rangeStatement.Body.List) != 1 {
		return nil, false
	}
	key, keyOK := rangeStatement.Key.(*ast.Ident)
	element, elementOK := rangeStatement.Value.(*ast.Ident)
	collectionName, collectionOK := rangeStatement.X.(*ast.Ident)
	if !keyOK || key.Name != "_" || !elementOK || !collectionOK || !isInt64(info.TypeOf(element)) {
		return nil, false
	}
	parameter := -1
	for index := 0; index < signature.Params().Len(); index++ {
		if info.Uses[collectionName] == signature.Params().At(index) {
			parameter = index
			break
		}
	}
	if parameter < 0 {
		return nil, false
	}
	if _, ok := fixedI64ArrayLength(signature.Params().At(parameter).Type()); !ok && !isI64Slice(signature.Params().At(parameter).Type()) {
		return nil, false
	}
	update, ok := rangeStatement.Body.List[0].(*ast.AssignStmt)
	if !ok || update.Tok != token.ADD_ASSIGN || len(update.Lhs) != 1 || len(update.Rhs) != 1 {
		return nil, false
	}
	updateAccumulator, aOK := update.Lhs[0].(*ast.Ident)
	updateElement, eOK := update.Rhs[0].(*ast.Ident)
	if !aOK || !eOK || info.Uses[updateAccumulator] != info.Defs[accumulator] || info.Uses[updateElement] != info.Defs[element] {
		return nil, false
	}
	returned, ok := statements[2].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return nil, false
	}
	returnName, ok := returned.Results[0].(*ast.Ident)
	if !ok || info.Uses[returnName] != info.Defs[accumulator] {
		return nil, false
	}
	body := &goExpression{kind: goIntegerAdd, left: &goExpression{kind: goIterationBindingRead, text: "accumulator"}, right: &goExpression{kind: goIterationBindingRead, text: "element"}}
	return &goExpression{kind: goFold, left: &goExpression{kind: goParameterRead, parameter: parameter}, initial: &goExpression{kind: goIntegerLiteral}, body: body, accName: accumulator.Name, elementName: element.Name}, true
}

func analyzeGoBlockScoped(statements []ast.Stmt, signature *types.Signature, info *types.Info, inherited map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int, requireReturn bool) (*goBlock, error) {
	locals := cloneLocalScope(inherited)
	block := &goBlock{}
	for index, raw := range statements {
		switch statement := raw.(type) {
		case *ast.AssignStmt:
			if (statement.Tok != token.DEFINE && statement.Tok != token.ASSIGN) || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
				return nil, fmt.Errorf("control.local_binding_shape")
			}
			name, ok := statement.Lhs[0].(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("control.local_binding_shape")
			}
			object := info.Defs[name]
			if statement.Tok == token.ASSIGN {
				object = info.Uses[name]
			}
			if object == nil {
				return nil, fmt.Errorf("control.local_binding_type")
			}
			named, isRecord := object.Type().(*types.Named)
			_, recordSupported := records[named]
			if !isInt64(object.Type()) && !isBool(object.Type()) && !isPureString(object.Type()) && !(isRecord && recordSupported) {
				return nil, fmt.Errorf("control.local_binding_type")
			}
			initializer, err := analyzeGoExpressionWithProgram(statement.Rhs[0], signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			if statement.Tok == token.ASSIGN {
				local, exists := locals[object]
				if !exists || !mutable[object] {
					return nil, fmt.Errorf("control.assignment_target")
				}
				block.statements = append(block.statements, &goStatement{local: local, mutable: true, assignment: initializer})
				continue
			}
			local := *next
			*next++
			localType := "i64"
			if isBool(object.Type()) {
				localType = "bool"
			}
			if isPureString(object.Type()) {
				localType = "string"
			}
			if isRecord && recordSupported {
				localType = records[named].id
			}
			block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializer, mutable: mutable[object]})
			locals[object] = local
		case *ast.ReturnStmt:
			if index != len(statements)-1 || len(statement.Results) != 1 {
				return nil, fmt.Errorf("control.return_arity")
			}
			expression, err := analyzeGoExpressionWithProgram(statement.Results[0], signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{returned: expression})
		case *ast.IfStmt:
			if statement.Init == nil && statement.Else == nil && !blockContainsReturn(statement.Body.List) {
				condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, records, mutable, next, false)
				if err != nil {
					return nil, err
				}
				block.statements = append(block.statements, &goStatement{condition: condition, whenBlock: body})
				continue
			}
			if index != 0 && len(block.statements) != index {
				return nil, fmt.Errorf("control.statement_order")
			}
			branch, err := analyzeTerminalIfScoped(statement, statements[index+1:], signature, info, locals, functions, records, mutable, next)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, branch.statements...)
			return block, nil
		case *ast.ForStmt:
			if statement.Init != nil || statement.Post != nil || statement.Cond == nil {
				return nil, fmt.Errorf("control.for_shape")
			}
			condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, records, mutable, next, false)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{condition: condition, loopBlock: body})
		case *ast.ExprStmt:
			call, ok := statement.X.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return nil, fmt.Errorf("control.effect_shape")
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return nil, fmt.Errorf("control.unsupported_effect")
			}
			function, functionOK := info.Uses[selector.Sel].(*types.Func)
			if !functionOK || function.Pkg() == nil || function.Pkg().Path() != "log" || function.Name() != "Print" {
				return nil, fmt.Errorf("control.unsupported_effect")
			}
			argument, err := analyzeGoExpressionWithProgram(call.Args[0], signature, info, locals, functions, records, mutable)
			if err != nil || !isBool(info.TypeOf(call.Args[0])) {
				return nil, fmt.Errorf("control.effect_argument")
			}
			block.statements = append(block.statements, &goStatement{effect: "observability.log", effectArgs: []*goExpression{argument}})
		default:
			return nil, fmt.Errorf("control.unsupported_statement")
		}
	}
	if requireReturn && (len(block.statements) == 0 || block.statements[len(block.statements)-1].returned == nil) {
		return nil, fmt.Errorf("control.block_not_total")
	}
	if !requireReturn && len(block.statements) == 0 {
		return nil, fmt.Errorf("control.loop_body_empty")
	}
	return block, nil
}

func blockContainsReturn(statements []ast.Stmt) bool {
	contains := false
	for _, statement := range statements {
		ast.Inspect(statement, func(node ast.Node) bool {
			if _, ok := node.(*ast.ReturnStmt); ok {
				contains = true
				return false
			}
			return !contains
		})
	}
	return contains
}

func cloneLocalScope(source map[types.Object]int) map[types.Object]int {
	result := make(map[types.Object]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func analyzeTerminalIf(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	next := 0
	return analyzeTerminalIfScoped(statement, following, signature, info, map[types.Object]int{}, nil, nil, nil, &next)
}

func analyzeTerminalIfScoped(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int) (*goBlock, error) {
	if statement.Init != nil {
		return nil, fmt.Errorf("control.if_init_unsupported")
	}
	condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, locals, functions, records, mutable)
	if err != nil {
		return nil, err
	}
	thenBlock, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, records, mutable, next, true)
	if err != nil {
		return nil, err
	}
	var elseBlock *goBlock
	if statement.Else == nil {
		if len(following) == 0 {
			return nil, fmt.Errorf("control.if_missing_fallthrough_return")
		}
		elseBlock, err = analyzeGoBlockScoped(following, signature, info, locals, functions, records, mutable, next, true)
	} else {
		if len(following) != 0 {
			return nil, fmt.Errorf("control.unreachable_following_statement")
		}
		switch alternate := statement.Else.(type) {
		case *ast.BlockStmt:
			elseBlock, err = analyzeGoBlockScoped(alternate.List, signature, info, locals, functions, records, mutable, next, true)
		case *ast.IfStmt:
			elseBlock, err = analyzeTerminalIfScoped(alternate, nil, signature, info, locals, functions, records, mutable, next)
		default:
			err = fmt.Errorf("control.else_unsupported")
		}
	}
	if err != nil {
		return nil, err
	}
	return &goBlock{statements: []*goStatement{{condition: condition, thenBlock: thenBlock, elseBlock: elseBlock}}}, nil
}

func emitCanonicalBlock(block *goBlock, owner, path string, parameterIDs []string, integerTypeID string, instances *[]graphEntity) (string, error) {
	return emitCanonicalBlockScoped(block, owner, path, parameterIDs, integerTypeID, map[int]string{}, instances)
}

func emitCanonicalBlockScoped(block *goBlock, owner, path string, parameterIDs []string, integerTypeID string, inherited map[int]string, instances *[]graphEntity) (string, error) {
	if block == nil || len(block.statements) == 0 {
		return "", fmt.Errorf("control.nil_block")
	}
	localIDs := make(map[int]string, len(inherited)+len(block.statements))
	for key, value := range inherited {
		localIDs[key] = value
	}
	statementIDs := make([]string, 0, len(block.statements))
	for index, statement := range block.statements {
		statementPath := path + ".statement"
		if len(block.statements) != 1 {
			statementPath += "." + strconv.Itoa(index)
		}
		statementID := ""
		if statement.initializer != nil {
			bindingID := stableID("execution", owner, path, "local", strconv.Itoa(statement.local))
			localTypeID := integerTypeID
			if statement.localType == "bool" {
				localTypeID = stableID("execution", "type", "bool")
			}
			if statement.localType == "string" {
				localTypeID = stableID("execution", "type", "string")
			}
			if len(statement.localType) == 32 {
				localTypeID = statement.localType
			}
			expressions, initializerID, err := emitCanonicalExpressionWithLocals(statement.initializer, owner+":"+path+":local:"+strconv.Itoa(statement.local), parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			if statement.mutable {
				*instances = append(*instances, graphEntity{bindingID, entity(bindingID, "000000000000000000000000000090e0", []graphField{bytesField(0x9e00, statement.localName), refField(0x9e01, localTypeID), refField(0x9e02, initializerID)})})
				statementID = stableID("execution", owner, statementPath, "declare-place")
				*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090e1", []graphField{refField(0x9e10, bindingID)})})
			} else {
				*instances = append(*instances, graphEntity{bindingID, entity(bindingID, "000000000000000000000000000090d0", []graphField{bytesField(0x9d00, statement.localName), refField(0x9d01, localTypeID), refField(0x9d02, initializerID)})})
				statementID = stableID("execution", owner, statementPath, "bind-local")
				*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090d1", []graphField{refField(0x9d10, bindingID)})})
			}
			localIDs[statement.local] = bindingID
		} else if statement.assignment != nil {
			placeID, ok := localIDs[statement.local]
			if !ok {
				return "", fmt.Errorf("control.assignment_out_of_scope")
			}
			expressions, valueID, err := emitCanonicalExpressionWithLocals(statement.assignment, owner+":"+path+":assignment:"+strconv.Itoa(index), parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			statementID = stableID("execution", owner, statementPath, "assign-place")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090e3", []graphField{refField(0x9e30, placeID), refField(0x9e31, valueID)})})
		} else if statement.returned != nil {
			expressions, expressionID, err := emitCanonicalExpressionWithLocals(statement.returned, owner+":"+path, parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			statementID = stableID("execution", owner, statementPath, "return")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{expressionID})})})
		} else if statement.loopBlock != nil {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+path+":loop-condition:"+strconv.Itoa(index), parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			bodyID, err := emitCanonicalBlockScoped(statement.loopBlock, owner, path+".loop."+strconv.Itoa(index), parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			statementID = stableID("execution", owner, statementPath, "while")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090e4", []graphField{refField(0x9e40, conditionID), refField(0x9e41, bodyID)})})
		} else if statement.whenBlock != nil {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+path+":condition:"+strconv.Itoa(index), parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			bodyID, err := emitCanonicalBlockScoped(statement.whenBlock, owner, path+".when."+strconv.Itoa(index), parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			statementID = stableID("execution", owner, statementPath, "when")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090f0", []graphField{refField(0x9f00, conditionID), refField(0x9f01, bodyID)})})
		} else if statement.effect != "" {
			capabilityID := stableID("capability", statement.effect)
			effectID := stableID("effect", statement.effect)
			argumentIDs := make([]string, len(statement.effectArgs))
			for argumentIndex, argument := range statement.effectArgs {
				expressions, argumentID, err := emitCanonicalExpressionWithLocals(argument, owner+":"+path+":effect:"+strconv.Itoa(index)+":"+strconv.Itoa(argumentIndex), parameterIDs, localIDs, integerTypeID)
				if err != nil {
					return "", err
				}
				*instances = append(*instances, expressions...)
				argumentIDs[argumentIndex] = argumentID
			}
			if !hasGraphEntity(*instances, capabilityID) {
				*instances = append(*instances,
					graphEntity{capabilityID, entity(capabilityID, "00000000000000000000000000000016", []graphField{bytesField(0x160, statement.effect)})},
					graphEntity{effectID, entity(effectID, "00000000000000000000000000000015", []graphField{bytesField(0x150, statement.effect), refField(0x151, capabilityID)})},
				)
			}
			statementID = stableID("execution", owner, statementPath, "effect-invoke")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090f1", []graphField{refField(0x9f10, effectID), refsField(0x9f11, argumentIDs)})})
		} else {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+path+":condition", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			thenID, err := emitCanonicalBlockScoped(statement.thenBlock, owner, path+".then", parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			elseID, err := emitCanonicalBlockScoped(statement.elseBlock, owner, path+".else", parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			statementID = stableID("execution", owner, statementPath, "if")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090c0", []graphField{
				refField(0x9c00, conditionID), refField(0x9c01, thenID), refField(0x9c02, elseID),
			})})
		}
		statementIDs = append(statementIDs, statementID)
	}
	blockID := stableID("execution", owner, path, "block")
	*instances = append(*instances, graphEntity{blockID, entity(blockID, "00000000000000000000000000009080", []graphField{refsField(0x9800, statementIDs)})})
	return blockID, nil
}

func hasGraphEntity(instances []graphEntity, id string) bool {
	for _, instance := range instances {
		if instance.id == id {
			return true
		}
	}
	return false
}

// LiftControlFunction lifts the bounded total-return control-flow profile. Go
// fallthrough after an if is normalized into the explicit canonical else block.
func LiftControlFunction(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	declaration, fn, signature, info, err := resolveFunction(project, manifest, functionName)
	if err != nil {
		return "", "", err
	}
	if signature.Results().Len() != 1 || (!isInt64(signature.Results().At(0).Type()) && !isBool(signature.Results().At(0).Type()) && !isPureString(signature.Results().At(0).Type())) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	for index := 0; index < signature.Params().Len(); index++ {
		_, fixedArray := fixedI64ArrayLength(signature.Params().At(index).Type())
		if !isInt64(signature.Params().At(index).Type()) && !isBool(signature.Params().At(index).Type()) && !isPureString(signature.Params().At(index).Type()) && !fixedArray && !isI64Slice(signature.Params().At(index).Type()) {
			return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
		}
	}
	block, err := analyzeGoBlock(fn.Body.List, signature, info)
	if err != nil {
		return "", "", fmt.Errorf("provider.execution_unsupported_control:%s:%w", functionName, err)
	}

	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	stringID := stableID("execution", "type", "string")
	resultTypeID := integerID
	if isBool(signature.Results().At(0).Type()) {
		resultTypeID = booleanID
	} else if isPureString(signature.Results().At(0).Type()) {
		resultTypeID = stringID
	}
	instances := []graphEntity{{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{
		unsignedField(0x9100, 64), {0x9101, "tr"}, unsignedField(0x9102, 0),
	})}}
	needsBoolean := isBool(signature.Results().At(0).Type())
	needsString := isPureString(signature.Results().At(0).Type())
	parameterIDs := make([]string, signature.Params().Len())
	for index := range parameterIDs {
		needsBoolean = needsBoolean || isBool(signature.Params().At(index).Type())
		needsString = needsString || isPureString(signature.Params().At(index).Type())
	}
	if needsBoolean {
		instances = append(instances, graphEntity{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)})
	}
	if needsString {
		instances = append(instances, graphEntity{stringID, entity(stringID, "00000000000000000000000000009040", nil)})
	}
	for index := range parameterIDs {
		typeID := integerID
		if isBool(signature.Params().At(index).Type()) {
			typeID = booleanID
		} else if isPureString(signature.Params().At(index).Type()) {
			typeID = stringID
		} else if length, ok := fixedI64ArrayLength(signature.Params().At(index).Type()); ok {
			typeID = stableID("execution", "type", "fixed-array", "i64", strconv.FormatUint(length, 10))
			if !hasGraphEntity(instances, typeID) {
				instances = append(instances, graphEntity{typeID, entity(typeID, "000000000000000000000000000090f2", []graphField{refField(0x9f20, integerID), unsignedField(0x9f21, length)})})
			}
		} else if isI64Slice(signature.Params().At(index).Type()) {
			typeID = stableID("execution", "type", "slice", "i64")
			if !hasGraphEntity(instances, typeID) {
				instances = append(instances, graphEntity{typeID, entity(typeID, "000000000000000000000000000090f8", []graphField{refField(0x9f80, integerID)})})
			}
		}
		parameterIDs[index] = stableID("execution", declaration.ID, "parameter", strconv.Itoa(index))
		instances = append(instances, graphEntity{parameterIDs[index], entity(parameterIDs[index], "00000000000000000000000000009012", []graphField{
			bytesField(0x9120, signature.Params().At(index).Name()), refField(0x9121, typeID), unsignedField(0x9122, uint64(index)),
		})})
	}
	bodyID, err := emitCanonicalBlock(block, declaration.ID, "body", parameterIDs, integerID, &instances)
	if err != nil {
		return "", "", err
	}
	programID := stableID("execution", declaration.ID, "program")
	instances = append(instances,
		graphEntity{declaration.ID, entity(declaration.ID, "00000000000000000000000000009011", []graphField{
			bytesField(0x9110, functionName), refsField(0x9111, parameterIDs), refField(0x9112, resultTypeID), refField(0x9113, bodyID),
		})},
		graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{
			refsField(0x9150, []string{declaration.ID}), refField(0x9151, declaration.ID),
		})},
	)
	return composeExecutionG1(moduleG1, manifest.Revision, instances), declaration.ID, nil
}
