package goprovider

import (
	"fmt"
	"go/ast"
	"strconv"
)

// LiftStructuredFunction lifts the first compositional statement profile. It
// accepts a typed function with int64 parameters and result and one return
// statement. The returned expression is analyzed recursively; the function is
// represented by a Block containing a Return rather than by a recognized
// whole-function shape.
func LiftStructuredFunction(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	declaration, fn, signature, info, err := resolveFunction(project, manifest, functionName)
	if err != nil {
		return "", "", err
	}
	if signature.Results().Len() != 1 || !isInt64(signature.Results().At(0).Type()) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	for index := 0; index < signature.Params().Len(); index++ {
		if !isInt64(signature.Params().At(index).Type()) {
			return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
		}
	}
	if len(fn.Body.List) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_body:%s", functionName)
	}
	returned, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_return:%s", functionName)
	}
	expression, err := analyzeGoExpression(returned.Results[0], signature, info)
	if err != nil {
		return "", "", fmt.Errorf("provider.execution_unsupported_expression:%s", functionName)
	}

	integerID := stableID("execution", "type", "i64")
	parameterIDs := make([]string, signature.Params().Len())
	instances := []graphEntity{{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{
		unsignedField(0x9100, 64),
		{0x9101, "tr"},
		unsignedField(0x9102, 0),
	})}}
	for index := range parameterIDs {
		parameterIDs[index] = stableID("execution", declaration.ID, "parameter", strconv.Itoa(index))
		instances = append(instances, graphEntity{parameterIDs[index], entity(parameterIDs[index], "00000000000000000000000000009012", []graphField{
			bytesField(0x9120, signature.Params().At(index).Name()),
			refField(0x9121, integerID),
			unsignedField(0x9122, uint64(index)),
		})})
	}
	expressions, expressionID, err := emitCanonicalExpression(expression, declaration.ID, parameterIDs, integerID)
	if err != nil {
		return "", "", err
	}
	instances = append(instances, expressions...)
	returnID := stableID("execution", declaration.ID, "statement", "return")
	blockID := stableID("execution", declaration.ID, "block", "body")
	programID := stableID("execution", declaration.ID, "program")
	instances = append(instances,
		graphEntity{returnID, entity(returnID, "00000000000000000000000000009081", []graphField{refsField(0x9810, []string{expressionID})})},
		graphEntity{blockID, entity(blockID, "00000000000000000000000000009080", []graphField{refsField(0x9800, []string{returnID})})},
		graphEntity{declaration.ID, entity(declaration.ID, "00000000000000000000000000009011", []graphField{
			bytesField(0x9110, functionName), refsField(0x9111, parameterIDs), refField(0x9112, integerID), refField(0x9113, blockID),
		})},
		graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{
			refsField(0x9150, []string{declaration.ID}), refField(0x9151, declaration.ID),
		})},
	)
	return composeExecutionG1(moduleG1, manifest.Revision, instances), declaration.ID, nil
}
