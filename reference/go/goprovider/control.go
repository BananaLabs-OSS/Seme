package goprovider

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

type goStatement struct {
	scopedType            string
	condition             *goExpression
	returned              *goExpression
	returns               []*goExpression
	thenBlock             *goBlock
	elseBlock             *goBlock
	localName             string
	localType             string
	local                 int
	initializer           *goExpression
	mutable               bool
	assignment            *goExpression
	loopBlock             *goBlock
	whenBlock             *goBlock
	effect                string
	effectArgs            []*goExpression
	evaluated             *goExpression
	deferred              *goExpression
	nativeFieldReceiver   *goExpression
	nativeFieldName       string
	nativeFieldValue      *goExpression
	nativeIndexCollection *goExpression
	nativeIndex           *goExpression
	nativeIndexValue      *goExpression
	nativeBindingName     string
	nativeBindingValue    *goExpression
	nativePointer         *goExpression
	nativePointerValue    *goExpression
	nativeSwitchSubject   *goExpression
	nativeSwitchCases     []goSwitchCase
	nativeSwitchDefault   *goBlock
	nativeBranch          string
	nativeRange           *goExpression
	nativeRangeKey        *goRangeBinding
	nativeRangeValue      *goRangeBinding
	nativeRangeBody       *goBlock
}

type goRangeBinding struct {
	name           string
	typeID         string
	local          int
	nativeSpelling string
}

type goSwitchCase struct {
	values []*goExpression
	body   *goBlock
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
	if signature.Results().Len() == 0 && len(statements) == 0 {
		return &goBlock{statements: []*goStatement{{returned: &goExpression{kind: goUnitValue}}}}, nil
	}
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
			if increment, ok := node.(*ast.IncDecStmt); ok && goExecutionModuleVersion(functions) >= 87 {
				if name, ok := ast.Unparen(increment.X).(*ast.Ident); ok && info.Uses[name] != nil {
					mutable[info.Uses[name]] = true
				}
				return true
			}
			assignment, ok := node.(*ast.AssignStmt)
			compound := false
			if ok && goExecutionModuleVersion(functions) >= 75 {
				switch assignment.Tok {
				case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN, token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN:
					compound = true
				}
			}
			mixedDefinition := ok && goExecutionModuleVersion(functions) >= 78 && assignment.Tok == token.DEFINE
			if !ok || assignment.Tok != token.ASSIGN && !compound && !mixedDefinition {
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
	inherited := map[types.Object]int{}
	prefix := []*goStatement{}
	if goExecutionModuleVersion(functions) >= 64 {
		for index := 0; index < signature.Params().Len(); index++ {
			parameter := signature.Params().At(index)
			if !mutable[parameter] {
				continue
			}
			localType, ok := goLocalSemanticType(parameter.Type(), records)
			if !ok {
				continue
			}
			local := next
			next++
			name := parameter.Name()
			if name == "" {
				name = "parameter_" + strconv.Itoa(index)
			}
			prefix = append(prefix, &goStatement{localName: "seme_mutable_" + name, localType: localType, local: local, initializer: &goExpression{kind: goParameterRead, parameter: index}, mutable: true})
			inherited[parameter] = local
		}
	}
	if goExecutionModuleVersion(functions) >= 85 {
		for index := 0; index < signature.Results().Len(); index++ {
			result := signature.Results().At(index)
			if result.Name() == "" {
				continue
			}
			initializer, err := zeroGoExpression(result.Type(), records, map[string]bool{}, 0)
			localType, typeOK := goLocalSemanticType(result.Type(), records)
			if err != nil || !typeOK {
				spelling, native := goNativeTypeSpelling(result.Type())
				if !native || functions[nil] != "native-default" {
					continue
				}
				localType = stableID("execution", "type", "native", "go", spelling)
				initializer = &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: localType, nativeTypes: map[string]string{localType: spelling}}
			}
			local := next
			next++
			prefix = append(prefix, &goStatement{localName: result.Name(), localType: localType, local: local, initializer: initializer, mutable: true})
			inherited[result] = local
			mutable[result] = true
		}
	}
	block, err := analyzeGoBlockScoped(statements, signature, info, inherited, functions, records, mutable, &next, true)
	if err != nil {
		return nil, err
	}
	block.statements = append(prefix, block.statements...)
	return block, nil
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
	if literal, ok := ast.Unparen(node).(*ast.CompositeLit); ok {
		if _, _, result := goResultTypes(info.TypeOf(literal)); result {
			fields, err := keyedCompositeFields(literal)
			if err != nil {
				return nil, false
			}
			tag, tagOK := fields["Ok"].(*ast.Ident)
			selected, kind := "Error", goResultError
			if tagOK && tag.Name == "true" {
				selected, kind = "Value", goResultOk
			} else if !tagOK || tag.Name != "false" {
				return nil, false
			}
			valueNode, exists := fields[selected]
			if !exists || len(fields) != 2 {
				return nil, false
			}
			value, ok := analyzeTaggedPayloadExpression(valueNode, source, field, binding, signature, info, functions, records)
			if !ok {
				return nil, false
			}
			return &goExpression{kind: kind, typeID: goSemanticTypeIdentity(info.TypeOf(literal)), left: value}, true
		}
	}
	value, err := analyzeGoExpressionWithProgram(node, signature, info, nil, functions, records, nil)
	return value, err == nil
}

