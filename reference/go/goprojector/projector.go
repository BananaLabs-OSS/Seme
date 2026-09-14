// Package goprojector renders a bounded canonical Seme program directly as
// ordinary Go source. It deliberately rejects semantics outside its declared
// profile instead of guessing a Go spelling.
package goprojector

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	sFunction            = "00000000000000000000000000009011"
	sParameter           = "00000000000000000000000000009012"
	sRead                = "00000000000000000000000000009013"
	sAdd                 = "00000000000000000000000000009014"
	sProgram             = "00000000000000000000000000009015"
	sInteger             = "00000000000000000000000000009010"
	sBoolean             = "00000000000000000000000000009020"
	sString              = "00000000000000000000000000009040"
	sStringLiteral       = "00000000000000000000000000009050"
	sStringEqual         = "000000000000000000000000000090c2"
	sConcat              = "000000000000000000000000000090c3"
	sBlock               = "00000000000000000000000000009080"
	sReturn              = "00000000000000000000000000009081"
	sCall                = "00000000000000000000000000009060"
	sRecordType          = "00000000000000000000000000009030"
	sRecordField         = "00000000000000000000000000009031"
	sFieldRead           = "00000000000000000000000000009032"
	sRecordConstruct     = "00000000000000000000000000009033"
	sBoolLiteral         = "000000000000000000000000000090b0"
	sIntegerLiteral      = "00000000000000000000000000009070"
	sFixedArrayType      = "000000000000000000000000000090f2"
	sFixedArrayConstruct = "000000000000000000000000000090f3"
	sIndexRead           = "000000000000000000000000000090f4"
	sSliceType           = "000000000000000000000000000090f8"
	sCollectionLength    = "000000000000000000000000000090f9"
	sDynamicIndexRead    = "000000000000000000000000000090fa"
	sCollectionAppend    = "000000000000000000000000000090fb"
	sCollectionUpdate    = "000000000000000000000000000090fc"
	sMapType             = "0000000000000000000000000000a040"
	sEmptyMap            = "0000000000000000000000000000a041"
	sMapLookup           = "0000000000000000000000000000a042"
	sMapLookupOption     = "0000000000000000000000000000a044"
	sMapUpdate           = "0000000000000000000000000000a043"
	sSliceRemove         = "0000000000000000000000000000a066"
	sMapRemove           = "0000000000000000000000000000a067"
	sNativeInvocation    = "0000000000000000000000000000a06d"
	sUnitType            = "0000000000000000000000000000a06a"
	sUnitValue           = "0000000000000000000000000000a06b"
	sNativeType          = "0000000000000000000000000000a071"
	sSliceConstruct      = "0000000000000000000000000000a068"
	sBooleanNot          = "0000000000000000000000000000a069"
	sBytes               = "00000000000000000000000000009041"
	sResultType          = "00000000000000000000000000009042"
	sResultOk            = "00000000000000000000000000009043"
	sResultError         = "00000000000000000000000000009044"
	sOptionType          = "0000000000000000000000000000a050"
	sOptionNone          = "0000000000000000000000000000a051"
	sOptionSome          = "0000000000000000000000000000a052"
	sBytesLiteral        = "0000000000000000000000000000a064"
	sBytesEqual          = "0000000000000000000000000000a065"
	sVariantBinding      = "0000000000000000000000000000a060"
	sVariantRead         = "0000000000000000000000000000a061"
	sResultMatch         = "0000000000000000000000000000a062"
	sOptionMatch         = "0000000000000000000000000000a063"
	sLocalBinding        = "000000000000000000000000000090d0"
	sBindLocal           = "000000000000000000000000000090d1"
	sLocalRead           = "000000000000000000000000000090d2"
	sPlace               = "000000000000000000000000000090e0"
	sDeclarePlace        = "000000000000000000000000000090e1"
	sPlaceRead           = "000000000000000000000000000090e2"
	sAssignPlace         = "000000000000000000000000000090e3"
	sWhile               = "000000000000000000000000000090e4"
	sNativeDefer         = "0000000000000000000000000000a075"
	sNativeFieldAssign   = "0000000000000000000000000000a076"
	sNativeSwitchCase    = "0000000000000000000000000000a077"
	sNativeSwitch        = "0000000000000000000000000000a078"
	sNativeRangeBinding  = "0000000000000000000000000000a07b"
	sNativeRange         = "0000000000000000000000000000a07c"
	sNativeSlice         = "0000000000000000000000000000a07d"
	sNativeDereference   = "0000000000000000000000000000a07e"
	sNativeBinary        = "0000000000000000000000000000a07f"
	sNativeBranch        = "0000000000000000000000000000a079"
	sNativeAddress       = "0000000000000000000000000000a07a"
	sWhen                = "000000000000000000000000000090f0"
	sIf                  = "000000000000000000000000000090c0"
	sLessEqual           = "00000000000000000000000000009021"
	sAnd                 = "000000000000000000000000000090b1"
	sOr                  = "000000000000000000000000000090c1"
	sMultiply            = "00000000000000000000000000009090"
	sSubtract            = "000000000000000000000000000090a0"
	sIterationBinding    = "000000000000000000000000000090f5"
	sIterationRead       = "000000000000000000000000000090f6"
	sFold                = "000000000000000000000000000090f7"
	sReceiverBinding     = "0000000000000000000000000000a000"
	sReceiverRead        = "0000000000000000000000000000a001"
	sMethod              = "0000000000000000000000000000a002"
	sMethodCall          = "0000000000000000000000000000a003"
	sInterfaceType       = "0000000000000000000000000000a010"
	sMethodRequirement   = "0000000000000000000000000000a011"
	sSatisfactionWitness = "0000000000000000000000000000a012"
	sInterfaceValue      = "0000000000000000000000000000a013"
	sDynamicMethodCall   = "0000000000000000000000000000a014"
	sFunctionType        = "0000000000000000000000000000a020"
	sCaptureBinding      = "0000000000000000000000000000a021"
	sCaptureRead         = "0000000000000000000000000000a022"
	sClosureConstruct    = "0000000000000000000000000000a023"
	sIndirectCall        = "0000000000000000000000000000a024"
	sTransitionState     = "0000000000000000000000000000a006"
	sTransitionResult    = "0000000000000000000000000000a007"
	sMutableCapture      = "0000000000000000000000000000a030"
	sMutableCaptureRead  = "0000000000000000000000000000a031"
	sCaptureUpdate       = "0000000000000000000000000000a032"
	sSequence            = "0000000000000000000000000000a033"
	sMutableClosure      = "0000000000000000000000000000a034"
	sStatefulCall        = "0000000000000000000000000000a035"
	sTransitionType      = "0000000000000000000000000000a004"
	sStateTransition     = "0000000000000000000000000000a005"
	sEffectInvoke        = "000000000000000000000000000090f1"
	sEffect              = "00000000000000000000000000000015"
	sCapability          = "00000000000000000000000000000016"
)

type entity struct {
	id, schema string
	fields     map[string][]string
}
type context struct {
	graph       map[string]entity
	functions   map[string]string
	parameters  map[string]string
	records     map[string]record
	locals      map[string]string
	iterations  map[string]string
	variants    map[string]string
	methods     map[string]string
	receivers   map[string]string
	captures    map[string]string
	transitions map[string]string
	typeNames   map[string]string
	familyNames map[string]string
}
type record struct {
	name   string
	fields []recordField
}
type recordField struct{ id, name, typ string }

// Project emits one gofmt-formatted file. packageName is projection metadata,
// not canonical meaning, and must be a valid Go identifier.
func Project(g1 []byte, packageName string) ([]byte, error) {
	return project(g1, packageName, false)
}

