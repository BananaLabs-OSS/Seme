package goprovider

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// DocumentSnapshot is a complete in-memory package snapshot. Revision is a
// caller-owned, strictly increasing sequence number; Files never touch disk.
type DocumentSnapshot struct {
	Revision    uint64
	ModulePath  string
	PackagePath string
	Entry       string
	Files       map[string]string
}

type SessionDiagnostic struct {
	Code     string
	Message  string
	File     string
	Line     int
	Column   int
	Severity string
}

type SourceIdentity struct {
	ID, Kind, Name, Document string
	Start, End               int
	Line, Column             int
}

type SessionResult struct {
	Revision          uint64
	Accepted          bool
	Valid             bool
	CanonicalG1       string
	LastValidRevision uint64
	Diagnostics       []SessionDiagnostic
	Sources           []SourceIdentity
	ContentDigest     string
	Disposition       string
	Packages          []PackageMetadata
	Resolution        ResolutionManifest
}

// PackageMetadata is the immutable language-neutral ownership/signature view
// proven by the same typed walk that produced CanonicalG1.
type PackageMetadata struct {
	Name         string
	Root         bool
	Dependencies []string
	Members      []PackageFunctionMetadata
	Functions    []PackageFunctionMetadata
	// Supplemental owns canonical declarations represented by Package v3 but
	// not by the Package v2 function-member surface.
	Supplemental []SemanticDeclarationMetadata
}
type PackageFunctionMetadata struct {
	ID, Name     string
	Parameters   []string
	Result       string
	Exported     bool
	Document     string
	Line, Column int
}

type SemanticDeclarationKind string

const (
	SemanticRecord             SemanticDeclarationKind = "record"
	SemanticInterface          SemanticDeclarationKind = "interface"
	SemanticMethod             SemanticDeclarationKind = "method"
	SemanticGenericRealization SemanticDeclarationKind = "generic-realization"
)

// SemanticDeclarationMetadata is derived from the same typed objects and AST
// positions used to emit CanonicalG1. Declaration is the emitted canonical
// identity; Package is the Go package path, not a guessed wire identity.
type SemanticDeclarationMetadata struct {
	Declaration, Package, Name, GenericDefinition string
	Kind                                          SemanticDeclarationKind
	Exported                                      bool
	Origin                                        ProjectLocation
	ReferencedImports                             []string
}

// IncrementalSession retains only the most recent valid canonical graph while
// still advancing past accepted but incomplete editor snapshots.
type IncrementalSession struct {
	mu                  sync.Mutex
	moduleG1            []byte
	currentRevision     uint64
	lastValidRevision   uint64
	lastValidGraph      string
	lastValidSources    []SourceIdentity
	lastValidPackages   []PackageMetadata
	lastValidResolution ResolutionManifest
}

func NewIncrementalSession(executionModuleG1 []byte) (*IncrementalSession, error) {
	if len(graphEntities(executionModuleG1)) == 0 {
		return nil, fmt.Errorf("session.execution_module_empty")
	}
	return &IncrementalSession{moduleG1: append([]byte(nil), executionModuleG1...)}, nil
}

func (session *IncrementalSession) Apply(snapshot DocumentSnapshot) SessionResult {
	session.mu.Lock()
	defer session.mu.Unlock()
	if snapshot.Revision == 0 || snapshot.Revision <= session.currentRevision {
		return SessionResult{
			Revision: snapshot.Revision, Accepted: false, Valid: false,
			CanonicalG1: session.lastValidGraph, LastValidRevision: session.lastValidRevision,
			Diagnostics:   []SessionDiagnostic{{Code: "session.stale_revision", Message: "revision must be strictly newer than the last accepted snapshot", Severity: "error"}},
			Sources:       cloneSources(session.lastValidSources),
			Packages:      clonePackageMetadata(session.lastValidPackages),
			Resolution:    cloneResolutionManifest(session.lastValidResolution),
			ContentDigest: snapshotDigest(snapshot), Disposition: "rejected-stale",
		}
	}
	session.currentRevision = snapshot.Revision
	resolution, resolutionDiagnostics := resolveSnapshot(snapshot)
	graph, sources, packages, diagnostics := "", []SourceIdentity(nil), []PackageMetadata(nil), resolutionDiagnostics
	if len(diagnostics) == 0 {
		graph, sources, packages, diagnostics = liftDocumentSnapshot(snapshot, session.moduleG1)
	}
	valid := graph != ""
	if valid {
		attachResolutionOwnership(&resolution, packages)
		session.lastValidRevision = snapshot.Revision
		session.lastValidGraph = graph
		session.lastValidSources = cloneSources(sources)
		session.lastValidPackages = clonePackageMetadata(packages)
		session.lastValidResolution = cloneResolutionManifest(resolution)
	}
	return SessionResult{
		Revision: snapshot.Revision, Accepted: true, Valid: valid,
		CanonicalG1: session.lastValidGraph, LastValidRevision: session.lastValidRevision,
		Diagnostics: diagnostics, Sources: cloneSources(session.lastValidSources),
		Packages:      clonePackageMetadata(session.lastValidPackages),
		Resolution:    cloneResolutionManifest(session.lastValidResolution),
		ContentDigest: snapshotDigest(snapshot), Disposition: map[bool]string{true: "accepted-valid", false: "accepted-invalid"}[valid],
	}
}