func analyzeTaggedPayloadExpression(node ast.Expr, source *ast.Ident, field, binding string, signature *types.Signature, info *types.Info, functions map[types.Object]string, records map[*types.Named]goRecordInfo) (*goExpression, bool) {
	if selector, ok := ast.Unparen(node).(*ast.SelectorExpr); ok {
		base, baseOK := ast.Unparen(selector.X).(*ast.Ident)
		if baseOK && info.Uses[base] == info.Uses[source] && selector.Sel.Name == field {
			return &goExpression{kind: goVariantRead, text: binding}, true
		}
	}
	if binary, ok := ast.Unparen(node).(*ast.BinaryExpr); ok && binary.Op == token.ADD {
		left, leftOK := analyzeTaggedPayloadExpression(binary.X, source, field, binding, signature, info, functions, records)
		right, rightOK := analyzeTaggedPayloadExpression(binary.Y, source, field, binding, signature, info, functions, records)
		if leftOK && rightOK {
			return &goExpression{kind: goIntegerAdd, left: left, right: right}, true
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
	mapType, mapOK := goUnderlying(info, empty).(*types.Map)
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
		case *ast.DeclStmt:
			if functions[nil] != "native-default" {
				return nil, fmt.Errorf("control.unsupported_statement:%T", raw)
			}
			declaration, ok := statement.Decl.(*ast.GenDecl)
			if ok && goExecutionModuleVersion(functions) >= 82 && declaration.Tok == token.CONST {
				for _, rawSpec := range declaration.Specs {
					spec, ok := rawSpec.(*ast.ValueSpec)
					if !ok || len(spec.Names) == 0 {
						return nil, fmt.Errorf("control.local_constant_declaration_shape")
					}
					for _, name := range spec.Names {
						if name.Name == "_" {
							continue
						}
						if _, ok := info.Defs[name].(*types.Const); !ok {
							return nil, fmt.Errorf("control.local_constant_declaration_binding")
						}
					}
				}
				continue
			}
			if ok && goExecutionModuleVersion(functions) >= 81 && declaration.Tok == token.TYPE {
				for _, rawSpec := range declaration.Specs {
					spec, ok := rawSpec.(*ast.TypeSpec)
					if !ok || spec.Assign.IsValid() {
						return nil, fmt.Errorf("control.local_type_declaration_shape")
					}
					object, ok := info.Defs[spec.Name].(*types.TypeName)
					if !ok {
						return nil, fmt.Errorf("control.local_type_declaration_binding")
					}
					named, ok := types.Unalias(object.Type()).(*types.Named)
					if !ok {
						return nil, fmt.Errorf("control.local_type_declaration_type")
					}
					record, ok := findGoRecord(records, named)
					if !ok {
						return nil, fmt.Errorf("control.local_type_declaration_type")
					}
					block.statements = append(block.statements, &goStatement{scopedType: record.id})
				}
				continue
			}
			if !ok || declaration.Tok != token.VAR {
				return nil, fmt.Errorf("control.local_declaration_shape")
			}
			for _, rawSpec := range declaration.Specs {
				spec, ok := rawSpec.(*ast.ValueSpec)
				if !ok || len(spec.Names) == 0 || len(spec.Values) != 0 && len(spec.Values) != len(spec.Names) {
					return nil, fmt.Errorf("control.local_declaration_shape")
				}
				for valueIndex, name := range spec.Names {
					if name.Name == "_" || info.Defs[name] == nil {
						return nil, fmt.Errorf("control.local_declaration_binding")
					}
					object := info.Defs[name]
					var initializer *goExpression
					var err error
					if len(spec.Values) != 0 {
						initializer, err = analyzeGoExpressionExpected(spec.Values[valueIndex], object.Type(), signature, info, locals, functions, records, mutable)
					} else {
						initializer, err = zeroGoExpression(object.Type(), records, map[string]bool{}, 0)
						if err != nil {
							spelling, native := goNativeTypeSpelling(object.Type())
							if !native {
								return nil, fmt.Errorf("control.local_declaration_type")
							}
							typeID := stableID("execution", "type", "native", "go", spelling)
							initializer = &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: typeID, nativeTypes: map[string]string{typeID: spelling}}
							err = nil
						}
					}
					if err != nil {
						return nil, err
					}
					localType, typeOK := goLocalSemanticType(object.Type(), records)
					if !typeOK && initializer.kind == goNativeDefaultValue {
						localType, typeOK = initializer.nativeResultType, true
					}
					if !typeOK && (initializer.kind == goNativeInvocation || initializer.kind == goNativeMethodInvocation) {
						localType = initializer.nativeResultType
						typeOK = localType != ""
					}
					if !typeOK && functions[nil] == "native-default" {
						initializer, localType, typeOK = nativeGoAssignmentValue(initializer, object.Type())
					}
					if !typeOK {
						return nil, fmt.Errorf("control.local_declaration_type")
					}
					local := *next
					*next++
					block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializer, mutable: mutable[object]})
					locals[object] = local
				}
			}
		case *ast.AssignStmt:
			if goExecutionModuleVersion(functions) >= 95 && functions[nil] == "native-default" && statement.Tok == token.ASSIGN && len(statement.Lhs) == 1 && len(statement.Rhs) == 1 {
				if dereference, ok := ast.Unparen(statement.Lhs[0]).(*ast.StarExpr); ok {
					pointer, err := analyzeGoExpressionWithProgram(dereference.X, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					value, err := analyzeGoExpressionExpected(statement.Rhs[0], info.TypeOf(dereference), signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					block.statements = append(block.statements, &goStatement{nativePointer: pointer, nativePointerValue: value})
					continue
				}
			}
			if goExecutionModuleVersion(functions) >= 94 && functions[nil] == "native-default" && statement.Tok == token.ASSIGN && len(statement.Lhs) == 1 && len(statement.Rhs) == 1 {
				if name, ok := ast.Unparen(statement.Lhs[0]).(*ast.Ident); ok {
					if object, ok := info.Uses[name].(*types.Var); ok && object.Pkg() != nil && object.Parent() == object.Pkg().Scope() {
						value, err := analyzeGoExpressionExpected(statement.Rhs[0], object.Type(), signature, info, locals, functions, records, mutable)
						if err != nil {
							return nil, err
						}
						block.statements = append(block.statements, &goStatement{nativeBindingName: object.Name(), nativeBindingValue: value})
						continue
					}
				}
			}
			if goExecutionModuleVersion(functions) >= 74 && statement.Tok == token.ASSIGN && len(statement.Lhs) == 1 && len(statement.Rhs) == 1 {
				if indexed, ok := ast.Unparen(statement.Lhs[0]).(*ast.IndexExpr); ok {
					collection, err := analyzeGoExpressionWithProgram(indexed.X, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					index, err := analyzeGoExpressionWithProgram(indexed.Index, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					value, err := analyzeGoExpressionExpected(statement.Rhs[0], info.TypeOf(indexed), signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					block.statements = append(block.statements, &goStatement{nativeIndexCollection: collection, nativeIndex: index, nativeIndexValue: value})
					continue
				}
			}
			if goExecutionModuleVersion(functions) >= 63 && statement.Tok == token.ASSIGN && len(statement.Lhs) == 1 && len(statement.Rhs) == 1 {
				if selector, ok := ast.Unparen(statement.Lhs[0]).(*ast.SelectorExpr); ok {
					field, fieldOK := info.Uses[selector.Sel].(*types.Var)
					if !fieldOK || !field.IsField() {
						return nil, fmt.Errorf("control.native_field_assignment_target")
					}
					receiver, err := analyzeGoExpressionWithProgram(selector.X, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					value, err := analyzeGoExpressionExpected(statement.Rhs[0], field.Type(), signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					block.statements = append(block.statements, &goStatement{nativeFieldReceiver: receiver, nativeFieldName: selector.Sel.Name, nativeFieldValue: value})
					continue
				}
			}
			// A multi-result call is one product-valued evaluation. Bind that product
			// once, then project each Go binding from it in source order.
			if (statement.Tok == token.DEFINE || goExecutionModuleVersion(functions) >= 62 && statement.Tok == token.ASSIGN) && len(statement.Lhs) > 1 && len(statement.Rhs) == 1 {
				_, callOK := ast.Unparen(statement.Rhs[0]).(*ast.CallExpr)
				_, assertionOK := ast.Unparen(statement.Rhs[0]).(*ast.TypeAssertExpr)
				if tuple, ok := types.Unalias(info.TypeOf(statement.Rhs[0])).(*types.Tuple); (callOK || assertionOK) && ok && tuple.Len() == len(statement.Lhs) {
					value, err := analyzeGoExpressionWithProgram(statement.Rhs[0], signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					productType, productTypes, supported := goProductTypeID(tuple, records)
					if !supported && goExecutionModuleVersion(functions) >= 76 && functions[nil] == "native-default" {
						productType, productTypes, _, supported = goProductTypeIDWithNative(tuple, records, "", true)
					}
					if !supported && (value.kind == goNativeInvocation || value.kind == goNativeMethodInvocation) && len(value.nativeResultTypes) == tuple.Len() {
						productType, productTypes, supported = value.nativeResultType, value.nativeResultTypes, true
					}
					if !supported {
						return nil, fmt.Errorf("control.multi_result_type")
					}
					allBlank := statement.Tok == token.ASSIGN
					for _, target := range statement.Lhs {
						name, nameOK := target.(*ast.Ident)
						allBlank = allBlank && nameOK && name.Name == "_"
					}
					if allBlank {
						block.statements = append(block.statements, &goStatement{evaluated: value})
						continue
					}
					productLocal := *next
					*next++
					block.statements = append(block.statements, &goStatement{localName: fmt.Sprintf("seme_product_%d", productLocal), localType: productType, local: productLocal, initializer: value})
					for resultIndex, target := range statement.Lhs {
						name, nameOK := target.(*ast.Ident)
						if !nameOK {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						if name.Name == "_" {
							continue
						}
						object := info.Defs[name]
						reassign := statement.Tok == token.ASSIGN
						if reassign || goExecutionModuleVersion(functions) >= 78 && object == nil {
							object = info.Uses[name]
							reassign = true
						}
						if object == nil {
							return nil, fmt.Errorf("control.local_binding_type")
						}
						localType, typeOK := goLocalSemanticType(object.Type(), records)
						if !typeOK && resultIndex < len(productTypes) && (goExecutionModuleVersion(functions) >= 76 || value.kind == goNativeInvocation || value.kind == goNativeMethodInvocation) {
							localType, typeOK = productTypes[resultIndex], true
						}
						if !typeOK {
							return nil, fmt.Errorf("control.local_binding_type")
						}
						projected := &goExpression{kind: goProductProject, left: &goExpression{kind: goLocalRead, local: productLocal}, typeID: productType, elementTypeID: goSemanticTypeIdentity(tuple.At(resultIndex).Type()), productIndex: uint64(resultIndex), productTypes: productTypes}
						if reassign {
							local, exists := locals[object]
							if !exists || !mutable[object] {
								return nil, fmt.Errorf("control.assignment_target")
							}
							block.statements = append(block.statements, &goStatement{local: local, mutable: true, assignment: projected})
						} else {
							local := *next
							*next++
							block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: projected, mutable: mutable[object]})
							locals[object] = local
						}
					}
					continue
				}
			}
			// Go's two-result map lookup is one semantic operation. Normalize its
			// value and presence results into ordinary immutable bindings backed by
			// MapLookupOption, so later stages never need Go's tuple convention.
			if statement.Tok == token.DEFINE && len(statement.Lhs) == 2 && len(statement.Rhs) == 1 {
				lookup, ok := ast.Unparen(statement.Rhs[0]).(*ast.IndexExpr)
				if !ok {
					return nil, fmt.Errorf("control.multi_binding_shape")
				}
				mapping, mapOK := goUnderlying(info, lookup.X).(*types.Map)
				if !mapOK {
					return nil, fmt.Errorf("control.multi_binding_shape")
				}
				if goExecutionModuleVersion(functions) >= 86 && (!isInt64(mapping.Key()) || !isInt64(mapping.Elem())) {
					mapExpression, err := analyzeGoExpressionWithProgram(lookup.X, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					mapExpression, mapTypeID, mapOK := nativeGoAssignmentValue(mapExpression, info.TypeOf(lookup.X))
					keyExpression, keyTypeID, keySpelling, err := analyzeNativeGoOperand(lookup.Index, mapping.Key(), signature, info, locals, functions, records, mutable)
					if err != nil || !mapOK {
						return nil, fmt.Errorf("control.multi_binding_shape")
					}
					boolType := types.Universe.Lookup("bool").Type()
					tuple := types.NewTuple(types.NewVar(token.NoPos, nil, "value", mapping.Elem()), types.NewVar(token.NoPos, nil, "present", boolType))
					productType, productTypes, nativeResultTypes, supported := goProductTypeIDWithNative(tuple, records, "", true)
					if !supported {
						return nil, fmt.Errorf("control.multi_binding_shape")
					}
					mapSpelling, _ := goNativeTypeSpelling(info.TypeOf(lookup.X))
					elementSpelling := types.TypeString(mapping.Elem(), nil)
					nativeTypes := map[string]string{mapTypeID: mapSpelling, keyTypeID: keySpelling}
					for _, nativeType := range nativeResultTypes {
						id, ok := goNativeTypeID(nativeType)
						spelling, native := goNativeTypeSpelling(nativeType)
						if ok && native {
							nativeTypes[id] = spelling
						}
					}
					value := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{mapExpression, keyExpression}, nativeLanguage: "go", nativeTarget: "builtin.map_lookup[" + mapSpelling + "]", nativeSignature: "func(" + mapSpelling + ", " + keySpelling + ") (" + elementSpelling + ", bool)", nativeResultType: productType, nativeResultTypes: productTypes, nativeTypes: nativeTypes}
					productLocal := *next
					*next++
					block.statements = append(block.statements, &goStatement{localName: fmt.Sprintf("seme_product_%d", productLocal), localType: productType, local: productLocal, initializer: value})
					for resultIndex, target := range statement.Lhs {
						name, nameOK := target.(*ast.Ident)
						if !nameOK {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						if name.Name == "_" {
							continue
						}
						object := info.Defs[name]
						if object == nil {
							return nil, fmt.Errorf("control.local_binding_type")
						}
						local := *next
						*next++
						projected := &goExpression{kind: goProductProject, left: &goExpression{kind: goLocalRead, local: productLocal}, typeID: productType, elementTypeID: productTypes[resultIndex], productIndex: uint64(resultIndex), productTypes: productTypes}
						block.statements = append(block.statements, &goStatement{localName: name.Name, localType: productTypes[resultIndex], local: local, initializer: projected, mutable: mutable[object]})
						locals[object] = local
					}
					continue
				}
				if !isInt64(mapping.Key()) || !isInt64(mapping.Elem()) {
					return nil, fmt.Errorf("control.multi_binding_shape")
				}
				mapExpression, err := analyzeGoExpressionWithProgram(lookup.X, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				keyExpression, err := analyzeGoExpressionWithProgram(lookup.Index, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				optionType := stableID("execution", "type", "option", goSemanticTypeIdentity(mapping.Elem()))
				newLookup := func() *goExpression {
					return &goExpression{kind: goMapLookupOption, left: mapExpression, right: keyExpression, typeID: optionType}
				}
				for resultIndex, target := range statement.Lhs {
					name, nameOK := target.(*ast.Ident)
					if !nameOK || name.Name == "_" {
						return nil, fmt.Errorf("control.multi_binding_target")
					}
					object := info.Defs[name]
					if object == nil {
						return nil, fmt.Errorf("control.local_binding_type")
					}
					local := *next
					*next++
					var initializer *goExpression
					localType := "i64"
					if resultIndex == 0 {
						initializer = &goExpression{kind: goOptionMatch, left: newLookup(), initial: &goExpression{kind: goIntegerLiteral}, body: &goExpression{kind: goVariantRead}, text: "value", typeID: goSemanticTypeIdentity(mapping.Elem())}
					} else {
						localType = "bool"
						initializer = &goExpression{kind: goOptionMatch, left: newLookup(), initial: &goExpression{kind: goBooleanLiteral}, body: &goExpression{kind: goBooleanLiteral, boolean: true}, text: "value", typeID: goSemanticTypeIdentity(mapping.Elem())}
					}
					block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializer, mutable: mutable[object]})
					locals[object] = local
				}
				continue
			}
			// Parallel ordinary assignments are one Go evaluation step: analyze every
			// right-hand side against the pre-assignment scope, then publish or update
			// the bindings in source order. This is general tuple-free Go syntax, not
			// Go's special multi-result convention.
			if len(statement.Lhs) > 1 && len(statement.Lhs) == len(statement.Rhs) && (statement.Tok == token.DEFINE || statement.Tok == token.ASSIGN) {
				receiverLocals := make([]int, len(statement.Lhs))
				for i := range receiverLocals {
					receiverLocals[i] = -1
				}
				if statement.Tok == token.ASSIGN && goExecutionModuleVersion(functions) >= 90 {
					// Go evaluates every field receiver before any RHS. Materialize
					// them first so later assignments cannot change which object an
					// earlier selector denoted.
					for i, target := range statement.Lhs {
						selector, selectorOK := ast.Unparen(target).(*ast.SelectorExpr)
						if !selectorOK {
							continue
						}
						field, fieldOK := info.Uses[selector.Sel].(*types.Var)
						if !fieldOK || !field.IsField() {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						receiver, err := analyzeGoExpressionWithProgram(selector.X, signature, info, locals, functions, records, mutable)
						if err != nil {
							return nil, err
						}
						receiverType := info.TypeOf(selector.X)
						localType, ok := goLocalSemanticType(receiverType, records)
						if !ok && functions[nil] == "native-default" {
							receiver, localType, ok = nativeGoAssignmentValue(receiver, receiverType)
						}
						if !ok {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						temporary := *next
						*next++
						block.statements = append(block.statements, &goStatement{localName: fmt.Sprintf("seme_parallel_receiver_%d", i), localType: localType, local: temporary, initializer: receiver})
						receiverLocals[i] = temporary
					}
				}
				initializers := make([]*goExpression, len(statement.Rhs))
				for i, rhs := range statement.Rhs {
					var expected types.Type
					switch target := ast.Unparen(statement.Lhs[i]).(type) {
					case *ast.Ident:
						object := info.Defs[target]
						if statement.Tok == token.ASSIGN {
							object = info.Uses[target]
						}
						if object != nil {
							expected = object.Type()
						}
					case *ast.SelectorExpr:
						if field, ok := info.Uses[target.Sel].(*types.Var); ok && field.IsField() {
							expected = field.Type()
						}
					}
					value, err := analyzeGoExpressionExpected(rhs, expected, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					initializers[i] = value
				}
				if statement.Tok == token.ASSIGN {
					for i, target := range statement.Lhs {
						name, ok := ast.Unparen(target).(*ast.Ident)
						if !ok && receiverLocals[i] < 0 {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						if !ok {
							field := info.Uses[ast.Unparen(target).(*ast.SelectorExpr).Sel].(*types.Var)
							localType, typeOK := goLocalSemanticType(field.Type(), records)
							if !typeOK && functions[nil] == "native-default" {
								initializers[i], localType, typeOK = nativeGoAssignmentValue(initializers[i], field.Type())
							}
							if !typeOK {
								return nil, fmt.Errorf("control.local_binding_type")
							}
							temporary := *next
							*next++
							block.statements = append(block.statements, &goStatement{localName: fmt.Sprintf("seme_parallel_%d", i), localType: localType, local: temporary, initializer: initializers[i]})
							initializers[i] = &goExpression{kind: goLocalRead, local: temporary}
							continue
						}
						if name.Name == "_" {
							if goExecutionModuleVersion(functions) < 62 {
								return nil, fmt.Errorf("control.multi_binding_target")
							}
							block.statements = append(block.statements, &goStatement{evaluated: initializers[i]})
							continue
						}
						object := info.Uses[name]
						if object == nil {
							return nil, fmt.Errorf("control.local_binding_type")
						}
						localType, ok := goLocalSemanticType(object.Type(), records)
						if !ok && (initializers[i].kind == goNativeInvocation || initializers[i].kind == goNativeMethodInvocation) {
							localType = initializers[i].nativeResultType
							ok = localType != ""
						}
						if !ok {
							if functions[nil] == "native-default" {
								initializers[i], localType, ok = nativeGoAssignmentValue(initializers[i], object.Type())
							}
							if !ok {
								return nil, fmt.Errorf("control.local_binding_type")
							}
						}
						temporary := *next
						*next++
						block.statements = append(block.statements, &goStatement{localName: fmt.Sprintf("seme_parallel_%d", i), localType: localType, local: temporary, initializer: initializers[i]})
						initializers[i] = &goExpression{kind: goLocalRead, local: temporary}
					}
				}
				for i, target := range statement.Lhs {
					if receiverLocals[i] >= 0 {
						selector := ast.Unparen(target).(*ast.SelectorExpr)
						block.statements = append(block.statements, &goStatement{nativeFieldReceiver: &goExpression{kind: goLocalRead, local: receiverLocals[i]}, nativeFieldName: selector.Sel.Name, nativeFieldValue: initializers[i]})
						continue
					}
					name, ok := ast.Unparen(target).(*ast.Ident)
					if !ok {
						return nil, fmt.Errorf("control.multi_binding_target")
					}
					if name.Name == "_" {
						if goExecutionModuleVersion(functions) < 62 {
							return nil, fmt.Errorf("control.multi_binding_target")
						}
						if statement.Tok == token.DEFINE {
							block.statements = append(block.statements, &goStatement{evaluated: initializers[i]})
						}
						continue
					}
					object := info.Defs[name]
					if statement.Tok == token.ASSIGN {
						object = info.Uses[name]
					}
					if object == nil {
						return nil, fmt.Errorf("control.local_binding_type")
					}
					localType, ok := goLocalSemanticType(object.Type(), records)
					if !ok && (initializers[i].kind == goNativeInvocation || initializers[i].kind == goNativeMethodInvocation) {
						localType = initializers[i].nativeResultType
						ok = localType != ""
					}
					if !ok {
						if functions[nil] == "native-default" {
							initializers[i], localType, ok = nativeGoAssignmentValue(initializers[i], object.Type())
						}
						if !ok {
							return nil, fmt.Errorf("control.local_binding_type")
						}
					}
					if statement.Tok == token.ASSIGN {
						local, exists := locals[object]
						if !exists || !mutable[object] {
							return nil, fmt.Errorf("control.assignment_target")
						}
						block.statements = append(block.statements, &goStatement{local: local, mutable: true, assignment: initializers[i]})
						continue
					}
					local := *next
					*next++
					block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializers[i], mutable: mutable[object]})
					locals[object] = local
				}
				continue
			}
			if goExecutionModuleVersion(functions) >= 59 && statement.Tok != token.DEFINE && statement.Tok != token.ASSIGN && len(statement.Lhs) == 1 && len(statement.Rhs) == 1 {
				name, nameOK := statement.Lhs[0].(*ast.Ident)
				object := info.Uses[name]
				local, exists := locals[object]
				if !nameOK || object == nil || !exists || !mutable[object] {
					return nil, fmt.Errorf("control.local_binding_shape")
				}
				right, err := analyzeGoExpressionWithProgram(statement.Rhs[0], signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				left := &goExpression{kind: goPlaceRead, local: local}
				var value *goExpression
				switch statement.Tok {
				case token.ADD_ASSIGN:
					if isInt64(object.Type()) {
						value = &goExpression{kind: goIntegerAdd, left: left, right: right}
					} else if isPureString(object.Type()) {
						value = &goExpression{kind: goStringConcat, left: left, right: right}
					}
				case token.SUB_ASSIGN:
					if isInt64(object.Type()) {
						value = &goExpression{kind: goIntegerSubtract, left: left, right: right}
					}
				case token.MUL_ASSIGN:
					if isInt64(object.Type()) {
						value = &goExpression{kind: goIntegerMultiply, left: left, right: right}
					}
				}
				if value == nil && functions[nil] == "native-default" {
					spelling, native := goNativeTypeSpelling(object.Type())
					resultID, idOK := goNativeTypeID(object.Type())
					if native && idOK {
						value = &goExpression{kind: goNativeInvocation, arguments: []*goExpression{left, right}, nativeLanguage: "go", nativeTarget: "builtin.compound[" + statement.Tok.String() + ";" + spelling + "]", nativeSignature: "func(" + spelling + ", ...) " + spelling, nativeResultType: resultID, nativeTypes: map[string]string{resultID: spelling}}
					}
				}
				if value == nil {
					return nil, fmt.Errorf("control.local_binding_shape")
				}
				block.statements = append(block.statements, &goStatement{local: local, mutable: true, assignment: value})
				continue
			}
			if (statement.Tok != token.DEFINE && statement.Tok != token.ASSIGN) || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
				return nil, fmt.Errorf("control.local_binding_shape:%s:%d:%d", statement.Tok, len(statement.Lhs), len(statement.Rhs))
			}
			name, ok := statement.Lhs[0].(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("control.local_binding_shape:%T", statement.Lhs[0])
			}
			if statement.Tok == token.ASSIGN && name.Name == "_" {
				evaluated, err := analyzeGoExpressionWithProgram(statement.Rhs[0], signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				block.statements = append(block.statements, &goStatement{evaluated: evaluated})
				continue
			}
			object := info.Defs[name]
			if statement.Tok == token.ASSIGN {
				object = info.Uses[name]
			}
			if object == nil {
				return nil, fmt.Errorf("control.local_binding_type")
			}
			initializer, err := analyzeGoExpressionExpected(statement.Rhs[0], object.Type(), signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			localType, typeOK := goLocalSemanticType(object.Type(), records)
			if !typeOK && (initializer.kind == goNativeInvocation || initializer.kind == goNativeMethodInvocation) {
				localType = initializer.nativeResultType
				typeOK = localType != ""
			}
			if !typeOK && functions[nil] == "native-default" {
				initializer, localType, typeOK = nativeGoAssignmentValue(initializer, object.Type())
			}
			if !typeOK {
				return nil, fmt.Errorf("control.local_binding_type")
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
			block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializer, mutable: mutable[object]})
			locals[object] = local
		case *ast.IncDecStmt:
			if goExecutionModuleVersion(functions) < 87 {
				return nil, fmt.Errorf("control.unsupported_statement:%T", raw)
			}
			name, ok := ast.Unparen(statement.X).(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("control.incdec_target")
			}
			object := info.Uses[name]
			local, exists := locals[object]
			if object == nil || !exists || !mutable[object] {
				return nil, fmt.Errorf("control.incdec_target")
			}
			left := &goExpression{kind: goPlaceRead, local: local}
			var value *goExpression
			if isInt64(object.Type()) {
				kind := goIntegerAdd
				if statement.Tok == token.DEC {
					kind = goIntegerSubtract
				} else if statement.Tok != token.INC {
					return nil, fmt.Errorf("control.incdec_operator")
				}
				value = &goExpression{kind: kind, left: left, right: &goExpression{kind: goIntegerLiteral, integer: 1}}
			} else {
				spelling, native := goNativeTypeSpelling(object.Type())
				resultID, idOK := goNativeTypeID(object.Type())
				if !native || !idOK || functions[nil] != "native-default" || statement.Tok != token.INC && statement.Tok != token.DEC {
					return nil, fmt.Errorf("control.incdec_type")
				}
				op := "+="
				if statement.Tok == token.DEC {
					op = "-="
				}
				value = &goExpression{kind: goNativeInvocation, arguments: []*goExpression{left, &goExpression{kind: goIntegerLiteral, integer: 1}}, nativeLanguage: "go", nativeTarget: "builtin.compound[" + op + ";" + spelling + "]", nativeSignature: "func(" + spelling + ", " + spelling + ") " + spelling, nativeResultType: resultID, nativeTypes: map[string]string{resultID: spelling}}
			}
			block.statements = append(block.statements, &goStatement{local: local, mutable: true, assignment: value})
		case *ast.ReturnStmt:
			forwardedProduct := len(statement.Results) == 1 && signature.Results().Len() > 1
			if forwardedProduct {
				tuple, ok := types.Unalias(info.TypeOf(statement.Results[0])).(*types.Tuple)
				forwardedProduct = ok && tuple.Len() == signature.Results().Len()
			}
			if index != len(statements)-1 || len(statement.Results) != signature.Results().Len() && !forwardedProduct {
				return nil, fmt.Errorf("control.return_arity")
			}
			if len(statement.Results) == 0 {
				block.statements = append(block.statements, &goStatement{returned: &goExpression{kind: goUnitValue}})
				continue
			}
			values := make([]*goExpression, len(statement.Results))
			for resultIndex, result := range statement.Results {
				var expression *goExpression
				var err error
				if identifier, nilResult := ast.Unparen(result).(*ast.Ident); nilResult && identifier.Name == "nil" && !forwardedProduct {
					expected := signature.Results().At(resultIndex).Type()
					expression, err = zeroGoExpression(expected, records, map[string]bool{}, 0)
					if err != nil && functions[nil] == "native-default" {
						if spelling, native := goNativeTypeSpelling(expected); native {
							typeID := stableID("execution", "type", "native", "go", spelling)
							expression = &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: typeID, nativeTypes: map[string]string{typeID: spelling}}
							err = nil
						}
					}
				} else {
					expression, err = analyzeGoExpressionWithProgram(result, signature, info, locals, functions, records, mutable)
				}
				if err != nil {
					return nil, err
				}
				values[resultIndex] = expression
			}
			if len(values) == 1 {
				block.statements = append(block.statements, &goStatement{returned: values[0]})
			} else {
				block.statements = append(block.statements, &goStatement{returns: values})
			}
		case *ast.SwitchStmt:
			if goExecutionModuleVersion(functions) < 65 || statement.Init != nil {
				return nil, fmt.Errorf("control.unsupported_statement:%T", raw)
			}
			subject := &goExpression{kind: goUnitValue}
			if statement.Tag != nil {
				var err error
				subject, err = analyzeGoExpressionWithProgram(statement.Tag, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
			}
			cases := []goSwitchCase{}
			var defaultBlock *goBlock
			for _, rawClause := range statement.Body.List {
				clause, ok := rawClause.(*ast.CaseClause)
				if !ok {
					return nil, fmt.Errorf("control.switch_clause")
				}
				bodyStatements := clause.Body
				if len(bodyStatements) > 0 {
					if branch, ok := bodyStatements[len(bodyStatements)-1].(*ast.BranchStmt); ok {
						if branch.Tok != token.BREAK || branch.Label != nil {
							return nil, fmt.Errorf("control.switch_branch:%s", branch.Tok)
						}
						bodyStatements = bodyStatements[:len(bodyStatements)-1]
					}
				}
				body, err := analyzeGoBlockScoped(bodyStatements, signature, info, locals, functions, records, mutable, next, false)
				if err != nil {
					return nil, err
				}
				if len(clause.List) == 0 {
					if defaultBlock != nil {
						return nil, fmt.Errorf("control.switch_multiple_default")
					}
					defaultBlock = body
					continue
				}
				values := make([]*goExpression, len(clause.List))
				for valueIndex, valueNode := range clause.List {
					value, err := analyzeGoExpressionWithProgram(valueNode, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, err
					}
					values[valueIndex] = value
				}
				cases = append(cases, goSwitchCase{values: values, body: body})
			}
			block.statements = append(block.statements, &goStatement{nativeSwitchSubject: subject, nativeSwitchCases: cases, nativeSwitchDefault: defaultBlock})
		case *ast.BranchStmt:
			if goExecutionModuleVersion(functions) < 66 || statement.Label != nil || statement.Tok != token.BREAK && statement.Tok != token.CONTINUE {
				return nil, fmt.Errorf("control.unsupported_statement:%T", raw)
			}
			block.statements = append(block.statements, &goStatement{nativeBranch: statement.Tok.String()})
		case *ast.IfStmt:
			ifLocals, ifMutable := locals, mutable
			var initializer []*goStatement
			nonterminalGuard := statement.Else == nil && (!blockContainsReturn(statement.Body.List) || goExecutionModuleVersion(functions) >= 80 && !requireReturn)
			if nonterminalGuard || statement.Else != nil && (index < len(statements)-1 || !requireReturn) {
				var err error
				ifLocals, ifMutable, initializer, err = analyzeGoIfInitializer(statement.Init, signature, info, locals, functions, records, mutable, next)
				if err != nil {
					return nil, err
				}
			}
			if statement.Else == nil && (!blockContainsReturn(statement.Body.List) || goExecutionModuleVersion(functions) >= 79 && !requireReturn) {
				condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, ifLocals, functions, records, ifMutable)
				if err != nil {
					return nil, err
				}
				body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, ifLocals, functions, records, ifMutable, next, false)
				if err != nil {
					return nil, err
				}
				block.statements = append(block.statements, initializer...)
				block.statements = append(block.statements, &goStatement{condition: condition, whenBlock: body})
				continue
			}
			if statement.Else != nil && (index < len(statements)-1 || !requireReturn) {
				var alternateStatements []ast.Stmt
				switch alternate := statement.Else.(type) {
				case *ast.BlockStmt:
					alternateStatements = alternate.List
				case *ast.IfStmt:
					if goExecutionModuleVersion(functions) < 84 {
						return nil, fmt.Errorf("control.else_unsupported")
					}
					alternateStatements = []ast.Stmt{alternate}
				default:
					return nil, fmt.Errorf("control.else_unsupported")
				}
				condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, ifLocals, functions, records, ifMutable)
				if err != nil {
					return nil, err
				}
				thenBlock, err := analyzeGoBlockScoped(statement.Body.List, signature, info, ifLocals, functions, records, ifMutable, next, false)
				if err != nil {
					return nil, err
				}
				elseBlock, err := analyzeGoBlockScoped(alternateStatements, signature, info, ifLocals, functions, records, ifMutable, next, false)
				if err != nil {
					return nil, err
				}
				block.statements = append(block.statements, initializer...)
				block.statements = append(block.statements, &goStatement{condition: condition, thenBlock: thenBlock, elseBlock: elseBlock})
				continue
			}
			branch, err := analyzeTerminalIfScoped(statement, statements[index+1:], signature, info, locals, functions, records, mutable, next)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, branch.statements...)
			return block, nil
		case *ast.ForStmt:
			if statement.Cond == nil && goExecutionModuleVersion(functions) < 83 {
				return nil, fmt.Errorf("control.for_shape")
			}
			if statement.Init != nil || statement.Post != nil {
				init, initOK := statement.Init.(*ast.AssignStmt)
				post, postOK := statement.Post.(*ast.IncDecStmt)
				if !initOK || init.Tok != token.DEFINE || len(init.Lhs) != 1 || len(init.Rhs) != 1 || !postOK {
					return nil, fmt.Errorf("control.for_shape")
				}
				name, nameOK := init.Lhs[0].(*ast.Ident)
				postName, postNameOK := post.X.(*ast.Ident)
				if !nameOK || !postNameOK {
					return nil, fmt.Errorf("control.for_binding")
				}
				object := info.Defs[name]
				if info.Uses[postName] != object || object == nil || !isInt64(object.Type()) {
					return nil, fmt.Errorf("control.for_binding")
				}
				initializer, err := analyzeGoExpressionWithProgram(init.Rhs[0], signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				local := *next
				*next++
				block.statements = append(block.statements, &goStatement{localName: name.Name, localType: "i64", local: local, initializer: initializer, mutable: true})
				locals[object] = local
				mutable[object] = true
				condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
				body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, records, mutable, next, false)
				if err != nil {
					return nil, err
				}
				kind := goIntegerAdd
				if post.Tok == token.DEC {
					kind = goIntegerSubtract
				} else if post.Tok != token.INC {
					return nil, fmt.Errorf("control.for_post")
				}
				postStep := func() *goStatement {
					return &goStatement{local: local, mutable: true, assignment: &goExpression{kind: kind, left: &goExpression{kind: goPlaceRead, local: local}, right: &goExpression{kind: goIntegerLiteral, integer: 1}}}
				}
				injectBeforeNativeContinue(body, postStep)
				body.statements = append(body.statements, postStep())
				block.statements = append(block.statements, &goStatement{condition: condition, loopBlock: body})
				continue
			}
			condition := &goExpression{kind: goBooleanLiteral, boolean: true}
			if statement.Cond != nil {
				var err error
				condition, err = analyzeGoExpressionWithProgram(statement.Cond, signature, info, locals, functions, records, mutable)
				if err != nil {
					return nil, err
				}
			}
			body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, records, mutable, next, false)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{condition: condition, loopBlock: body})
		case *ast.RangeStmt:
			if ranged, ok, err := analyzeGoNativeMapRange(statement, signature, info, locals, functions, records, mutable, next); ok || err != nil {
				if err != nil {
					return nil, err
				}
				block.statements = append(block.statements, ranged)
				continue
			}
			rangeStatements, err := analyzeGoIndexedRange(statement, signature, info, locals, functions, records, mutable, next)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, rangeStatements...)
		case *ast.DeferStmt:
			if goExecutionModuleVersion(functions) < 61 {
				return nil, fmt.Errorf("control.unsupported_statement:%T", statement)
			}
			invocation, err := analyzeGoExpressionWithProgram(statement.Call, signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{deferred: invocation})
		case *ast.ExprStmt:
			call, ok := statement.X.(*ast.CallExpr)
			if !ok {
				return nil, fmt.Errorf("control.effect_shape")
			}
			if selector, selectorOK := call.Fun.(*ast.SelectorExpr); selectorOK && len(call.Args) == 1 {
				function, functionOK := info.Uses[selector.Sel].(*types.Func)
				if functionOK && function.Pkg() != nil && function.Pkg().Path() == "log" && function.Name() == "Print" {
					argument, err := analyzeGoExpressionWithProgram(call.Args[0], signature, info, locals, functions, records, mutable)
					if err != nil || !isBool(info.TypeOf(call.Args[0])) {
						return nil, fmt.Errorf("control.effect_argument")
					}
					block.statements = append(block.statements, &goStatement{effect: "observability.log", effectArgs: []*goExpression{argument}})
					continue
				}
			}
			evaluated, err := analyzeGoExpressionWithProgram(statement.X, signature, info, locals, functions, records, mutable)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{evaluated: evaluated})
		default:
			return nil, fmt.Errorf("control.unsupported_statement:%T", raw)
		}
	}
	total := len(block.statements) > 0 && (block.statements[len(block.statements)-1].returned != nil || len(block.statements[len(block.statements)-1].returns) != 0 || nativeSwitchAlwaysReturns(block.statements[len(block.statements)-1]))
	if requireReturn && !total {
		if signature.Results().Len() == 0 {
			block.statements = append(block.statements, &goStatement{returned: &goExpression{kind: goUnitValue}})
		} else {
			return nil, fmt.Errorf("control.block_not_total")
		}
	}
	return block, nil
}

