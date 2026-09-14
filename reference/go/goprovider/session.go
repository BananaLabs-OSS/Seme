package goprovider

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
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
	// GOOS and GOARCH select the source files that belong to this Go realization.
	// Empty values use the Go toolchain's current target.
	GOOS   string
	GOARCH string
	// ExternalImporter supplies types for dependencies that remain outside
	// canonical Seme ownership, such as module-aware native islands.
	ExternalImporter types.Importer
	// IdentityBindings are explicit reconciliation authority supplied by a
	// semantic patch. Ordinary ingestion leaves this empty and derives IDs.
	IdentityBindings []IdentityBinding
}

func snapshotTarget(snapshot DocumentSnapshot) (string, string) {
	goos, goarch := snapshot.GOOS, snapshot.GOARCH
	if goos == "" {
		goos = build.Default.GOOS
	}
	if goarch == "" {
		goarch = build.Default.GOARCH
	}
	return goos, goarch
}

var knownGOOS = map[string]bool{"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true, "hurd": true, "illumos": true, "ios": true, "js": true, "linux": true, "netbsd": true, "openbsd": true, "plan9": true, "solaris": true, "wasip1": true, "windows": true}
var knownGOARCH = map[string]bool{"386": true, "amd64": true, "arm": true, "arm64": true, "loong64": true, "mips": true, "mips64": true, "mips64le": true, "mipsle": true, "ppc64": true, "ppc64le": true, "riscv64": true, "s390x": true, "sparc64": true, "wasm": true}

func snapshotFileMatches(snapshot DocumentSnapshot, name, source string) bool {
	goos, goarch := snapshotTarget(snapshot)
	base := strings.TrimSuffix(filepath.Base(name), ".go")
	parts := strings.Split(base, "_")
	if n := len(parts); n > 1 {
		last := parts[n-1]
		if knownGOOS[last] && last != goos || knownGOARCH[last] && last != goarch {
			return false
		}
		if n > 2 && knownGOOS[parts[n-2]] && parts[n-2] != goos {
			return false
		}
	}
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//go:build ") {
			expr, err := constraint.Parse(trimmed)
			if err != nil {
				return true
			}
			return expr.Eval(func(tag string) bool { return tag == goos || tag == goarch || tag == "gc" })
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			break
		}
	}
	return true
}