func attachResolutionOwnership(resolution *ResolutionManifest, packages []PackageMetadata) {
	byName := make(map[string][]SemanticDeclarationMetadata, len(packages))
	for _, item := range packages {
		byName[item.Name] = item.Supplemental
	}
	for i := range resolution.Packages {
		resolution.Packages[i].Supplemental = cloneSemanticDeclarations(byName[resolution.Packages[i].Name])
	}
}

func clonePackageMetadata(in []PackageMetadata) []PackageMetadata {
	out := make([]PackageMetadata, len(in))
	for i, p := range in {
		out[i] = p
		out[i].Dependencies = append([]string(nil), p.Dependencies...)
		out[i].Members = clonePackageFunctions(p.Members)
		out[i].Functions = clonePackageFunctions(p.Functions)
		out[i].Supplemental = cloneSemanticDeclarations(p.Supplemental)
	}
	return out
}

func cloneSemanticDeclarations(in []SemanticDeclarationMetadata) []SemanticDeclarationMetadata {
	out := make([]SemanticDeclarationMetadata, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].ReferencedImports = append([]string(nil), in[i].ReferencedImports...)
	}
	return out
}

func clonePackageFunctions(in []PackageFunctionMetadata) []PackageFunctionMetadata {
	out := make([]PackageFunctionMetadata, len(in))
	for i, f := range in {
		out[i] = f
		out[i].Parameters = append([]string(nil), f.Parameters...)
	}
	return out
}

func cloneSources(sources []SourceIdentity) []SourceIdentity {
	return append([]SourceIdentity(nil), sources...)
}

type sessionFunction struct {
	id, name    string
	packagePath string
	fn          *ast.FuncDecl
	sig         *types.Signature
	info        *types.Info
	file        string
	fset        *token.FileSet
	method      bool
	receiverID  string
}

type checkedSessionPackage struct {
	path        string
	pkg         *types.Package
	files       []*ast.File
	info        *types.Info
	fset        *token.FileSet
	semanticIDs map[types.Object]string
	receiverIDs map[types.Object]string
}

type snapshotSourceImporter struct {
	snapshot    DocumentSnapshot
	groups      map[string][]string
	loaded      map[string]*checkedSessionPackage
	checking    map[string]bool
	diagnostics *[]SessionDiagnostic
}

func (loader *snapshotSourceImporter) Import(path string) (*types.Package, error) {
	if unit := loader.loaded[path]; unit != nil {
		return unit.pkg, nil
	}
	paths, local := loader.groups[path]
	if !local {
		module := loader.snapshot.ModulePath
		if module == "" {
			module = loader.snapshot.PackagePath
		}
		if path == module || strings.HasPrefix(path, module+"/") {
			return nil, fmt.Errorf("go.local_import_missing:%s", path)
		}
		pkg, err := build.Default.Import(path, "", build.FindOnly)
		if err != nil || !pkg.Goroot {
			return nil, fmt.Errorf("go.external_import_unsupported:%s", path)
		}
		return importer.Default().Import(path)
	}
	if loader.checking[path] {
		return nil, fmt.Errorf("go.import_cycle:%s", path)
	}
	loader.checking[path] = true
	defer delete(loader.checking, path)
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(paths))
	for _, name := range paths {
		file, err := parser.ParseFile(fset, name, loader.snapshot.Files[name], parser.AllErrors|parser.ParseComments)
		if err != nil {
			*loader.diagnostics = append(*loader.diagnostics, parseDiagnostics(err)...)
			return nil, err
		}
		files = append(files, file)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	config := types.Config{Importer: loader, Error: func(err error) { *loader.diagnostics = append(*loader.diagnostics, typeDiagnostic(err)) }}
	pkg, err := config.Check(path, fset, files, info)
	if err != nil {
		return nil, err
	}
	semanticIDs := map[types.Object]string{}
	receiverIDs := map[types.Object]string{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			switch node := declaration.(type) {
			case *ast.FuncDecl:
				if id := semeIdentityDirective(node.Doc); id != "" {
					semanticIDs[info.Defs[node.Name]] = id
				}
				if id := semeReceiverDirective(node.Doc); id != "" {
					receiverIDs[info.Defs[node.Name]] = id
				}
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					if typed, ok := spec.(*ast.TypeSpec); ok {
						doc := typed.Doc
						if doc == nil {
							doc = node.Doc
						}
						if id := semeIdentityDirective(doc); id != "" {
							semanticIDs[info.Defs[typed.Name]] = id
						}
					}
				}
			}
		}
	}
	loader.loaded[path] = &checkedSessionPackage{path: path, pkg: pkg, files: files, info: info, fset: fset, semanticIDs: semanticIDs, receiverIDs: receiverIDs}
	return pkg, nil
}

func semeIdentityDirective(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	for _, comment := range group.List {
		value := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if !strings.HasPrefix(value, "seme:id ") {
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(value, "seme:id "))
		if len(id) != 32 {
			continue
		}
		if _, err := hex.DecodeString(id); err == nil {
			return id
		}
	}
	return ""
}

func semeReceiverDirective(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	for _, comment := range group.List {
		value := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if !strings.HasPrefix(value, "seme:receiver ") {
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(value, "seme:receiver "))
		if len(id) != 32 {
			continue
		}
		if _, err := hex.DecodeString(id); err == nil {
			return id
		}
	}
	return ""
}

