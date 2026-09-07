package wasmtarget

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

type PureABI struct {
	Contract          string         `json:"contract"`
	Provider          string         `json:"provider"`
	Target            string         `json:"target"`
	Fidelity          string         `json:"fidelity"`
	CanonicalModule   string         `json:"canonical_module"`
	CanonicalRevision string         `json:"canonical_revision"`
	CanonicalProgram  string         `json:"canonical_program"`
	Function          string         `json:"function"`
	ProgramSHA256     string         `json:"program_sha256,omitempty"`
	ArtifactSHA256    string         `json:"artifact_sha256,omitempty"`
	RequestSize       uint64         `json:"request_size"`
	ResponseSize      uint64         `json:"response_size"`
	Parameters        []PureABIField `json:"parameters"`
	Result            PureABIField   `json:"result"`
}

type PureABIField struct {
	Index    uint64 `json:"index"`
	Type     string `json:"type"`
	Offset   uint64 `json:"offset"`
	Size     uint64 `json:"size"`
	Encoding string `json:"encoding"`
}

type pureValueType struct {
	name string
	wasm byte
	size uint64
}

// PureFunctionCertificate is an immutable executable plan. Its private fields
// ensure that Wasm emission cannot accept an unvalidated canonical graph.
type PureFunctionCertificate struct {
	wasm []byte
	abi  PureABI
}

// ABI returns a defensive copy of the certified target layout.
func (certificate PureFunctionCertificate) ABI() PureABI {
	abi := certificate.abi
	abi.Parameters = append([]PureABIField(nil), certificate.abi.Parameters...)
	return abi
}

// CertifyPureFunction checks the complete bounded structured pure-function
// contract and produces an immutable executable plan.
func CertifyPureFunction(graph wire.Envelope) (PureFunctionCertificate, error) {
	wasm, abi, err := certifyPureFunction(graph)
	if err != nil {
		return PureFunctionCertificate{}, err
	}
	return PureFunctionCertificate{wasm: append([]byte(nil), wasm...), abi: abi}, nil
}

// LowerCertifiedPureFunction emits only a previously certified plan.
func LowerCertifiedPureFunction(certificate PureFunctionCertificate) ([]byte, PureABI, error) {
	if len(certificate.wasm) == 0 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_certificate_invalid")
	}
	return append([]byte(nil), certificate.wasm...), certificate.ABI(), nil
}