func project(g1 []byte, packageName string, allowDuplicateNames bool) ([]byte, error) {
	if !identifier(packageName) {
		return nil, fmt.Errorf("go_projection.invalid_package_name")
	}
	graph, err := parse(g1)
	if err != nil {
		return nil, err
	}
	if allowDuplicateNames {
		uniquifyDeclarationNames(graph)
	}
	var programs []entity
	for _, e := range graph {
		if e.schema == sProgram {
			programs = append(programs, e)
		}
	}
	if len(programs) != 1 {
		return nil, fmt.Errorf("go_projection.requires_one_program")
	}
	ids, err := refs(programs[0], "00000000000000000000000000009150")
	if err != nil || len(ids) == 0 {
		return nil, fmt.Errorf("go_projection.invalid_program_members")
	}
	entry, err := ref(programs[0], "00000000000000000000000000009151")
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	seen := map[string]bool{}
	for _, id := range ids {
		fn, ok := graph[id]
		if !ok || fn.schema != sFunction {
			return nil, fmt.Errorf("go_projection.invalid_function_member")
		}
		name, err := text(fn, "00000000000000000000000000009110")
		if err != nil || !identifier(name) || (!allowDuplicateNames && seen[name]) {
			return nil, fmt.Errorf("go_projection.invalid_function_name")
		}
		rendered := name
		if allowDuplicateNames && seen[name] {
			rendered = name + "__seme_" + id[:8]
		}
		names[id], seen[name] = rendered, true
	}
	if _, ok := names[entry]; !ok {
		return nil, fmt.Errorf("go_projection.entry_membership")
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	writeProjectionEnvelope(&out, g1)
	importSet := map[string]bool{}
	if graphHasSchema(graph, sBytesEqual) {
		importSet["bytes"] = true
	}
	if graphHasSchema(graph, sCollectionUpdate) || graphHasSchema(graph, sSliceRemove) {
		importSet["slices"] = true
	}
	if graphHasUnfoldedMapUpdate(graph) || graphHasSchema(graph, sMapRemove) {
		importSet["maps"] = true
	}
	if graphHasSchema(graph, sEffectInvoke) {
		importSet["log"] = true
	}
	for _, e := range graph {
		if e.schema != sNativeInvocation {
			continue
		}
		language, languageErr := text(e, "000000000000000000000000000a06d0")
		callable, callableErr := text(e, "000000000000000000000000000a06d1")
		if languageErr == nil && callableErr == nil && language == "go" {
			if strings.HasPrefix(callable, "builtin.") {
				continue
			}
			if dot := strings.LastIndexByte(callable, '.'); dot > 0 {
				importSet[callable[:dot]] = true
			}
		}
	}
	imports := make([]string, 0, len(importSet))
	for path := range importSet {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	if len(imports) == 1 {
		fmt.Fprintf(&out, "import %q\n\n", imports[0])
	} else if len(imports) > 1 {
		out.WriteString("import (\n")
		for _, name := range imports {
			fmt.Fprintf(&out, "\t%q\n", name)
		}
		out.WriteString(")\n\n")
	}
	if graphHasSchema(graph, sOptionType) {
		out.WriteString("type Option[T any] struct {\n\tSome bool\n\tValue T\n}\n\n")
	}
	if graphHasSchema(graph, sResultType) {
		out.WriteString("type Result[T, E any] struct {\n\tOk bool\n\tValue T\n\tError E\n}\n\n")
	}
	if graphHasSchema(graph, sTransitionType) {
		out.WriteString("type Transition[S, R any] struct {\n\tState S\n\tResult R\n}\n\n")
	}
	records, err := collectRecords(graph)
	if err != nil {
		return nil, err
	}
	for _, id := range sortedRecordIDs(records) {
		r := records[id]
		fmt.Fprintf(&out, "//seme:id %s\n", id)
		fmt.Fprintf(&out, "type %s struct {\n", r.name)
		for _, field := range r.fields {
			fmt.Fprintf(&out, "\t%s %s\n", field.name, field.typ)
		}
		out.WriteString("}\n\n")
	}
	methods := map[string]string{}
	methodIDs := []string{}
	for id, value := range graph {
		if value.schema == sMethod {
			name, e := text(value, "000000000000000000000000000a0020")
			if e != nil || !identifier(name) {
				return nil, fmt.Errorf("go_projection.invalid_method")
			}
			methods[id] = name
			methodIDs = append(methodIDs, id)
		}
	}
	sort.Strings(methodIDs)
	interfaceIDs := []string{}
	for id, value := range graph {
		if value.schema == sInterfaceType {
			interfaceIDs = append(interfaceIDs, id)
		}
	}
	sort.Strings(interfaceIDs)
	for _, iid := range interfaceIDs {
		value := graph[iid]
		name, e := text(value, "000000000000000000000000000a0100")
		if e != nil || !identifier(name) {
			return nil, fmt.Errorf("go_projection.invalid_interface")
		}
		requirements, e := refs(value, "000000000000000000000000000a0101")
		if e != nil || len(requirements) == 0 {
			return nil, fmt.Errorf("go_projection.invalid_interface")
		}
		fmt.Fprintf(&out, "//seme:id %s\n", iid)
		fmt.Fprintf(&out, "type %s interface {\n", name)
		for _, rid := range requirements {
			requirement, ok := graph[rid]
			if !ok || requirement.schema != sMethodRequirement {
				return nil, fmt.Errorf("go_projection.invalid_requirement")
			}
			methodName, e := text(requirement, "000000000000000000000000000a0110")
			if e != nil || !identifier(methodName) {
				return nil, fmt.Errorf("go_projection.invalid_requirement")
			}
			types, e := refs(requirement, "000000000000000000000000000a0111")
			if e != nil {
				return nil, e
			}
			rendered := make([]string, len(types))
			for i, t := range types {
				rendered[i], e = typeName(graph, t)
				if e != nil {
					return nil, e
				}
			}
			resultID, e := ref(requirement, "000000000000000000000000000a0112")
			if e != nil {
				return nil, e
			}
			result, e := typeName(graph, resultID)
			if e != nil {
				return nil, e
			}
			params := make([]string, len(rendered))
			for i, t := range rendered {
				params[i] = "argument" + strconv.Itoa(i) + " " + t
			}
			fmt.Fprintf(&out, "\t%s(%s) %s\n", methodName, strings.Join(params, ", "), result)
		}
		out.WriteString("}\n\n")
	}
	for _, mid := range methodIDs {
		method := graph[mid]
		receiverID, e := ref(method, "000000000000000000000000000a0021")
		if e != nil {
			return nil, e
		}
		receiver, ok := graph[receiverID]
		if !ok || receiver.schema != sReceiverBinding {
			return nil, fmt.Errorf("go_projection.invalid_receiver")
		}
		receiverTypeID, e := ref(receiver, "000000000000000000000000000a0001")
		if e != nil {
			return nil, e
		}
		receiverType, e := typeName(graph, receiverTypeID)
		if e != nil {
			return nil, e
		}
		parameterIDs, e := refs(method, "000000000000000000000000000a0022")
		if e != nil {
			return nil, e
		}
		params := map[string]string{}
		declarations := []string{}
		for _, pid := range parameterIDs {
			p := graph[pid]
			name, e := text(p, "00000000000000000000000000009120")
			if e != nil || !identifier(name) {
				return nil, fmt.Errorf("go_projection.invalid_parameter")
			}
			tid, e := ref(p, "00000000000000000000000000009121")
			if e != nil {
				return nil, e
			}
			typ, e := typeName(graph, tid)
			if e != nil {
				return nil, e
			}
			params[pid] = name
			declarations = append(declarations, name+" "+typ)
		}
		resultID, e := ref(method, "000000000000000000000000000a0023")
		if e != nil {
			return nil, e
		}
		result, e := typeName(graph, resultID)
		if e != nil {
			return nil, e
		}
		bodyID, e := ref(method, "000000000000000000000000000a0024")
		if e != nil {
			return nil, e
		}
		body, e := projectBlock(bodyID, context{graph: graph, functions: names, parameters: params, records: records, locals: map[string]string{}, iterations: map[string]string{}, variants: map[string]string{}, methods: methods, receivers: map[string]string{receiverID: "self"}, captures: map[string]string{}, transitions: map[string]string{}})
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&out, "//seme:id %s\n", mid)
		fmt.Fprintf(&out, "//seme:receiver %s\n", receiverID)
		fmt.Fprintf(&out, "func (self %s) %s(%s) %s {\n%s\n}\n\n", receiverType, methods[mid], strings.Join(declarations, ", "), result, body)
	}
	for _, id := range ids {
		fn := graph[id]
		paramIDs, err := refs(fn, "00000000000000000000000000009111")
		if err != nil {
			return nil, err
		}
		params := map[string]string{}
		var declarations []string
		for _, pid := range paramIDs {
			p, ok := graph[pid]
			if !ok || p.schema != sParameter {
				return nil, fmt.Errorf("go_projection.invalid_parameter")
			}
			name, e := text(p, "00000000000000000000000000009120")
			if e != nil || !identifier(name) {
				return nil, fmt.Errorf("go_projection.invalid_parameter_name")
			}
			tid, e := ref(p, "00000000000000000000000000009121")
			if e != nil {
				return nil, e
			}
			typ, e := typeName(graph, tid)
			if e != nil {
				return nil, e
			}
			params[pid] = name
			declarations = append(declarations, name+" "+typ)
		}
		resultID, e := ref(fn, "00000000000000000000000000009112")
		if e != nil {
			return nil, e
		}
		result, e := typeName(graph, resultID)
		if e != nil {
			return nil, e
		}
		bodyID, e := ref(fn, "00000000000000000000000000009113")
		if e != nil {
			return nil, e
		}
		body, e := projectBlock(bodyID, context{graph: graph, functions: names, parameters: params, records: records, locals: map[string]string{}, iterations: map[string]string{}, variants: map[string]string{}, methods: methods, receivers: map[string]string{}, captures: map[string]string{}, transitions: map[string]string{}})
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&out, "//seme:id %s\n", id)
		fmt.Fprintf(&out, "func %s(%s) %s {\n%s\n}\n\n", names[id], strings.Join(declarations, ", "), result, body)
	}
	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, fmt.Errorf("go_projection.format: %w\n%s", err, out.String())
	}
	return formatted, nil
}

// uniquifyDeclarationNames makes the rich projector's temporary single-file
// syntax tree unambiguous. Authenticated owner names are restored when that
// tree is partitioned; the canonical graph and embedded envelope are unchanged.
func uniquifyDeclarationNames(graph map[string]entity) {
	fields := map[string]string{sRecordType: "00000000000000000000000000009300", sInterfaceType: "000000000000000000000000000a0100"}
	seen := map[string]bool{
		"Option":     graphHasSchema(graph, sOptionType),
		"Result":     graphHasSchema(graph, sResultType),
		"Transition": graphHasSchema(graph, sTransitionType),
	}
	ids := make([]string, 0, len(graph))
	for id := range graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := graph[id]
		field, ok := fields[e.schema]
		if !ok {
			continue
		}
		name, err := text(e, field)
		if err != nil {
			continue
		}
		if seen[name] {
			e.fields[field] = []string{"by " + hex.EncodeToString([]byte(name+"__seme_"+id[:8]))}
			graph[id] = e
		}
		seen[name] = true
	}
}

func writeProjectionEnvelope(out *strings.Builder, graph []byte) {
	digest := sha256.Sum256(graph)
	fmt.Fprintf(out, "//seme:projection-v1 %x\n", digest[:])
	encoded := base64.RawStdEncoding.EncodeToString(graph)
	for len(encoded) > 0 {
		count := 120
		if len(encoded) < count {
			count = len(encoded)
		}
		fmt.Fprintf(out, "//seme:graph %s\n", encoded[:count])
		encoded = encoded[count:]
	}
	out.WriteString("\n")
}

// VerifyProjectionEnvelope returns the exact canonical graph carried by a
// projector-produced Go source file only when independently projecting that
// graph reproduces the complete source byte-for-byte. The envelope therefore
// preserves canonical identity without trusting editable provenance comments.
func VerifyProjectionEnvelope(source []byte) ([]byte, bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "projected.go", source, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	var digestText string
	var encoded strings.Builder
	for _, group := range file.Comments {
		for _, comment := range group.List {
			value := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(value, "seme:projection-v1 ") {
				digestText = strings.TrimSpace(strings.TrimPrefix(value, "seme:projection-v1 "))
			}
			if strings.HasPrefix(value, "seme:graph ") {
				encoded.WriteString(strings.TrimSpace(strings.TrimPrefix(value, "seme:graph ")))
			}
		}
	}
	if digestText == "" && encoded.Len() == 0 {
		return nil, false, nil
	}
	if len(digestText) != 64 || encoded.Len() == 0 {
		return nil, true, fmt.Errorf("go_projection.envelope_incomplete")
	}
	graph, err := base64.RawStdEncoding.DecodeString(encoded.String())
	if err != nil {
		return nil, true, fmt.Errorf("go_projection.envelope_encoding")
	}
	digest := sha256.Sum256(graph)
	if fmt.Sprintf("%x", digest[:]) != digestText {
		return nil, true, fmt.Errorf("go_projection.envelope_digest")
	}
	reprojected, err := Project(graph, file.Name.Name)
	if err != nil {
		return nil, true, fmt.Errorf("go_projection.envelope_graph:%w", err)
	}
	if !bytes.Equal(reprojected, source) {
		return nil, true, fmt.Errorf("go_projection.envelope_source_mismatch")
	}
	return graph, true, nil
}

func graphHasUnfoldedMapUpdate(graph map[string]entity) bool {
	foldBodies := map[string]bool{}
	for _, item := range graph {
		if item.schema != sFold {
			continue
		}
		if body, err := ref(item, "00000000000000000000000000009f74"); err == nil {
			foldBodies[body] = true
		}
	}
	for id, item := range graph {
		if item.schema == sMapUpdate && !foldBodies[id] {
			return true
		}
	}
	return false
}