func checkSessionPackages(snapshot DocumentSnapshot) ([]*checkedSessionPackage, []SessionDiagnostic) {
	module := snapshot.ModulePath
	if module == "" {
		module = snapshot.PackagePath
	}
	groups := map[string][]string{}
	var diagnostics []SessionDiagnostic
	for name, source := range snapshot.Files {
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		directory := filepath.ToSlash(filepath.Dir(name))
		if directory == "." {
			directory = ""
		}
		path := module
		if directory != "" {
			path += "/" + directory
		}
		groups[path] = append(groups[path], filepath.ToSlash(name))
		_ = source
	}
	for path := range groups {
		sort.Strings(groups[path])
	}
	// An editor snapshot may give its root package a semantic identity that
	// intentionally differs from its on-disk go.mod path. Remap only that root;
	// genuine imported subpackages retain their module-relative identities.
	if snapshot.PackagePath != module {
		if rootFiles, exists := groups[module]; exists {
			delete(groups, module)
			groups[snapshot.PackagePath] = rootFiles
		}
	}
	loader := &snapshotSourceImporter{snapshot: snapshot, groups: groups, loaded: map[string]*checkedSessionPackage{}, checking: map[string]bool{}, diagnostics: &diagnostics}
	if _, err := loader.Import(snapshot.PackagePath); err != nil && len(diagnostics) == 0 {
		diagnostics = append(diagnostics, SessionDiagnostic{Code: "go.type", Message: err.Error(), Severity: "error"})
	}
	paths := make([]string, 0, len(loader.loaded))
	for path := range loader.loaded {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	units := make([]*checkedSessionPackage, 0, len(paths))
	for _, path := range paths {
		units = append(units, loader.loaded[path])
	}
	return units, sortedDiagnostics(diagnostics)
}

type goInterfaceInfo struct {
	named          *types.Named
	id             string
	requirements   []*types.Func
	requirementIDs []string
}

func liftDocumentSnapshot(snapshot DocumentSnapshot, moduleG1 []byte) (string, []SourceIdentity, []PackageMetadata, []SessionDiagnostic) {
	if snapshot.PackagePath == "" {
		return "", nil, nil, []SessionDiagnostic{{Code: "session.package_path_missing", Message: "package path is required", Severity: "error"}}
	}
	units, diagnostics := checkSessionPackages(snapshot)
	if len(diagnostics) != 0 || len(units) == 0 {
		return "", nil, nil, diagnostics
	}
	var functions []sessionFunction
	for _, unit := range units {
		for _, file := range unit.files {
			for _, declaration := range file.Decls {
				fn, ok := declaration.(*ast.FuncDecl)
				if !ok {
					continue
				}
				object, ok := unit.info.Defs[fn.Name].(*types.Func)
				if !ok {
					continue
				}
				signature, ok := object.Type().(*types.Signature)
				if !ok {
					continue
				}
				position := unit.fset.Position(fn.Pos())
				declarationID := unit.semanticIDs[object]
				if declarationID == "" {
					declarationID = stableID("session-declaration", unit.path, fn.Name.Name)
				}
				if fn.Recv != nil {
					receiverName := receiverTypeName(signature.Recv().Type())
					if unit.semanticIDs[object] == "" {
						declarationID = stableID("session-method", unit.path, receiverName, fn.Name.Name)
					}
				}
				functions = append(functions, sessionFunction{
					id: declarationID, name: fn.Name.Name, packagePath: unit.path,
					fn: fn, sig: signature, info: unit.info, file: filepath.ToSlash(position.Filename), fset: unit.fset, method: fn.Recv != nil,
					receiverID: unit.receiverIDs[object],
				})
			}
		}
	}
	sort.Slice(functions, func(left, right int) bool { return functions[left].id < functions[right].id })
	functionObjects := make(map[types.Object]string, len(functions))
	for _, function := range functions {
		functionObjects[function.info.Defs[function.fn.Name]] = function.id
	}
	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	stringID := stableID("execution", "type", "string")
	bytesID := stableID("execution", "type", "bytes")
	instances := []graphEntity{
		{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{unsignedField(0x9100, 64), {0x9101, "tr"}, unsignedField(0x9102, 0)})},
		{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)},
		{stringID, entity(stringID, "00000000000000000000000000009040", nil)},
		{bytesID, entity(bytesID, "00000000000000000000000000009041", nil)},
	}
	records := make(map[*types.Named]goRecordInfo)
	for _, unit := range units {
		for identifier, object := range unit.info.Defs {
			typeName, ok := object.(*types.TypeName)
			if !ok {
				continue
			}
			named, namedOK := typeName.Type().(*types.Named)
			if !namedOK {
				continue
			}
			structure, structOK := named.Underlying().(*types.Struct)
			if !structOK || structure.NumFields() == 0 {
				continue
			}
			recordID := unit.semanticIDs[object]
			if recordID == "" {
				recordID = stableID("execution", "record", unit.path, identifier.Name)
			}
			record := goRecordInfo{id: recordID, fields: map[*types.Var]string{}}
			fieldIDs := make([]string, structure.NumFields())
			var fieldEntities []graphEntity
			valid := true
			for index := 0; index < structure.NumFields(); index++ {
				field := structure.Field(index)
				typeID := integerID
				if isBool(field.Type()) {
					typeID = booleanID
				} else if isPureString(field.Type()) {
					typeID = stringID
				} else if isI64Slice(field.Type()) || isI64Map(field.Type()) || isBytes(field.Type()) {
					typeID = goSemanticTypeIdentity(field.Type())
					for _, typeEntity := range goBridgeTypeEntities(field.Type(), integerID, booleanID, stringID) {
						if !hasGraphEntity(instances, typeEntity.id) {
							instances = append(instances, typeEntity)
						}
					}
				} else if !isInt64(field.Type()) {
					valid = false
					break
				}
				fieldID := stableID("execution", recordID, "field", strconv.Itoa(index))
				fieldIDs[index] = fieldID
				record.fields[field] = fieldID
				record.ordered = append(record.ordered, field)
				fieldEntities = append(fieldEntities, graphEntity{fieldID, entity(fieldID, "00000000000000000000000000009031", []graphField{
					bytesField(0x9310, field.Name()), refField(0x9311, typeID), unsignedField(0x9312, uint64(index)),
				})})
			}
			if valid {
				records[named] = record
				instances = append(instances, fieldEntities...)
				instances = append(instances, graphEntity{recordID, entity(recordID, "00000000000000000000000000009030", []graphField{bytesField(0x9300, identifier.Name), refsField(0x9301, fieldIDs)})})
			}
		}
	}
	for _, function := range functions {
		if !function.method || function.receiverID == "" {
			continue
		}
		value := function.sig.Recv().Type()
		if pointer, ok := value.(*types.Pointer); ok {
			value = pointer.Elem()
		}
		if named, ok := value.(*types.Named); ok {
			if record, exists := findGoRecord(records, named); exists {
				record.receiverID = function.receiverID
				records[named.Origin()] = record
			}
		}
	}
	var interfaces []goInterfaceInfo
	for _, unit := range units {
		for identifier, object := range unit.info.Defs {
			typeName, ok := object.(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}
			contract, ok := named.Underlying().(*types.Interface)
			if !ok {
				continue
			}
			contract = contract.Complete()
			interfaceID := unit.semanticIDs[object]
			if interfaceID == "" {
				interfaceID = stableID("execution", "interface", unit.path, identifier.Name)
			}
			declaration := goInterfaceInfo{named: named, id: interfaceID}
			valid := contract.NumMethods() > 0
			for index := 0; valid && index < contract.NumMethods(); index++ {
				method := contract.Method(index)
				signature, ok := method.Type().(*types.Signature)
				if !ok || signature.Params().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || signature.Results().Len() != 1 || !isInt64(signature.Results().At(0).Type()) {
					valid = false
					break
				}
				requirementID := stableID("execution", interfaceID, "requirement", method.Name())
				declaration.requirements = append(declaration.requirements, method)
				declaration.requirementIDs = append(declaration.requirementIDs, requirementID)
				instances = append(instances, graphEntity{requirementID, entity(requirementID, "0000000000000000000000000000a011", []graphField{
					bytesField(0xa0110, method.Name()), refsField(0xa0111, []string{integerID}), refField(0xa0112, integerID),
				})})
				functionObjects[method] = requirementID
			}
			if !valid {
				continue
			}
			instances = append(instances, graphEntity{interfaceID, entity(interfaceID, "0000000000000000000000000000a010", []graphField{
				bytesField(0xa0100, identifier.Name), refsField(0xa0101, declaration.requirementIDs),
			})})
			records[named] = goRecordInfo{id: interfaceID}
			interfaces = append(interfaces, declaration)
		}
	}
	for named, record := range records {
		if _, isInterface := named.Underlying().(*types.Interface); isInterface {
			continue
		}
		for _, contract := range interfaces {
			if !types.Implements(named, contract.named.Underlying().(*types.Interface)) {
				continue
			}
			methodIDs := make([]string, len(contract.requirements))
			valid := true
			for index, requirement := range contract.requirements {
				method, _, _ := types.LookupFieldOrMethod(named, true, named.Obj().Pkg(), requirement.Name())
				methodID, exists := functionObjects[method]
				if !exists {
					valid = false
					break
				}
				methodIDs[index] = methodID
			}
			if valid {
				witnessID := stableID("execution", "witness", record.id, contract.id)
				instances = append(instances, graphEntity{witnessID, entity(witnessID, "0000000000000000000000000000a012", []graphField{
					refField(0xa0120, record.id), refField(0xa0121, contract.id), refsField(0xa0122, methodIDs),
				})})
			}
		}
	}
	var functionIDs []string
	var sources []SourceIdentity
	for _, function := range functions {
		entities, source, diagnostic := liftSessionFunction(function, integerID, booleanID, stringID, functionObjects, records)
		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
			continue
		}
		instances = append(instances, entities...)
		if !function.method {
			functionIDs = append(functionIDs, function.id)
		}
		sources = append(sources, source)
	}
	supportedFunctions := make(map[string]bool, len(functionIDs))
	for _, id := range functionIDs {
		supportedFunctions[id] = true
	}
	for _, instance := range instances {
		if callee, ok := graphFunctionCallCallee(instance.text); ok && !supportedFunctions[callee] {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.call_target_unsupported", Message: "supported function calls an omitted declaration", Severity: "error"})
			return "", nil, nil, sortedDiagnostics(diagnostics)
		}
	}
	if len(functionIDs) == 0 {
		if len(diagnostics) == 0 {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.no_supported_declarations", Message: "snapshot contains no supported package functions", Severity: "error"})
		}
		return "", nil, nil, sortedDiagnostics(diagnostics)
	}
	entryID := ""
	for _, function := range functions {
		if !function.method && function.packagePath == snapshot.PackagePath && supportedFunctions[function.id] {
			entryID = function.id
			break
		}
	}
	if snapshot.Entry != "" {
		entryID = ""
		for _, function := range functions {
			if function.packagePath == snapshot.PackagePath && function.name == snapshot.Entry {
				for _, supported := range functionIDs {
					if supported == function.id {
						entryID = supported
					}
				}
			}
		}
	}
	if entryID == "" {
		diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.entry_missing", Message: "requested root-package entry function is not supported", Severity: "error"})
		return "", nil, nil, sortedDiagnostics(diagnostics)
	}
	programID := stableID("session-program", snapshot.PackagePath)
	instances = append(instances, graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{
		refsField(0x9150, functionIDs), refField(0x9151, entryID),
	})})
	revision := stableID("session-revision", snapshot.PackagePath, strconv.FormatUint(snapshot.Revision, 10))
	metadata := buildPackageMetadata(snapshot.PackagePath, units, functions, supportedFunctions)
	if diagnostic := attachSemanticOwnership(metadata, units, functions, instances); diagnostic != nil {
		diagnostics = append(diagnostics, *diagnostic)
		return "", nil, nil, sortedDiagnostics(diagnostics)
	}
	return composeExecutionG1(moduleG1, revision, instances), sources, metadata, sortedDiagnostics(diagnostics)
}