func certifyPureFunction(graph wire.Envelope) ([]byte, PureABI, error) {
	programs := bySchema(graph, 0x9015)
	if len(programs) != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_program_cardinality")
	}
	functionsValue, err := field(programs[0], 0x9150)
	if err != nil || functionsValue.Tag != 7 || len(functionsValue.List) != 1 || functionsValue.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_program_functions")
	}
	entryValue, err := field(programs[0], 0x9151)
	if err != nil || entryValue.Tag != 6 || functionsValue.List[0].Reference != entryValue.Reference {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_entry_membership")
	}
	function, ok := graph.Entities[entryValue.Reference]
	if !ok || function.Schema != identity(0x9011) {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_entry_function")
	}
	functionName, err := field(function, 0x9110)
	if err != nil || functionName.Tag != 5 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_function_name")
	}
	parametersValue, err := field(function, 0x9111)
	if err != nil || parametersValue.Tag != 7 || len(parametersValue.List) > 32 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_parameters")
	}
	parameterTypes := make([]pureValueType, len(parametersValue.List))
	parameterLocals := make(map[wire.ID]byte, len(parametersValue.List))
	parameterTypeNames := make(map[wire.ID]string, len(parametersValue.List))
	abi := PureABI{
		Contract: "seme.pure-abi/v1", Provider: "seme.function-v1",
		Target: "wasm32-pulp-reactor-v1", Fidelity: "exact",
		CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(),
		CanonicalProgram: programs[0].ID.String(), Function: function.ID.String(),
	}
	seenParameters := make(map[wire.ID]bool, len(parametersValue.List))
	for index, item := range parametersValue.List {
		if item.Tag != 6 || seenParameters[item.Reference] {
			return nil, PureABI{}, fmt.Errorf("wasm.pure_parameter_membership")
		}
		seenParameters[item.Reference] = true
		parameter, exists := graph.Entities[item.Reference]
		if !exists || parameter.Schema != identity(0x9012) {
			return nil, PureABI{}, fmt.Errorf("wasm.pure_parameter_missing")
		}
		name, nameErr := field(parameter, 0x9120)
		position, positionErr := field(parameter, 0x9122)
		typeValue, typeErr := field(parameter, 0x9121)
		if nameErr != nil || name.Tag != 5 || positionErr != nil || position.Tag != 3 || typeErr != nil || typeValue.Tag != 6 || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.pure_parameter_order")
		}
		valueType, err := pureType(graph, typeValue.Reference)
		if err != nil {
			return nil, PureABI{}, err
		}
		parameterTypes[index] = valueType
		parameterLocals[parameter.ID] = byte(index)
		parameterTypeNames[parameter.ID] = valueType.name
		abi.Parameters = append(abi.Parameters, PureABIField{Index: uint64(index), Type: valueType.name, Offset: abi.RequestSize, Size: valueType.size, Encoding: pureEncoding(valueType.name)})
		abi.RequestSize += valueType.size
	}
	resultValue, err := field(function, 0x9112)
	if err != nil || resultValue.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_result")
	}
	resultType, err := pureType(graph, resultValue.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	abi.Result = PureABIField{Index: 0, Type: resultType.name, Offset: 0, Size: resultType.size, Encoding: pureEncoding(resultType.name)}
	abi.ResponseSize = resultType.size
	bodyValue, err := field(function, 0x9113)
	if err != nil || bodyValue.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_body")
	}
	body, ok := graph.Entities[bodyValue.Reference]
	if !ok || body.Schema != identity(0x9080) {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_body")
	}
	statements, err := field(body, 0x9800)
	if err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_block")
	}
	returned, ok := graph.Entities[statements.List[0].Reference]
	if !ok || returned.Schema != identity(0x9081) {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_return")
	}
	values, err := field(returned, 0x9810)
	if err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_return_values")
	}
	used := map[byte]bool{}
	visiting := map[wire.ID]bool{}
	budget := 4096
	if err := validatePureExpression(graph, values.List[0].Reference, resultType.name, parameterTypeNames, map[wire.ID]bool{}, &budget); err != nil {
		return nil, PureABI{}, err
	}
	budget = 4096
	var instructions []byte
	if resultType.name == "i64" {
		instructions, err = lowerHelperInteger(graph, values.List[0].Reference, parameterLocals, used, visiting, &budget)
	} else {
		instructions, err = lowerHelperBoolean(graph, values.List[0].Reference, parameterLocals, used, visiting, &budget)
	}
	if err != nil {
		return nil, PureABI{}, err
	}
	wasm, err := pureModule(parameterTypes, resultType, instructions, abi)
	return wasm, abi, err
}

func pureEncoding(name string) string {
	if name == "bool" {
		return "canonical-u8-0-or-1"
	}
	return "little-endian-twos-complement-i64-modular"
}