func nativeSwitchAlwaysReturns(statement *goStatement) bool {
	if statement == nil || statement.nativeSwitchSubject == nil || statement.nativeSwitchDefault == nil || !goBlockAlwaysReturns(statement.nativeSwitchDefault) {
		return false
	}
	for _, switchCase := range statement.nativeSwitchCases {
		if !goBlockAlwaysReturns(switchCase.body) {
			return false
		}
	}
	return true
}

func goBlockAlwaysReturns(block *goBlock) bool {
	if block == nil || len(block.statements) == 0 {
		return false
	}
	last := block.statements[len(block.statements)-1]
	return last.returned != nil || len(last.returns) != 0 || nativeSwitchAlwaysReturns(last)
}

func analyzeGoNativeMapRange(statement *ast.RangeStmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int) (*goStatement, bool, error) {
	if goExecutionModuleVersion(functions) < 70 || functions[nil] != "native-default" {
		return nil, false, nil
	}
	var keyType, valueType types.Type
	switch ranged := info.TypeOf(statement.X).Underlying().(type) {
	case *types.Map:
		keyType, valueType = ranged.Key(), ranged.Elem()
	case *types.Basic:
		if goExecutionModuleVersion(functions) < 96 || ranged.Info()&types.IsString == 0 {
			return nil, false, nil
		}
		// Go defines string range as byte-index plus decoded Unicode code point.
		// Keep that mechanic native and retain its exact `int`/`rune` bindings.
		keyType, valueType = types.Typ[types.Int], types.Typ[types.Rune]
	default:
		return nil, false, nil
	}
	if statement.Tok != token.DEFINE {
		return nil, true, fmt.Errorf("control.native_range_assignment")
	}
	collection, err := analyzeGoExpressionWithProgram(statement.X, signature, info, locals, functions, records, mutable)
	if err != nil {
		return nil, true, err
	}
	loopLocals := cloneLocalScope(locals)
	loopMutable := cloneMutableScope(mutable)
	bind := func(node ast.Expr, typ types.Type) (*goRangeBinding, error) {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("control.range_binding")
		}
		if identifier.Name == "_" {
			return nil, nil
		}
		object := info.Defs[identifier]
		if object == nil {
			return nil, fmt.Errorf("control.range_binding")
		}
		typeID, ok := goLocalSemanticType(typ, records)
		spelling := ""
		if ok {
			switch typeID {
			case "i64":
				typeID = stableID("execution", "type", "i64")
			case "bool":
				typeID = stableID("execution", "type", "bool")
			case "string":
				typeID = stableID("execution", "type", "string")
			}
		}
		if !ok {
			typeID, ok = goNativeTypeID(typ)
			spelling, _ = goNativeTypeSpelling(typ)
		}
		if !ok {
			return nil, fmt.Errorf("control.range_element_type")
		}
		local := *next
		*next++
		loopLocals[object] = local
		loopMutable[object] = mutable[object]
		return &goRangeBinding{name: identifier.Name, typeID: typeID, local: local, nativeSpelling: spelling}, nil
	}
	key, err := bind(statement.Key, keyType)
	if err != nil {
		return nil, true, err
	}
	var value *goRangeBinding
	if statement.Value != nil {
		value, err = bind(statement.Value, valueType)
		if err != nil {
			return nil, true, err
		}
	}
	body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, loopLocals, functions, records, loopMutable, next, false)
	if err != nil {
		return nil, true, err
	}
	return &goStatement{nativeRange: collection, nativeRangeKey: key, nativeRangeValue: value, nativeRangeBody: body}, true, nil
}