func buildPackageMetadata(root string, units []*checkedSessionPackage, functions []sessionFunction, supported map[string]bool) []PackageMetadata {
	local := map[string]bool{}
	for _, u := range units {
		local[u.path] = true
	}
	out := make([]PackageMetadata, 0, len(units))
	for _, u := range units {
		p := PackageMetadata{Name: u.path}
		if u.path == root {
			p.Root = true
		}
		for _, dep := range u.pkg.Imports() {
			if local[dep.Path()] {
				p.Dependencies = append(p.Dependencies, dep.Path())
			}
		}
		sort.Strings(p.Dependencies)
		for _, f := range functions {
			if f.packagePath != u.path || f.method || !supported[f.id] {
				continue
			}
			position := f.fset.Position(f.fn.Name.Pos())
			m := PackageFunctionMetadata{ID: f.id, Name: f.name, Result: goSemanticTypeIdentity(f.sig.Results().At(0).Type()), Exported: ast.IsExported(f.name), Document: f.file, Line: position.Line, Column: position.Column}
			for i := 0; i < f.sig.Params().Len(); i++ {
				m.Parameters = append(m.Parameters, goSemanticTypeIdentity(f.sig.Params().At(i).Type()))
			}
			p.Members = append(p.Members, m)
			if m.Exported {
				p.Functions = append(p.Functions, m)
			}
		}
		sort.Slice(p.Members, func(i, j int) bool { return p.Members[i].ID < p.Members[j].ID })
		sort.Slice(p.Functions, func(i, j int) bool { return p.Functions[i].ID < p.Functions[j].ID })
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func graphFunctionCallCallee(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return "", false
	}
	header := strings.Fields(lines[0])
	if len(header) < 3 || header[2] != "00000000000000000000000000009060" {
		return "", false
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == "fi" && fields[1] == "00000000000000000000000000009600" && fields[2] == "rf" {
			return fields[3], true
		}
	}
	return "", false
}

func liftSessionFunction(function sessionFunction, integerID, booleanID, stringID string, functions map[types.Object]string, records map[*types.Named]goRecordInfo) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
	position := function.fset.Position(function.fn.Pos())
	diagnostic := func(code, message string) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
		return nil, SourceIdentity{}, &SessionDiagnostic{Code: code, Message: message, File: function.file, Line: position.Line, Column: position.Column, Severity: "warning"}
	}
	if function.sig.Results().Len() != 1 {
		return diagnostic("session.unsupported_function_shape", "supported functions require one result")
	}
	resultType := function.sig.Results().At(0).Type()
	resultTypeID := integerID
	transitionState, transitionResult, isTransition := goTransitionTypes(resultType)
	if isTransition {
		var ok bool
		resultTypeID, ok = goSupportedTypeID(resultType, integerID, booleanID, stringID, records)
		if !ok {
			return diagnostic("session.unsupported_result_type", "transition state and result types must be supported")
		}
	} else if isBool(resultType) {
		resultTypeID = booleanID
	} else if isPureString(function.sig.Results().At(0).Type()) {
		resultTypeID = stringID
	} else if isI64Slice(function.sig.Results().At(0).Type()) {
		resultTypeID = stableID("execution", "type", "slice", "i64")
	} else if isBytes(resultType) {
		resultTypeID = stableID("execution", "type", "bytes")
	} else if _, ok := goOptionValueType(resultType); ok {
		resultTypeID = goSemanticTypeIdentity(resultType)
	} else if _, _, ok := goResultTypes(resultType); ok {
		resultTypeID = goSemanticTypeIdentity(resultType)
	} else if functionSignature, ok := goFunctionSignature(resultType); ok {
		if !isUnaryI64Function(functionSignature) {
			return diagnostic("session.unsupported_result_type", "only unary i64 function values are supported")
		}
		resultTypeID = goFunctionTypeID(functionSignature)
	} else if _, named := resultType.(*types.Named); named {
		var ok bool
		resultTypeID, ok = goSupportedTypeID(resultType, integerID, booleanID, stringID, records)
		if !ok {
			return diagnostic("session.unsupported_result_type", "unsupported record result")
		}
	} else if !isInt64(resultType) {
		return diagnostic("session.unsupported_result_type", "supported result types are int64, bool, string, records, i64 slices, and state transitions")
	}
	block, err := analyzeGoBlockWithProgram(function.fn.Body.List, function.sig, function.info, functions, records)
	if err != nil {
		return diagnostic("session.unsupported_function_body", err.Error())
	}
	parameterIDs := make([]string, function.sig.Params().Len())
	var instances []graphEntity
	if isTransition {
		stateTypeID, stateOK := goSupportedTypeID(transitionState, integerID, booleanID, stringID, records)
		valueTypeID, valueOK := goSupportedTypeID(transitionResult, integerID, booleanID, stringID, records)
		if !stateOK || !valueOK {
			return diagnostic("session.unsupported_result_type", "unsupported transition type arguments")
		}
		instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "0000000000000000000000000000a004", []graphField{refField(0xa0040, stateTypeID), refField(0xa0041, valueTypeID)})})
	}
	instances = append(instances, goBridgeTypeEntities(resultType, integerID, booleanID, stringID)...)
	if isI64Slice(function.sig.Results().At(0).Type()) {
		instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "000000000000000000000000000090f8", []graphField{refField(0x9f80, integerID)})})
	}
	if _, ok := goFunctionSignature(resultType); ok {
		instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})})
	}
	for index := range parameterIDs {
		parameterTypeID := integerID
		if isBool(function.sig.Params().At(index).Type()) {
			parameterTypeID = booleanID
		} else if isPureString(function.sig.Params().At(index).Type()) {
			parameterTypeID = stringID
		} else if length, ok := fixedI64ArrayLength(function.sig.Params().At(index).Type()); ok {
			parameterTypeID = stableID("execution", "type", "fixed-array", "i64", strconv.FormatUint(length, 10))
			if !hasGraphEntity(instances, parameterTypeID) {
				instances = append(instances, graphEntity{parameterTypeID, entity(parameterTypeID, "000000000000000000000000000090f2", []graphField{refField(0x9f20, integerID), unsignedField(0x9f21, length)})})
			}
		} else if isI64Slice(function.sig.Params().At(index).Type()) {
			parameterTypeID = stableID("execution", "type", "slice", "i64")
			if !hasGraphEntity(instances, parameterTypeID) {
				instances = append(instances, graphEntity{parameterTypeID, entity(parameterTypeID, "000000000000000000000000000090f8", []graphField{refField(0x9f80, integerID)})})
			}
		} else if supported, ok := goSupportedTypeID(function.sig.Params().At(index).Type(), integerID, booleanID, stringID, records); ok {
			parameterTypeID = supported
			instances = append(instances, goBridgeTypeEntities(function.sig.Params().At(index).Type(), integerID, booleanID, stringID)...)
		} else if functionSignature, ok := goFunctionSignature(function.sig.Params().At(index).Type()); ok {
			if !isUnaryI64Function(functionSignature) {
				return diagnostic("session.unsupported_parameter_type", "only unary i64 function values are supported")
			}
			parameterTypeID = goFunctionTypeID(functionSignature)
			if !hasGraphEntity(instances, parameterTypeID) {
				instances = append(instances, graphEntity{parameterTypeID, entity(parameterTypeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})})
			}
		} else if named, ok := function.sig.Params().At(index).Type().(*types.Named); ok {
			record, exists := records[named]
			if !exists {
				return diagnostic("session.unsupported_parameter_type", "unsupported named parameter type")
			}
			parameterTypeID = record.id
		} else if !isInt64(function.sig.Params().At(index).Type()) {
			return diagnostic("session.unsupported_parameter_type", "supported parameter types are int64, bool, string, fixed i64 arrays, and i64 slices")
		}
		parameterIDs[index] = stableID("execution", function.id, "parameter", strconv.Itoa(index))
		instances = append(instances, graphEntity{parameterIDs[index], entity(parameterIDs[index], "00000000000000000000000000009012", []graphField{
			bytesField(0x9120, function.sig.Params().At(index).Name()), refField(0x9121, parameterTypeID), unsignedField(0x9122, uint64(index)),
		})})
	}
	bodyID, err := emitCanonicalBlock(block, function.id, "body", parameterIDs, integerID, &instances)
	if err != nil {
		return diagnostic("session.expression_emission", err.Error())
	}
	if function.method {
		receiverTypeID, ok := goSupportedTypeID(function.sig.Recv().Type(), integerID, booleanID, stringID, records)
		if !ok {
			return diagnostic("session.unsupported_receiver_type", "value receiver must have a supported record type")
		}
		receiverID := goReceiverID(function.sig, records)
		instances = append(instances,
			graphEntity{receiverID, entity(receiverID, "0000000000000000000000000000a000", []graphField{bytesField(0xa0000, "self"), refField(0xa0001, receiverTypeID)})},
			graphEntity{function.id, entity(function.id, "0000000000000000000000000000a002", []graphField{bytesField(0xa0020, function.name), refField(0xa0021, receiverID), refsField(0xa0022, parameterIDs), refField(0xa0023, resultTypeID), refField(0xa0024, bodyID)})},
		)
	} else {
		instances = append(instances, graphEntity{function.id, entity(function.id, "00000000000000000000000000009011", []graphField{
			bytesField(0x9110, function.name), refsField(0x9111, parameterIDs), refField(0x9112, resultTypeID), refField(0x9113, bodyID),
		})})
	}
	start := function.fset.Position(function.fn.Pos())
	end := function.fset.Position(function.fn.End())
	kind := "function"
	if function.method {
		kind = "method"
	}
	source := SourceIdentity{ID: function.id, Kind: kind, Name: function.name, Document: function.file, Start: start.Offset, End: end.Offset, Line: start.Line, Column: start.Column}
	return instances, source, nil
}

