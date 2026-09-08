package wasmtarget

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"seme.local/reference/wire"
)

type PureABI struct {
	Contract             string         `json:"contract"`
	Provider             string         `json:"provider"`
	Target               string         `json:"target"`
	Fidelity             string         `json:"fidelity"`
	CanonicalModule      string         `json:"canonical_module"`
	CanonicalRevision    string         `json:"canonical_revision"`
	CanonicalProgram     string         `json:"canonical_program"`
	Function             string         `json:"function"`
	ProgramSHA256        string         `json:"program_sha256,omitempty"`
	ArtifactSHA256       string         `json:"artifact_sha256,omitempty"`
	RequestSize          uint64         `json:"request_size"`
	ResponseSize         uint64         `json:"response_size"`
	FixedHeaderSize      uint64         `json:"fixed_header_size,omitempty"`
	MaximumRequestSize   uint64         `json:"maximum_request_size,omitempty"`
	VariablePayload      bool           `json:"variable_payload,omitempty"`
	RequiredCapabilities []string       `json:"required_capabilities,omitempty"`
	Parameters           []PureABIField `json:"parameters"`
	Result               PureABIField   `json:"result"`
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
	abi.RequiredCapabilities = append([]string(nil), certificate.abi.RequiredCapabilities...)
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
	listed, listedErr := field(programs[0], 0x9150)
	if listedErr == nil && listed.Tag == 7 && len(listed.List) > 1 {
		normalized, callErr := normalizePureCalls(graph, programs[0])
		if callErr != nil {
			return nil, PureABI{}, callErr
		}
		graph = normalized
		programs = bySchema(graph, 0x9015)
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
	if result, resultErr := field(function, 0x9112); resultErr == nil && result.Tag == 6 {
		if resultEntity, exists := graph.Entities[result.Reference]; exists && resultEntity.Schema == identity(0xa004) {
			return certifyPureTransitionFunction(graph, programs[0], function)
		}
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
	hasVariableValues := false
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
		hasVariableValues = hasVariableValues || valueType.name == "string" || valueType.name == "slice:i64"
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
	hasVariableValues = hasVariableValues || resultType.name == "string" || resultType.name == "slice:i64"
	abi.Result = PureABIField{Index: 0, Type: resultType.name, Offset: 0, Size: resultType.size, Encoding: pureEncoding(resultType.name)}
	abi.ResponseSize = resultType.size
	bodyValue, err := field(function, 0x9113)
	if err != nil || bodyValue.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.pure_body")
	}
	graph, err = normalizePureLocals(graph, bodyValue.Reference, parameterTypeNames)
	if err != nil {
		return nil, PureABI{}, err
	}
	graph, err = normalizePureRecords(graph, bodyValue.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	if len(bySchema(graph, 0x90f1)) > 0 {
		return certifyPureEffectFunction(graph, bodyValue.Reference, parameterTypes, resultType, parameterTypeNames, parameterLocals, abi)
	}
	hasState := len(bySchema(graph, 0x90e0))+len(bySchema(graph, 0x90e1))+len(bySchema(graph, 0x90e2))+len(bySchema(graph, 0x90e3))+len(bySchema(graph, 0x90e4))+len(bySchema(graph, 0x90f0)) > 0
	if hasVariableValues || hasState {
		abi.Contract = "seme.pure-abi/v2"
		abi.Provider = "seme.function-v2"
		abi.FixedHeaderSize = abi.RequestSize
		abi.MaximumRequestSize = 7160
		abi.VariablePayload = true
		if resultType.name == "string" || resultType.name == "slice:i64" {
			abi.ResponseSize = 0
		}
		return certifyPureStringFunction(graph, bodyValue.Reference, parameterTypes, resultType, parameterTypeNames, parameterLocals, abi)
	}
	used := map[byte]bool{}
	budget := 4096
	instructions, err := lowerPureBlock(graph, bodyValue.Reference, resultType.name, parameterTypeNames, parameterLocals, used, map[wire.ID]bool{}, &budget)
	if err != nil {
		return nil, PureABI{}, err
	}
	wasm, err := pureModule(parameterTypes, resultType, instructions, abi, false)
	return wasm, abi, err
}

func lowerPureBlock(graph wire.Envelope, id wire.ID, resultType string, parameterTypes map[wire.ID]string, parameterLocals map[wire.ID]byte, used map[byte]bool, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.pure_control_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	block, ok := graph.Entities[id]
	if !ok || block.Schema != identity(0x9080) {
		return nil, fmt.Errorf("wasm.pure_block")
	}
	statements, err := field(block, 0x9800)
	if err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return nil, fmt.Errorf("wasm.pure_block")
	}
	statement, ok := graph.Entities[statements.List[0].Reference]
	if !ok {
		return nil, fmt.Errorf("wasm.pure_statement_missing")
	}
	switch statement.Schema {
	case identity(0x9081):
		values, err := field(statement, 0x9810)
		if err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
			return nil, fmt.Errorf("wasm.pure_return_values")
		}
		if err := validatePureExpression(graph, values.List[0].Reference, resultType, parameterTypes, map[wire.ID]bool{}, budget); err != nil {
			return nil, err
		}
		if resultType == "i64" {
			return lowerHelperInteger(graph, values.List[0].Reference, parameterLocals, used, map[wire.ID]bool{}, budget)
		}
		return lowerHelperBoolean(graph, values.List[0].Reference, parameterLocals, used, map[wire.ID]bool{}, budget)
	case identity(0x90c0):
		condition, conditionErr := field(statement, 0x9c00)
		thenValue, thenErr := field(statement, 0x9c01)
		elseValue, elseErr := field(statement, 0x9c02)
		if conditionErr != nil || thenErr != nil || elseErr != nil || condition.Tag != 6 || thenValue.Tag != 6 || elseValue.Tag != 6 {
			return nil, fmt.Errorf("wasm.pure_if_fields")
		}
		if err := validatePureExpression(graph, condition.Reference, "bool", parameterTypes, map[wire.ID]bool{}, budget); err != nil {
			return nil, err
		}
		conditionCode, err := lowerHelperBoolean(graph, condition.Reference, parameterLocals, used, map[wire.ID]bool{}, budget)
		if err != nil {
			return nil, err
		}
		thenCode, err := lowerPureBlock(graph, thenValue.Reference, resultType, parameterTypes, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		elseCode, err := lowerPureBlock(graph, elseValue.Reference, resultType, parameterTypes, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		wasmType := byte(0x7f)
		if resultType == "i64" {
			wasmType = 0x7e
		}
		instructions := append(conditionCode, 0x04, wasmType)
		instructions = append(instructions, thenCode...)
		instructions = append(instructions, 0x05)
		instructions = append(instructions, elseCode...)
		return append(instructions, 0x0b), nil
	default:
		return nil, fmt.Errorf("wasm.pure_return")
	}
}

func pureEncoding(name string) string {
	if name == "bool" {
		return "canonical-u8-0-or-1"
	}
	if name == "string" {
		return "u32le-offset-u32le-length/utf8-scalar-exact"
	}
	if strings.HasPrefix(name, "fixed-array:i64:") {
		return "packed-little-endian-twos-complement-i64"
	}
	if name == "slice:i64" {
		return "u32le-offset-u32le-element-count/packed-i64"
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
	case identity(0x90c1):
		if expected != "bool" {
			return fmt.Errorf("wasm.pure_boolean_or_type")
		}
		return validatePair(0x9c10, 0x9c11, "bool")
	case identity(0x90c2):
		if expected != "bool" {
			return fmt.Errorf("wasm.pure_string_equal_type")
		}
		return validatePair(0x9c20, 0x9c21, "string")
	case identity(0x90c3):
		if expected != "string" {
			return fmt.Errorf("wasm.pure_string_concat_type")
		}
		return validatePair(0x9c30, 0x9c31, "string")
	case identity(0x9050):
		literal, err := field(expression, 0x9500)
		if expected != "string" || err != nil || literal.Tag != 5 || !utf8.Valid(literal.Bytes) {
			return fmt.Errorf("wasm.pure_string_literal_type")
		}
		return nil
	case identity(0x90f4):
		if expected != "i64" {
			return fmt.Errorf("wasm.fixed_array_result_type")
		}
		collection, collectionErr := field(expression, 0x9f40)
		index, indexErr := field(expression, 0x9f41)
		if collectionErr != nil || indexErr != nil || collection.Tag != 6 || index.Tag != 6 {
			return fmt.Errorf("wasm.index_read_fields")
		}
		collectionEntity, exists := graph.Entities[collection.Reference]
		if !exists {
			return fmt.Errorf("wasm.index_read_collection")
		}
		if collectionEntity.Schema == identity(0x90f3) {
			values, err := fixedI64ArrayValues(graph, collection.Reference)
			if err != nil {
				return err
			}
			for _, value := range values {
				if err := validatePureExpression(graph, value, "i64", parameterTypes, visiting, budget); err != nil {
					return err
				}
			}
		} else if collectionEntity.Schema == identity(0x9013) {
			parameter, err := field(collectionEntity, 0x9130)
			if err != nil || parameter.Tag != 6 || !strings.HasPrefix(parameterTypes[parameter.Reference], "fixed-array:i64:") {
				return fmt.Errorf("wasm.index_read_collection_type")
			}
		} else {
			return fmt.Errorf("wasm.index_read_collection")
		}
		return validatePureExpression(graph, index.Reference, "i64", parameterTypes, visiting, budget)
	case identity(0x90f7):
		if expected != "i64" {
			return fmt.Errorf("wasm.fold_result_type")
		}
		collection, initial, accumulator, element, body, err := foldFields(graph, expression)
		if err != nil {
			return err
		}
		if _, _, err := fixedI64Collection(graph, collection, parameterTypes); err != nil {
			return err
		}
		if err := validatePureExpression(graph, initial, "i64", parameterTypes, visiting, budget); err != nil {
			return err
		}
		return validateI64AddFoldBody(graph, body, accumulator, element)
	case identity(0x90f9):
		if expected != "i64" {
			return fmt.Errorf("wasm.collection_length_result_type")
		}
		collection, err := field(expression, 0x9f90)
		if err != nil || collection.Tag != 6 || validateI64Collection(graph, collection.Reference, parameterTypes) != nil {
			return fmt.Errorf("wasm.collection_length_collection")
		}
		return nil
	case identity(0x90fa):
		if expected != "i64" {
			return fmt.Errorf("wasm.dynamic_index_result_type")
		}
		collection, collectionErr := field(expression, 0x9fa0)
		index, indexErr := field(expression, 0x9fa1)
		if collectionErr != nil || indexErr != nil || collection.Tag != 6 || index.Tag != 6 || validateI64Collection(graph, collection.Reference, parameterTypes) != nil {
			return fmt.Errorf("wasm.dynamic_index_fields")
		}
		return validatePureExpression(graph, index.Reference, "i64", parameterTypes, visiting, budget)
	case identity(0x90fb):
		if expected != "slice:i64" {
			return fmt.Errorf("wasm.collection_append_result_type")
		}
		collection, collectionErr := field(expression, 0x9fb0)
		value, valueErr := field(expression, 0x9fb1)
		if collectionErr != nil || valueErr != nil || collection.Tag != 6 || value.Tag != 6 {
			return fmt.Errorf("wasm.collection_append_fields")
		}
		if err := validatePureExpression(graph, collection.Reference, "slice:i64", parameterTypes, visiting, budget); err != nil {
			return err
		}
		return validatePureExpression(graph, value.Reference, "i64", parameterTypes, visiting, budget)
	case identity(0x90fc):
		if expected != "slice:i64" {
			return fmt.Errorf("wasm.collection_update_result_type")
		}
		collection, collectionErr := field(expression, 0x9fc0)
		index, indexErr := field(expression, 0x9fc1)
		value, valueErr := field(expression, 0x9fc2)
		if collectionErr != nil || indexErr != nil || valueErr != nil || collection.Tag != 6 || index.Tag != 6 || value.Tag != 6 {
			return fmt.Errorf("wasm.collection_update_fields")
		}
		if err := validatePureExpression(graph, collection.Reference, "slice:i64", parameterTypes, visiting, budget); err != nil {
			return err
		}
		if err := validatePureExpression(graph, index.Reference, "i64", parameterTypes, visiting, budget); err != nil {
			return err
		}
		return validatePureExpression(graph, value.Reference, "i64", parameterTypes, visiting, budget)
	default:
		return fmt.Errorf("wasm.pure_unsupported_expression")
	}
}

func validateI64Collection(graph wire.Envelope, id wire.ID, parameterTypes map[wire.ID]string) error {
	collection, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.collection_missing")
	}
	if collection.Schema == identity(0x90f3) {
		_, err := fixedI64ArrayValues(graph, id)
		return err
	}
	if collection.Schema != identity(0x9013) {
		return fmt.Errorf("wasm.collection_expression")
	}
	parameter, err := field(collection, 0x9130)
	typeName := parameterTypes[parameter.Reference]
	if err != nil || parameter.Tag != 6 || (typeName != "slice:i64" && !strings.HasPrefix(typeName, "fixed-array:i64:")) {
		return fmt.Errorf("wasm.collection_type")
	}
	return nil
}

func foldFields(graph wire.Envelope, fold wire.Entity) (wire.ID, wire.ID, wire.ID, wire.ID, wire.ID, error) {
	fields := make([]wire.ID, 5)
	for index, fieldID := range []uint64{0x9f70, 0x9f71, 0x9f72, 0x9f73, 0x9f74} {
		value, err := field(fold, fieldID)
		if err != nil || value.Tag != 6 {
			return wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, fmt.Errorf("wasm.fold_fields")
		}
		fields[index] = value.Reference
	}
	if fields[2] == fields[3] {
		return wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, fmt.Errorf("wasm.fold_binding_alias")
	}
	for _, id := range fields[2:4] {
		binding, ok := graph.Entities[id]
		if !ok || binding.Schema != identity(0x90f5) {
			return wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, fmt.Errorf("wasm.fold_binding")
		}
		name, nameErr := field(binding, 0x9f50)
		typeValue, typeErr := field(binding, 0x9f51)
		if nameErr != nil || typeErr != nil || name.Tag != 5 || len(name.Bytes) == 0 || typeValue.Tag != 6 {
			return wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, fmt.Errorf("wasm.fold_binding_fields")
		}
		type_, err := pureType(graph, typeValue.Reference)
		if err != nil || type_.name != "i64" {
			return wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, wire.ID{}, fmt.Errorf("wasm.fold_binding_type")
		}
	}
	return fields[0], fields[1], fields[2], fields[3], fields[4], nil
}

func validateI64AddFoldBody(graph wire.Envelope, bodyID, accumulatorID, elementID wire.ID) error {
	body, ok := graph.Entities[bodyID]
	if !ok || body.Schema != identity(0x9014) || validateIntegerTypeReference(graph, body, 0x9142) != nil {
		return fmt.Errorf("wasm.fold_body")
	}
	left, leftErr := field(body, 0x9140)
	right, rightErr := field(body, 0x9141)
	if leftErr != nil || rightErr != nil || left.Tag != 6 || right.Tag != 6 {
		return fmt.Errorf("wasm.fold_body")
	}
	for _, expectedRead := range []struct {
		id        wire.ID
		bindingID wire.ID
	}{
		{id: left.Reference, bindingID: accumulatorID},
		{id: right.Reference, bindingID: elementID},
	} {
		read, ok := graph.Entities[expectedRead.id]
		if !ok || read.Schema != identity(0x90f6) {
			return fmt.Errorf("wasm.fold_binding_read")
		}
		binding, err := field(read, 0x9f60)
		if err != nil || binding.Tag != 6 || binding.Reference != expectedRead.bindingID {
			return fmt.Errorf("wasm.fold_binding_scope")
		}
	}
	return nil
}

func fixedI64Collection(graph wire.Envelope, collectionID wire.ID, parameterTypes map[wire.ID]string) ([]wire.ID, wire.ID, error) {
	collection, ok := graph.Entities[collectionID]
	if !ok {
		return nil, wire.ID{}, fmt.Errorf("wasm.fold_collection")
	}
	if collection.Schema == identity(0x90f3) {
		values, err := fixedI64ArrayValues(graph, collectionID)
		return values, wire.ID{}, err
	}
	if collection.Schema != identity(0x9013) {
		return nil, wire.ID{}, fmt.Errorf("wasm.fold_collection")
	}
	parameter, err := field(collection, 0x9130)
	if err != nil || parameter.Tag != 6 || (!strings.HasPrefix(parameterTypes[parameter.Reference], "fixed-array:i64:") && parameterTypes[parameter.Reference] != "slice:i64") {
		return nil, wire.ID{}, fmt.Errorf("wasm.fold_collection_type")
	}
	return nil, parameter.Reference, nil
}

func fixedI64ArrayValues(graph wire.Envelope, constructID wire.ID) ([]wire.ID, error) {
	construct, ok := graph.Entities[constructID]
	if !ok || construct.Schema != identity(0x90f3) {
		return nil, fmt.Errorf("wasm.fixed_array_construct")
	}
	typeValue, typeErr := field(construct, 0x9f30)
	valuesValue, valuesErr := field(construct, 0x9f31)
	if typeErr != nil || valuesErr != nil || typeValue.Tag != 6 || valuesValue.Tag != 7 || len(valuesValue.List) > 32 {
		return nil, fmt.Errorf("wasm.fixed_array_fields")
	}
	arrayType, ok := graph.Entities[typeValue.Reference]
	if !ok || arrayType.Schema != identity(0x90f2) {
		return nil, fmt.Errorf("wasm.fixed_array_type")
	}
	element, elementErr := field(arrayType, 0x9f20)
	length, lengthErr := field(arrayType, 0x9f21)
	elementType, elementTypeErr := pureType(graph, element.Reference)
	if elementErr != nil || lengthErr != nil || element.Tag != 6 || length.Tag != 3 || length.Unsigned != uint64(len(valuesValue.List)) || elementTypeErr != nil || elementType.name != "i64" {
		return nil, fmt.Errorf("wasm.fixed_array_profile")
	}
	values := make([]wire.ID, len(valuesValue.List))
	for index, value := range valuesValue.List {
		if value.Tag != 6 {
			return nil, fmt.Errorf("wasm.fixed_array_value")
		}
		values[index] = value.Reference
	}
	return values, nil
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
	case identity(0x9040):
		return pureValueType{"string", 0x7e, 8}, nil
	case identity(0x90f2):
		element, elementErr := field(entity, 0x9f20)
		length, lengthErr := field(entity, 0x9f21)
		if elementErr != nil || lengthErr != nil || element.Tag != 6 || length.Tag != 3 || length.Unsigned > 32 {
			return pureValueType{}, fmt.Errorf("wasm.fixed_array_type")
		}
		elementType, err := pureType(graph, element.Reference)
		if err != nil || elementType.name != "i64" {
			return pureValueType{}, fmt.Errorf("wasm.fixed_array_element_type")
		}
		return pureValueType{fmt.Sprintf("fixed-array:i64:%d", length.Unsigned), 0x7f, length.Unsigned * 8}, nil
	case identity(0x90f8):
		element, err := field(entity, 0x9f80)
		if err != nil || element.Tag != 6 {
			return pureValueType{}, fmt.Errorf("wasm.slice_type")
		}
		elementType, err := pureType(graph, element.Reference)
		if err != nil || elementType.name != "i64" {
			return pureValueType{}, fmt.Errorf("wasm.slice_element_type")
		}
		return pureValueType{"slice:i64", 0x7e, 8}, nil
	default:
		return pureValueType{}, fmt.Errorf("wasm.pure_unsupported_type")
	}
}

