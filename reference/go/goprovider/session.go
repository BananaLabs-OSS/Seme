package goprovider

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
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
}

// IncrementalSession retains only the most recent valid canonical graph while
// still advancing past accepted but incomplete editor snapshots.
type IncrementalSession struct {
	mu                sync.Mutex
	moduleG1          []byte
	currentRevision   uint64
	lastValidRevision uint64
	lastValidGraph    string
	lastValidSources  []SourceIdentity
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
			ContentDigest: snapshotDigest(snapshot), Disposition: "rejected-stale",
		}
	}
	session.currentRevision = snapshot.Revision
	graph, sources, diagnostics := liftDocumentSnapshot(snapshot, session.moduleG1)
	valid := graph != ""
	if valid {
		session.lastValidRevision = snapshot.Revision
		session.lastValidGraph = graph
		session.lastValidSources = cloneSources(sources)
	}
	return SessionResult{
		Revision: snapshot.Revision, Accepted: true, Valid: valid,
		CanonicalG1: session.lastValidGraph, LastValidRevision: session.lastValidRevision,
		Diagnostics: diagnostics, Sources: cloneSources(session.lastValidSources),
		ContentDigest: snapshotDigest(snapshot), Disposition: map[bool]string{true: "accepted-valid", false: "accepted-invalid"}[valid],
	}
}

func cloneSources(sources []SourceIdentity) []SourceIdentity {
	return append([]SourceIdentity(nil), sources...)
}

type sessionFunction struct {
	id, name string
	fn       *ast.FuncDecl
	sig      *types.Signature
	info     *types.Info
	file     string
	fset     *token.FileSet
}

func liftDocumentSnapshot(snapshot DocumentSnapshot, moduleG1 []byte) (string, []SourceIdentity, []SessionDiagnostic) {
	if snapshot.PackagePath == "" {
		return "", nil, []SessionDiagnostic{{Code: "session.package_path_missing", Message: "package path is required", Severity: "error"}}
	}
	paths := make([]string, 0, len(snapshot.Files))
	for path := range snapshot.Files {
		if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") {
			paths = append(paths, filepath.ToSlash(path))
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return "", nil, []SessionDiagnostic{{Code: "session.no_go_files", Message: "snapshot contains no non-test Go files", Severity: "error"}}
	}
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(paths))
	var diagnostics []SessionDiagnostic
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, snapshot.Files[path], parser.AllErrors)
		if err != nil {
			diagnostics = append(diagnostics, parseDiagnostics(err)...)
			continue
		}
		files = append(files, file)
	}
	if len(diagnostics) != 0 || len(files) != len(paths) {
		return "", nil, sortedDiagnostics(diagnostics)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	config := types.Config{Importer: importer.Default(), Error: func(err error) {
		diagnostics = append(diagnostics, typeDiagnostic(err))
	}}
	if _, err := config.Check(snapshot.PackagePath, fset, files, info); err != nil {
		return "", nil, sortedDiagnostics(diagnostics)
	}
	var functions []sessionFunction
	for _, file := range files {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			object, ok := info.Defs[fn.Name].(*types.Func)
			if !ok {
				continue
			}
			signature, ok := object.Type().(*types.Signature)
			if !ok {
				continue
			}
			position := fset.Position(fn.Pos())
			functions = append(functions, sessionFunction{
				id: stableID("session-declaration", snapshot.PackagePath, fn.Name.Name), name: fn.Name.Name,
				fn: fn, sig: signature, info: info, file: filepath.ToSlash(position.Filename), fset: fset,
			})
		}
	}
	sort.Slice(functions, func(left, right int) bool { return functions[left].id < functions[right].id })
	functionObjects := make(map[types.Object]string, len(functions))
	for _, function := range functions {
		functionObjects[info.Defs[function.fn.Name]] = function.id
	}
	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	stringID := stableID("execution", "type", "string")
	instances := []graphEntity{
		{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{unsignedField(0x9100, 64), {0x9101, "tr"}, unsignedField(0x9102, 0)})},
		{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)},
		{stringID, entity(stringID, "00000000000000000000000000009040", nil)},
	}
	var functionIDs []string
	var sources []SourceIdentity
	for _, function := range functions {
		entities, source, diagnostic := liftSessionFunction(function, integerID, booleanID, stringID, functionObjects)
		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
			continue
		}
		instances = append(instances, entities...)
		functionIDs = append(functionIDs, function.id)
		sources = append(sources, source)
	}
	if len(functionIDs) == 0 {
		if len(diagnostics) == 0 {
			diagnostics = append(diagnostics, SessionDiagnostic{Code: "session.no_supported_declarations", Message: "snapshot contains no supported package functions", Severity: "error"})
		}
		return "", nil, sortedDiagnostics(diagnostics)
	}
	entryID := functionIDs[0]
	if snapshot.Entry != "" {
		entryID = ""
		for _, function := range functions {
			if function.name == snapshot.Entry {
				for _, supported := range functionIDs {
					if supported == function.id {
						entryID = supported
					}
				}
			}
		}
		if entryID == "" {
			return "", nil, []SessionDiagnostic{{Code: "session.entry_missing", Message: "requested entry function is not supported", Severity: "error"}}
		}
	}
	programID := stableID("session-program", snapshot.PackagePath)
	instances = append(instances, graphEntity{programID, entity(programID, "00000000000000000000000000009015", []graphField{
		refsField(0x9150, functionIDs), refField(0x9151, entryID),
	})})
	revision := stableID("session-revision", snapshot.PackagePath, strconv.FormatUint(snapshot.Revision, 10))
	return composeExecutionG1(moduleG1, revision, instances), sources, sortedDiagnostics(diagnostics)
}