func goReceiverID(signature *types.Signature, records map[*types.Named]goRecordInfo) string {
	receiver := signature.Recv()
	value := receiver.Type()
	if pointer, ok := value.(*types.Pointer); ok {
		value = pointer.Elem()
	}
	if named, ok := value.(*types.Named); ok {
		if record, exists := findGoRecord(records, named); exists && record.receiverID != "" {
			return record.receiverID
		}
	}
	return stableID("execution", "receiver", receiver.Pkg().Path(), receiverTypeName(receiver.Type()))
}

func receiverTypeName(value types.Type) string {
	if pointer, ok := value.(*types.Pointer); ok {
		value = pointer.Elem()
	}
	if named, ok := value.(*types.Named); ok && named.Obj() != nil {
		return named.Obj().Name()
	}
	return types.TypeString(value, nil)
}

func goTransitionTypes(value types.Type) (types.Type, types.Type, bool) {
	value = types.Unalias(value)
	named, ok := value.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Name() != "Transition" || named.TypeArgs() == nil || named.TypeArgs().Len() != 2 {
		return nil, nil, false
	}
	structure, ok := named.Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 2 || structure.Field(0).Name() != "State" || structure.Field(1).Name() != "Result" {
		return nil, nil, false
	}
	state, result := named.TypeArgs().At(0), named.TypeArgs().At(1)
	if !types.Identical(structure.Field(0).Type(), state) || !types.Identical(structure.Field(1).Type(), result) {
		return nil, nil, false
	}
	return state, result, true
}

