package goprovider

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
)

type goStatement struct {
	condition *goExpression
	returned  *goExpression
	thenBlock *goBlock
	elseBlock *goBlock
}

type goBlock struct {
	statement *goStatement
}

func analyzeGoBlock(statements []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	if len(statements) == 1 {
		switch statement := statements[0].(type) {
		case *ast.ReturnStmt:
			if len(statement.Results) != 1 {
				return nil, fmt.Errorf("control.return_arity")
			}
			expression, err := analyzeGoExpression(statement.Results[0], signature, info)
			if err != nil {
				return nil, err
			}
			return &goBlock{statement: &goStatement{returned: expression}}, nil
		case *ast.IfStmt:
			return analyzeTerminalIf(statement, nil, signature, info)
		}
	}
	if len(statements) >= 2 {
		if statement, ok := statements[0].(*ast.IfStmt); ok && statement.Else == nil {
			return analyzeTerminalIf(statement, statements[1:], signature, info)
		}
	}
	return nil, fmt.Errorf("control.block_not_total")
}

func analyzeTerminalIf(statement *ast.IfStmt, following []ast.Stmt, signature *types.Signature, info *types.Info) (*goBlock, error) {
	if statement.Init != nil {
		return nil, fmt.Errorf("control.if_init_unsupported")
	}
	condition, err := analyzeGoExpression(statement.Cond, signature, info)
	if err != nil {
		return nil, err
	}
	thenBlock, err := analyzeGoBlock(statement.Body.List, signature, info)
	if err != nil {
		return nil, err
	}
	var elseBlock *goBlock
	if statement.Else == nil {
		if len(following) == 0 {
			return nil, fmt.Errorf("control.if_missing_fallthrough_return")
		}
		elseBlock, err = analyzeGoBlock(following, signature, info)
	} else {
		if len(following) != 0 {
			return nil, fmt.Errorf("control.unreachable_following_statement")
		}
		switch alternate := statement.Else.(type) {
		case *ast.BlockStmt:
			elseBlock, err = analyzeGoBlock(alternate.List, signature, info)
		case *ast.IfStmt:
			elseBlock, err = analyzeTerminalIf(alternate, nil, signature, info)
		default:
			err = fmt.Errorf("control.else_unsupported")
		}
	}
	if err != nil {
		return nil, err
	}
	return &goBlock{statement: &goStatement{condition: condition, thenBlock: thenBlock, elseBlock: elseBlock}}, nil
}

func emitCanonicalBlock(block *goBlock, owner, path string, parameterIDs []string, integerTypeID string, instances *[]graphEntity) (string, error) {
	if block == nil || block.statement == nil {
		return "", fmt.Errorf("control.nil_block")
	}
	statementPath := path + ".statement"
	statementID := ""
	if block.statement.returned != nil {
		expressions, expressionID, err := emitCanonicalExpression(block.statement.returned, owner+":"+path, parameterIDs, integerTypeID)
		if err != nil {
			return "", err
		}
		*instances = append(*instances, expressions...)
		statementID = stableID("execution", owner, statementPath, "return")
		*instances = append(*instances, graphEntity{statementID, entity(statementID, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{expressionID})})})
	} else {
		conditionEntities, conditionID, err := emitCanonicalExpression(block.statement.condition, owner+":"+path+":condition", parameterIDs, integerTypeID)
		if err != nil {
			return "", err
		}
		*instances = append(*instances, conditionEntities...)
		thenID, err := emitCanonicalBlock(block.statement.thenBlock, owner, path+".then", parameterIDs, integerTypeID, instances)
		if err != nil {
			return "", err
		}
		elseID, err := emitCanonicalBlock(block.statement.elseBlock, owner, path+".else", parameterIDs, integerTypeID, instances)
		if err != nil {
			return "", err
		}
		statementID = stableID("execution", owner, statementPath, "if")
		*instances = append(*instances, graphEntity{statementID, entity(statementID, "000000000000000000000000000090c0", []graphField{
			refField(0x9c00, conditionID), refField(0x9c01, thenID), refField(0x9c02, elseID),
		})})
	}
	blockID := stableID("execution", owner, path, "block")
	*instances = append(*instances, graphEntity{blockID, entity(blockID, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{statementID})})})
	return blockID, nil
}

// LiftControlFunction lifts the bounded total-return control-flow profile. Go
// fallthrough after an if is normalized into the explicit canonical else block.
func LiftControlFunction(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	declaration, fn, signature, info, err := resolveFunction(project, manifest, functionName)
	if err != nil {
		return "", "", err
	}
	if signature.Results().Len() != 1 || (!isInt64(signature.Results().At(0).Type()) && !isBool(signature.Results().At(0).Type())) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	for index := 0; index < signature.Params().Len(); index++ {
		if !isInt64(signature.Params().At(index).Type()) && !isBool(signature.Params().At(index).Type()) {
			return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
		}
	}
	block, err := analyzeGoBlock(fn.Body.List, signature, info)
	if err != nil {
		return "", "", fmt.Errorf("provider.execution_unsupported_control:%s:%w", functionName, err)
	}

	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	resultTypeID := integerID
	if isBool(signature.Results().At(0).Type()) {
		resultTypeID = booleanID
	}
	instances := []graphEntity{{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{
		unsignedField(0x9100, 64), {0x9101, "tr"}, unsignedField(0x9102, 0),
	})}}
	needsBoolean := isBool(signature.Results().At(0).Type())
	parameterIDs := make([]string, signature.Params().Len())
	for index := range parameterIDs {
		needsBoolean = needsBoolean || isBool(signature.Params().At(index).Type())
	}
	if needsBoolean {
		instances = append(instances, graphEntity{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)})
	}
	for index := range parameterIDs {
		typeID := integerID
		if isBool(signature.Params().At(index).Type()) {
			typeID = booleanID
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