type IdentityBinding struct {
	ID, Kind, Document, Name string
	Start                    int
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

// ReferenceOccurrence binds a source-language use site to a stable semantic
// declaration identity. Consumers navigate by TargetID, never by name guesses.
type ReferenceOccurrence struct {
	TargetID string `json:"target_id"`
	Document string `json:"document"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type SessionResult struct {
	Revision          uint64
	Accepted          bool
	Valid             bool
	CanonicalG1       string
	LastValidRevision uint64
	Diagnostics       []SessionDiagnostic
	Sources           []SourceIdentity
	References        []ReferenceOccurrence
	ContentDigest     string
	Disposition       string
	Packages          []PackageMetadata
	Resolution        ResolutionManifest
	NativeIslands     []NativeIslandDeclaration
}

// NativeIslandDeclaration preserves a fully type-checked Go declaration whose
// body is not yet canonical. It is an honest realization boundary, not omitted
// source and not a claim of cross-language equivalence.
type NativeIslandDeclaration struct {
	ID, Name, Package, Document, Signature, Reason, Detail, Receiver string
	Parameters, ParameterTypes, ResultTypes                          []string
	Line, Column                                                     int
	Method                                                           bool
	Variadic                                                         bool
}

func nativeParameterNames(signature *types.Signature) []string {
	parameters := make([]string, signature.Params().Len())
	for index := range parameters {
		name := signature.Params().At(index).Name()
		if name == "" || name == "_" {
			name = "arg" + strconv.Itoa(index+1)
		}
		parameters[index] = name
	}
	return parameters
}

func nativeTupleTypes(tuple *types.Tuple) []string {
	values := make([]string, tuple.Len())
	for index := range values {
		values[index] = types.TypeString(tuple.At(index).Type(), func(pkg *types.Package) string {
			if pkg == nil {
				return ""
			}
			return pkg.Path()
		})
	}
	return values
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
	Aliases      []SourceAliasMetadata
}

// SourceAliasMetadata preserves a source-language presentation declaration
// whose runtime meaning is exactly its canonical Execution target. Aliases do
// not become Core or Execution types.
type SourceAliasMetadata struct {
	Name, Package, Target, Document              string
	Exported, Generic                            bool
	Start, End, Line, Column, EndLine, EndColumn int
	ReferencedImports                            []string
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
	Declaration, Package, Name, OwnershipName, ExportName, GenericDefinition string
	Kind                                                                     SemanticDeclarationKind
	Exported                                                                 bool
	Origin                                                                   ProjectLocation
	ReferencedImports                                                        []string
	// ImportReferences retain the exact checked import-spec locations needed
	// to reconcile paths to neutral Package ImportBinding identities.
	ImportReferences []SemanticImportReference
}

type SemanticImportReference struct {
	Alias, Requested, Resolved string
	Local                      bool
	Location                   ProjectLocation
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
	lastValidReferences []ReferenceOccurrence
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
			References:    cloneReferences(session.lastValidReferences),
			Packages:      clonePackageMetadata(session.lastValidPackages),
			Resolution:    cloneResolutionManifest(session.lastValidResolution),
			ContentDigest: snapshotDigest(snapshot), Disposition: "rejected-stale",
		}
	}
	session.currentRevision = snapshot.Revision
	resolution, resolutionDiagnostics := resolveSnapshot(snapshot)
	graph, sources, references, packages, islands, diagnostics := "", []SourceIdentity(nil), []ReferenceOccurrence(nil), []PackageMetadata(nil), []NativeIslandDeclaration(nil), resolutionDiagnostics
	if len(diagnostics) == 0 {
		graph, sources, references, packages, islands, diagnostics = liftDocumentSnapshot(snapshot, session.moduleG1)
	}
	valid := graph != ""
	if valid {
		attachResolutionOwnership(&resolution, packages)
		session.lastValidRevision = snapshot.Revision
		session.lastValidGraph = graph
		session.lastValidSources = cloneSources(sources)
		session.lastValidReferences = cloneReferences(references)
		session.lastValidPackages = clonePackageMetadata(packages)
		session.lastValidResolution = cloneResolutionManifest(resolution)
	}
	return SessionResult{
		Revision: snapshot.Revision, Accepted: true, Valid: valid,
		CanonicalG1: session.lastValidGraph, LastValidRevision: session.lastValidRevision,
		Diagnostics: diagnostics, Sources: cloneSources(session.lastValidSources),
		References:    cloneReferences(session.lastValidReferences),
		Packages:      clonePackageMetadata(session.lastValidPackages),
		Resolution:    cloneResolutionManifest(session.lastValidResolution),
		NativeIslands: append([]NativeIslandDeclaration(nil), islands...),
		ContentDigest: snapshotDigest(snapshot), Disposition: map[bool]string{true: "accepted-valid", false: "accepted-invalid"}[valid],
	}
}

func attachResolutionOwnership(resolution *ResolutionManifest, packages []PackageMetadata) {
	byName := make(map[string][]SemanticDeclarationMetadata, len(packages))
	for _, item := range packages {
		byName[item.Name] = item.Supplemental
	}
	for i := range resolution.Packages {
		supplemental := cloneSemanticDeclarations(byName[resolution.Packages[i].Name])
		resolution.Packages[i].Supplemental = supplemental
		for _, item := range supplemental {
			resolution.Packages[i].Declarations = append(resolution.Packages[i].Declarations, ResolvedDeclaration{ID: item.Declaration, Name: item.OwnershipName, Exported: item.Exported, Location: item.Origin})
		}
		sort.Slice(resolution.Packages[i].Declarations, func(a, b int) bool {
			return resolution.Packages[i].Declarations[a].ID < resolution.Packages[i].Declarations[b].ID
		})
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
		out[i].Aliases = append([]SourceAliasMetadata(nil), p.Aliases...)
		for j := range out[i].Aliases {
			out[i].Aliases[j].ReferencedImports = append([]string(nil), p.Aliases[j].ReferencedImports...)
		}
	}
	return out
}

func cloneSemanticDeclarations(in []SemanticDeclarationMetadata) []SemanticDeclarationMetadata {
	out := make([]SemanticDeclarationMetadata, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].ReferencedImports = append([]string(nil), in[i].ReferencedImports...)
		out[i].ImportReferences = append([]SemanticImportReference(nil), in[i].ImportReferences...)
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

func cloneReferences(references []ReferenceOccurrence) []ReferenceOccurrence {
	return append([]ReferenceOccurrence(nil), references...)
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
	standard    types.Importer
	matched     map[string]bool
}

func (loader *snapshotSourceImporter) boundIdentity(node *ast.FuncDecl, fset *token.FileSet) string {
	position := fset.Position(node.Pos())
	document := filepath.ToSlash(position.Filename)
	kind := "function"
	if node.Recv != nil {
		kind = "method"
	}
	for _, binding := range loader.snapshot.IdentityBindings {
		if binding.Document == document && binding.Start == position.Offset && binding.Name == node.Name.Name && binding.Kind == kind {
			loader.matched[binding.Document+"\x00"+strconv.Itoa(binding.Start)] = true
			return binding.ID
		}
	}
	return ""
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
		if loader.snapshot.ExternalImporter != nil {
			pkg, err := loader.snapshot.ExternalImporter.Import(path)
			if err != nil {
				return nil, fmt.Errorf("go.external_import_unsupported:%s", path)
			}
			return pkg, nil
		}
		// Native Go dependencies may supply type information without claiming
		// canonical Seme ownership. The configured build context bounds where the
		// source importer can resolve them; missing packages remain unsupported.
		_, err := build.Default.Import(path, "", build.FindOnly)
		if err != nil {
			return nil, fmt.Errorf("go.external_import_unsupported:%s", path)
		}
		if loader.standard == nil {
			loader.standard = importer.ForCompiler(token.NewFileSet(), "source", nil)
		}
		return loader.standard.Import(path)
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
				id := semeIdentityDirective(node.Doc)
				if id == "" {
					id = loader.boundIdentity(node, fset)
				}
				if id != "" {
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
	seenBindings, seenBindingIDs := map[string]bool{}, map[string]bool{}
	for _, binding := range snapshot.IdentityBindings {
		key := binding.Document + "\x00" + strconv.Itoa(binding.Start)
		if len(binding.ID) != 32 || binding.Start < 0 || binding.Document == "" || binding.Name == "" || (binding.Kind != "function" && binding.Kind != "method") {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.identity_binding_invalid", Message: "identity continuity binding is malformed", File: binding.Document, Severity: "error"})
			continue
		}
		if _, err := hex.DecodeString(binding.ID); err != nil || seenBindings[key] || seenBindingIDs[binding.ID] {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.identity_binding_invalid", Message: "identity continuity binding is malformed or duplicated", File: binding.Document, Severity: "error"})
			continue
		}
		seenBindings[key] = true
		seenBindingIDs[binding.ID] = true
	}
	if len(diagnostics) != 0 {
		return nil, sortedDiagnostics(diagnostics)
	}
	for name, source := range snapshot.Files {
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") || !snapshotFileMatches(snapshot, name, source) {
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
	loader := &snapshotSourceImporter{snapshot: snapshot, groups: groups, loaded: map[string]*checkedSessionPackage{}, checking: map[string]bool{}, diagnostics: &diagnostics, matched: map[string]bool{}}
	if _, err := loader.Import(snapshot.PackagePath); err != nil && len(diagnostics) == 0 {
		diagnostics = append(diagnostics, SessionDiagnostic{Code: "go.type", Message: err.Error(), Severity: "error"})
	}
	// A project snapshot is the complete declared source set, not merely the
	// transitive import closure of its selected executable entry. Load every
	// package group deterministically so disconnected libraries remain visible
	// to package contracts and later project-wide selections.
	allPaths := make([]string, 0, len(groups))
	for path := range groups {
		allPaths = append(allPaths, path)
	}
	sort.Strings(allPaths)
	for _, path := range allPaths {
		if _, present := loader.loaded[path]; present {
			continue
		}
		if _, err := loader.Import(path); err != nil && len(diagnostics) == 0 {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "go.type", Message: err.Error(), Severity: "error"})
		}
	}
	for key := range seenBindings {
		if !loader.matched[key] {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.identity_binding_unmatched", Message: "identity continuity binding did not resolve exactly", Severity: "error"})
		}
	}
	if len(diagnostics) != 0 {
		return nil, sortedDiagnostics(diagnostics)
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

func liftDocumentSnapshot(snapshot DocumentSnapshot, moduleG1 []byte) (string, []SourceIdentity, []ReferenceOccurrence, []PackageMetadata, []NativeIslandDeclaration, []SessionDiagnostic) {
	if snapshot.PackagePath == "" {
		return "", nil, nil, nil, nil, []SessionDiagnostic{{Code: "session.package_path_missing", Message: "package path is required", Severity: "error"}}
	}
	units, diagnostics := checkSessionPackages(snapshot)
	if len(diagnostics) != 0 || len(units) == 0 {
		return "", nil, nil, nil, nil, diagnostics
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
	if executionModuleVersion(moduleG1) >= 43 {
		// nil is reserved metadata, never a Go declaration object.
		functionObjects[nil] = "core-v43"
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
	type recordCandidate struct {
		named     *types.Named
		name      string
		id        string
		structure *types.Struct
	}
	var recordCandidates []recordCandidate
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
			recordCandidates = append(recordCandidates, recordCandidate{named: named, name: identifier.Name, id: recordID, structure: structure})
		}
	}
	sort.Slice(recordCandidates, func(i, j int) bool { return recordCandidates[i].id < recordCandidates[j].id })
	records := make(map[*types.Named]goRecordInfo)
	visitingRecords := make(map[*types.Named]bool)
	invalidRecords := make(map[*types.Named]bool)
	var buildRecord func(*types.Named, int) (goRecordInfo, bool)
	findCandidate := func(named *types.Named) (recordCandidate, bool) {
		origin := named.Origin()
		for _, candidate := range recordCandidates {
			if candidate.named == named || candidate.named == origin || sameNamedType(candidate.named, origin) {
				return candidate, true
			}
		}
		return recordCandidate{}, false
	}
	var validateRecord func(*types.Named, map[*types.Named]bool, int) bool
	validateRecord = func(named *types.Named, visiting map[*types.Named]bool, depth int) bool {
		candidate, exists := findCandidate(named)
		if !exists || depth > 32 || visiting[candidate.named] {
			return false
		}
		visiting[candidate.named] = true
		defer delete(visiting, candidate.named)
		for index := 0; index < candidate.structure.NumFields(); index++ {
			fieldType := candidate.structure.Field(index).Type()
			if isBool(fieldType) || isPureString(fieldType) || isPrimitiveSlice(fieldType) || isI64Map(fieldType) || isBytes(fieldType) || isInt64(fieldType) {
				continue
			}
			nested, ok := types.Unalias(fieldType).(*types.Named)
			if !ok || !validateRecord(nested, visiting, depth+1) {
				return false
			}
		}
		return true
	}
	buildRecord = func(named *types.Named, depth int) (goRecordInfo, bool) {
		candidate, exists := findCandidate(named)
		if !exists || depth > 32 || len(records) >= 512 || invalidRecords[candidate.named] || visitingRecords[candidate.named] {
			return goRecordInfo{}, false
		}
		if record, exists := records[candidate.named]; exists {
			return record, true
		}
		visitingRecords[candidate.named] = true
		defer delete(visitingRecords, candidate.named)
		record := goRecordInfo{id: candidate.id, fields: map[*types.Var]string{}}
		fieldIDs := make([]string, candidate.structure.NumFields())
		var fieldEntities []graphEntity
		for index := 0; index < candidate.structure.NumFields(); index++ {
			field := candidate.structure.Field(index)
			typeID := integerID
			switch {
			case isBool(field.Type()):
				typeID = booleanID
			case isPureString(field.Type()):
				typeID = stringID
			case isPrimitiveSlice(field.Type()) || isI64Map(field.Type()) || isBytes(field.Type()):
				typeID = goSemanticTypeIdentity(field.Type())
				for _, typeEntity := range goBridgeTypeEntities(field.Type(), integerID, booleanID, stringID) {
					if !hasGraphEntity(instances, typeEntity.id) {
						instances = append(instances, typeEntity)
					}
				}
			case isInt64(field.Type()):
			default:
				nested, ok := types.Unalias(field.Type()).(*types.Named)
				if !ok {
					invalidRecords[candidate.named] = true
					return goRecordInfo{}, false
				}
				nestedRecord, ok := buildRecord(nested, depth+1)
				if !ok {
					invalidRecords[candidate.named] = true
					return goRecordInfo{}, false
				}
				typeID = nestedRecord.id
			}
			fieldID := stableID("execution", candidate.id, "field", strconv.Itoa(index))
			fieldIDs[index] = fieldID
			record.fields[field] = fieldID
			record.ordered = append(record.ordered, field)
			fieldEntities = append(fieldEntities, graphEntity{fieldID, entity(fieldID, "00000000000000000000000000009031", []graphField{
				bytesField(0x9310, field.Name()), refField(0x9311, typeID), unsignedField(0x9312, uint64(index)),
			})})
		}
		records[candidate.named] = record
		records[candidate.named.Origin()] = record
		instances = append(instances, fieldEntities...)
		instances = append(instances, graphEntity{candidate.id, entity(candidate.id, "00000000000000000000000000009030", []graphField{bytesField(0x9300, candidate.name), refsField(0x9301, fieldIDs)})})
		return record, true
	}
	for _, candidate := range recordCandidates {
		if validateRecord(candidate.named, map[*types.Named]bool{}, 1) {
			buildRecord(candidate.named, 1)
		} else {
			invalidRecords[candidate.named] = true
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
	var islands []NativeIslandDeclaration
	for _, function := range functions {
		version := executionModuleVersion(moduleG1)
		entities, source, diagnostic := liftSessionFunction(function, integerID, booleanID, stringID, functionObjects, records, version >= 41, version >= 42)
		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
			position := function.fset.Position(function.fn.Pos())
			receiver := ""
			if function.sig.Recv() != nil {
				receiver = types.TypeString(function.sig.Recv().Type(), func(pkg *types.Package) string {
					if pkg == nil {
						return ""
					}
					return pkg.Path()
				})
			}
			islands = append(islands, NativeIslandDeclaration{ID: function.id, Name: function.name, Package: function.packagePath, Document: function.file, Signature: types.TypeString(function.sig, func(pkg *types.Package) string {
				if pkg == nil {
					return ""
				}
				return pkg.Path()
			}), Reason: diagnostic.Code, Detail: diagnostic.Message, Receiver: receiver, Parameters: nativeParameterNames(function.sig), ParameterTypes: nativeTupleTypes(function.sig.Params()), ResultTypes: nativeTupleTypes(function.sig.Results()), Variadic: function.sig.Variadic(), Line: position.Line, Column: position.Column, Method: function.method})
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
			return "", nil, nil, nil, islands, sortedDiagnostics(diagnostics)
		}
	}
	if len(functionIDs) == 0 {
		if len(diagnostics) == 0 {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.no_supported_declarations", Message: "snapshot contains no supported package functions", Severity: "error"})
		}
		return "", nil, nil, nil, islands, sortedDiagnostics(diagnostics)
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
		return "", nil, nil, nil, islands, sortedDiagnostics(diagnostics)
	}
	programID := stableID("session-program", snapshot.PackagePath)
	instances = append(instances, graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{
		refsField(0x9150, functionIDs), refField(0x9151, entryID),
	})})
	revision := stableID("session-revision", snapshot.PackagePath, strconv.FormatUint(snapshot.Revision, 10))
	metadata := buildPackageMetadata(snapshot.PackagePath, units, functions, supportedFunctions, instances)
	if diagnostic := attachSemanticOwnership(metadata, units, functions, instances); diagnostic != nil {
		diagnostics = append(diagnostics, *diagnostic)
		return "", nil, nil, nil, islands, sortedDiagnostics(diagnostics)
	}
	sourceIDs := map[string]bool{}
	for _, source := range sources {
		sourceIDs[source.ID] = true
	}
	var references []ReferenceOccurrence
	for _, unit := range units {
		for identifier, object := range unit.info.Uses {
			target := functionObjects[object]
			if target == "" || !sourceIDs[target] {
				continue
			}
			start, end := unit.fset.Position(identifier.Pos()), unit.fset.Position(identifier.End())
			references = append(references, ReferenceOccurrence{TargetID: target, Document: filepath.ToSlash(start.Filename), Start: start.Offset, End: end.Offset, Line: start.Line, Column: start.Column})
		}
	}
	sort.Slice(references, func(i, j int) bool {
		if references[i].Document != references[j].Document {
			return references[i].Document < references[j].Document
		}
		if references[i].Start != references[j].Start {
			return references[i].Start < references[j].Start
		}
		return references[i].TargetID < references[j].TargetID
	})
	return composeExecutionG1(moduleG1, revision, instances), sources, references, metadata, islands, sortedDiagnostics(diagnostics)
}

func buildPackageMetadata(root string, units []*checkedSessionPackage, functions []sessionFunction, supported map[string]bool, instances []graphEntity) []PackageMetadata {
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
			result := stableID("execution", "type", "unit")
			if f.sig.Results().Len() != 0 {
				result = goSemanticTypeIdentity(f.sig.Results().At(0).Type())
			}
			m := PackageFunctionMetadata{ID: f.id, Name: f.name, Result: result, Exported: ast.IsExported(f.name), Document: f.file, Line: position.Line, Column: position.Column}
			for i := 0; i < f.sig.Params().Len(); i++ {
				m.Parameters = append(m.Parameters, goSemanticTypeIdentity(f.sig.Params().At(i).Type()))
			}
			p.Members = append(p.Members, m)
			if m.Exported {
				p.Functions = append(p.Functions, m)
			}
		}
		for identifier, object := range u.info.Defs {
			typeName, ok := object.(*types.TypeName)
			if !ok || !typeName.IsAlias() {
				continue
			}
			target := goSemanticTypeIdentity(typeName.Type())
			if !hasGraphEntity(instances, target) {
				continue
			}
			start := u.fset.Position(identifier.Pos())
			end := u.fset.Position(identifier.End())
			imports := typePackagePaths(typeName.Type(), u.path)
			p.Aliases = append(p.Aliases, SourceAliasMetadata{Name: identifier.Name, Package: u.path, Target: target, Document: filepath.ToSlash(start.Filename), Exported: ast.IsExported(identifier.Name), Generic: false, Start: start.Offset, End: end.Offset, Line: start.Line, Column: start.Column, EndLine: end.Line, EndColumn: end.Column, ReferencedImports: imports})
		}
		sort.Slice(p.Members, func(i, j int) bool { return p.Members[i].ID < p.Members[j].ID })
		sort.Slice(p.Functions, func(i, j int) bool { return p.Functions[i].ID < p.Functions[j].ID })
		sort.Slice(p.Aliases, func(i, j int) bool { return p.Aliases[i].Name < p.Aliases[j].Name })
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func typePackagePaths(value types.Type, self string) []string {
	seen := map[types.Type]bool{}
	paths := map[string]bool{}
	var visit func(types.Type)
	visit = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch x := t.(type) {
		case *types.Alias:
			visit(types.Unalias(x))
		case *types.Named:
			if x.Obj() != nil && x.Obj().Pkg() != nil && x.Obj().Pkg().Path() != self {
				paths[x.Obj().Pkg().Path()] = true
			}
			if x.TypeArgs() != nil {
				for i := 0; i < x.TypeArgs().Len(); i++ {
					visit(x.TypeArgs().At(i))
				}
			}
		case *types.Pointer:
			visit(x.Elem())
		case *types.Slice:
			visit(x.Elem())
		case *types.Array:
			visit(x.Elem())
		case *types.Map:
			visit(x.Key())
			visit(x.Elem())
		case *types.Signature:
			for i := 0; i < x.Params().Len(); i++ {
				visit(x.Params().At(i).Type())
			}
			for i := 0; i < x.Results().Len(); i++ {
				visit(x.Results().At(i).Type())
			}
		}
	}
	visit(value)
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
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

func liftSessionFunction(function sessionFunction, integerID, booleanID, stringID string, functions map[types.Object]string, records map[*types.Named]goRecordInfo, allowProducts, allowNativeOwnedTypes bool) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
	position := function.fset.Position(function.fn.Pos())
	diagnostic := func(code, message string) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
		return nil, SourceIdentity{}, &SessionDiagnostic{Code: code, Message: message, File: function.file, Line: position.Line, Column: position.Column, Severity: "warning"}
	}
	unitResult := function.sig.Results().Len() == 0
	multiResult := function.sig.Results().Len() > 1
	if multiResult && !allowProducts {
		return diagnostic("session.unsupported_function_shape", "multiple results require Core Execution v41")
	}
	var resultType types.Type
	if multiResult {
		resultType = function.sig.Results()
	} else if !unitResult {
		resultType = function.sig.Results().At(0).Type()
	}
	resultTypeID := integerID
	resultNative := false
	var transitionState, transitionResult types.Type
	isTransition := false
	if !unitResult {
		transitionState, transitionResult, isTransition = goTransitionTypes(resultType)
	}
	if unitResult {
		resultTypeID = stableID("execution", "type", "unit")
	} else if multiResult {
		var ok bool
		resultTypeID, _, ok = goProductTypeID(function.sig.Results(), records)
		if !ok {
			return diagnostic("session.unsupported_result_type", "product result items must have supported semantic types")
		}
	} else if isTransition {
		var ok bool
		resultTypeID, ok = goSupportedTypeID(resultType, integerID, booleanID, stringID, records)
		if !ok {
			return diagnostic("session.unsupported_result_type", "transition state and result types must be supported")
		}
	} else if isBool(resultType) {
		resultTypeID = booleanID
	} else if isPureString(function.sig.Results().At(0).Type()) {
		resultTypeID = stringID
	} else if element, ok := goPrimitiveSliceElement(function.sig.Results().At(0).Type()); ok {
		tag, _ := goPrimitiveTypeTag(element)
		resultTypeID = stableID("execution", "type", "slice", tag)
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
		if !ok && allowNativeOwnedTypes && goTypeOwnedOutsidePackage(resultType, function.packagePath) {
			resultTypeID, ok = goNativeTypeID(resultType)
			resultNative = ok
		}
		if !ok {
			return diagnostic("session.unsupported_result_type", "unsupported record result")
		}
	} else if !isInt64(resultType) {
		var ok bool
		if allowNativeOwnedTypes && goTypeOwnedOutsidePackage(resultType, function.packagePath) {
			resultTypeID, ok = goNativeTypeID(resultType)
			resultNative = ok
		}
		if !ok {
			return diagnostic("session.unsupported_result_type", "supported result types are neutral values or explicitly Go-owned native values")
		}
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
	if !unitResult {
		instances = append(instances, goBridgeTypeEntities(resultType, integerID, booleanID, stringID)...)
		if resultNative {
			instances = append(instances, goNativeTypeEntity(resultType))
		}
	}
	if unitResult {
		instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "0000000000000000000000000000a06a", nil)})
	}
	if !unitResult {
		if _, ok := goFunctionSignature(resultType); ok {
			instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})})
		}
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
		} else if supported, ok := goSupportedTypeID(function.sig.Params().At(index).Type(), integerID, booleanID, stringID, records); ok {
			parameterTypeID = supported
			instances = append(instances, goBridgeTypeEntities(function.sig.Params().At(index).Type(), integerID, booleanID, stringID)...)
		} else if allowNativeOwnedTypes && goTypeOwnedOutsidePackage(function.sig.Params().At(index).Type(), function.packagePath) {
			parameterTypeID, _ = goNativeTypeID(function.sig.Params().At(index).Type())
			instances = append(instances, goNativeTypeEntity(function.sig.Params().At(index).Type()))
		} else if functionSignature, ok := goFunctionSignature(function.sig.Params().At(index).Type()); ok {
			if !isUnaryI64Function(functionSignature) {
				return diagnostic("session.unsupported_parameter_type", "only unary i64 function values are supported")
			}
			parameterTypeID = goFunctionTypeID(functionSignature)
			if !hasGraphEntity(instances, parameterTypeID) {
				instances = append(instances, graphEntity{parameterTypeID, entity(parameterTypeID, "0000000000000000000000000000a020", []graphField{refsField(0xa0200, []string{integerID}), refField(0xa0201, integerID)})})
			}
		} else if named, ok := types.Unalias(function.sig.Params().At(index).Type()).(*types.Named); ok {
			record, exists := findGoRecord(records, named)
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
	if tuple, ok := value.(*types.Tuple); ok {
		parts := []string{"execution", "type", "product"}
		for index := 0; index < tuple.Len(); index++ {
			parts = append(parts, goSemanticTypeIdentity(tuple.At(index).Type()))
		}
		return stableID(parts...)
	}
	if isInt64(value) {
		return stableID("execution", "type", "i64")
	}
	if isGoErrorType(value) {
		return stableID("execution", "type", "native", "go", "error")
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
	if element, ok := goPrimitiveSliceElement(value); ok {
		tag, _ := goPrimitiveTypeTag(element)
		return stableID("execution", "type", "slice", tag)
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
	if tuple, ok := value.(*types.Tuple); ok {
		id, _, supported := goProductTypeID(tuple, records)
		return id, supported
	}
	if isGoErrorType(value) {
		return stableID("execution", "type", "native", "go", "error"), true
	}
	if isInt64(value) {
		return integerID, true
	}
	if isBool(value) {
		return booleanID, true
	}
	if isPureString(value) {
		return stringID, true
	}
	if element, ok := goPrimitiveSliceElement(value); ok {
		tag, _ := goPrimitiveTypeTag(element)
		return stableID("execution", "type", "slice", tag), true
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
	if named, ok := types.Unalias(value).(*types.Named); ok {
		if record, exists := findGoRecord(records, named); exists {
			return record.id, true
		}
	}
	return "", false
}

func goNativeTypeID(value types.Type) (string, bool) {
	spelling, ok := goNativeTypeSpelling(value)
	if !ok {
		return "", false
	}
	return stableID("execution", "type", "native", "go", spelling), true
}

func goNativeTypeEntity(value types.Type) graphEntity {
	spelling, _ := goNativeTypeSpelling(value)
	id := stableID("execution", "type", "native", "go", spelling)
	return graphEntity{id, entity(id, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, spelling)})}
}

// goTypeOwnedOutsidePackage prevents an unsupported local record from being
// mislabeled as a runtime boundary. Predeclared Go mechanics (such as int) and
// types declared by imported packages are legitimately owned outside the
// package currently being lifted.
func goTypeOwnedOutsidePackage(value types.Type, packagePath string) bool {
	value = types.Unalias(value)
	switch typed := value.(type) {
	case *types.Basic:
		return true
	case *types.Named:
		return typed.Obj() != nil && (typed.Obj().Pkg() == nil || typed.Obj().Pkg().Path() != packagePath)
	case *types.Pointer:
		return goTypeOwnedOutsidePackage(typed.Elem(), packagePath)
	case *types.Slice:
		return goTypeOwnedOutsidePackage(typed.Elem(), packagePath)
	case *types.Array:
		return goTypeOwnedOutsidePackage(typed.Elem(), packagePath)
	case *types.Map:
		return goTypeOwnedOutsidePackage(typed.Key(), packagePath) && goTypeOwnedOutsidePackage(typed.Elem(), packagePath)
	case *types.Chan, *types.Signature, *types.Interface:
		return true
	default:
		return false
	}
}

// goNativeTypeSpelling retains a type that Go owns without importing its
// mechanics into neutral Core. The fully-qualified go/types spelling is the
// realization contract; aliases are resolved so the identity is stable.
func goNativeTypeSpelling(value types.Type) (string, bool) {
	if value == nil {
		return "", false
	}
	value = types.Unalias(value)
	if _, tuple := value.(*types.Tuple); tuple {
		return "", false
	}
	spelling := types.TypeString(value, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
	return spelling, spelling != "" && spelling != "invalid type"
}

func goProductTypeID(tuple *types.Tuple, records map[*types.Named]goRecordInfo) (string, []string, bool) {
	if tuple == nil || tuple.Len() < 2 || tuple.Len() > 16 {
		return "", nil, false
	}
	itemTypes := make([]string, tuple.Len())
	parts := []string{"execution", "type", "product"}
	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	stringID := stableID("execution", "type", "string")
	for index := 0; index < tuple.Len(); index++ {
		item, ok := goSupportedTypeID(tuple.At(index).Type(), integerID, booleanID, stringID, records)
		if !ok {
			return "", nil, false
		}
		itemTypes[index] = item
		parts = append(parts, item)
	}
	return stableID(parts...), itemTypes, true
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
	if isGoErrorType(value) {
		id := stableID("execution", "type", "native", "go", "error")
		return []graphEntity{{id, entity(id, "0000000000000000000000000000a071", []graphField{bytesField(0xa0710, "go"), bytesField(0xa0711, "error")})}}
	}
	if tuple, ok := types.Unalias(value).(*types.Tuple); ok {
		id := goSemanticTypeIdentity(tuple)
		items := make([]string, tuple.Len())
		entities := []graphEntity{}
		for index := 0; index < tuple.Len(); index++ {
			items[index] = goSemanticTypeIdentity(tuple.At(index).Type())
			entities = append(entities, goBridgeTypeEntities(tuple.At(index).Type(), integerID, booleanID, stringID)...)
		}
		return append(entities, graphEntity{id, entity(id, "0000000000000000000000000000a06f", []graphField{refsField(0xa06f0, items)})})
	}
	if element, ok := goPrimitiveSliceElement(value); ok {
		elementID := goSemanticTypeIdentity(element)
		tag, _ := goPrimitiveTypeTag(element)
		id := stableID("execution", "type", "slice", tag)
		return []graphEntity{{id, entity(id, "000000000000000000000000000090f8", []graphField{refField(0x9f80, elementID)})}}
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