func validatePureExpression(graph wire.Envelope, id wire.ID, expected string, parameterTypes map[wire.ID]string, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.pure_expression_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.pure_expression_missing")
	}
	validatePair := func(leftField, rightField uint64, childType string) error {
		left, leftErr := field(expression, leftField)
		right, rightErr := field(expression, rightField)
		if leftErr != nil || rightErr != nil || left.Tag != 6 || right.Tag != 6 {
			return fmt.Errorf("wasm.pure_expression_fields")
		}
		if err := validatePureExpression(graph, left.Reference, childType, parameterTypes, visiting, budget); err != nil {
			return err
		}
		return validatePureExpression(graph, right.Reference, childType, parameterTypes, visiting, budget)
	}
	switch expression.Schema {
	case identity(0x9013):
		parameter, err := field(expression, 0x9130)
		if err != nil || parameter.Tag != 6 || parameterTypes[parameter.Reference] != expected {
			return fmt.Errorf("wasm.pure_parameter_read_type")
		}
		return nil
	case identity(0x9070):
		if expected != "i64" || validateIntegerTypeReference(graph, expression, 0x9701) != nil {
			return fmt.Errorf("wasm.pure_integer_literal_type")
		}
		return nil
	case identity(0x9014):
		if expected != "i64" || validateIntegerTypeReference(graph, expression, 0x9142) != nil {
			return fmt.Errorf("wasm.pure_integer_add_type")
		}
		return validatePair(0x9140, 0x9141, "i64")
	case identity(0x9090):
		if expected != "i64" || validateIntegerTypeReference(graph, expression, 0x9902) != nil {
			return fmt.Errorf("wasm.pure_integer_multiply_type")
		}
		return validatePair(0x9900, 0x9901, "i64")
	case identity(0x90a0):
		if expected != "i64" || validateIntegerTypeReference(graph, expression, 0x9a02) != nil {
			return fmt.Errorf("wasm.pure_integer_subtract_type")
		}
		return validatePair(0x9a00, 0x9a01, "i64")
	case identity(0x9021):
		if expected != "bool" || validateIntegerTypeReference(graph, expression, 0x9162) != nil {
			return fmt.Errorf("wasm.pure_integer_comparison_type")
		}
		return validatePair(0x9160, 0x9161, "i64")
	case identity(0x90b0):
		literal, err := field(expression, 0x9b00)
		if expected != "bool" || err != nil || (literal.Tag != 1 && literal.Tag != 2) {
			return fmt.Errorf("wasm.pure_boolean_literal_type")
		}
		return nil
	case identity(0x90b1):
		if expected != "bool" {
			return fmt.Errorf("wasm.pure_boolean_and_type")
		}
		return validatePair(0x9b10, 0x9b11, "bool")
	default:
		return fmt.Errorf("wasm.pure_unsupported_expression")
	}
}

func pureType(graph wire.Envelope, id wire.ID) (pureValueType, error) {
	entity, ok := graph.Entities[id]
	if !ok {
		return pureValueType{}, fmt.Errorf("wasm.pure_type_missing")
	}
	switch entity.Schema {
	case identity(0x9010):
		width, widthErr := field(entity, 0x9100)
		signed, signedErr := field(entity, 0x9101)
		overflow, overflowErr := field(entity, 0x9102)
		if widthErr != nil || signedErr != nil || overflowErr != nil || width.Tag != 3 || signed.Tag != 2 || overflow.Tag != 3 || width.Unsigned != 64 || overflow.Unsigned != 0 {
			return pureValueType{}, fmt.Errorf("wasm.pure_integer_profile")
		}
		return pureValueType{"i64", 0x7e, 8}, nil
	case identity(0x9020):
		return pureValueType{"bool", 0x7f, 1}, nil
	default:
		return pureValueType{}, fmt.Errorf("wasm.pure_unsupported_type")
	}
}

func pureModule(parameters []pureValueType, result pureValueType, expression []byte, abi PureABI) ([]byte, error) {
	if len(expression) == 0 || abi.RequestSize > 65535 || abi.ResponseSize > 8 {
		return nil, fmt.Errorf("wasm.pure_layout")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 6)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	parameterWasm := make([]byte, len(parameters))
	for index, parameter := range parameters {
		parameterWasm[index] = parameter.wasm
	}
	functionType(&types, parameterWasm, []byte{result.wasm})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{7, 0, 5, 3, 3, 2, 4, 1})
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, 1024)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	var exports bytes.Buffer
	uleb(&exports, 7)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 0)
	export(&exports, "pulp_free", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, 6)
	section(&wasm, 7, exports.Bytes())
	bodies := [][]byte{
		pureAllocatorBody(), pureFreeBody(),
		{0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b},
		append(append([]byte{0}, expression...), 0x0b),
		pureProviderBody(parameters, result, abi),
	}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), nil
}