func goTransitionTypeID(state, result types.Type) string {
	return stableID("execution", "type", "state-transition", goSemanticTypeIdentity(state), goSemanticTypeIdentity(result))
}

func goSemanticTypeIdentity(value types.Type) string {
	value = types.Unalias(value)
	if isInt64(value) {
		return stableID("execution", "type", "i64")
	}
	if isBool(value) {
		return stableID("execution", "type", "bool")
	}
	if isPureString(value) {
		return stableID("execution", "type", "string")
	}
	if isBytes(value) {
		return stableID("execution", "type", "bytes")
	}
	if isI64Slice(value) {
		return stableID("execution", "type", "slice", "i64")
	}
	if isI64Map(value) {
		return stableID("execution", "type", "map", "i64", "i64")
	}
	if item, ok := goOptionValueType(value); ok {
		return stableID("execution", "type", "option", goSemanticTypeIdentity(item))
	}
	if success, failure, ok := goResultTypes(value); ok {
		return stableID("execution", "type", "result", goSemanticTypeIdentity(success), goSemanticTypeIdentity(failure))
	}
	if state, result, ok := goTransitionTypes(value); ok {
		return goTransitionTypeID(state, result)
	}
	if named, ok := value.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
		return stableID("execution", "record", named.Obj().Pkg().Path(), named.Obj().Name())
	}
	return types.TypeString(value, func(pkg *types.Package) string { return pkg.Path() })
}