func liftSessionFunction(function sessionFunction, integerID, booleanID, stringID string, functions map[types.Object]string) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
	position := function.fset.Position(function.fn.Pos())
	diagnostic := func(code, message string) ([]graphEntity, SourceIdentity, *SessionDiagnostic) {
		return nil, SourceIdentity{}, &SessionDiagnostic{Code: code, Message: message, File: function.file, Line: position.Line, Column: position.Column, Severity: "warning"}
	}
	if function.sig.Results().Len() != 1 {
		return diagnostic("session.unsupported_function_shape", "supported functions require one result")
	}
	resultTypeID := integerID
	if isBool(function.sig.Results().At(0).Type()) {
		resultTypeID = booleanID
	} else if isPureString(function.sig.Results().At(0).Type()) {
		resultTypeID = stringID
	} else if !isInt64(function.sig.Results().At(0).Type()) {
		return diagnostic("session.unsupported_result_type", "supported result types are int64, bool, and string")
	}
	block, err := analyzeGoBlockWithCalls(function.fn.Body.List, function.sig, function.info, functions)
	if err != nil {
		return diagnostic("session.unsupported_function_body", err.Error())
	}
	parameterIDs := make([]string, function.sig.Params().Len())
	var instances []graphEntity
	for index := range parameterIDs {
		parameterTypeID := integerID
		if isBool(function.sig.Params().At(index).Type()) {
			parameterTypeID = booleanID
		} else if isPureString(function.sig.Params().At(index).Type()) {
			parameterTypeID = stringID
		} else if !isInt64(function.sig.Params().At(index).Type()) {
			return diagnostic("session.unsupported_parameter_type", "supported parameter types are int64, bool, and string")
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
	instances = append(instances, graphEntity{function.id, entity(function.id, "00000000000000000000000000009011", []graphField{
		bytesField(0x9110, function.name), refsField(0x9111, parameterIDs), refField(0x9112, resultTypeID), refField(0x9113, bodyID),
	})})
	start := function.fset.Position(function.fn.Pos())
	end := function.fset.Position(function.fn.End())
	source := SourceIdentity{ID: function.id, Kind: "function", Name: function.name, Document: function.file, Start: start.Offset, End: end.Offset, Line: start.Line, Column: start.Column}
	return instances, source, nil
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
		return SessionDiagnostic{Code: "go.type", Message: typed.Msg, File: filepath.ToSlash(typed.Fset.Position(typed.Pos).Filename), Line: typed.Fset.Position(typed.Pos).Line, Column: typed.Fset.Position(typed.Pos).Column, Severity: "error"}
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