// analyzeGoIndexedRange preserves Go's array/slice range mechanics while
// expressing the control flow through Seme's ordinary places and while loop.
// The range expression is evaluated exactly once. Its native Go type is retained
// because Go's index is `int` and application-owned element types must not be
// silently recast as neutral Seme values. Map and string range have different
// ordering and rune semantics and deliberately remain native islands here.
func analyzeGoIndexedRange(statement *ast.RangeStmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int) ([]*goStatement, error) {
	if goExecutionModuleVersion(functions) < 59 || functions[nil] != "native-default" || statement.Tok != token.DEFINE {
		return nil, fmt.Errorf("control.unsupported_statement:%T", statement)
	}
	collectionType := info.TypeOf(statement.X)
	underlying := collectionType.Underlying()
	var elementType types.Type
	switch value := underlying.(type) {
	case *types.Array:
		elementType = value.Elem()
	case *types.Slice:
		elementType = value.Elem()
	default:
		return nil, fmt.Errorf("control.unsupported_statement:%T", statement)
	}
	collection, err := analyzeGoExpressionWithProgram(statement.X, signature, info, locals, functions, records, mutable)
	if err != nil {
		return nil, err
	}
	collection, collectionID, ok := nativeGoAssignmentValue(collection, collectionType)
	if !ok {
		return nil, fmt.Errorf("control.range_collection_type")
	}
	collectionSpelling, _ := goNativeTypeSpelling(collectionType)
	intType := types.Universe.Lookup("int").Type()
	intID, _ := goNativeTypeID(intType)
	intSpelling, _ := goNativeTypeSpelling(intType)
	collectionLocal := *next
	*next++
	indexLocal := *next
	*next++
	out := []*goStatement{
		{localName: "seme_range_collection", localType: collectionID, local: collectionLocal, initializer: collection},
		{localName: "seme_range_index", localType: intID, local: indexLocal, initializer: &goExpression{kind: goNativeDefaultValue, nativeLanguage: "go", nativeResultType: intID, nativeTypes: map[string]string{intID: intSpelling}}, mutable: true},
	}
	loopLocals := cloneLocalScope(locals)
	loopMutable := cloneMutableScope(mutable)
	key, keyOK := statement.Key.(*ast.Ident)
	if !keyOK {
		return nil, fmt.Errorf("control.range_binding")
	}
	bodyPrefix := []*goStatement{}
	if key.Name != "_" {
		object := info.Defs[key]
		if object == nil {
			return nil, fmt.Errorf("control.range_binding")
		}
		local := *next
		*next++
		bodyPrefix = append(bodyPrefix, &goStatement{localName: key.Name, localType: intID, local: local, initializer: &goExpression{kind: goLocalRead, local: indexLocal}})
		loopLocals[object] = local
	}
	if statement.Value != nil {
		value, valueOK := statement.Value.(*ast.Ident)
		if !valueOK {
			return nil, fmt.Errorf("control.range_binding")
		}
		if value.Name != "_" {
			object := info.Defs[value]
			if object == nil {
				return nil, fmt.Errorf("control.range_binding")
			}
			resultID, resultOK := goLocalSemanticType(elementType, records)
			nativeTypes := map[string]string{collectionID: collectionSpelling, intID: intSpelling}
			if !resultOK {
				resultID, resultOK = goNativeTypeID(elementType)
				if spelling, native := goNativeTypeSpelling(elementType); native {
					nativeTypes[resultID] = spelling
				}
			}
			if !resultOK {
				return nil, fmt.Errorf("control.range_element_type")
			}
			local := *next
			*next++
			index := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{{kind: goLocalRead, local: collectionLocal}, {kind: goLocalRead, local: indexLocal}}, nativeLanguage: "go", nativeTarget: "builtin.index[" + collectionSpelling + "]", nativeSignature: "func(" + collectionSpelling + ", int) " + types.TypeString(elementType, nil), nativeResultType: resultID, nativeTypes: nativeTypes}
			bodyPrefix = append(bodyPrefix, &goStatement{localName: value.Name, localType: resultID, local: local, initializer: index, mutable: mutable[object]})
			loopLocals[object] = local
			loopMutable[object] = mutable[object]
		}
	}
	body, err := analyzeGoBlockScoped(statement.Body.List, signature, info, loopLocals, functions, records, loopMutable, next, false)
	if err != nil {
		return nil, err
	}
	body.statements = append(bodyPrefix, body.statements...)
	one, _, _ := nativeGoAssignmentValue(&goExpression{kind: goIntegerLiteral, integer: 1}, intType)
	add := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{{kind: goPlaceRead, local: indexLocal}, one}, nativeLanguage: "go", nativeTarget: "builtin.add[int]", nativeSignature: "func(int, int) int", nativeResultType: intID, nativeTypes: map[string]string{intID: intSpelling}}
	postStep := func() *goStatement {
		return &goStatement{local: indexLocal, mutable: true, assignment: add}
	}
	injectBeforeNativeContinue(body, postStep)
	body.statements = append(body.statements, postStep())
	length := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{{kind: goLocalRead, local: collectionLocal}}, nativeLanguage: "go", nativeTarget: "builtin.len[" + collectionSpelling + "]", nativeSignature: "func(" + collectionSpelling + ") int", nativeResultType: intID, nativeTypes: map[string]string{collectionID: collectionSpelling, intID: intSpelling}}
	condition := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{{kind: goPlaceRead, local: indexLocal}, length}, nativeLanguage: "go", nativeTarget: "builtin.compare[<;int;int]", nativeSignature: "func(int, int) bool", nativeResultType: stableID("execution", "type", "bool"), nativeTypes: map[string]string{intID: intSpelling}}
	out = append(out, &goStatement{condition: condition, loopBlock: body})
	return out, nil
}