func pureAllocatorBody() []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 2, 0x7f}) // allocation pointer and next pointer
	// Reject zero and all sizes that cannot fit in the bounded 1024..8192 arena.
	body.Write([]byte{0x20, 0, 0x45, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b})
	body.Write([]byte{0x20, 0, 0x41})
	sleb(&body, 7160)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b}) // size > 7160
	body.Write([]byte{0x23, 0, 0x22, 1, 0x20, 0, 0x6a, 0x41, 8, 0x6a, 0x22, 2, 0x41})
	sleb(&body, 8192)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 0, 0x0f, 0x0b}) // next > arena end
	body.Write([]byte{0x20, 2, 0x24, 0, 0x20, 1, 0x0b})
	return body.Bytes()
}

func pureFreeBody() []byte {
	var body bytes.Buffer
	body.WriteByte(0) // no locals
	// Reclaim only a valid top-of-stack arena allocation. Foreign response
	// scratch pointers and out-of-order frees are deliberately ignored.
	body.Write([]byte{0x20, 0, 0x41})
	sleb(&body, 1024)
	body.Write([]byte{0x4f, 0x20, 0, 0x41}) // ptr >= 1024, ptr <= 8192
	sleb(&body, 8192)
	body.Write([]byte{0x4d, 0x71, 0x20, 1, 0x45, 0x45, 0x71, 0x20, 1, 0x41}) // size != 0
	sleb(&body, 7160)
	body.Write([]byte{0x4d, 0x71, 0x04, 0x40}) // size <= 7160
	body.Write([]byte{0x20, 0, 0x20, 1, 0x6a, 0x41, 8, 0x6a, 0x23, 0, 0x46, 0x04, 0x40})
	body.Write([]byte{0x20, 0, 0x24, 0, 0x0b, 0x0b, 0x0b})
	return body.Bytes()
}

func functionType(output *bytes.Buffer, parameters, results []byte) {
	output.WriteByte(0x60)
	uleb(output, uint64(len(parameters)))
	output.Write(parameters)
	uleb(output, uint64(len(results)))
	output.Write(results)
}

func pureProviderBody(parameters []pureValueType, result pureValueType, abi PureABI) []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 1, result.wasm}) // one result temporary at local 6
	body.Write([]byte{0x20, 3, 0x41})
	sleb(&body, int64(abi.RequestSize))
	body.Write([]byte{0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	var offset uint64
	for _, parameter := range parameters {
		if parameter.name == "bool" {
			body.Write([]byte{0x20, 2, 0x2d, 0})
			uleb(&body, offset)
			body.Write([]byte{0x41, 1, 0x4d, 0x04, 0x40, 0x05, 0x41, 3, 0x0f, 0x0b})
		}
		offset += parameter.size
	}
	offset = 0
	for _, parameter := range parameters {
		body.Write([]byte{0x20, 2})
		if parameter.name == "i64" {
			body.Write([]byte{0x29, 3})
		} else {
			body.Write([]byte{0x2d, 0})
		}
		uleb(&body, offset)
		offset += parameter.size
	}
	body.Write([]byte{0x10, 5, 0x21, 6})
	constI32(&body, 8192)
	body.Write([]byte{0x20, 6})
	if result.name == "i64" {
		body.Write([]byte{0x37, 3, 0})
	} else {
		body.Write([]byte{0x3a, 0, 0})
	}
	body.Write([]byte{0x20, 4})
	constI32(&body, 8192)
	body.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41})
	sleb(&body, int64(abi.ResponseSize))
	body.Write([]byte{0x36, 2, 0, 0x41, 0, 0x0b})
	return body.Bytes()
}