func projectBlock(id string, c context) (string, error) {
	b, ok := c.graph[id]
	if !ok || b.schema != sBlock {
		return "", fmt.Errorf("go_projection.invalid_block")
	}
	statements, err := refs(b, "00000000000000000000000000009800")
	if err != nil {
		return "", fmt.Errorf("go_projection.unsupported_block")
	}
	if len(statements) == 1 {
		if returned, ok := c.graph[statements[0]]; ok && returned.schema == sReturn {
			values, e := refs(returned, "00000000000000000000000000009810")
			if e == nil && len(values) == 1 {
				if closure, ok := c.graph[values[0]]; ok && closure.schema == sMutableClosure {
					return projectMutableClosureReturn(closure, c)
				}
			}
		}
	}
	lines := []string{}
	for index, statementID := range statements {
		statement, ok := c.graph[statementID]
		if !ok {
			return "", fmt.Errorf("go_projection.unsupported_statement")
		}
		switch statement.schema {
		case sBindLocal:
			bindingID, err := ref(statement, "00000000000000000000000000009d10")
			if err != nil {
				return "", err
			}
			binding, ok := c.graph[bindingID]
			if !ok || binding.schema != sLocalBinding {
				return "", fmt.Errorf("go_projection.invalid_local")
			}
			name, err := text(binding, "00000000000000000000000000009d00")
			if err != nil || !identifier(name) {
				return "", fmt.Errorf("go_projection.invalid_local_name:%s:%q", bindingID, name)
			}
			name = availableLocalName(name, c)
			initializerID, err := ref(binding, "00000000000000000000000000009d02")
			if err != nil {
				return "", err
			}
			initializer, err := expr(initializerID, c)
			if err != nil {
				return "", err
			}
			if initialEntity, ok := c.graph[initializerID]; ok && initialEntity.schema == sStatefulCall {
				if statefulResultIsObserved(bindingID, c.graph) {
					lines = append(lines, "\t"+name+" := "+initializer)
				} else {
					lines = append(lines, "\t"+initializer)
				}
				c.transitions[bindingID] = name
				c.locals[bindingID] = name
				continue
			}
			if typeID, x := ref(binding, "00000000000000000000000000009d01"); x == nil {
				if typ, exists := c.graph[typeID]; exists && typ.schema == sInteger {
					if initial, exists := c.graph[initializerID]; exists && initial.schema == sIntegerLiteral {
						initializer = "int64(" + initializer + ")"
					}
				}
			}
			lines = append(lines, "\t"+name+" := "+initializer)
			c.locals[bindingID] = name
		case sDeclarePlace:
			placeID, err := ref(statement, "00000000000000000000000000009e10")
			if err != nil {
				return "", err
			}
			place, ok := c.graph[placeID]
			if !ok || place.schema != sPlace {
				return "", fmt.Errorf("go_projection.invalid_place")
			}
			name, err := text(place, "00000000000000000000000000009e00")
			if err != nil || !identifier(name) {
				return "", fmt.Errorf("go_projection.invalid_place_name")
			}
			initializerID, err := ref(place, "00000000000000000000000000009e02")
			if err != nil {
				return "", err
			}
			initializer, err := expr(initializerID, c)
			if err != nil {
				return "", err
			}
			if typeID, x := ref(place, "00000000000000000000000000009e01"); x == nil {
				if typ, exists := c.graph[typeID]; exists && typ.schema == sInteger {
					if initial, exists := c.graph[initializerID]; exists && initial.schema == sIntegerLiteral {
						initializer = "int64(" + initializer + ")"
					}
				}
			}
			lines = append(lines, "\t"+name+" := "+initializer)
			c.locals[placeID] = name
		case sAssignPlace:
			placeID, err := ref(statement, "00000000000000000000000000009e30")
			if err != nil {
				return "", err
			}
			name, ok := c.locals[placeID]
			if !ok {
				return "", fmt.Errorf("go_projection.place_scope")
			}
			valueID, err := ref(statement, "00000000000000000000000000009e31")
			if err != nil {
				return "", err
			}
			if state, ok := c.graph[valueID]; ok && state.schema == sTransitionState {
				continue
			}
			value, err := expr(valueID, c)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\t"+name+" = "+value)
		case sWhile, sWhen:
			conditionField, bodyField := "00000000000000000000000000009e40", "00000000000000000000000000009e41"
			keyword := "for "
			if statement.schema == sWhen {
				conditionField, bodyField, keyword = "00000000000000000000000000009f00", "00000000000000000000000000009f01", "if "
			}
			conditionID, err := ref(statement, conditionField)
			if err != nil {
				return "", err
			}
			bodyID, err := ref(statement, bodyField)
			if err != nil {
				return "", err
			}
			condition, err := expr(conditionID, c)
			if err != nil {
				return "", err
			}
			child := c
			child.locals = cloneNames(c.locals)
			body, err := projectBlock(bodyID, child)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\t"+keyword+condition+" {", indentBlock(body), "\t}")
		case sNativeRange:
			if language, err := text(statement, "000000000000000000000000000a07c0"); err != nil || language != "go" {
				return "", fmt.Errorf("go_projection.native_range_language")
			}
			collectionID, err := ref(statement, "000000000000000000000000000a07c1")
			if err != nil {
				return "", err
			}
			collection, err := expr(collectionID, c)
			if err != nil {
				return "", err
			}
			child := c
			child.locals = cloneNames(c.locals)
			bindingName := func(field string) (string, error) {
				ids, err := refs(statement, field)
				if err != nil || len(ids) > 1 {
					return "", fmt.Errorf("go_projection.native_range_binding")
				}
				if len(ids) == 0 {
					return "_", nil
				}
				binding, ok := c.graph[ids[0]]
				if !ok || binding.schema != sNativeRangeBinding {
					return "", fmt.Errorf("go_projection.native_range_binding")
				}
				name, err := text(binding, "000000000000000000000000000a07b0")
				if err != nil || !identifier(name) {
					return "", fmt.Errorf("go_projection.native_range_binding")
				}
				child.locals[ids[0]] = name
				return name, nil
			}
			key, err := bindingName("000000000000000000000000000a07c2")
			if err != nil {
				return "", err
			}
			valueIDs, err := refs(statement, "000000000000000000000000000a07c3")
			if err != nil || len(valueIDs) > 1 {
				return "", fmt.Errorf("go_projection.native_range_binding")
			}
			value := ""
			if len(valueIDs) == 1 {
				binding, ok := c.graph[valueIDs[0]]
				if !ok || binding.schema != sNativeRangeBinding {
					return "", fmt.Errorf("go_projection.native_range_binding")
				}
				value, err = text(binding, "000000000000000000000000000a07b0")
				if err != nil || !identifier(value) {
					return "", fmt.Errorf("go_projection.native_range_binding")
				}
				child.locals[valueIDs[0]] = value
			}
			bodyID, err := ref(statement, "000000000000000000000000000a07c4")
			if err != nil {
				return "", err
			}
			body, err := projectBlock(bodyID, child)
			if err != nil {
				return "", err
			}
			header := key
			if value != "" {
				header += ", " + value
			}
			lines = append(lines, "\tfor "+header+" := range "+collection+" {", indentBlock(body), "\t}")
		case sNativeDefer:
			if language, err := text(statement, "000000000000000000000000000a0750"); err != nil || language != "go" {
				return "", fmt.Errorf("go_projection.native_defer_language")
			}
			invocationID, err := ref(statement, "000000000000000000000000000a0751")
			if err != nil {
				return "", err
			}
			invocation, err := expr(invocationID, c)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\tdefer "+invocation)
		case sNativeFieldAssign:
			if language, err := text(statement, "000000000000000000000000000a0760"); err != nil || language != "go" {
				return "", fmt.Errorf("go_projection.native_field_assignment_language")
			}
			name, err := text(statement, "000000000000000000000000000a0761")
			if err != nil || !identifier(name) {
				return "", fmt.Errorf("go_projection.native_field_assignment_field")
			}
			receiverID, err := ref(statement, "000000000000000000000000000a0762")
			if err != nil {
				return "", err
			}
			valueID, err := ref(statement, "000000000000000000000000000a0763")
			if err != nil {
				return "", err
			}
			receiver, err := expr(receiverID, c)
			if err != nil {
				return "", err
			}
			value, err := expr(valueID, c)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\t"+receiver+"."+name+" = "+value)
		case sNativeSwitch:
			if language, err := text(statement, "000000000000000000000000000a0780"); err != nil || language != "go" {
				return "", fmt.Errorf("go_projection.native_switch_language")
			}
			subjectID, err := ref(statement, "000000000000000000000000000a0781")
			if err != nil {
				return "", err
			}
			header := "\tswitch"
			if subject, ok := c.graph[subjectID]; !ok || subject.schema != sUnitValue {
				projected, err := expr(subjectID, c)
				if err != nil {
					return "", err
				}
				header += " " + projected
			}
			lines = append(lines, header+" {")
			caseIDs, err := refs(statement, "000000000000000000000000000a0782")
			if err != nil {
				return "", err
			}
			for _, caseID := range caseIDs {
				switchCase, ok := c.graph[caseID]
				if !ok || switchCase.schema != sNativeSwitchCase {
					return "", fmt.Errorf("go_projection.native_switch_case")
				}
				valueIDs, err := refs(switchCase, "000000000000000000000000000a0770")
				if err != nil || len(valueIDs) == 0 {
					return "", fmt.Errorf("go_projection.native_switch_case_values")
				}
				values := make([]string, len(valueIDs))
				for valueIndex, valueID := range valueIDs {
					values[valueIndex], err = expr(valueID, c)
					if err != nil {
						return "", err
					}
				}
				bodyID, err := ref(switchCase, "000000000000000000000000000a0771")
				if err != nil {
					return "", err
				}
				child := c
				child.locals = cloneNames(c.locals)
				body, err := projectBlock(bodyID, child)
				if err != nil {
					return "", err
				}
				lines = append(lines, "\tcase "+strings.Join(values, ", ")+":")
				if body != "" {
					lines = append(lines, indentBlock(body))
				}
			}
			defaultIDs, err := refs(statement, "000000000000000000000000000a0783")
			if err != nil || len(defaultIDs) > 1 {
				return "", fmt.Errorf("go_projection.native_switch_default")
			}
			if len(defaultIDs) == 1 {
				child := c
				child.locals = cloneNames(c.locals)
				body, err := projectBlock(defaultIDs[0], child)
				if err != nil {
					return "", err
				}
				lines = append(lines, "\tdefault:")
				if body != "" {
					lines = append(lines, indentBlock(body))
				}
			}
			lines = append(lines, "\t}")
		case sNativeBranch:
			language, languageErr := text(statement, "000000000000000000000000000a0790")
			operation, operationErr := text(statement, "000000000000000000000000000a0791")
			target, targetErr := text(statement, "000000000000000000000000000a0792")
			if languageErr != nil || operationErr != nil || targetErr != nil || language != "go" || target != "nearest" || operation != "break" && operation != "continue" {
				return "", fmt.Errorf("go_projection.native_branch")
			}
			lines = append(lines, "\t"+operation)
		case sIf:
			conditionID, err := ref(statement, "00000000000000000000000000009c00")
			if err != nil {
				return "", err
			}
			thenID, err := ref(statement, "00000000000000000000000000009c01")
			if err != nil {
				return "", err
			}
			elseID, err := ref(statement, "00000000000000000000000000009c02")
			if err != nil {
				return "", err
			}
			condition, err := expr(conditionID, c)
			if err != nil {
				return "", err
			}
			thenContext := c
			thenContext.locals = cloneNames(c.locals)
			elseContext := c
			elseContext.locals = cloneNames(c.locals)
			thenBody, err := projectBlock(thenID, thenContext)
			if err != nil {
				return "", err
			}
			elseBody, err := projectBlock(elseID, elseContext)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\tif "+condition+" {", indentBlock(thenBody), "\t} else {", indentBlock(elseBody), "\t}")
		case sReturn:
			if index != len(statements)-1 {
				return "", fmt.Errorf("go_projection.return_not_terminal")
			}
			values, err := refs(statement, "00000000000000000000000000009810")
			if err != nil || len(values) != 1 {
				return "", fmt.Errorf("go_projection.return_arity")
			}
			if value, ok := c.graph[values[0]]; ok && value.schema == sUnitValue {
				lines = append(lines, "\treturn")
				continue
			}
			if folded, ok, err := projectFoldReturn(values[0], c); err != nil {
				return "", err
			} else if ok {
				lines = append(lines, folded...)
				continue
			}
			if matched, ok, err := projectTaggedReturn(values[0], c); err != nil {
				return "", err
			} else if ok {
				lines = append(lines, matched...)
				continue
			}
			expression, err := expr(values[0], c)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\treturn "+expression)
		case sEffectInvoke:
			effectID, err := ref(statement, "00000000000000000000000000009f10")
			if err != nil {
				return "", err
			}
			effect, ok := c.graph[effectID]
			if !ok || effect.schema != sEffect {
				return "", fmt.Errorf("go_projection.effect")
			}
			name, err := text(effect, "00000000000000000000000000000150")
			if err != nil || name != "observability.log" {
				return "", fmt.Errorf("go_projection.unsupported_effect")
			}
			capabilityID, err := ref(effect, "00000000000000000000000000000151")
			if err != nil {
				return "", err
			}
			capability, ok := c.graph[capabilityID]
			capabilityName, capabilityErr := text(capability, "00000000000000000000000000000160")
			if !ok || capability.schema != sCapability || capabilityErr != nil || capabilityName != name {
				return "", fmt.Errorf("go_projection.effect_capability")
			}
			arguments, err := refs(statement, "00000000000000000000000000009f11")
			if err != nil || len(arguments) != 1 {
				return "", fmt.Errorf("go_projection.effect_arity")
			}
			argument, err := expr(arguments[0], c)
			if err != nil {
				return "", err
			}
			lines = append(lines, "\tlog.Print("+argument+")")
		default:
			return "", fmt.Errorf("go_projection.unsupported_statement")
		}
	}
	return strings.Join(lines, "\n"), nil
}