// injectBeforeNativeContinue preserves source-language loop-post semantics after
// a Go for/range loop is lowered to canonical while control. Nested loop bodies
// are deliberately excluded: their continue statements belong to that loop.
func injectBeforeNativeContinue(block *goBlock, postStep func() *goStatement) {
	if block == nil {
		return
	}
	statements := make([]*goStatement, 0, len(block.statements))
	for _, statement := range block.statements {
		if statement == nil {
			continue
		}
		if statement.nativeBranch == "continue" {
			statements = append(statements, postStep())
		}
		injectBeforeNativeContinue(statement.whenBlock, postStep)
		injectBeforeNativeContinue(statement.thenBlock, postStep)
		injectBeforeNativeContinue(statement.elseBlock, postStep)
		for index := range statement.nativeSwitchCases {
			injectBeforeNativeContinue(statement.nativeSwitchCases[index].body, postStep)
		}
		injectBeforeNativeContinue(statement.nativeSwitchDefault, postStep)
		statements = append(statements, statement)
	}
	block.statements = statements
}

func nativeGoAssignmentValue(value *goExpression, target types.Type) (*goExpression, string, bool) {
	if value == nil || target == nil {
		return value, "", false
	}
	typeID, ok := goNativeTypeID(target)
	if !ok {
		return value, "", false
	}
	spelling, ok := goNativeTypeSpelling(target)
	if !ok {
		return value, "", false
	}
	return &goExpression{
		kind: goNativeInvocation, arguments: []*goExpression{value}, nativeLanguage: "go",
		nativeTarget:    "builtin.assignment_convert[" + spelling + "]",
		nativeSignature: "func(...) " + spelling, nativeResultType: typeID,
		nativeTypes: map[string]string{typeID: spelling},
	}, typeID, true
}

