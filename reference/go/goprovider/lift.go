package goprovider

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LiftAdd lifts the deliberately narrow Core Execution v1 Go profile. The
// returned G1 is a checked construction projection; compiled .seme is authority.
func LiftAdd(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	declaration, fn, signature, info, err := resolveFunction(project, manifest, functionName)
	if err != nil {
		return "", "", err
	}
	if signature.Params().Len() != 2 || signature.Results().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || !isInt64(signature.Results().At(0).Type()) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	if len(fn.Body.List) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_body:%s", functionName)
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_return:%s", functionName)
	}
	expression, err := analyzeGoExpression(ret.Results[0], signature, info)
	if err != nil {
		return "", "", fmt.Errorf("provider.execution_unsupported_expression:%s", functionName)
	}
	leftIndex, rightIndex, ok := matchAddParameters(expression)
	if !ok {
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

// LiftAdmit lifts the Core Execution v2 quota-policy profile. The accepted Go
// body is exactly: return parameter + parameter <= parameter.
func LiftAdmit(project string, manifest Manifest, moduleG1 []byte, functionName string) (string, string, error) {
	declaration, fn, signature, info, err := resolveFunction(project, manifest, functionName)
	if err != nil {
		return "", "", err
	}
	if signature.Params().Len() != 3 || signature.Results().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || !isInt64(signature.Params().At(2).Type()) || !isBool(signature.Results().At(0).Type()) {
		return "", "", fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	if len(fn.Body.List) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_body:%s", functionName)
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", "", fmt.Errorf("provider.execution_unsupported_return:%s", functionName)
	}
	expression, err := analyzeGoExpression(ret.Results[0], signature, info)
	if err != nil {
		return "", "", fmt.Errorf("provider.execution_unsupported_expression:%s", functionName)
	}
	profile, ok := matchAddLessEqualParameters(expression)
	if !ok {
		return "", "", fmt.Errorf("provider.execution_unsupported_expression:%s", functionName)
	}
	addLeft, addRight, limitIndex := profile.addLeft, profile.addRight, profile.limit

	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	parameterIDs := make([]string, 3)
	readIDs := make([]string, 3)
	instances := []graphEntity{
		{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{unsignedField(0x9100, 64), graphField{0x9101, "tr"}, unsignedField(0x9102, 0)})},
		{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)},
	}
	for i := 0; i < 3; i++ {
		parameterIDs[i] = stableID("execution", declaration.ID, "parameter", strconv.Itoa(i))
		readIDs[i] = stableID("execution", declaration.ID, "read", strconv.Itoa(i))
		instances = append(instances,
			graphEntity{parameterIDs[i], entity(parameterIDs[i], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(i).Name()), refField(0x9121, integerID), unsignedField(0x9122, uint64(i))})},
			graphEntity{readIDs[i], entity(readIDs[i], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[i])})},
		)
	}
	addID := stableID("execution", declaration.ID, "add")
	comparisonID := stableID("execution", declaration.ID, "less-equal")
	programID := stableID("execution", declaration.ID, "program")
	instances = append(instances,
		graphEntity{addID, entity(addID, "00000000000000000000000000009014", []graphField{refField(0x9140, readIDs[addLeft]), refField(0x9141, readIDs[addRight]), refField(0x9142, integerID)})},
		graphEntity{comparisonID, entity(comparisonID, "00000000000000000000000000009021", []graphField{refField(0x9160, addID), refField(0x9161, readIDs[limitIndex]), refField(0x9162, integerID)})},
		graphEntity{declaration.ID, entity(declaration.ID, "00000000000000000000000000009011", []graphField{bytesField(0x9110, functionName), refsField(0x9111, parameterIDs), refField(0x9112, booleanID), refField(0x9113, comparisonID)})},
		graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{refsField(0x9150, []string{declaration.ID}), refField(0x9151, declaration.ID)})},
	)
	return composeExecutionG1(moduleG1, manifest.Revision, instances), declaration.ID, nil
}

// AttachPackageContract composes canonical Package Contract declarations and
// evidence with a lifted executable graph. Empty dependency/effect lists are
// explicit claims, not omitted metadata.
func AttachPackageContract(programG1 string, packageModule []byte, manifest Manifest, functionID string) (string, error) {
	var declaration *Declaration
	for i := range manifest.Declarations {
		if manifest.Declarations[i].ID == functionID {
			declaration = &manifest.Declarations[i]
			break
		}
	}
	if declaration == nil {
		return "", fmt.Errorf("provider.package_unknown_function:%s", functionID)
	}
	packageName := packagePathOf(declaration.NativeKey)
	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	packageID := stableID("package", packageName)
	interfaceID := stableID("package-interface", functionID)
	runtimeID := stableID("runtime-assumption", packageID, "integer.i64.modular")
	mappingID := stableID("fidelity", functionID, "core-execution-v2")
	instances := []graphEntity{
		{packageID, entity(packageID, "0000000000000000000000000000b010", []graphField{bytesField(0xb100, packageName), bytesField(0xb101, manifest.Revision), refsField(0xb102, []string{interfaceID}), refsField(0xb103, nil), refsField(0xb104, nil), refsField(0xb105, []string{runtimeID}), refsField(0xb106, []string{mappingID})})},
		{interfaceID, entity(interfaceID, "0000000000000000000000000000b011", []graphField{bytesField(0xb110, declaration.Name), refField(0xb111, functionID), refsField(0xb112, []string{integerID, integerID, integerID}), refField(0xb113, booleanID)})},
		{runtimeID, entity(runtimeID, "0000000000000000000000000000b013", []graphField{bytesField(0xb130, "integer.i64.modular"), bytesField(0xb131, "signed 64-bit two's-complement wrapping")})},
		{mappingID, entity(mappingID, "0000000000000000000000000000b014", []graphField{refField(0xb140, functionID), refField(0xb141, functionID), unsignedField(0xb142, 0), bytesField(0xb143, "go/types exact lift; differential Go vectors")})},
	}
	instances = append(graphEntities(packageModule), instances...)
	return composeExecutionG1([]byte(programG1), manifest.Revision, instances), nil
}