func projectTaggedReturn(id string, c context) ([]string, bool, error) {
	match, ok := c.graph[id]
	if !ok || (match.schema != sOptionMatch && match.schema != sResultMatch) {
		return nil, false, nil
	}
	valueField := "000000000000000000000000000a0630"
	if match.schema == sResultMatch {
		valueField = "000000000000000000000000000a0620"
	}
	valueID, err := ref(match, valueField)
	if err != nil {
		return nil, false, err
	}
	value, err := expr(valueID, c)
	if err != nil {
		return nil, false, err
	}
	if match.schema == sOptionMatch {
		noneID, err := ref(match, "000000000000000000000000000a0631")
		if err != nil {
			return nil, false, err
		}
		bindingID, err := ref(match, "000000000000000000000000000a0632")
		if err != nil {
			return nil, false, err
		}
		someID, err := ref(match, "000000000000000000000000000a0633")
		if err != nil {
			return nil, false, err
		}
		someContext := c
		someContext.variants = cloneNames(c.variants)
		someContext.variants[bindingID] = value + ".Value"
		some, err := projectBlock(someID, someContext)
		if err != nil {
			return nil, false, err
		}
		none, err := projectBlock(noneID, c)
		if err != nil {
			return nil, false, err
		}
		return []string{"\tif " + value + ".Some {", indentBlock(some), "\t}", none}, true, nil
	}
	okBinding, err := ref(match, "000000000000000000000000000a0621")
	if err != nil {
		return nil, false, err
	}
	okBody, err := ref(match, "000000000000000000000000000a0622")
	if err != nil {
		return nil, false, err
	}
	errorBinding, err := ref(match, "000000000000000000000000000a0623")
	if err != nil {
		return nil, false, err
	}
	errorBody, err := ref(match, "000000000000000000000000000a0624")
	if err != nil {
		return nil, false, err
	}
	okContext, errorContext := c, c
	okContext.variants = cloneNames(c.variants)
	errorContext.variants = cloneNames(c.variants)
	okContext.variants[okBinding] = value + ".Value"
	errorContext.variants[errorBinding] = value + ".Error"
	okText, err := projectBlock(okBody, okContext)
	if err != nil {
		return nil, false, err
	}
	errorText, err := projectBlock(errorBody, errorContext)
	if err != nil {
		return nil, false, err
	}
	return []string{"\tif " + value + ".Ok {", indentBlock(okText), "\t}", errorText}, true, nil
}
func indentBlock(value string) string { return "\t" + strings.ReplaceAll(value, "\n", "\n\t") }

func projectMutableClosureReturn(closure entity, c context) (string, error) {
	parameters, err := refs(closure, "000000000000000000000000000a0341")
	if err != nil || len(parameters) != 1 {
		return "", fmt.Errorf("go_projection.mutable_closure_parameter")
	}
	captures, err := refs(closure, "000000000000000000000000000a0342")
	if err != nil || len(captures) != 1 {
		return "", fmt.Errorf("go_projection.mutable_closure_capture")
	}
	bodyID, err := ref(closure, "000000000000000000000000000a0343")
	if err != nil {
		return "", err
	}
	capture, ok := c.graph[captures[0]]
	if !ok || capture.schema != sMutableCapture {
		return "", fmt.Errorf("go_projection.mutable_closure_capture")
	}
	captureName, err := text(capture, "000000000000000000000000000a0300")
	if err != nil || !identifier(captureName) {
		return "", fmt.Errorf("go_projection.mutable_closure_capture")
	}
	initialID, err := ref(capture, "000000000000000000000000000a0302")
	if err != nil {
		return "", err
	}
	initial, err := expr(initialID, c)
	if err != nil {
		return "", err
	}
	parameter, ok := c.graph[parameters[0]]
	if !ok || parameter.schema != sParameter {
		return "", fmt.Errorf("go_projection.mutable_closure_parameter")
	}
	parameterName, err := text(parameter, "00000000000000000000000000009120")
	if err != nil || !identifier(parameterName) {
		return "", fmt.Errorf("go_projection.mutable_closure_parameter")
	}
	typeID, err := ref(parameter, "00000000000000000000000000009121")
	if err != nil {
		return "", err
	}
	typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
	if err != nil {
		return "", err
	}
	sequence, ok := c.graph[bodyID]
	if !ok || sequence.schema != sSequence {
		return "", fmt.Errorf("go_projection.mutable_closure_sequence")
	}
	steps, err := refs(sequence, "000000000000000000000000000a0330")
	if err != nil || len(steps) != 1 {
		return "", fmt.Errorf("go_projection.mutable_closure_sequence")
	}
	update, ok := c.graph[steps[0]]
	if !ok || update.schema != sCaptureUpdate {
		return "", fmt.Errorf("go_projection.mutable_closure_update")
	}
	updatedCapture, err := ref(update, "000000000000000000000000000a0320")
	if err != nil || updatedCapture != captures[0] {
		return "", fmt.Errorf("go_projection.mutable_closure_update")
	}
	updatedID, err := ref(update, "000000000000000000000000000a0321")
	if err != nil {
		return "", err
	}
	resultID, err := ref(sequence, "000000000000000000000000000a0331")
	if err != nil {
		return "", err
	}
	next := c
	next.parameters = cloneNames(c.parameters)
	next.captures = cloneNames(c.captures)
	next.parameters[parameters[0]] = parameterName
	next.captures[captures[0]] = captureName
	updated, err := expr(updatedID, next)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(updated, "(") && strings.HasSuffix(updated, ")") {
		updated = updated[1 : len(updated)-1]
	}
	result, err := expr(resultID, next)
	if err != nil {
		return "", err
	}
	return "\t" + captureName + " := " + initial + "\n\treturn func(" + parameterName + " " + typ + ") int64 {\n\t\t" + captureName + " = " + updated + "\n\t\treturn " + result + "\n\t}", nil
}

func projectFoldReturn(resultID string, c context) ([]string, bool, error) {
	foldID := resultID
	lookupKey := ""
	result := c.graph[resultID]
	if result.schema == sMapLookup {
		var err error
		foldID, err = ref(result, "000000000000000000000000000a0420")
		if err != nil {
			return nil, false, err
		}
		lookupKey, err = ref(result, "000000000000000000000000000a0421")
		if err != nil {
			return nil, false, err
		}
	}
	fold, ok := c.graph[foldID]
	if !ok || fold.schema != sFold {
		return nil, false, nil
	}
	collectionID, err := ref(fold, "00000000000000000000000000009f70")
	if err != nil {
		return nil, false, err
	}
	initialID, err := ref(fold, "00000000000000000000000000009f71")
	if err != nil {
		return nil, false, err
	}
	accID, err := ref(fold, "00000000000000000000000000009f72")
	if err != nil {
		return nil, false, err
	}
	elementID, err := ref(fold, "00000000000000000000000000009f73")
	if err != nil {
		return nil, false, err
	}
	bodyID, err := ref(fold, "00000000000000000000000000009f74")
	if err != nil {
		return nil, false, err
	}
	accEntity, aok := c.graph[accID]
	elementEntity, eok := c.graph[elementID]
	if !aok || !eok || accEntity.schema != sIterationBinding || elementEntity.schema != sIterationBinding {
		return nil, false, fmt.Errorf("go_projection.invalid_fold_binding")
	}
	acc, err := text(accEntity, "00000000000000000000000000009f50")
	if err != nil || !identifier(acc) {
		return nil, false, fmt.Errorf("go_projection.invalid_fold_binding")
	}
	element, err := text(elementEntity, "00000000000000000000000000009f50")
	if err != nil || !identifier(element) || element == acc {
		return nil, false, fmt.Errorf("go_projection.invalid_fold_binding")
	}
	collection, err := expr(collectionID, c)
	if err != nil {
		return nil, false, err
	}
	initial, err := expr(initialID, c)
	if err != nil {
		return nil, false, err
	}
	if initialEntity, ok := c.graph[initialID]; ok && initialEntity.schema == sIntegerLiteral {
		initial = "int64(" + initial + ")"
	}
	c.iterations = cloneNames(c.iterations)
	c.iterations[accID], c.iterations[elementID] = acc, element
	lines := []string{"\t" + acc + " := " + initial, "\tfor _, " + element + " := range " + collection + " {"}
	body := c.graph[bodyID]
	if body.schema == sMapUpdate {
		mapID, err := ref(body, "000000000000000000000000000a0430")
		if err != nil {
			return nil, false, err
		}
		keyID, err := ref(body, "000000000000000000000000000a0431")
		if err != nil {
			return nil, false, err
		}
		valueID, err := ref(body, "000000000000000000000000000a0432")
		if err != nil {
			return nil, false, err
		}
		mapValue, err := expr(mapID, c)
		if err != nil {
			return nil, false, err
		}
		if mapValue != acc {
			return nil, false, fmt.Errorf("go_projection.fold_accumulator_mismatch")
		}
		key, err := expr(keyID, c)
		if err != nil {
			return nil, false, err
		}
		value, err := expr(valueID, c)
		if err != nil {
			return nil, false, err
		}
		// A map-fold update is already delimited by assignment. Avoid an outer
		// parenthesized binary expression so the Go provider observes the native
		// assignment shape without losing operator structure.
		if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
			value = strings.TrimSuffix(strings.TrimPrefix(value, "("), ")")
		}
		lines = append(lines, "\t\t"+acc+"["+key+"] = "+value)
	} else {
		if isFoldAddition(body, accID, elementID, c.graph) {
			lines = append(lines, "\t\t"+acc+" += "+element)
		} else {
			value, err := expr(bodyID, c)
			if err != nil {
				return nil, false, err
			}
			lines = append(lines, "\t\t"+acc+" = "+value)
		}
	}
	lines = append(lines, "\t}")
	if lookupKey != "" {
		key, err := expr(lookupKey, c)
		if err != nil {
			return nil, false, err
		}
		lines = append(lines, "\treturn "+acc+"["+key+"]")
	} else {
		lines = append(lines, "\treturn "+acc)
	}
	return lines, true, nil
}

func isFoldAddition(body entity, accumulatorID, elementID string, graph map[string]entity) bool {
	if body.schema != sAdd {
		return false
	}
	left, err := ref(body, "00000000000000000000000000009140")
	if err != nil {
		return false
	}
	right, err := ref(body, "00000000000000000000000000009141")
	if err != nil {
		return false
	}
	l, lok := graph[left]
	r, rok := graph[right]
	if !lok || !rok || l.schema != sIterationRead || r.schema != sIterationRead {
		return false
	}
	lid, le := ref(l, "00000000000000000000000000009f60")
	rid, re := ref(r, "00000000000000000000000000009f60")
	return le == nil && re == nil && lid == accumulatorID && rid == elementID
}