func goLocalSemanticType(t types.Type, records map[*types.Named]goRecordInfo) (string, bool) {
	if tuple, ok := types.Unalias(t).(*types.Tuple); ok {
		id, _, supported := goProductTypeID(tuple, records)
		return id, supported
	}
	if isInt64(t) {
		return "i64", true
	}
	if isBool(t) {
		return "bool", true
	}
	if isPureString(t) {
		return "string", true
	}
	if isBytes(t) {
		return stableID("execution", "type", "bytes"), true
	}
	if isGoErrorType(t) {
		return stableID("execution", "type", "native", "go", "error"), true
	}
	if element, ok := goPrimitiveSliceElement(t); ok {
		tag, _ := goPrimitiveTypeTag(element)
		return stableID("execution", "type", "slice", tag), true
	}
	if isI64Map(t) {
		return stableID("execution", "type", "map", "i64", "i64"), true
	}
	if _, _, ok := goResultTypes(t); ok {
		return goSemanticTypeIdentity(t), true
	}
	if signature, ok := goFunctionSignature(t); ok && isUnaryI64Function(signature) {
		return goFunctionTypeID(signature), true
	}
	if named, ok := types.Unalias(t).(*types.Named); ok {
		if record, exists := findGoRecord(records, named); exists {
			return record.id, true
		}
	}
	return "", false
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

func analyzeGoIfInitializer(initializer ast.Stmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int) (map[types.Object]int, map[types.Object]bool, []*goStatement, error) {
	if initializer == nil {
		return locals, mutable, nil, nil
	}
	assignment, ok := initializer.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE && assignment.Tok != token.ASSIGN || len(assignment.Rhs) != 1 {
		return nil, nil, nil, fmt.Errorf("control.if_init_shape")
	}
	if len(assignment.Lhs) > 1 {
		if goExecutionModuleVersion(functions) >= 97 && assignment.Tok == token.DEFINE && len(assignment.Lhs) == 2 {
			if lookup, lookupOK := ast.Unparen(assignment.Rhs[0]).(*ast.IndexExpr); lookupOK {
				if mapping, mapOK := goUnderlying(info, lookup.X).(*types.Map); mapOK {
					value, productType, productTypes, err := analyzeGoNativeMapLookupProduct(lookup, mapping, signature, info, locals, functions, records, mutable)
					if err != nil {
						return nil, nil, nil, err
					}
					scopedLocals := cloneLocalScope(locals)
					scopedMutable := cloneMutableScope(mutable)
					productLocal := *next
					*next++
					statements := []*goStatement{{localName: fmt.Sprintf("seme_product_%d", productLocal), localType: productType, local: productLocal, initializer: value}}
					for resultIndex, target := range assignment.Lhs {
						name, nameOK := target.(*ast.Ident)
						if !nameOK {
							return nil, nil, nil, fmt.Errorf("control.if_init_binding")
						}
						if name.Name == "_" {
							continue
						}
						object := info.Defs[name]
						if object == nil {
							return nil, nil, nil, fmt.Errorf("control.if_init_binding")
						}
						local := *next
						*next++
						projected := &goExpression{kind: goProductProject, left: &goExpression{kind: goLocalRead, local: productLocal}, typeID: productType, elementTypeID: productTypes[resultIndex], productIndex: uint64(resultIndex), productTypes: productTypes}
						statements = append(statements, &goStatement{localName: name.Name, localType: productTypes[resultIndex], local: local, initializer: projected, mutable: scopedMutable[object]})
						scopedLocals[object] = local
					}
					return scopedLocals, scopedMutable, statements, nil
				}
			}
		}
		_, callOK := ast.Unparen(assignment.Rhs[0]).(*ast.CallExpr)
		_, assertionOK := ast.Unparen(assignment.Rhs[0]).(*ast.TypeAssertExpr)
		tuple, tupleOK := types.Unalias(info.TypeOf(assignment.Rhs[0])).(*types.Tuple)
		if !callOK && !assertionOK || !tupleOK || tuple.Len() != len(assignment.Lhs) {
			return nil, nil, nil, fmt.Errorf("control.if_init_shape")
		}
		value, err := analyzeGoExpressionWithProgram(assignment.Rhs[0], signature, info, locals, functions, records, mutable)
		if err != nil {
			return nil, nil, nil, err
		}
		productType, productTypes, supported := goProductTypeID(tuple, records)
		if !supported && goExecutionModuleVersion(functions) >= 76 && functions[nil] == "native-default" {
			productType, productTypes, _, supported = goProductTypeIDWithNative(tuple, records, "", true)
		}
		if !supported && (value.kind == goNativeInvocation || value.kind == goNativeMethodInvocation) && len(value.nativeResultTypes) == tuple.Len() {
			productType, productTypes, supported = value.nativeResultType, value.nativeResultTypes, true
		}
		if !supported {
			return nil, nil, nil, fmt.Errorf("control.if_init_type")
		}
		scopedLocals := cloneLocalScope(locals)
		scopedMutable := cloneMutableScope(mutable)
		productLocal := *next
		*next++
		statements := []*goStatement{{localName: fmt.Sprintf("seme_product_%d", productLocal), localType: productType, local: productLocal, initializer: value}}
		for resultIndex, target := range assignment.Lhs {
			name, nameOK := target.(*ast.Ident)
			if !nameOK {
				return nil, nil, nil, fmt.Errorf("control.if_init_binding")
			}
			if name.Name == "_" {
				continue
			}
			object := info.Defs[name]
			if assignment.Tok == token.ASSIGN {
				object = info.Uses[name]
			}
			if object == nil {
				return nil, nil, nil, fmt.Errorf("control.if_init_binding")
			}
			localType, typeOK := goLocalSemanticType(object.Type(), records)
			if !typeOK && (goExecutionModuleVersion(functions) >= 76 || value.kind == goNativeInvocation || value.kind == goNativeMethodInvocation) {
				localType, typeOK = productTypes[resultIndex], true
			}
			if !typeOK {
				return nil, nil, nil, fmt.Errorf("control.if_init_type")
			}
			projected := &goExpression{kind: goProductProject, left: &goExpression{kind: goLocalRead, local: productLocal}, typeID: productType, elementTypeID: productTypes[resultIndex], productIndex: uint64(resultIndex), productTypes: productTypes}
			if assignment.Tok == token.ASSIGN {
				local, exists := scopedLocals[object]
				if !exists || !scopedMutable[object] {
					return nil, nil, nil, fmt.Errorf("control.if_init_assignment_target")
				}
				statements = append(statements, &goStatement{local: local, mutable: true, assignment: projected})
			} else {
				local := *next
				*next++
				statements = append(statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: projected, mutable: scopedMutable[object]})
				scopedLocals[object] = local
			}
		}
		return scopedLocals, scopedMutable, statements, nil
	}
	if assignment.Tok != token.DEFINE {
		return nil, nil, nil, fmt.Errorf("control.if_init_shape")
	}
	if len(assignment.Lhs) != 1 {
		return nil, nil, nil, fmt.Errorf("control.if_init_shape")
	}
	name, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || name.Name == "_" {
		return nil, nil, nil, fmt.Errorf("control.if_init_binding")
	}
	object := info.Defs[name]
	if object == nil {
		return nil, nil, nil, fmt.Errorf("control.if_init_binding")
	}
	value, err := analyzeGoExpressionWithProgram(assignment.Rhs[0], signature, info, locals, functions, records, mutable)
	if err != nil {
		return nil, nil, nil, err
	}
	localType, ok := goLocalSemanticType(object.Type(), records)
	if !ok && goExecutionModuleVersion(functions) >= 76 && functions[nil] == "native-default" {
		// The Go type checker has already proved the initializer assignable to
		// this binding. Retain an exact native type without manufacturing a Go
		// conversion (which is invalid for identities such as *T -> *T).
		localType, ok = goNativeTypeID(object.Type())
	}
	if !ok {
		return nil, nil, nil, fmt.Errorf("control.if_init_type")
	}
	scopedLocals := cloneLocalScope(locals)
	scopedMutable := cloneMutableScope(mutable)
	local := *next
	*next++
	scopedLocals[object] = local
	return scopedLocals, scopedMutable, []*goStatement{{localName: name.Name, localType: localType, local: local, initializer: value, mutable: scopedMutable[object]}}, nil
}