func resolveFunction(project string, manifest Manifest, functionName string) (*Declaration, *ast.FuncDecl, *types.Signature, *types.Info, error) {
	var declaration *Declaration
	for i := range manifest.Declarations {
		if manifest.Declarations[i].Name == functionName {
			declaration = &manifest.Declarations[i]
			break
		}
	}
	if declaration == nil {
		return nil, nil, nil, nil, fmt.Errorf("provider.execution_unknown_function:%s", functionName)
	}
	return resolvePackageFunction(project, manifest, packagePathOf(declaration.NativeKey), functionName, declaration)
}

func resolveImportedFunction(project string, manifest Manifest, packagePath, functionName string) (*Declaration, *ast.FuncDecl, *types.Signature, *types.Info, error) {
	declaration := &Declaration{ID: stableID("declaration", packagePath+"\x00func\x00"+functionName), Name: functionName, NativeKey: packagePath + "\x00func\x00" + functionName}
	return resolvePackageFunction(project, manifest, packagePath, functionName, declaration)
}

func resolvePackageFunction(project string, manifest Manifest, packagePath, functionName string, declaration *Declaration) (*Declaration, *ast.FuncDecl, *types.Signature, *types.Info, error) {
	fset := token.NewFileSet()
	var parsed []*ast.File
	rootPackage, err := modulePath(project)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	relativePackage := strings.TrimPrefix(packagePath, rootPackage)
	relativePackage = strings.TrimPrefix(relativePackage, "/")
	for _, file := range manifest.Files {
		if !strings.HasSuffix(file.Path, ".go") || strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		directory := filepath.ToSlash(filepath.Dir(file.Path))
		if directory == "." {
			directory = ""
		}
		if directory != relativePackage {
			continue
		}
		node, err := parser.ParseFile(fset, filepath.Join(project, filepath.FromSlash(file.Path)), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		parsed = append(parsed, node)
	}
	info := &types.Info{
		Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{},
		Types: map[ast.Expr]types.TypeAndValue{}, Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	config := types.Config{Importer: newSourceImporter(project, manifest, rootPackage)}
	pkg, err := config.Check(packagePath, fset, parsed, info)
	if err != nil {
		return nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, fmt.Errorf("provider.execution_unresolved_function:%s", functionName)
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("provider.execution_unsupported_signature:%s", functionName)
	}
	return declaration, fn, signature, info, nil
}

func modulePath(project string) (string, error) {
	data, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	for index := 0; index+1 < len(fields); index++ {
		if fields[index] == "module" {
			return fields[index+1], nil
		}
	}
	return "", fmt.Errorf("provider.module_path_missing")
}

type sourceImporter struct {
	project, root string
	manifest      Manifest
	standard      types.Importer
	cache         map[string]*types.Package
	loading       map[string]bool
}

func newSourceImporter(project string, manifest Manifest, root string) *sourceImporter {
	return &sourceImporter{project: project, root: root, manifest: manifest, standard: importer.Default(), cache: map[string]*types.Package{}, loading: map[string]bool{}}
}

func (loader *sourceImporter) Import(path string) (*types.Package, error) {
	if cached := loader.cache[path]; cached != nil {
		return cached, nil
	}
	if loader.loading[path] {
		return nil, fmt.Errorf("provider.import_cycle:%s", path)
	}
	if path != loader.root && !strings.HasPrefix(path, loader.root+"/") {
		return loader.standard.Import(path)
	}
	directory := strings.TrimPrefix(strings.TrimPrefix(path, loader.root), "/")
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, file := range loader.manifest.Files {
		if !strings.HasSuffix(file.Path, ".go") || strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		fileDirectory := filepath.ToSlash(filepath.Dir(file.Path))
		if fileDirectory == "." {
			fileDirectory = ""
		}
		if fileDirectory != directory {
			continue
		}
		node, err := parser.ParseFile(fset, filepath.Join(loader.project, filepath.FromSlash(file.Path)), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, node)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("provider.imported_package_missing:%s", path)
	}
	loader.loading[path] = true
	defer delete(loader.loading, path)
	pkg, err := (&types.Config{Importer: loader}).Check(path, fset, parsed, nil)
	if err != nil {
		return nil, err
	}
	loader.cache[path] = pkg
	return pkg, nil
}

func isInt64(value types.Type) bool {
	basic, ok := value.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Int64
}
func isBool(value types.Type) bool {
	basic, ok := value.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}
func packagePathOf(nativeKey string) string {
	parts := strings.Split(nativeKey, "\x00")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
func composeExecutionG1(module []byte, revision string, instances []graphEntity) string {
	entities := graphEntities(module)
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

func graphEntities(module []byte) []graphEntity {
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
	return entities
}
