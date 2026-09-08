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
}

type goBlock struct {
	statements []*goStatement
}

func analyzeGoBlock(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	return analyzeGoBlockWithCalls(statements, signature, info, nil)
}

func analyzeGoBlockWithCalls(statements []ast.Stmt, signature *types.Signature, info *types.Info, functions map[types.Object]string) (*goBlock, error) {
	next := 0
	return analyzeGoBlockScoped(statements, signature, info, map[types.Object]int{}, functions, &next)
}

func analyzeGoBlockScoped(statements []ast.Stmt, signature *types.Signature, info *types.Info, inherited map[types.Object]int, functions map[types.Object]string, next *int) (*goBlock, error) {
	locals := cloneLocalScope(inherited)
	block := &goBlock{}
	for index, raw := range statements {
		switch statement := raw.(type) {
		case *ast.AssignStmt:
			if statement.Tok != token.DEFINE || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
				return nil, fmt.Errorf("control.local_binding_shape")
			}
			name, ok := statement.Lhs[0].(*ast.Ident)
			object := info.Defs[name]
			if !ok || object == nil || (!isInt64(object.Type()) && !isBool(object.Type()) && !isPureString(object.Type())) {
				return nil, fmt.Errorf("control.local_binding_type")
			}
			initializer, err := analyzeGoExpressionWithContext(statement.Rhs[0], signature, info, locals, functions)
			if err != nil {
				return nil, err
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
			block.statements = append(block.statements, &goStatement{localName: name.Name, localType: localType, local: local, initializer: initializer})
			locals[object] = local
		case *ast.ReturnStmt:
			if index != len(statements)-1 || len(statement.Results) != 1 {
				return nil, fmt.Errorf("control.return_arity")
			}
			expression, err := analyzeGoExpressionWithContext(statement.Results[0], signature, info, locals, functions)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, &goStatement{returned: expression})
		case *ast.IfStmt:
			if index != 0 && len(block.statements) != index {
				return nil, fmt.Errorf("control.statement_order")
			}
			branch, err := analyzeTerminalIfScoped(statement, statements[index+1:], signature, info, locals, functions, next)
			if err != nil {
				return nil, err
			}
			block.statements = append(block.statements, branch.statements...)
			return block, nil
		default:
			return nil, fmt.Errorf("control.unsupported_statement")
		}
	}
	if len(block.statements) == 0 || block.statements[len(block.statements)-1].returned == nil {
		return nil, fmt.Errorf("control.block_not_total")
	}
	return block, nil
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
	return analyzeTerminalIfScoped(statement, following, signature, info, map[types.Object]int{}, nil, &next)
}

func analyzeTerminalIfScoped(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info, locals map[types.Object]int, functions map[types.Object]string, next *int) (*goBlock, error) {
	if statement.Init != nil {
		return nil, fmt.Errorf("control.if_init_unsupported")
	}
	condition, err := analyzeGoExpressionWithContext(statement.Cond, signature, info, locals, functions)
	if err != nil {
		return nil, err
	}
	thenBlock, err := analyzeGoBlockScoped(statement.Body.List, signature, info, locals, functions, next)
	if err != nil {
		return nil, err
	}
	var elseBlock *goBlock
	if statement.Else == nil {
		if len(following) == 0 {
			return nil, fmt.Errorf("control.if_missing_fallthrough_return")
		}
		elseBlock, err = analyzeGoBlockScoped(following, signature, info, locals, functions, next)
	} else {
		if len(following) != 0 {
			return nil, fmt.Errorf("control.unreachable_following_statement")
		}
		switch alternate := statement.Else.(type) {
		case *ast.BlockStmt:
			elseBlock, err = analyzeGoBlockScoped(alternate.List, signature, info, locals, functions, next)
		case *ast.IfStmt:
			elseBlock, err = analyzeTerminalIfScoped(alternate, nil, signature, info, locals, functions, next)
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
			expressions, initializerID, err := emitCanonicalExpressionWithLocals(statement.initializer, owner+":"+path+":local:"+strconv.Itoa(statement.local), parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			*instances = append(*instances, graphEntity{bindingID, entity(bindingID, "000000000000000000000000000090d0", []graphField{
				bytesField(0x9d00, statement.localName), refField(0x9d01, localTypeID), refField(0x9d02, initializerID),
			})})
			statementID = stableID("execution", owner, statementPath, "bind-local")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090d1", []graphField{refField(0x9d10, bindingID)})})
			localIDs[statement.local] = bindingID
		} else if statement.returned != nil {
			expressions, expressionID, err := emitCanonicalExpressionWithLocals(statement.returned, owner+":"+path, parameterIDs, localIDs, integerTypeID)
			if err != nil {
				return "", err
			}
			*instances = append(*instances, expressions...)
			statementID = stableID("execution", owner, statementPath, "return")
			*instances = append(*instances, graphEntity{statementID, entity(statementID, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{expressionID})})})
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
		if !isInt64(signature.Params().At(index).Type()) && !isBool(signature.Params().At(index).Type()) && !isPureString(signature.Params().At(index).Type()) {
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