func analyzeGoNativeMapLookupProduct(lookup *ast.IndexExpr, mapping *types.Map, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool) (*goExpression, string, []string, error) {
	mapExpression, err := analyzeGoExpressionWithProgram(lookup.X, signature, info, locals, functions, records, mutable)
	if err != nil {
		return nil, "", nil, err
	}
	mapExpression, mapTypeID, mapOK := nativeGoAssignmentValue(mapExpression, info.TypeOf(lookup.X))
	keyExpression, keyTypeID, keySpelling, err := analyzeNativeGoOperand(lookup.Index, mapping.Key(), signature, info, locals, functions, records, mutable)
	if err != nil || !mapOK {
		return nil, "", nil, fmt.Errorf("control.if_init_shape")
	}
	boolType := types.Universe.Lookup("bool").Type()
	tuple := types.NewTuple(types.NewVar(token.NoPos, nil, "value", mapping.Elem()), types.NewVar(token.NoPos, nil, "present", boolType))
	productType, productTypes, nativeResultTypes, supported := goProductTypeIDWithNative(tuple, records, "", true)
	if !supported {
		return nil, "", nil, fmt.Errorf("control.if_init_type")
	}
	mapSpelling, _ := goNativeTypeSpelling(info.TypeOf(lookup.X))
	elementSpelling := types.TypeString(mapping.Elem(), nil)
	nativeTypes := map[string]string{mapTypeID: mapSpelling, keyTypeID: keySpelling}
	for _, nativeType := range nativeResultTypes {
		id, ok := goNativeTypeID(nativeType)
		spelling, native := goNativeTypeSpelling(nativeType)
		if ok && native {
			nativeTypes[id] = spelling
		}
	}
	value := &goExpression{kind: goNativeInvocation, arguments: []*goExpression{mapExpression, keyExpression}, nativeLanguage: "go", nativeTarget: "builtin.map_lookup[" + mapSpelling + "]", nativeSignature: "func(" + mapSpelling + ", " + keySpelling + ") (" + elementSpelling + ", bool)", nativeResultType: productType, nativeResultTypes: productTypes, nativeTypes: nativeTypes}
	return value, productType, productTypes, nil
}

func cloneMutableScope(mutable map[types.Object]bool) map[types.Object]bool {
	cloned := make(map[types.Object]bool, len(mutable))
	for object, value := range mutable {
		cloned[object] = value
	}
	return cloned
}

func analyzeTerminalIf(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	next := 0
	return analyzeTerminalIfScoped(statement, following, signature, info, map[types.Object]int{}, nil, nil, nil, &next)
}

func analyzeTerminalIfScoped(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, records map[*types.Named]goRecordInfo, mutable map[types.Object]bool, next *int) (*goBlock, error) {
	ifLocals, ifMutable, initializer, err := analyzeGoIfInitializer(statement.Init, signature, info, locals, functions, records, mutable, next)
	if err != nil {
		return nil, err
	}
	condition, err := analyzeGoExpressionWithProgram(statement.Cond, signature, info, ifLocals, functions, records, ifMutable)
	if err != nil {
		return nil, err
	}
	thenStatements := statement.Body.List
	if statement.Else == nil && !blockAlwaysReturns(statement.Body.List) {
		thenStatements = append(append([]ast.Stmt(nil), statement.Body.List...), following...)
	}
	thenBlock, err := analyzeGoBlockScoped(thenStatements, signature, info, ifLocals, functions, records, ifMutable, next, true)
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
			elseBlock, err = analyzeGoBlockScoped(alternate.List, signature, info, ifLocals, functions, records, ifMutable, next, true)
		case *ast.IfStmt:
			elseBlock, err = analyzeTerminalIfScoped(alternate, nil, signature, info, ifLocals, functions, records, ifMutable, next)
		default:
			err = fmt.Errorf("control.else_unsupported")
		}
	}
	if err != nil {
		return nil, err
	}
	return &goBlock{statements: append(initializer, &goStatement{condition: condition, thenBlock: thenBlock, elseBlock: elseBlock})}, nil
}

func blockAlwaysReturns(statements []ast.Stmt) bool {
	if len(statements) == 0 {
		return false
	}
	switch last := statements[len(statements)-1].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BlockStmt:
		return blockAlwaysReturns(last.List)
	case *ast.IfStmt:
		if last.Else == nil || !blockAlwaysReturns(last.Body.List) {
			return false
		}
		switch alternate := last.Else.(type) {
		case *ast.BlockStmt:
			return blockAlwaysReturns(alternate.List)
		case *ast.IfStmt:
			return blockAlwaysReturns([]ast.Stmt{alternate})
		}
	}
	return false
}

func emitCanonicalBlock(block *goBlock, owner, path string, parameterIDs []string, integerTypeID string, instances *[]graphEntity) (string, error) {
	return emitCanonicalBlockScoped(block, owner, path, parameterIDs, integerTypeID, map[int]string{}, instances)
}