func cloneNames(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

// availableLocalName preserves the canonical binding name when Go's lexical
// namespace permits it and deterministically disambiguates it otherwise. Seme
// binding identities are independent of projection spelling, while ordinary Go
// cannot redeclare a parameter with := in the same function body.
func availableLocalName(name string, c context) string {
	used := map[string]bool{}
	for _, names := range []map[string]string{c.parameters, c.locals, c.iterations, c.variants, c.receivers, c.captures} {
		for _, existing := range names {
			used[existing] = true
		}
	}
	if !used[name] {
		return name
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s%d", name, suffix)
		if !used[candidate] {
			return candidate
		}
	}
}

func statefulResultIsObserved(bindingID string, graph map[string]entity) bool {
	reads := map[string]bool{}
	for id, item := range graph {
		if item.schema != sLocalRead {
			continue
		}
		if binding, err := ref(item, "00000000000000000000000000009d20"); err == nil && binding == bindingID {
			reads[id] = true
		}
	}
	for _, item := range graph {
		if item.schema != sTransitionResult {
			continue
		}
		if value, err := ref(item, "000000000000000000000000000a0070"); err == nil && reads[value] {
			return true
		}
	}
	return false
}

func expr(id string, c context) (string, error) {
	e, ok := c.graph[id]
	if !ok {
		return "", fmt.Errorf("go_projection.missing_expression")
	}
	switch e.schema {
	case sUnitValue:
		return "", nil
	case sRead:
		pid, err := ref(e, "00000000000000000000000000009130")
		if err != nil {
			return "", err
		}
		name, ok := c.parameters[pid]
		if !ok {
			return "", fmt.Errorf("go_projection.parameter_scope")
		}
		return name, nil
	case sLocalRead, sPlaceRead:
		field := "00000000000000000000000000009d20"
		if e.schema == sPlaceRead {
			field = "00000000000000000000000000009e20"
		}
		id, err := ref(e, field)
		if err != nil {
			return "", err
		}
		name, ok := c.locals[id]
		if !ok {
			return "", fmt.Errorf("go_projection.local_scope")
		}
		return name, nil
	case sIterationRead:
		id, err := ref(e, "00000000000000000000000000009f60")
		if err != nil {
			return "", err
		}
		name, ok := c.iterations[id]
		if !ok {
			return "", fmt.Errorf("go_projection.iteration_scope")
		}
		return name, nil
	case sVariantRead:
		id, err := ref(e, "000000000000000000000000000a0610")
		if err != nil {
			return "", err
		}
		value, ok := c.variants[id]
		if !ok {
			return "", fmt.Errorf("go_projection.variant_scope")
		}
		return value, nil
	case sOptionMatch, sResultMatch:
		return projectMatchExpression(e, c)
	case sMapLookupOption:
		mappingID, err := ref(e, "000000000000000000000000000a0440")
		if err != nil {
			return "", err
		}
		keyID, err := ref(e, "000000000000000000000000000a0441")
		if err != nil {
			return "", err
		}
		mapping, err := expr(mappingID, c)
		if err != nil {
			return "", err
		}
		key, err := expr(keyID, c)
		if err != nil {
			return "", err
		}
		return "func() Option[int64] { value, found := " + mapping + "[" + key + "]; return Option[int64]{Some: found, Value: value} }()", nil
	case sCall:
		fid, err := ref(e, "00000000000000000000000000009600")
		if err != nil {
			return "", err
		}
		name, ok := c.functions[fid]
		if !ok {
			return "", fmt.Errorf("go_projection.call_membership")
		}
		args, err := refs(e, "00000000000000000000000000009601")
		if err != nil {
			return "", err
		}
		rendered := make([]string, len(args))
		for i, arg := range args {
			rendered[i], err = expr(arg, c)
			if err != nil {
				return "", err
			}
		}
		return name + "(" + strings.Join(rendered, ", ") + ")", nil
	case sStringLiteral:
		value, err := text(e, "00000000000000000000000000009500")
		if err != nil {
			return "", err
		}
		return strconv.Quote(value), nil
	case sStringEqual:
		left, err := ref(e, "00000000000000000000000000009c20")
		if err != nil {
			return "", err
		}
		right, err := ref(e, "00000000000000000000000000009c21")
		if err != nil {
			return "", err
		}
		a, err := expr(left, c)
		if err != nil {
			return "", err
		}
		b, err := expr(right, c)
		if err != nil {
			return "", err
		}
		return a + " == " + b, nil
	case sBytesLiteral:
		value, err := rawBytes(e, "000000000000000000000000000a0640")
		if err != nil {
			return "", err
		}
		items := make([]string, len(value))
		for i, item := range value {
			items[i] = strconv.FormatUint(uint64(item), 10)
		}
		return "[]byte{" + strings.Join(items, ", ") + "}", nil
	case sBytesEqual:
		left, err := ref(e, "000000000000000000000000000a0650")
		if err != nil {
			return "", err
		}
		right, err := ref(e, "000000000000000000000000000a0651")
		if err != nil {
			return "", err
		}
		a, err := expr(left, c)
		if err != nil {
			return "", err
		}
		b, err := expr(right, c)
		if err != nil {
			return "", err
		}
		return "bytes.Equal(" + a + ", " + b + ")", nil
	case sOptionNone:
		typeID, err := ref(e, "000000000000000000000000000a0510")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		return typ + "{}", nil
	case sOptionSome:
		typeID, err := ref(e, "000000000000000000000000000a0520")
		if err != nil {
			return "", err
		}
		valueID, err := ref(e, "000000000000000000000000000a0521")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		value, err := expr(valueID, c)
		if err != nil {
			return "", err
		}
		return typ + "{Some: true, Value: " + value + "}", nil
	case sResultOk, sResultError:
		typeField, valueField, tag, name := "00000000000000000000000000009410", "00000000000000000000000000009411", "true", "Value"
		if e.schema == sResultError {
			typeField, valueField, tag, name = "00000000000000000000000000009420", "00000000000000000000000000009421", "false", "Error"
		}
		typeID, err := ref(e, typeField)
		if err != nil {
			return "", err
		}
		valueID, err := ref(e, valueField)
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		value, err := expr(valueID, c)
		if err != nil {
			return "", err
		}
		return typ + "{Ok: " + tag + ", " + name + ": " + value + "}", nil
	case sIntegerLiteral:
		value, err := unsigned(e, "00000000000000000000000000009700")
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(int64(value), 10), nil
	case sBoolLiteral:
		value, err := scalar(e, "00000000000000000000000000009b00")
		if err != nil || (value != "tr" && value != "fa") {
			return "", fmt.Errorf("go_projection.invalid_boolean")
		}
		return strconv.FormatBool(value == "tr"), nil
	case sRecordConstruct:
		typeID, err := ref(e, "00000000000000000000000000009330")
		if err != nil {
			return "", err
		}
		r, ok := c.records[typeID]
		if !ok {
			return "", fmt.Errorf("go_projection.record_type")
		}
		values, err := refs(e, "00000000000000000000000000009331")
		if err != nil || len(values) != len(r.fields) {
			return "", fmt.Errorf("go_projection.record_arity")
		}
		items := make([]string, len(values))
		for i, value := range values {
			rendered, x := expr(value, c)
			if x != nil {
				return "", x
			}
			items[i] = r.fields[i].name + ": " + rendered
		}
		return r.name + "{" + strings.Join(items, ", ") + "}", nil
	case sStateTransition:
		typeID, err := ref(e, "000000000000000000000000000a0050")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		stateID, err := ref(e, "000000000000000000000000000a0051")
		if err != nil {
			return "", err
		}
		resultID, err := ref(e, "000000000000000000000000000a0052")
		if err != nil {
			return "", err
		}
		state, err := expr(stateID, c)
		if err != nil {
			return "", err
		}
		result, err := expr(resultID, c)
		if err != nil {
			return "", err
		}
		return typ + "{State: " + state + ", Result: " + result + "}", nil
	case sFieldRead:
		valueID, err := ref(e, "00000000000000000000000000009320")
		if err != nil {
			return "", err
		}
		fieldID, err := ref(e, "00000000000000000000000000009321")
		if err != nil {
			return "", err
		}
		fieldName := ""
		for _, r := range c.records {
			for _, field := range r.fields {
				if field.id == fieldID {
					fieldName = field.name
				}
			}
		}
		if fieldName == "" {
			return "", fmt.Errorf("go_projection.record_field")
		}
		value, err := expr(valueID, c)
		if err != nil {
			return "", err
		}
		return value + "." + fieldName, nil
	case sFixedArrayConstruct:
		typeID, err := ref(e, "00000000000000000000000000009f30")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		values, err := refs(e, "00000000000000000000000000009f31")
		if err != nil {
			return "", err
		}
		rendered := make([]string, len(values))
		for i, value := range values {
			rendered[i], err = expr(value, c)
			if err != nil {
				return "", err
			}
		}
		return typ + "{" + strings.Join(rendered, ", ") + "}", nil
	case sIndexRead:
		collection, err := ref(e, "00000000000000000000000000009f40")
		if err != nil {
			return "", err
		}
		index, err := ref(e, "00000000000000000000000000009f41")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(index, c)
		if err != nil {
			return "", err
		}
		return a + "[" + b + "]", nil
	case sCollectionLength:
		collection, err := ref(e, "00000000000000000000000000009f90")
		if err != nil {
			return "", err
		}
		value, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		return "int64(len(" + value + "))", nil
	case sDynamicIndexRead:
		collection, err := ref(e, "00000000000000000000000000009fa0")
		if err != nil {
			return "", err
		}
		index, err := ref(e, "00000000000000000000000000009fa1")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(index, c)
		if err != nil {
			return "", err
		}
		return a + "[" + b + "]", nil
	case sReceiverRead:
		binding, err := ref(e, "000000000000000000000000000a0010")
		if err != nil {
			return "", err
		}
		name, ok := c.receivers[binding]
		if !ok {
			return "", fmt.Errorf("go_projection.receiver_scope")
		}
		return name, nil
	case sMethodCall:
		receiver, err := ref(e, "000000000000000000000000000a0030")
		if err != nil {
			return "", err
		}
		method, err := ref(e, "000000000000000000000000000a0031")
		if err != nil {
			return "", err
		}
		arguments, err := refs(e, "000000000000000000000000000a0032")
		if err != nil {
			return "", err
		}
		name, ok := c.methods[method]
		if !ok {
			return "", fmt.Errorf("go_projection.method_scope")
		}
		r, err := expr(receiver, c)
		if err != nil {
			return "", err
		}
		args := make([]string, len(arguments))
		for i, a := range arguments {
			args[i], err = expr(a, c)
			if err != nil {
				return "", err
			}
		}
		return r + "." + name + "(" + strings.Join(args, ", ") + ")", nil
	case sInterfaceValue:
		contract, err := ref(e, "000000000000000000000000000a0130")
		if err != nil {
			return "", err
		}
		value, err := ref(e, "000000000000000000000000000a0131")
		if err != nil {
			return "", err
		}
		witnessID, err := ref(e, "000000000000000000000000000a0132")
		if err != nil {
			return "", err
		}
		witness, ok := c.graph[witnessID]
		if !ok || witness.schema != sSatisfactionWitness {
			return "", fmt.Errorf("go_projection.interface_witness")
		}
		wContract, err := ref(witness, "000000000000000000000000000a0121")
		if err != nil || wContract != contract {
			return "", fmt.Errorf("go_projection.interface_witness")
		}
		return expr(value, c)
	case sDynamicMethodCall:
		receiver, err := ref(e, "000000000000000000000000000a0140")
		if err != nil {
			return "", err
		}
		requirementID, err := ref(e, "000000000000000000000000000a0141")
		if err != nil {
			return "", err
		}
		requirement, ok := c.graph[requirementID]
		if !ok || requirement.schema != sMethodRequirement {
			return "", fmt.Errorf("go_projection.dynamic_requirement")
		}
		name, err := text(requirement, "000000000000000000000000000a0110")
		if err != nil || !identifier(name) {
			return "", fmt.Errorf("go_projection.dynamic_requirement")
		}
		arguments, err := refs(e, "000000000000000000000000000a0142")
		if err != nil {
			return "", err
		}
		r, err := expr(receiver, c)
		if err != nil {
			return "", err
		}
		args := make([]string, len(arguments))
		for i, a := range arguments {
			args[i], err = expr(a, c)
			if err != nil {
				return "", err
			}
		}
		return r + "." + name + "(" + strings.Join(args, ", ") + ")", nil
	case sCaptureRead:
		binding, err := ref(e, "000000000000000000000000000a0220")
		if err != nil {
			return "", err
		}
		value, ok := c.captures[binding]
		if !ok {
			return "", fmt.Errorf("go_projection.capture_scope")
		}
		return value, nil
	case sClosureConstruct:
		typeID, err := ref(e, "000000000000000000000000000a0230")
		if err != nil {
			return "", err
		}
		parameters, err := refs(e, "000000000000000000000000000a0231")
		if err != nil {
			return "", err
		}
		captures, err := refs(e, "000000000000000000000000000a0232")
		if err != nil {
			return "", err
		}
		bodyID, err := ref(e, "000000000000000000000000000a0233")
		if err != nil {
			return "", err
		}
		functionType, ok := c.graph[typeID]
		if !ok || functionType.schema != sFunctionType {
			return "", fmt.Errorf("go_projection.closure_type")
		}
		typeParams, err := refs(functionType, "000000000000000000000000000a0200")
		if err != nil || len(typeParams) != len(parameters) {
			return "", fmt.Errorf("go_projection.closure_arity")
		}
		resultID, err := ref(functionType, "000000000000000000000000000a0201")
		if err != nil {
			return "", err
		}
		result, err := typeNameRelative(c.graph, resultID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		next := c
		next.parameters = cloneNames(c.parameters)
		next.captures = cloneNames(c.captures)
		declarations := make([]string, len(parameters))
		for i, pid := range parameters {
			p, ok := c.graph[pid]
			if !ok || p.schema != sParameter {
				return "", fmt.Errorf("go_projection.closure_parameter")
			}
			name, er := text(p, "00000000000000000000000000009120")
			if er != nil || !identifier(name) {
				return "", fmt.Errorf("go_projection.closure_parameter")
			}
			typ, er := typeNameRelative(c.graph, typeParams[i], c.typeNames, c.familyNames)
			if er != nil {
				return "", er
			}
			next.parameters[pid] = name
			declarations[i] = name + " " + typ
		}
		for _, captureID := range captures {
			binding, ok := c.graph[captureID]
			if !ok || binding.schema != sCaptureBinding {
				return "", fmt.Errorf("go_projection.closure_capture")
			}
			captured, er := ref(binding, "000000000000000000000000000a0212")
			if er != nil {
				return "", er
			}
			rendered, er := expr(captured, c)
			if er != nil {
				return "", er
			}
			next.captures[captureID] = rendered
		}
		body, err := expr(bodyID, next)
		if err != nil {
			return "", err
		}
		return "func(" + strings.Join(declarations, ", ") + ") " + result + " { return " + body + " }", nil
	case sIndirectCall:
		callee, err := ref(e, "000000000000000000000000000a0240")
		if err != nil {
			return "", err
		}
		arguments, err := refs(e, "000000000000000000000000000a0241")
		if err != nil {
			return "", err
		}
		fn, err := expr(callee, c)
		if err != nil {
			return "", err
		}
		args := make([]string, len(arguments))
		for i, a := range arguments {
			args[i], err = expr(a, c)
			if err != nil {
				return "", err
			}
		}
		return fn + "(" + strings.Join(args, ", ") + ")", nil
	case sMutableCaptureRead:
		binding, err := ref(e, "000000000000000000000000000a0310")
		if err != nil {
			return "", err
		}
		value, ok := c.captures[binding]
		if !ok {
			return "", fmt.Errorf("go_projection.mutable_capture_scope")
		}
		return value, nil
	case sStatefulCall:
		callee, err := ref(e, "000000000000000000000000000a0350")
		if err != nil {
			return "", err
		}
		arguments, err := refs(e, "000000000000000000000000000a0351")
		if err != nil {
			return "", err
		}
		fn, err := expr(callee, c)
		if err != nil {
			return "", err
		}
		args := make([]string, len(arguments))
		for i, a := range arguments {
			args[i], err = expr(a, c)
			if err != nil {
				return "", err
			}
		}
		return fn + "(" + strings.Join(args, ", ") + ")", nil
	case sTransitionResult, sTransitionState:
		fieldID := "000000000000000000000000000a0070"
		if e.schema == sTransitionState {
			fieldID = "000000000000000000000000000a0060"
		}
		valueID, err := ref(e, fieldID)
		if err != nil {
			return "", err
		}
		if read, ok := c.graph[valueID]; ok && read.schema == sLocalRead {
			binding, er := ref(read, "00000000000000000000000000009d20")
			if er == nil {
				if value, yes := c.transitions[binding]; yes {
					return value, nil
				}
			}
		}
		return expr(valueID, c)
	case sSliceConstruct:
		typeID, err := ref(e, "000000000000000000000000000a0680")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", err
		}
		values, err := refs(e, "000000000000000000000000000a0681")
		if err != nil {
			return "", err
		}
		rendered := make([]string, len(values))
		for i, value := range values {
			rendered[i], err = expr(value, c)
			if err != nil {
				return "", err
			}
		}
		return typ + "{" + strings.Join(rendered, ", ") + "}", nil
	case sCollectionAppend:
		collection, err := ref(e, "00000000000000000000000000009fb0")
		if err != nil {
			return "", err
		}
		value, err := ref(e, "00000000000000000000000000009fb1")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(value, c)
		if err != nil {
			return "", err
		}
		return "append(" + a + ", " + b + ")", nil
	case sCollectionUpdate:
		collection, err := ref(e, "00000000000000000000000000009fc0")
		if err != nil {
			return "", err
		}
		index, err := ref(e, "00000000000000000000000000009fc1")
		if err != nil {
			return "", err
		}
		value, err := ref(e, "00000000000000000000000000009fc2")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(index, c)
		if err != nil {
			return "", err
		}
		v, err := expr(value, c)
		if err != nil {
			return "", err
		}
		return "slices.Replace(slices.Clone(" + a + "), int(" + b + "), int(" + b + ")+1, " + v + ")", nil
	case sSliceRemove:
		collection, err := ref(e, "000000000000000000000000000a0660")
		if err != nil {
			return "", err
		}
		index, err := ref(e, "000000000000000000000000000a0661")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(index, c)
		if err != nil {
			return "", err
		}
		return "slices.Delete(slices.Clone(" + a + "), int(" + b + "), int(" + b + ")+1)", nil
	case sEmptyMap:
		typeID, err := ref(e, "000000000000000000000000000a0410")
		if err != nil {
			return "", fmt.Errorf("go_projection.empty_map_type: %w", err)
		}
		typ, err := typeNameRelative(c.graph, typeID, c.typeNames, c.familyNames)
		if err != nil {
			return "", fmt.Errorf("go_projection.empty_map_type: %w", err)
		}
		return typ + "{}", nil
	case sMapLookup:
		collection, err := ref(e, "000000000000000000000000000a0420")
		if err != nil {
			return "", err
		}
		key, err := ref(e, "000000000000000000000000000a0421")
		if err != nil {
			return "", err
		}
		a, err := expr(collection, c)
		if err != nil {
			return "", err
		}
		b, err := expr(key, c)
		if err != nil {
			return "", err
		}
		return a + "[" + b + "]", nil
	case sMapUpdate:
		mapping, err := ref(e, "000000000000000000000000000a0430")
		if err != nil {
			return "", err
		}
		key, err := ref(e, "000000000000000000000000000a0431")
		if err != nil {
			return "", err
		}
		value, err := ref(e, "000000000000000000000000000a0432")
		if err != nil {
			return "", err
		}
		m, err := expr(mapping, c)
		if err != nil {
			return "", err
		}
		k, err := expr(key, c)
		if err != nil {
			return "", err
		}
		v, err := expr(value, c)
		if err != nil {
			return "", err
		}
		return "func(m map[int64]int64, k, v int64) map[int64]int64 { out := maps.Clone(m); out[k] = v; return out }(" + m + ", " + k + ", " + v + ")", nil
	case sMapRemove:
		mapping, err := ref(e, "000000000000000000000000000a0670")
		if err != nil {
			return "", err
		}
		key, err := ref(e, "000000000000000000000000000a0671")
		if err != nil {
			return "", err
		}
		m, err := expr(mapping, c)
		if err != nil {
			return "", err
		}
		k, err := expr(key, c)
		if err != nil {
			return "", err
		}
		return "func(m map[int64]int64, k int64) map[int64]int64 { out := maps.Clone(m); delete(out, k); return out }(" + m + ", " + k + ")", nil
	case sFold:
		return "", fmt.Errorf("go_projection.fold_requires_statement_context")
	case sBooleanNot:
		value, err := ref(e, "000000000000000000000000000a0690")
		if err != nil {
			return "", err
		}
		projected, err := expr(value, c)
		if err != nil {
			return "", err
		}
		return "!(" + projected + ")", nil
	case sNativeInvocation:
		language, err := text(e, "000000000000000000000000000a06d0")
		if err != nil || language != "go" {
			return "", fmt.Errorf("go_projection.native_invocation_language")
		}
		callable, err := text(e, "000000000000000000000000000a06d1")
		if err != nil {
			return "", err
		}
		argumentIDs, err := refs(e, "000000000000000000000000000a06d3")
		if err != nil {
			return "", err
		}
		arguments := make([]string, len(argumentIDs))
		for index, argumentID := range argumentIDs {
			arguments[index], err = expr(argumentID, c)
			if err != nil {
				return "", err
			}
		}
		if strings.HasPrefix(callable, "builtin.") {
			if callable == "builtin.error.is_nil" {
				if len(arguments) != 1 {
					return "", fmt.Errorf("go_projection.native_builtin_arity")
				}
				return "(" + arguments[0] + " == nil)", nil
			}
			open, close := strings.IndexByte(callable, '['), strings.LastIndexByte(callable, ']')
			if open <= len("builtin.") || close < open {
				return "", fmt.Errorf("go_projection.native_builtin_callable")
			}
			name, payload := callable[len("builtin."):open], localizeNativeSpelling(callable[open+1:close], c)
			suffix := callable[close+1:]
			switch name {
			case "make":
				arguments = append([]string{payload}, arguments...)
			case "new":
				if !strings.HasPrefix(payload, "*") || len(arguments) != 0 {
					return "", fmt.Errorf("go_projection.native_builtin_new")
				}
				arguments = []string{strings.TrimPrefix(payload, "*")}
			case "append", "delete", "copy", "cap", "clear", "len":
			case "assignment_convert":
				if len(arguments) != 1 {
					return "", fmt.Errorf("go_projection.native_builtin_arity")
				}
				return payload + "(" + arguments[0] + ")", nil
			case "index":
				if len(arguments) != 2 {
					return "", fmt.Errorf("go_projection.native_builtin_arity")
				}
				return arguments[0] + "[" + arguments[1] + "]", nil
			case "add":
				if len(arguments) != 2 {
					return "", fmt.Errorf("go_projection.native_builtin_arity")
				}
				return "(" + arguments[0] + " + " + arguments[1] + ")", nil
			case "compare":
				parts := strings.Split(payload, ";")
				if len(parts) < 1 || len(arguments) != 2 {
					return "", fmt.Errorf("go_projection.native_builtin_compare")
				}
				return "(" + arguments[0] + " " + parts[0] + " " + arguments[1] + ")", nil
			case "compound":
				parts := strings.Split(payload, ";")
				if len(parts) < 1 || len(arguments) != 2 || !strings.HasSuffix(parts[0], "=") {
					return "", fmt.Errorf("go_projection.native_builtin_compound")
				}
				return "(" + arguments[0] + " " + strings.TrimSuffix(parts[0], "=") + " " + arguments[1] + ")", nil
			case "type_assert_comma_ok":
				if len(arguments) != 1 {
					return "", fmt.Errorf("go_projection.native_builtin_arity")
				}
				return arguments[0] + ".(" + payload + ")", nil
			case "composite_literal":
				parts := strings.SplitN(payload, ";", 2)
				if len(parts) != 2 {
					return "", fmt.Errorf("go_projection.native_builtin_composite")
				}
				shapes, values, argument := strings.Split(parts[1], ","), make([]string, 0), 0
				for _, shape := range shapes {
					if argument >= len(arguments) {
						return "", fmt.Errorf("go_projection.native_builtin_composite_arity")
					}
					switch {
					case shape == "position":
						values = append(values, arguments[argument])
						argument++
					case strings.HasPrefix(shape, "field:"):
						values = append(values, strings.TrimPrefix(shape, "field:")+": "+arguments[argument])
						argument++
					case shape == "key":
						if argument+1 >= len(arguments) {
							return "", fmt.Errorf("go_projection.native_builtin_composite_arity")
						}
						values = append(values, arguments[argument]+": "+arguments[argument+1])
						argument += 2
					default:
						return "", fmt.Errorf("go_projection.native_builtin_composite_shape")
					}
				}
				if argument != len(arguments) {
					return "", fmt.Errorf("go_projection.native_builtin_composite_arity")
				}
				return parts[0] + "{" + strings.Join(values, ", ") + "}", nil
			default:
				return "", fmt.Errorf("go_projection.native_builtin_name")
			}
			if suffix == ".ellipsis" {
				if name != "append" || len(arguments) == 0 {
					return "", fmt.Errorf("go_projection.native_builtin_ellipsis")
				}
				arguments[len(arguments)-1] += "..."
			} else if suffix != "" {
				return "", fmt.Errorf("go_projection.native_builtin_suffix")
			}
			return name + "(" + strings.Join(arguments, ", ") + ")", nil
		}
		dot := strings.LastIndexByte(callable, '.')
		if dot <= 0 || dot == len(callable)-1 {
			return "", fmt.Errorf("go_projection.native_invocation_callable")
		}
		path, name := callable[:dot], callable[dot+1:]
		if !identifier(name) {
			return "", fmt.Errorf("go_projection.native_invocation_callable")
		}
		// slices.Replace's indices are native Go ints. Seme's neutral integer
		// projections are int64, so restore the checked native call boundary.
		if callable == "slices.Replace" && len(arguments) >= 3 {
			arguments[1] = "int(" + arguments[1] + ")"
			arguments[2] = "int(" + arguments[2] + ")"
		}
		return filepath.Base(path) + "." + name + "(" + strings.Join(arguments, ", ") + ")", nil
	case sNativeAddress:
		language, languageErr := text(e, "000000000000000000000000000a07a0")
		operandID, operandErr := ref(e, "000000000000000000000000000a07a1")
		_, typeErr := ref(e, "000000000000000000000000000a07a2")
		if languageErr != nil || operandErr != nil || typeErr != nil || language != "go" {
			return "", fmt.Errorf("go_projection.native_address")
		}
		operand, err := expr(operandID, c)
		if err != nil {
			return "", err
		}
		return "&(" + operand + ")", nil
	case sNativeSlice:
		language, err := text(e, "000000000000000000000000000a07d0")
		if err != nil || language != "go" {
			return "", fmt.Errorf("go_projection.native_slice_language")
		}
		collectionID, err := ref(e, "000000000000000000000000000a07d1")
		if err != nil {
			return "", err
		}
		collection, err := expr(collectionID, c)
		if err != nil {
			return "", err
		}
		optional := func(field string) (string, bool, error) {
			ids, err := refs(e, field)
			if err != nil || len(ids) > 1 {
				return "", false, fmt.Errorf("go_projection.native_slice_bound")
			}
			if len(ids) == 0 {
				return "", false, nil
			}
			value, err := expr(ids[0], c)
			return value, true, err
		}
		low, _, err := optional("000000000000000000000000000a07d2")
		if err != nil {
			return "", err
		}
		high, _, err := optional("000000000000000000000000000a07d3")
		if err != nil {
			return "", err
		}
		maximum, hasMax, err := optional("000000000000000000000000000a07d4")
		if err != nil {
			return "", err
		}
		if _, err := ref(e, "000000000000000000000000000a07d5"); err != nil {
			return "", err
		}
		inside := low + ":" + high
		if hasMax {
			inside += ":" + maximum
		}
		return collection + "[" + inside + "]", nil
	case sNativeDereference:
		language, languageErr := text(e, "000000000000000000000000000a07e0")
		operandID, operandErr := ref(e, "000000000000000000000000000a07e1")
		_, typeErr := ref(e, "000000000000000000000000000a07e2")
		if languageErr != nil || operandErr != nil || typeErr != nil || language != "go" {
			return "", fmt.Errorf("go_projection.native_dereference")
		}
		operand, err := expr(operandID, c)
		if err != nil {
			return "", err
		}
		return "*(" + operand + ")", nil
	case sNativeBinary:
		language, languageErr := text(e, "000000000000000000000000000a07f0")
		operator, operatorErr := text(e, "000000000000000000000000000a07f1")
		leftID, leftErr := ref(e, "000000000000000000000000000a07f2")
		rightID, rightErr := ref(e, "000000000000000000000000000a07f3")
		_, typeErr := ref(e, "000000000000000000000000000a07f4")
		allowed := map[string]bool{"/": true, "%": true, "<<": true, ">>": true, "|": true, "&": true, "^": true, "&^": true}
		if languageErr != nil || operatorErr != nil || leftErr != nil || rightErr != nil || typeErr != nil || language != "go" || !allowed[operator] {
			return "", fmt.Errorf("go_projection.native_binary")
		}
		left, err := expr(leftID, c)
		if err != nil {
			return "", err
		}
		right, err := expr(rightID, c)
		if err != nil {
			return "", err
		}
		return "(" + left + " " + operator + " " + right + ")", nil
	case sAdd, sConcat, sMultiply, sSubtract, sLessEqual, sAnd, sOr:
		leftField, rightField := "00000000000000000000000000009140", "00000000000000000000000000009141"
		op := "+"
		if e.schema == sConcat {
			leftField, rightField = "00000000000000000000000000009c30", "00000000000000000000000000009c31"
		} else if e.schema == sMultiply {
			leftField, rightField, op = "00000000000000000000000000009900", "00000000000000000000000000009901", "*"
		} else if e.schema == sSubtract {
			leftField, rightField, op = "00000000000000000000000000009a00", "00000000000000000000000000009a01", "-"
		} else if e.schema == sLessEqual {
			leftField, rightField, op = "00000000000000000000000000009160", "00000000000000000000000000009161", "<="
		} else if e.schema == sAnd {
			leftField, rightField, op = "00000000000000000000000000009b10", "00000000000000000000000000009b11", "&&"
		} else if e.schema == sOr {
			leftField, rightField, op = "00000000000000000000000000009c10", "00000000000000000000000000009c11", "||"
		}
		left, err := ref(e, leftField)
		if err != nil {
			return "", err
		}
		right, err := ref(e, rightField)
		if err != nil {
			return "", err
		}
		l, err := expr(left, c)
		if err != nil {
			return "", err
		}
		r, err := expr(right, c)
		if err != nil {
			return "", err
		}
		return "(" + l + " " + op + " " + r + ")", nil
	default:
		return "", fmt.Errorf("go_projection.unsupported_expression:%s", e.schema)
	}
}

func localizeNativeSpelling(spelling string, c context) string {
	names := make([]string, 0, len(c.records)+len(c.typeNames))
	for _, record := range c.records {
		names = append(names, record.name)
	}
	for _, name := range c.typeNames {
		names = append(names, name)
	}
	for _, name := range names {
		qualified := regexp.MustCompile(`[[:alnum:]_./-]+\.` + regexp.QuoteMeta(name) + `\b`)
		spelling = qualified.ReplaceAllString(spelling, name)
	}
	return spelling
}

func projectMatchExpression(match entity, c context) (string, error) {
	valueField, firstBindingField, firstBlockField, secondBindingField, secondBlockField :=
		"000000000000000000000000000a0630", "000000000000000000000000000a0632", "000000000000000000000000000a0633", "", "000000000000000000000000000a0631"
	tag, firstField := ".Some", ".Value"
	if match.schema == sResultMatch {
		valueField, firstBindingField, firstBlockField = "000000000000000000000000000a0620", "000000000000000000000000000a0621", "000000000000000000000000000a0622"
		secondBindingField, secondBlockField = "000000000000000000000000000a0623", "000000000000000000000000000a0624"
		tag, firstField = ".Ok", ".Value"
	}
	valueID, err := ref(match, valueField)
	if err != nil {
		return "", err
	}
	firstBinding, err := ref(match, firstBindingField)
	if err != nil {
		return "", err
	}
	firstBlock, err := ref(match, firstBlockField)
	if err != nil {
		return "", err
	}
	secondBlock, err := ref(match, secondBlockField)
	if err != nil {
		return "", err
	}
	value, err := expr(valueID, c)
	if err != nil {
		return "", err
	}
	firstExprID, err := singleReturnExpression(c.graph, firstBlock)
	if err != nil {
		return "", err
	}
	secondExprID, err := singleReturnExpression(c.graph, secondBlock)
	if err != nil {
		return "", err
	}
	firstContext := c
	firstContext.variants = cloneNames(c.variants)
	firstContext.variants[firstBinding] = "matched" + firstField
	if secondBindingField != "" {
		secondBinding, x := ref(match, secondBindingField)
		if x != nil {
			return "", x
		}
		firstContext.variants[secondBinding] = "matched.Error"
	}
	first, err := expr(firstExprID, firstContext)
	if err != nil {
		return "", err
	}
	second, err := expr(secondExprID, firstContext)
	if err != nil {
		return "", err
	}
	returnType, err := expressionTypeName(firstExprID, firstBinding, c.graph)
	if err != nil {
		return "", err
	}
	secondType, secondTypeErr := expressionTypeName(secondExprID, firstBinding, c.graph)
	if secondTypeErr == nil && secondType != returnType && second == "0" {
		// Canonical optional/result payload matches use the scalar zero literal
		// for an absent payload. Go requires a type-correct zero for aggregate
		// payloads, so realize that neutral zero without naming constructors.
		second = returnType + "{}"
	}
	return "func() " + returnType + " { matched := " + value + "; if matched" + tag + " { return " + first + " }; return " + second + " }()", nil
}

func singleReturnExpression(graph map[string]entity, blockID string) (string, error) {
	block, ok := graph[blockID]
	if !ok || block.schema != sBlock {
		return "", fmt.Errorf("go_projection.match_block")
	}
	statements, err := refs(block, "00000000000000000000000000009800")
	if err != nil || len(statements) != 1 {
		return "", fmt.Errorf("go_projection.match_block")
	}
	returned, ok := graph[statements[0]]
	if !ok || returned.schema != sReturn {
		return "", fmt.Errorf("go_projection.match_return")
	}
	values, err := refs(returned, "00000000000000000000000000009810")
	if err != nil || len(values) != 1 {
		return "", fmt.Errorf("go_projection.match_return")
	}
	return values[0], nil
}

func expressionTypeName(expressionID, bindingID string, graph map[string]entity) (string, error) {
	expression, ok := graph[expressionID]
	if !ok {
		return "", fmt.Errorf("go_projection.match_expression")
	}
	switch expression.schema {
	case sBoolLiteral:
		return "bool", nil
	case sIntegerLiteral:
		return "int64", nil
	case sStringLiteral:
		return "string", nil
	case sVariantRead:
		binding, ok := graph[bindingID]
		if !ok || binding.schema != sVariantBinding {
			return "", fmt.Errorf("go_projection.match_binding")
		}
		typeID, err := ref(binding, "000000000000000000000000000a0601")
		if err != nil {
			return "", err
		}
		return typeName(graph, typeID)
	default:
		return "", fmt.Errorf("go_projection.match_result_type:%s", expression.schema)
	}
}

func typeName(g map[string]entity, id string) (string, error) {
	return typeNameRelative(g, id, nil, nil)
}

// typeNameRelative renders declaration and generic-family names relative to
// one package without changing their canonical identities.
func typeNameRelative(g map[string]entity, id string, names, families map[string]string) (string, error) {
	e, ok := g[id]
	if !ok {
		return "", fmt.Errorf("go_projection.missing_type:%s", id)
	}
	// Concrete applications of neutral generic families still need their type
	// arguments. Package ownership may attach the family declaration name to
	// the application identity; do not mistake that alias for a complete type.
	if name := names[id]; name != "" && e.schema != sOptionType && e.schema != sResultType && e.schema != sTransitionType {
		return name, nil
	}
	switch e.schema {
	case sInteger:
		return "int64", nil
	case sBoolean:
		return "bool", nil
	case sString:
		return "string", nil
	case sBytes:
		return "[]byte", nil
	case sOptionType:
		value, err := ref(e, "000000000000000000000000000a0500")
		if err != nil {
			return "", err
		}
		typ, err := typeNameRelative(g, value, names, families)
		if err != nil {
			return "", err
		}
		family := "Option"
		if families["option"] != "" {
			family = families["option"]
		}
		return family + "[" + typ + "]", nil
	case sResultType:
		success, err := ref(e, "00000000000000000000000000009400")
		if err != nil {
			return "", err
		}
		failure, err := ref(e, "00000000000000000000000000009401")
		if err != nil {
			return "", err
		}
		st, err := typeNameRelative(g, success, names, families)
		if err != nil {
			return "", err
		}
		ft, err := typeNameRelative(g, failure, names, families)
		if err != nil {
			return "", err
		}
		family := "Result"
		if families["result"] != "" {
			family = families["result"]
		}
		return family + "[" + st + ", " + ft + "]", nil
	case sRecordType:
		name, err := text(e, "00000000000000000000000000009300")
		if err != nil || !identifier(name) {
			return "", fmt.Errorf("go_projection.invalid_record_name")
		}
		return name, nil
	case sInterfaceType:
		name, err := text(e, "000000000000000000000000000a0100")
		if err != nil || !identifier(name) {
			return "", fmt.Errorf("go_projection.invalid_interface_name")
		}
		return name, nil
	case sFunctionType:
		parameters, err := refs(e, "000000000000000000000000000a0200")
		if err != nil {
			return "", err
		}
		resultID, err := ref(e, "000000000000000000000000000a0201")
		if err != nil {
			return "", err
		}
		rendered := make([]string, len(parameters))
		for i, p := range parameters {
			rendered[i], err = typeNameRelative(g, p, names, families)
			if err != nil {
				return "", err
			}
		}
		result, err := typeNameRelative(g, resultID, names, families)
		if err != nil {
			return "", err
		}
		return "func(" + strings.Join(rendered, ", ") + ") " + result, nil
	case sTransitionType:
		stateID, err := ref(e, "000000000000000000000000000a0040")
		if err != nil {
			return "", err
		}
		resultID, err := ref(e, "000000000000000000000000000a0041")
		if err != nil {
			return "", err
		}
		state, err := typeNameRelative(g, stateID, names, families)
		if err != nil {
			return "", err
		}
		result, err := typeNameRelative(g, resultID, names, families)
		if err != nil {
			return "", err
		}
		family := "Transition"
		if families["transition"] != "" {
			family = families["transition"]
		}
		return family + "[" + state + ", " + result + "]", nil
	case sFixedArrayType:
		element, err := ref(e, "00000000000000000000000000009f20")
		if err != nil {
			return "", err
		}
		if typ, err := typeNameRelative(g, element, names, families); err != nil || typ != "int64" {
			return "", fmt.Errorf("go_projection.unsupported_array_element")
		}
		length, err := unsigned(e, "00000000000000000000000000009f21")
		if err != nil || length > 32 {
			return "", fmt.Errorf("go_projection.invalid_array_length")
		}
		return fmt.Sprintf("[%d]int64", length), nil
	case sSliceType:
		element, err := ref(e, "00000000000000000000000000009f80")
		if err != nil {
			return "", err
		}
		if typ, err := typeNameRelative(g, element, names, families); err != nil || typ != "int64" {
			return "", fmt.Errorf("go_projection.unsupported_slice_element")
		}
		return "[]int64", nil
	case sMapType:
		key, err := ref(e, "000000000000000000000000000a0400")
		if err != nil {
			return "", err
		}
		value, err := ref(e, "000000000000000000000000000a0401")
		if err != nil {
			return "", err
		}
		kt, ke := typeNameRelative(g, key, names, families)
		vt, ve := typeNameRelative(g, value, names, families)
		if ke != nil || ve != nil || kt != "int64" || vt != "int64" {
			return "", fmt.Errorf("go_projection.unsupported_map_type")
		}
		return "map[int64]int64", nil
	case sNativeType:
		language, err := text(e, "000000000000000000000000000a0710")
		if err != nil || language != "go" {
			return "", fmt.Errorf("go_projection.native_type_language")
		}
		spelling, err := text(e, "000000000000000000000000000a0711")
		if err != nil || spelling == "" {
			return "", fmt.Errorf("go_projection.native_type_spelling")
		}
		localTypeNames := make([]string, 0, len(names))
		for _, localName := range names {
			localTypeNames = append(localTypeNames, localName)
		}
		for _, candidate := range g {
			if candidate.schema == sRecordType {
				if localName, nameErr := text(candidate, "00000000000000000000000000009300"); nameErr == nil {
					localTypeNames = append(localTypeNames, localName)
				}
			}
		}
		for _, localName := range localTypeNames {
			qualified := regexp.MustCompile(`[[:alnum:]_./-]+\.` + regexp.QuoteMeta(localName) + `\b`)
			spelling = qualified.ReplaceAllString(spelling, localName)
		}
		if _, err := parser.ParseExpr(spelling); err != nil {
			return "", fmt.Errorf("go_projection.native_type_spelling")
		}
		return spelling, nil
	case sUnitType:
		return "", nil
	}
	return "", fmt.Errorf("go_projection.unsupported_type:%s", e.schema)
}

func collectRecords(graph map[string]entity) (map[string]record, error) {
	result := map[string]record{}
	for id, e := range graph {
		if e.schema != sRecordType {
			continue
		}
		name, err := text(e, "00000000000000000000000000009300")
		if err != nil || !identifier(name) {
			return nil, fmt.Errorf("go_projection.invalid_record_name")
		}
		fieldIDs, err := refs(e, "00000000000000000000000000009301")
		if err != nil {
			return nil, err
		}
		r := record{name: name}
		for position, fid := range fieldIDs {
			field, ok := graph[fid]
			if !ok || field.schema != sRecordField {
				return nil, fmt.Errorf("go_projection.invalid_record_field")
			}
			fieldName, err := text(field, "00000000000000000000000000009310")
			if err != nil || !identifier(fieldName) {
				return nil, fmt.Errorf("go_projection.invalid_record_field_name")
			}
			typeID, err := ref(field, "00000000000000000000000000009311")
			if err != nil {
				return nil, err
			}
			typ, err := typeName(graph, typeID)
			if err != nil {
				return nil, err
			}
			actual, err := unsigned(field, "00000000000000000000000000009312")
			if err != nil || actual != uint64(position) {
				return nil, fmt.Errorf("go_projection.record_field_order")
			}
			r.fields = append(r.fields, recordField{fid, fieldName, typ})
		}
		result[id] = r
	}
	return result, nil
}
func sortedRecordIDs(records map[string]record) []string {
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func graphHasSchema(graph map[string]entity, schema string) bool {
	for _, item := range graph {
		if item.schema == schema {
			return true
		}
	}
	return false
}
func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func parse(source []byte) (map[string]entity, error) {
	lines := strings.Split(strings.TrimSpace(string(source)), "\n")
	graph := map[string]entity{}
	for i := 0; i < len(lines); {
		parts := strings.Fields(lines[i])
		if len(parts) == 0 || parts[0] != "en" {
			i++
			continue
		}
		if len(parts) != 5 {
			return nil, fmt.Errorf("go_projection.malformed_entity")
		}
		count, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, fmt.Errorf("go_projection.malformed_entity")
		}
		e := entity{parts[1], parts[2], map[string][]string{}}
		i++
		for n := 0; n < count; n++ {
			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
			}
			if i >= len(lines) {
				return nil, fmt.Errorf("go_projection.truncated_field")
			}
			p := strings.Fields(lines[i])
			if len(p) < 3 || p[0] != "fi" {
				return nil, fmt.Errorf("go_projection.malformed_field:%q", lines[i])
			}
			key := p[1]
			value := []string{strings.Join(p[2:], " ")}
			i++
			if p[2] == "li" {
				if len(p) != 4 {
					return nil, fmt.Errorf("go_projection.malformed_list")
				}
				size, x := strconv.Atoi(p[3])
				if x != nil || i+size > len(lines) {
					return nil, fmt.Errorf("go_projection.malformed_list")
				}
				for j := 0; j < size; j++ {
					value = append(value, strings.TrimSpace(lines[i]))
					i++
				}
			}
			if _, exists := e.fields[key]; exists {
				return nil, fmt.Errorf("go_projection.duplicate_field")
			}
			e.fields[key] = value
		}
		if _, exists := graph[e.id]; exists {
			return nil, fmt.Errorf("go_projection.duplicate_entity")
		}
		graph[e.id] = e
	}
	return graph, nil
}
func scalar(e entity, field string) (string, error) {
	v, ok := e.fields[field]
	if !ok || len(v) != 1 {
		return "", fmt.Errorf("go_projection.missing_field:%s:%s", e.schema, field)
	}
	return v[0], nil
}
func ref(e entity, field string) (string, error) {
	v, err := scalar(e, field)
	if err != nil {
		return "", err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "rf" {
		return "", fmt.Errorf("go_projection.invalid_reference")
	}
	return p[1], nil
}
func refs(e entity, field string) ([]string, error) {
	v, ok := e.fields[field]
	if !ok || len(v) < 1 {
		return nil, fmt.Errorf("go_projection.missing_field:%s:%s", e.schema, field)
	}
	p := strings.Fields(v[0])
	if len(p) != 2 || p[0] != "li" {
		return nil, fmt.Errorf("go_projection.invalid_reference_list")
	}
	count, err := strconv.Atoi(p[1])
	if err != nil || len(v) != count+1 {
		return nil, fmt.Errorf("go_projection.invalid_reference_list")
	}
	out := make([]string, count)
	for i := range out {
		p = strings.Fields(v[i+1])
		if len(p) != 2 || p[0] != "rf" {
			return nil, fmt.Errorf("go_projection.invalid_reference")
		}
		out[i] = p[1]
	}
	return out, nil
}
func unsigned(e entity, field string) (uint64, error) {
	v, err := scalar(e, field)
	if err != nil {
		return 0, err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "uu" {
		return 0, fmt.Errorf("go_projection.invalid_unsigned")
	}
	value, err := strconv.ParseUint(p[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("go_projection.invalid_unsigned")
	}
	return value, nil
}
func text(e entity, field string) (string, error) {
	v, err := scalar(e, field)
	if err != nil {
		return "", err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "by" {
		return "", fmt.Errorf("go_projection.invalid_text")
	}
	if p[1] == "-" {
		return "", nil
	}
	b, err := hex.DecodeString(p[1])
	if err != nil || !utf8.Valid(b) {
		return "", fmt.Errorf("go_projection.invalid_utf8")
	}
	return string(b), nil
}
func rawBytes(e entity, field string) ([]byte, error) {
	v, err := scalar(e, field)
	if err != nil {
		return nil, err
	}
	p := strings.Fields(v)
	if len(p) != 2 || p[0] != "by" {
		return nil, fmt.Errorf("go_projection.invalid_bytes")
	}
	if p[1] == "-" {
		return []byte{}, nil
	}
	value, err := hex.DecodeString(p[1])
	if err != nil {
		return nil, fmt.Errorf("go_projection.invalid_bytes")
	}
	return value, nil
}