func goSupportedTypeID(value types.Type, integerID, booleanID, stringID string, records map[*types.Named]goRecordInfo) (string, bool) {
	value = types.Unalias(value)
	if isInt64(value) {
		return integerID, true
	}
	if isBool(value) {
		return booleanID, true
	}
	if isPureString(value) {
		return stringID, true
	}
	if isI64Slice(value) {
		return stableID("execution", "type", "slice", "i64"), true
	}
	if isI64Map(value) {
		return stableID("execution", "type", "map", "i64", "i64"), true
	}
	if isBytes(value) {
		return stableID("execution", "type", "bytes"), true
	}
	if _, ok := goOptionValueType(value); ok {
		return goSemanticTypeIdentity(value), true
	}
	if _, _, ok := goResultTypes(value); ok {
		return goSemanticTypeIdentity(value), true
	}
	if state, result, ok := goTransitionTypes(value); ok {
		return goTransitionTypeID(state, result), true
	}
	if signature, ok := goFunctionSignature(value); ok && isUnaryI64Function(signature) {
		return goFunctionTypeID(signature), true
	}
	if named, ok := value.(*types.Named); ok {
		if record, exists := records[named]; exists {
			return record.id, true
		}
	}
	return "", false
}

func goOptionValueType(value types.Type) (types.Type, bool) {
	value = types.Unalias(value)
	named, ok := value.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Name() != "Option" || named.TypeArgs().Len() != 1 {
		return nil, false
	}
	structure, ok := named.Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 2 || structure.Field(0).Name() != "Some" || !isBool(structure.Field(0).Type()) || structure.Field(1).Name() != "Value" {
		return nil, false
	}
	item := named.TypeArgs().At(0)
	return item, types.Identical(structure.Field(1).Type(), item)
}
func goResultTypes(value types.Type) (types.Type, types.Type, bool) {
	value = types.Unalias(value)
	named, ok := value.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Name() != "Result" || named.TypeArgs().Len() != 2 {
		return nil, nil, false
	}
	structure, ok := named.Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 3 || structure.Field(0).Name() != "Ok" || !isBool(structure.Field(0).Type()) || structure.Field(1).Name() != "Value" || structure.Field(2).Name() != "Error" {
		return nil, nil, false
	}
	success, failure := named.TypeArgs().At(0), named.TypeArgs().At(1)
	return success, failure, types.Identical(structure.Field(1).Type(), success) && types.Identical(structure.Field(2).Type(), failure)
}
func goBridgeTypeEntities(value types.Type, integerID, booleanID, stringID string) []graphEntity {
	if isI64Slice(value) {
		id := stableID("execution", "type", "slice", "i64")
		return []graphEntity{{id, entity(id, "000000000000000000000000000090f8", []graphField{refField(0x9f80, integerID)})}}
	}
	if isI64Map(value) {
		id := stableID("execution", "type", "map", "i64", "i64")
		return []graphEntity{{id, entity(id, "0000000000000000000000000000a040", []graphField{refField(0xa0400, integerID), refField(0xa0401, integerID)})}}
	}
	if isBytes(value) {
		id := stableID("execution", "type", "bytes")
		return []graphEntity{{id, entity(id, "00000000000000000000000000009041", nil)}}
	}
	if item, ok := goOptionValueType(value); ok {
		id, itemID := goSemanticTypeIdentity(value), goSemanticTypeIdentity(item)
		return append(goBridgeTypeEntities(item, integerID, booleanID, stringID), graphEntity{id, entity(id, "0000000000000000000000000000a050", []graphField{refField(0xa0500, itemID)})})
	}
	if success, failure, ok := goResultTypes(value); ok {
		id, successID, failureID := goSemanticTypeIdentity(value), goSemanticTypeIdentity(success), goSemanticTypeIdentity(failure)
		entities := append(goBridgeTypeEntities(success, integerID, booleanID, stringID), goBridgeTypeEntities(failure, integerID, booleanID, stringID)...)
		return append(entities, graphEntity{id, entity(id, "00000000000000000000000000009042", []graphField{refField(0x9400, successID), refField(0x9401, failureID)})})
	}
	if state, result, ok := goTransitionTypes(value); ok {
		id, stateID, resultID := goTransitionTypeID(state, result), goSemanticTypeIdentity(state), goSemanticTypeIdentity(result)
		entities := append(goBridgeTypeEntities(state, integerID, booleanID, stringID), goBridgeTypeEntities(result, integerID, booleanID, stringID)...)
		return append(entities, graphEntity{id, entity(id, "0000000000000000000000000000a004", []graphField{refField(0xa0040, stateID), refField(0xa0041, resultID)})})
	}
	return nil
}