func emitCanonicalBlockScoped(block *goBlock, owner, path string, parameterIDs []string, integerTypeID string, inherited map[int]string, instances *[]graphEntity) (string, error) {
	if block == nil {
		return "", fmt.Errorf("control.nil_block")
	}
	localIDs := make(map[int]string, len(inherited)+len(block.statements))
	for key, value := range inherited {
		localIDs[key] = value
	}
	statementIDs := make([]string, 0, len(block.statements))
	for index, statement := range block.statements {
		statementPath := path + ".statement." + strconv.Itoa(index)
		statementID := ""
		if statement.scopedType != "" {
			statementID = stableID("execution", owner, statementPath, "scoped-type-declaration")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a081", []graphField{refField(0xa0810, statement.scopedType)})})
		} else if statement.initializer != nil {
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
		} else if len(statement.returns) != 0 {
			values := make([]string, len(statement.returns))
			for valueIndex, value := range statement.returns {
				expressions, expressionID, err := emitCanonicalExpressionWithLocals(value, owner+":"+path+":return:"+strconv.Itoa(valueIndex), parameterIDs, localIDs, integerTypeID)
				if err != nil {
					return "", err
				}
				*instances = append(*instances, expressions...)
				values[valueIndex] = expressionID
			}
			statementID = stableID("execution", owner, statementPath, "return")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "00000000000000000000000000009081", []graphField{refsField(0x9810, values)})})
		} else if statement.nativeRange != nil {
			expressions, collectionID, err := emitCanonicalExpressionWithLocals(statement.nativeRange, owner+":"+statementPath+":native-range-collection", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			bodyLocals := make(map[int]string, len(localIDs)+2)
			for key, value := range localIDs {
				bodyLocals[key] = value
			}
			emitBinding := func(binding *goRangeBinding, role string) []string {
				if binding == nil {
					return nil
				}
				id := stableID("execution", owner, statementPath, "native-range-"+role)
				if binding.nativeSpelling != "" {
					*instances = append(*instances, graphEntity{binding.typeID, entity(binding.typeID, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, binding.nativeSpelling)})})
				}
				*instances = append(*instances, graphEntity{id, entity(id, "0000000000000000000000000000a07b", []graphField{bytesField(0xa07b0, binding.name), refField(0xa07b1, binding.typeID)})})
				bodyLocals[binding.local] = id
				return []string{id}
			}
			keyIDs := emitBinding(statement.nativeRangeKey, "key")
			valueIDs := emitBinding(statement.nativeRangeValue, "value")
			bodyID, err := emitCanonicalBlockScoped(statement.nativeRangeBody, owner, statementPath+".native-range", parameterIDs, integerTypeID, bodyLocals, instances)
			if err != nil {
				return "", err
			}
			statementID = stableID("execution", owner, statementPath, "native-range")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a07c", []graphField{bytesField(0xa07c0, "go"), refField(0xa07c1, collectionID), refsField(0xa07c2, keyIDs), refsField(0xa07c3, valueIDs), refField(0xa07c4, bodyID)})})
		} else if statement.loopBlock != nil {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+statementPath+":loop-condition", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			bodyID, err := emitCanonicalBlockScoped(statement.loopBlock, owner, statementPath+".loop", parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			statementID = stableID("execution", owner, statementPath, "while")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090e4", []graphField{refField(0x9e40, conditionID), refField(0x9e41, bodyID)})})
		} else if statement.whenBlock != nil {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+statementPath+":condition", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			bodyID, err := emitCanonicalBlockScoped(statement.whenBlock, owner, statementPath+".when", parameterIDs, integerTypeID, localIDs, instances)
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
		} else if statement.evaluated != nil {
			expressions, valueID, err := emitCanonicalExpressionWithLocals(statement.evaluated, owner+":"+statementPath+":evaluate", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			statementID = stableID("execution", owner, statementPath, "evaluate")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a06c", []graphField{refField(0xa06c0, valueID)})})
		} else if statement.deferred != nil {
			expressions, invocationID, err := emitCanonicalExpressionWithLocals(statement.deferred, owner+":"+statementPath+":defer", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			statementID = stableID("execution", owner, statementPath, "native-defer")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a075", []graphField{bytesField(0xa0750, "go"), refField(0xa0751, invocationID)})})
		} else if statement.nativeFieldReceiver != nil {
			receiverEntities, receiverID, err := emitCanonicalExpressionWithLocals(statement.nativeFieldReceiver, owner+":"+statementPath+":native-field-receiver", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			valueEntities, valueID, err := emitCanonicalExpressionWithLocals(statement.nativeFieldValue, owner+":"+statementPath+":native-field-value", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, receiverEntities...)
			*instances = append(*instances, valueEntities...)
			statementID = stableID("execution", owner, statementPath, "native-field-assignment")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a076", []graphField{bytesField(0xa0760, "go"), bytesField(0xa0761, statement.nativeFieldName), refField(0xa0762, receiverID), refField(0xa0763, valueID)})})
		} else if statement.nativePointer != nil {
			pointerEntities, pointerID, err := emitCanonicalExpressionWithLocals(statement.nativePointer, owner+":"+statementPath+":native-pointer", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			valueEntities, valueID, err := emitCanonicalExpressionWithLocals(statement.nativePointerValue, owner+":"+statementPath+":native-pointer-value", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, pointerEntities...)
			*instances = append(*instances, valueEntities...)
			statementID = stableID("execution", owner, statementPath, "native-dereference-assignment")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a083", []graphField{bytesField(0xa0830, "go"), refField(0xa0831, pointerID), refField(0xa0832, valueID)})})
		} else if statement.nativeBindingValue != nil {
			valueEntities, valueID, err := emitCanonicalExpressionWithLocals(statement.nativeBindingValue, owner+":"+statementPath+":native-binding-value", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, valueEntities...)
			statementID = stableID("execution", owner, statementPath, "native-binding-assignment")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a082", []graphField{bytesField(0xa0820, "go"), bytesField(0xa0821, statement.nativeBindingName), refField(0xa0822, valueID)})})
		} else if statement.nativeIndexCollection != nil {
			collectionEntities, collectionID, err := emitCanonicalExpressionWithLocals(statement.nativeIndexCollection, owner+":"+statementPath+":native-index-collection", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			indexEntities, indexID, err := emitCanonicalExpressionWithLocals(statement.nativeIndex, owner+":"+statementPath+":native-index-index", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			valueEntities, valueID, err := emitCanonicalExpressionWithLocals(statement.nativeIndexValue, owner+":"+statementPath+":native-index-value", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, collectionEntities...)
			*instances = append(*instances, indexEntities...)
			*instances = append(*instances, valueEntities...)
			statementID = stableID("execution", owner, statementPath, "native-index-assignment")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a080", []graphField{bytesField(0xa0800, "go"), refField(0xa0801, collectionID), refField(0xa0802, indexID), refField(0xa0803, valueID)})})
		} else if statement.nativeSwitchSubject != nil {
			subjectEntities, subjectID, err := emitCanonicalExpressionWithLocals(statement.nativeSwitchSubject, owner+":"+statementPath+":native-switch-subject", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, subjectEntities...)
			caseIDs := make([]string, len(statement.nativeSwitchCases))
			for caseIndex, switchCase := range statement.nativeSwitchCases {
				valueIDs := make([]string, len(switchCase.values))
				for valueIndex, value := range switchCase.values {
					valueEntities, valueID, err := emitCanonicalExpressionWithLocals(value, owner+":"+statementPath+":native-switch-case:"+strconv.Itoa(caseIndex)+":"+strconv.Itoa(valueIndex), parameterIDs, localIDs, integerTypeID)
					if err != nil {
						return "", err
					}
					*instances = append(*instances, valueEntities...)
					valueIDs[valueIndex] = valueID
				}
				bodyID, err := emitCanonicalBlockScoped(switchCase.body, owner, statementPath+".native-switch-case."+strconv.Itoa(caseIndex), parameterIDs, integerTypeID, localIDs, instances)
				if err != nil {
					return "", err
				}
				caseID := stableID("execution", owner, statementPath, "native-switch-case", strconv.Itoa(caseIndex))
				*instances = append(*instances, graphEntity{caseID, entity(caseID, "0000000000000000000000000000a077", []graphField{refsField(0xa0770, valueIDs), refField(0xa0771, bodyID)})})
				caseIDs[caseIndex] = caseID
			}
			defaultIDs := []string{}
			if statement.nativeSwitchDefault != nil {
				defaultID, err := emitCanonicalBlockScoped(statement.nativeSwitchDefault, owner, statementPath+".native-switch-default", parameterIDs, integerTypeID, localIDs, instances)
				if err != nil {
					return "", err
				}
				defaultIDs = append(defaultIDs, defaultID)
			}
			statementID = stableID("execution", owner, statementPath, "native-switch")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a078", []graphField{bytesField(0xa0780, "go"), refField(0xa0781, subjectID), refsField(0xa0782, caseIDs), refsField(0xa0783, defaultIDs)})})
		} else if statement.nativeBranch != "" {
			statementID = stableID("execution", owner, statementPath, "native-branch")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "0000000000000000000000000000a079", []graphField{bytesField(0xa0790, "go"), bytesField(0xa0791, statement.nativeBranch), bytesField(0xa0792, "nearest")})})
		} else {
			conditionEntities, conditionID, err := emitCanonicalExpressionWithLocals(statement.condition, owner+":"+statementPath+":condition", parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, conditionEntities...)
			thenID, err := emitCanonicalBlockScoped(statement.thenBlock, owner, statementPath+".then", parameterIDs, integerTypeID, localIDs, instances)
			if err != nil {
				return "", err
			}
			elseID, err := emitCanonicalBlockScoped(statement.elseBlock, owner, statementPath+".else", parameterIDs, integerTypeID, localIDs, instances)
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
