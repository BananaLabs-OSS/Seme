package goprovider

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LiftAdd lifts the deliberately narrow Core Execution v1 Go profile. The
// returned G1 is a checked construction projection; compiled .seme is authority.
func LiftAdd(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	var declaration *Declaration
	for i := range manifest.Declarations {
		if manifest.Declarations[i].Name == functionName {
			declaration = &manifest.Declarations[i]
			break
		}
	}
	if declaration == nil {
		return "", "", fmt.Errorf("provider.execution_unknown_function:%s", functionName)
	}
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, file := range manifest.Files {
		if !strings.HasSuffix(file.Path, ".go") || strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, filepath.Join(project, filepath.FromSlash(file.Path)), nil, parser.SkipObjectResolution)
		if err != nil {
			return "", "", err
		}
		parsed = append(parsed, node)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	config := types.Config{Importer: importer.Default()}
	pkg, err := config.Check(packagePathOf(declaration.NativeKey), fset, parsed, info)
	if err != nil {
		return "", "", err
	}
	var fn *ast.FuncDecl
	var object *types.Func
	for _, file := range parsed {
		for _, item := range file.Decls {
			candidate, ok := item.(*ast.FuncDecl)
			if !ok || candidate.Recv != nil || candidate.Name.Name != functionName {
				continue
			}
			candidateObject, ok := info.Defs[candidate.Name].(*types.Func)
			if ok && candidateObject.Parent() == pkg.Scope() {
				fn = candidate
				object = candidateObject
			}
		}
	}
	if fn == nil {
		return "", "", fmt.Errorf("provider.execution_unresolved_function:%s", functionName)
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Params().Len() != 2 || signature.Results().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || !isInt64(signature.Results().At(0).Type()) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	if len(fn.Body.List) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_body:%s", functionName)
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_return:%s", functionName)
	}
	add, ok := ret.Results[0].(*ast.BinaryExpr)
	if !ok || add.Op != token.ADD {
		return "", "", fmt.Errorf("provider.execution_unsupported_expression:%s", functionName)
	}
	left, ok := add.X.(*ast.Ident)
	if !ok {
		return "", "", fmt.Errorf("provider.execution_unsupported_left:%s", functionName)
	}
	right, ok := add.Y.(*ast.Ident)
	if !ok {
		return "", "", fmt.Errorf("provider.execution_unsupported_right:%s", functionName)
	}
	leftVar, ok := info.Uses[left].(*types.Var)
	if !ok {
		return "", "", fmt.Errorf("provider.execution_unresolved_left:%s", functionName)
	}
	rightVar, ok := info.Uses[right].(*types.Var)
	if !ok {
		return "", "", fmt.Errorf("provider.execution_unresolved_right:%s", functionName)
	}
	leftIndex, rightIndex := -1, -1
	for i := 0; i < 2; i++ {
		parameter := signature.Params().At(i)
		if parameter == leftVar {
			leftIndex = i
		}
		if parameter == rightVar {
			rightIndex = i
		}
	}
	if leftIndex < 0 || rightIndex < 0 {
		return "", "", fmt.Errorf("provider.execution_nonparameter_add:%s", functionName)
	}
	typeID := stableID("execution", "type", "i64")
	parameterIDs := []string{stableID("execution", declaration.ID, "parameter", "0"), stableID("execution", declaration.ID, "parameter", "1")}
	readIDs := []string{stableID("execution", declaration.ID, "read", "left"), stableID("execution", declaration.ID, "read", "right")}
	addID := stableID("execution", declaration.ID, "add")
	programID := stableID("execution", declaration.ID, "program")
	instances := []graphEntity{
		{declaration.ID, entity(declaration.ID, "00000000000000000000000000009011", []graphField{bytesField(0x9110, functionName), refsField(0x9111, parameterIDs), refField(0x9112, typeID), refField(0x9113, addID)})},
		{typeID, entity(typeID, "00000000000000000000000000009010", []graphField{unsignedField(0x9100, 64), graphField{0x9101, "tr"}, unsignedField(0x9102, 0)})},
		{parameterIDs[0], entity(parameterIDs[0], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(0).Name()), refField(0x9121, typeID), unsignedField(0x9122, 0)})},
		{parameterIDs[1], entity(parameterIDs[1], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(1).Name()), refField(0x9121, typeID), unsignedField(0x9122, 1)})},
		{readIDs[0], entity(readIDs[0], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[leftIndex])})},
		{readIDs[1], entity(readIDs[1], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[rightIndex])})},
		{addID, entity(addID, "00000000000000000000000000009014", []graphField{refField(0x9140, readIDs[0]), refField(0x9141, readIDs[1]), refField(0x9142, typeID)})},
		{programID, entity(programID, "00000000000000000000000000009015", []graphField{refsField(0x9150, []string{declaration.ID}), refField(0x9151, declaration.ID)})},
	}
	return composeExecutionG1(moduleG1, manifest.Revision, instances), declaration.ID, nil
}
func isInt64(value types.Type) bool {
	basic, ok := value.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Int64
}
func packagePathOf(nativeKey string) string {
	parts := strings.Split(nativeKey, "\x00")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
func composeExecutionG1(module []byte, revision string, instances []graphEntity) string {
	var entities []graphEntity
	var current *graphEntity
	for _, line := range strings.Split(strings.TrimSpace(string(module)), "\n") {
		if strings.HasPrefix(line, "en ") {
			parts := strings.Fields(line)
			entities = append(entities, graphEntity{id: parts[1]})
			current = &entities[len(entities)-1]
		}
		if current != nil {
			current.text += line + "\n"
		}
	}
	entities = append(entities, instances...)
	sort.Slice(entities, func(i, j int) bool { return entities[i].id < entities[j].id })
	var out strings.Builder
	fmt.Fprintf(&out, "# Generated exact Go to Core Execution v1 lift.\nve 1\nmo %032x\nrv %s\npc 0\nec %s\n", 0x9000, revision, strconv.Itoa(len(entities)))
	for _, item := range entities {
		out.WriteString("\n")
		out.WriteString(item.text)
	}
	return out.String()
}
