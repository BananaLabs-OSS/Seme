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
		if !isInt64(signature.Params().At(index).Type()) && !isBool(signature.Params().At(index).Type()) && !isPureString(signature.Params().At(index).Type()) && !fixedArray {
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
