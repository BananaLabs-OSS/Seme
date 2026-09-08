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
	method   bool
}

type goInterfaceInfo struct {
	named          *types.Named
	id             string
	requirements   []*types.Func
	requirementIDs []string
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
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
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
			if !ok {
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
			declarationID := stableID("session-declaration", snapshot.PackagePath, fn.Name.Name)
			if fn.Recv != nil {
				receiverName := receiverTypeName(signature.Recv().Type())
				declarationID = stableID("session-method", snapshot.PackagePath, receiverName, fn.Name.Name)
			}
			functions = append(functions, sessionFunction{
				id: declarationID, name: fn.Name.Name,
				fn: fn, sig: signature, info: info, file: filepath.ToSlash(position.Filename), fset: fset, method: fn.Recv != nil,
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
	records := make(map[*types.Named]goRecordInfo)
	for identifier, object := range info.Defs {
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
		recordID := stableID("execution", "record", snapshot.PackagePath, identifier.Name)
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
	var interfaces []goInterfaceInfo
	for identifier, object := range info.Defs {
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
		interfaceID := stableID("execution", "interface", snapshot.PackagePath, identifier.Name)
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
	if isI64Slice(function.sig.Results().At(0).Type()) {
		instances = append(instances, graphEntity{resultTypeID, entity(resultTypeID, "000000000000000000000000000090f8", []graphField{refField(0x9f80, integerID)})})
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
		receiverID := goReceiverID(function.sig)
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

func goReceiverID(signature *types.Signature) string {
	receiver := signature.Recv()
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
	if isInt64(value) {
		return stableID("execution", "type", "i64")
	}
	if isBool(value) {
		return stableID("execution", "type", "bool")
	}
	if isPureString(value) {
		return stableID("execution", "type", "string")
	}
	if named, ok := value.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
		return stableID("execution", "record", named.Obj().Pkg().Path(), named.Obj().Name())
	}
	return types.TypeString(value, func(pkg *types.Package) string { return pkg.Path() })
}

func goSupportedTypeID(value types.Type, integerID, booleanID, stringID string, records map[*types.Named]goRecordInfo) (string, bool) {
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
	if state, result, ok := goTransitionTypes(value); ok {
		return goTransitionTypeID(state, result), true
	}
	if named, ok := value.(*types.Named); ok {
		if record, exists := records[named]; exists {
			return record.id, true
		}
	}
	return "", false
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