func pureModule(parameters []pureValueType, result pureValueType, expression []byte, abi PureABI, effectImport bool) ([]byte, error) {
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
	functionOffset := uint64(0)
	if effectImport {
		var imports bytes.Buffer
		uleb(&imports, 1)
		name(&imports, "pulp")
		name(&imports, "log_bool")
		imports.Write([]byte{0, 0})
		section(&wasm, 2, imports.Bytes())
		functionOffset = 1
	}
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
	export(&exports, "pulp_alloc", 0, functionOffset+0)
	export(&exports, "pulp_free", 0, functionOffset+1)
	export(&exports, "pulp_init", 0, functionOffset+2)
	export(&exports, "pulp_step", 0, functionOffset+3)
	export(&exports, "pulp_shutdown", 0, functionOffset+4)
	export(&exports, "pulp_on_call", 0, functionOffset+6)
	section(&wasm, 7, exports.Bytes())
	bodies := [][]byte{
		pureAllocatorBody(), pureFreeBody(),
		{0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b},
		append(append([]byte{0}, expression...), 0x0b),
		pureProviderBody(parameters, result, abi, byte(functionOffset+5)),
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

func pureProviderBody(parameters []pureValueType, result pureValueType, abi PureABI, helperIndex byte) []byte {
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
		if strings.HasPrefix(parameter.name, "fixed-array:i64:") {
			body.Write([]byte{0x20, 2, 0x41})
			sleb(&body, int64(offset))
			body.WriteByte(0x6a)
		} else {
			body.Write([]byte{0x20, 2})
			if parameter.name == "i64" {
				body.Write([]byte{0x29, 3})
			} else {
				body.Write([]byte{0x2d, 0})
			}
			uleb(&body, offset)
		}
		offset += parameter.size
	}
	body.Write([]byte{0x10, helperIndex, 0x21, 6})
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