func isI64Map(value types.Type) bool {
	mapping, ok := value.Underlying().(*types.Map)
	return ok && isInt64(mapping.Key()) && isInt64(mapping.Elem())
}

func goFunctionSignature(value types.Type) (*types.Signature, bool) {
	value = types.Unalias(value)
	if named, ok := value.(*types.Named); ok {
		value = named.Underlying()
	}
	signature, ok := value.(*types.Signature)
	return signature, ok
}

func isUnaryI64Function(signature *types.Signature) bool {
	return signature != nil && signature.Params().Len() == 1 && isInt64(signature.Params().At(0).Type()) && signature.Results().Len() == 1 && isInt64(signature.Results().At(0).Type())
}

func goFunctionTypeID(signature *types.Signature) string {
	return stableID("execution", "type", "function", "i64", "i64")
}

func snapshotDigest(snapshot DocumentSnapshot) string {
	paths := make([]string, 0, len(snapshot.Files))
	for path := range snapshot.Files {
		paths = append(paths, filepath.ToSlash(path))
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, path := range paths {
		digest.Write([]byte(path))
		digest.Write([]byte{0})
		digest.Write([]byte(snapshot.Files[path]))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func parseDiagnostics(err error) []SessionDiagnostic {
	var result []SessionDiagnostic
	if list, ok := err.(scanner.ErrorList); ok {
		for _, item := range list {
			result = append(result, SessionDiagnostic{Code: "go.parse", Message: item.Msg, File: filepath.ToSlash(item.Pos.Filename), Line: item.Pos.Line, Column: item.Pos.Column, Severity: "error"})
		}
		return result
	}
	return []SessionDiagnostic{{Code: "go.parse", Message: err.Error(), Severity: "error"}}
}

func typeDiagnostic(err error) SessionDiagnostic {
	if typed, ok := err.(types.Error); ok {
		code := "go.type"
		if strings.Contains(typed.Msg, "go.local_import_missing:") {
			code = "go.local_import_missing"
		} else if strings.Contains(typed.Msg, "go.import_cycle:") {
			code = "go.import_cycle"
		} else if strings.Contains(typed.Msg, "go.external_import_unsupported:") {
			code = "go.external_import_unsupported"
		}
		return SessionDiagnostic{Code: code, Message: typed.Msg, File: filepath.ToSlash(typed.Fset.Position(typed.Pos).Filename), Line: typed.Fset.Position(typed.Pos).Line, Column: typed.Fset.Position(typed.Pos).Column, Severity: "error"}
	}
	return SessionDiagnostic{Code: "go.type", Message: err.Error(), Severity: "error"}
}

func sortedDiagnostics(diagnostics []SessionDiagnostic) []SessionDiagnostic {
	sort.SliceStable(diagnostics, func(left, right int) bool {
		if diagnostics[left].File != diagnostics[right].File {
			return diagnostics[left].File < diagnostics[right].File
		}
		if diagnostics[left].Line != diagnostics[right].Line {
			return diagnostics[left].Line < diagnostics[right].Line
		}
		if diagnostics[left].Column != diagnostics[right].Column {
			return diagnostics[left].Column < diagnostics[right].Column
		}
		return diagnostics[left].Code < diagnostics[right].Code
	})
	return diagnostics
}
