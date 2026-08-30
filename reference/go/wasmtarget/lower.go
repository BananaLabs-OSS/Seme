// Package wasmtarget implements the first deliberately scoped canonical
// Seme-to-WebAssembly backend. It consumes only a validated semantic plan.
package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

func Lower(plan wire.Envelope) ([]byte, error) {
	plans := bySchema(plan, 0xc014)
	boundaries := bySchema(plan, 0xc015)
	functions := bySchema(plan, 0x9011)
	effects := bySchema(plan, 0x15)
	if len(plans) != 1 || len(functions) != 2 || len(effects) != 1 {
		return nil, fmt.Errorf("wasm.profile_cardinality")
	}
	if value, err := field(plans[0], 0xc144); err != nil || value.Tag != 2 {
		return nil, fmt.Errorf("wasm.plan_not_executable")
	}
	if len(boundaries) != 1 {
		return nil, fmt.Errorf("wasm.boundary_cardinality")
	}
	resolutionList, err := field(plans[0], 0xc142)
	if err != nil || len(resolutionList.List) != 2 {
		return nil, fmt.Errorf("wasm.resolutions")
	}
	for _, item := range resolutionList.List {
		resolution, ok := plan.Entities[item.Reference]
		if !ok || resolution.Schema != identity(0xc013) {
			return nil, fmt.Errorf("wasm.resolution_missing")
		}
		fidelity, err := field(resolution, 0xc133)
		if err != nil || fidelity.Unsigned != 2 {
			return nil, fmt.Errorf("wasm.resolution_not_adapted")
		}
		if selected, err := field(resolution, 0xc132); err != nil || selected.Tag != 6 {
			return nil, fmt.Errorf("wasm.resolution_rule_missing")
		}
	}
	transport, err := field(boundaries[0], 0xc153)
	if err != nil {
		return nil, err
	}
	transportEntity, ok := plan.Entities[transport.Reference]
	if !ok || transportEntity.Schema != identity(0xb013) {
		return nil, fmt.Errorf("wasm.transport_missing")
	}
	transportName, err := field(transportEntity, 0xb130)
	if err != nil || string(transportName.Bytes) != "seme.pulp.log-v1" {
		return nil, fmt.Errorf("wasm.unsupported_transport")
	}
	effectName, err := field(effects[0], 0x150)
	if err != nil || string(effectName.Bytes) != "observability.log" {
		return nil, fmt.Errorf("wasm.unsupported_effect")
	}
	var entry, helper wire.Entity
	for _, function := range functions {
		name, nameErr := field(function, 0x9110)
		if nameErr == nil && string(name.Bytes) == "Admit" {
			entry = function
		}
		if nameErr == nil && string(name.Bytes) != "Admit" {
			helper = function
		}
	}
	if entry.ID == (wire.ID{}) || helper.ID == (wire.ID{}) {
		return nil, fmt.Errorf("wasm.function_names")
	}
	if err := validateDecisionHelper(plan, helper); err != nil {
		return nil, err
	}
	layout, err := validateFunction(plan, entry, helper)
	if err != nil {
		return nil, err
	}
	return module(layout)
}

type applicationLayout struct {
	requestOffsets     [3]uint64
	requestHeaderSize  uint64
	responseBoolOffset uint64
	responseHeaderSize uint64
	errorMessage       []byte
}

func validateFunction(graph wire.Envelope, function, helper wire.Entity) (applicationLayout, error) {
	var layout applicationLayout
	parameters, err := field(function, 0x9111)
	if err != nil || len(parameters.List) != 1 {
		return layout, fmt.Errorf("wasm.function_parameters")
	}
	parameter, ok := graph.Entities[parameters.List[0].Reference]
	if !ok || parameter.Schema != identity(0x9012) {
		return layout, fmt.Errorf("wasm.parameter_missing")
	}
	position, err := field(parameter, 0x9122)
	if err != nil || position.Unsigned != 0 {
		return layout, fmt.Errorf("wasm.parameter_order")
	}
	requestType, err := referenced(graph, parameter, 0x9121, 0x9030)
	if err != nil {
		return layout, err
	}
	requestFields, err := validateRecord(graph, requestType, "AdmitRequest", []recordFieldProfile{{"Current", 0x9010, 8}, {"Delta", 0x9010, 8}, {"Limit", 0x9010, 8}, {"Subject", 0x9040, 4}, {"Evidence", 0x9041, 4}})
	if err != nil {
		return layout, err
	}
	for index, item := range requestFields[:3] {
		layout.requestOffsets[index] = item.offset
	}
	layout.requestHeaderSize = requestFields[len(requestFields)-1].offset + requestFields[len(requestFields)-1].size
	resultValue, err := field(function, 0x9112)
	if err != nil {
		return layout, fmt.Errorf("wasm.function_result")
	}
	resultType, ok := graph.Entities[resultValue.Reference]
	if !ok || resultType.Schema != identity(0x9042) {
		return layout, fmt.Errorf("wasm.function_result")
	}
	okType, okErr := field(resultType, 0x9400)
	errorType, errorErr := field(resultType, 0x9401)
	responseType, responseOK := graph.Entities[okType.Reference]
	errorTypeEntity, errorOK := graph.Entities[errorType.Reference]
	if okErr != nil || errorErr != nil || !responseOK || responseType.Schema != identity(0x9030) || !errorOK || errorTypeEntity.Schema != identity(0x9030) {
		return layout, fmt.Errorf("wasm.result_profile")
	}
	responseFields, err := validateRecord(graph, responseType, "AdmitResponse", []recordFieldProfile{{"Accepted", 0x9020, 1}, {"Subject", 0x9040, 4}, {"Evidence", 0x9041, 4}})
	if err != nil {
		return layout, err
	}
	layout.responseBoolOffset = 1 + responseFields[0].offset
	layout.responseHeaderSize = 1 + responseFields[len(responseFields)-1].offset + responseFields[len(responseFields)-1].size
	errorFields, err := validateRecord(graph, errorTypeEntity, "AdmitError", []recordFieldProfile{{"Message", 0x9040, 4}})
	if err != nil {
		return layout, err
	}
	conditional, err := referenced(graph, function, 0x9113, 0x9052)
	if err != nil {
		return layout, err
	}
	condition, err := referenced(graph, conditional, 0x9520, 0x9051)
	if err != nil {
		return layout, err
	}
	conditionLeft, err := referenced(graph, condition, 0x9510, 0x9032)
	if err != nil {
		return layout, err
	}
	conditionField, conditionFieldErr := field(conditionLeft, 0x9321)
	if conditionFieldErr != nil || conditionField.Reference != requestFields[3].entity.ID {
		return layout, fmt.Errorf("wasm.error_condition_field")
	}
	resultError, err := referenced(graph, conditional, 0x9521, 0x9044)
	if err != nil {
		return layout, err
	}
	resultErrorType, typeErr := field(resultError, 0x9420)
	if typeErr != nil || resultErrorType.Reference != resultType.ID {
		return layout, fmt.Errorf("wasm.result_error_type")
	}
	errorConstruct, err := referenced(graph, resultError, 0x9421, 0x9033)
	if err != nil {
		return layout, err
	}
	errorConstructType, errorConstructTypeErr := field(errorConstruct, 0x9330)
	errorValues, errorValuesErr := field(errorConstruct, 0x9331)
	if errorConstructTypeErr != nil || errorValuesErr != nil || errorConstructType.Reference != errorTypeEntity.ID || len(errorValues.List) != len(errorFields) {
		return layout, fmt.Errorf("wasm.error_construct")
	}
	errorLiteral, ok := graph.Entities[errorValues.List[0].Reference]
	if !ok || errorLiteral.Schema != identity(0x9050) {
		return layout, fmt.Errorf("wasm.error_literal")
	}
	errorMessage, err := field(errorLiteral, 0x9500)
	if err != nil || len(errorMessage.Bytes) == 0 || len(errorMessage.Bytes) > 4096 {
		return layout, fmt.Errorf("wasm.error_message")
	}
	layout.errorMessage = append([]byte(nil), errorMessage.Bytes...)
	resultOk, err := referenced(graph, conditional, 0x9522, 0x9043)
	if err != nil {
		return layout, err
	}
	resultOKType, resultTypeErr := field(resultOk, 0x9410)
	if resultTypeErr != nil || resultOKType.Reference != resultType.ID {
		return layout, fmt.Errorf("wasm.result_ok_type")
	}
	body, err := referenced(graph, resultOk, 0x9411, 0x9033)
	if err != nil {
		return layout, err
	}
	constructedType, err := field(body, 0x9330)
	if err != nil || constructedType.Reference != responseType.ID {
		return layout, fmt.Errorf("wasm.response_construct_type")
	}
	values, err := field(body, 0x9331)
	if err != nil || len(values.List) != 3 {
		return layout, fmt.Errorf("wasm.response_construct_values")
	}
	call, ok := graph.Entities[values.List[0].Reference]
	if !ok || call.Schema != identity(0x9060) {
		return layout, fmt.Errorf("wasm.response_value")
	}
	callee, calleeErr := field(call, 0x9600)
	arguments, argumentsErr := field(call, 0x9601)
	if calleeErr != nil || argumentsErr != nil || callee.Reference != helper.ID || len(arguments.List) != 3 {
		return layout, fmt.Errorf("wasm.helper_call")
	}
	for index, argument := range arguments.List {
		expression, exists := graph.Entities[argument.Reference]
		if !exists || expression.Schema != identity(0x9032) {
			return layout, fmt.Errorf("wasm.helper_argument")
		}
		read, err := referenced(graph, expression, 0x9320, 0x9013)
		if err != nil {
			return layout, err
		}
		readParameter, err := field(read, 0x9130)
		selectedField, selectedErr := field(expression, 0x9321)
		if err != nil || selectedErr != nil || readParameter.Reference != parameter.ID || selectedField.Reference != requestFields[index].entity.ID {
			return layout, fmt.Errorf("wasm.expression_record_field")
		}
	}
	for index, fieldIndex := range []int{3, 4} {
		value, ok := graph.Entities[values.List[index+1].Reference]
		if !ok || value.Schema != identity(0x9032) {
			return layout, fmt.Errorf("wasm.response_variable_value")
		}
		selected, selectedErr := field(value, 0x9321)
		if selectedErr != nil || selected.Reference != requestFields[fieldIndex].entity.ID {
			return layout, fmt.Errorf("wasm.response_variable_field")
		}
	}
	return layout, nil
}

func validateDecisionHelper(graph wire.Envelope, function wire.Entity) error {
	parameters, err := field(function, 0x9111)
	if err != nil || len(parameters.List) != 3 {
		return fmt.Errorf("wasm.helper_parameters")
	}
	parameterIDs := make([]wire.ID, 3)
	for index, reference := range parameters.List {
		parameter, ok := graph.Entities[reference.Reference]
		if !ok || parameter.Schema != identity(0x9012) {
			return fmt.Errorf("wasm.helper_parameter")
		}
		position, positionErr := field(parameter, 0x9122)
		if positionErr != nil || position.Unsigned != uint64(index) || validateIntegerTypeReference(graph, parameter, 0x9121) != nil {
			return fmt.Errorf("wasm.helper_parameter_type")
		}
		parameterIDs[index] = parameter.ID
	}
	if _, err := referenced(graph, function, 0x9112, 0x9020); err != nil {
		return fmt.Errorf("wasm.helper_result")
	}
	comparison, err := referenced(graph, function, 0x9113, 0x9021)
	if err != nil {
		return err
	}
	addition, err := referenced(graph, comparison, 0x9160, 0x9014)
	if err != nil {
		return err
	}
	reads := make([]wire.Entity, 3)
	reads[0], err = referenced(graph, addition, 0x9140, 0x9013)
	if err != nil {
		return err
	}
	reads[1], err = referenced(graph, addition, 0x9141, 0x9013)
	if err != nil {
		return err
	}
	reads[2], err = referenced(graph, comparison, 0x9161, 0x9013)
	if err != nil {
		return err
	}
	if validateIntegerTypeReference(graph, addition, 0x9142) != nil || validateIntegerTypeReference(graph, comparison, 0x9162) != nil {
		return fmt.Errorf("wasm.helper_integer_type")
	}
	for index, read := range reads {
		parameter, readErr := field(read, 0x9130)
		if readErr != nil || parameter.Reference != parameterIDs[index] {
			return fmt.Errorf("wasm.helper_parameter_order")
		}
	}
	return nil
}

type recordFieldProfile struct {
	name   string
	schema uint64
	size   uint64
}
type recordFieldLayout struct {
	entity wire.Entity
	offset uint64
	size   uint64
}

func validateRecord(graph wire.Envelope, record wire.Entity, name string, profile []recordFieldProfile) ([]recordFieldLayout, error) {
	nameValue, err := field(record, 0x9300)
	fieldsValue, fieldsErr := field(record, 0x9301)
	if err != nil || fieldsErr != nil || string(nameValue.Bytes) != name || len(fieldsValue.List) != len(profile) {
		return nil, fmt.Errorf("wasm.record_profile:%s", name)
	}
	layout := make([]recordFieldLayout, len(profile))
	var offset uint64
	for index, expected := range profile {
		entity, ok := graph.Entities[fieldsValue.List[index].Reference]
		if !ok || entity.Schema != identity(0x9031) {
			return nil, fmt.Errorf("wasm.record_field_missing:%s", expected.name)
		}
		fieldName, nameErr := field(entity, 0x9310)
		fieldType, typeErr := field(entity, 0x9311)
		fieldIndex, indexErr := field(entity, 0x9312)
		typeEntity, typeOK := graph.Entities[fieldType.Reference]
		if nameErr != nil || typeErr != nil || indexErr != nil || string(fieldName.Bytes) != expected.name || fieldIndex.Unsigned != uint64(index) || !typeOK || typeEntity.Schema != identity(expected.schema) {
			return nil, fmt.Errorf("wasm.record_field_profile:%s", expected.name)
		}
		if expected.schema == 0x9010 {
			if err := validateIntegerTypeReference(graph, entity, 0x9311); err != nil {
				return nil, err
			}
		}
		layout[index] = recordFieldLayout{entity, offset, expected.size}
		offset += expected.size
	}
	return layout, nil
}

func validateIntegerTypeReference(graph wire.Envelope, entity wire.Entity, fieldID uint64) error {
	typeValue, err := field(entity, fieldID)
	if err != nil {
		return err
	}
	typeEntity, ok := graph.Entities[typeValue.Reference]
	if !ok || typeEntity.Schema != identity(0x9010) {
		return fmt.Errorf("wasm.integer_type_missing")
	}
	width, widthErr := field(typeEntity, 0x9100)
	signed, signedErr := field(typeEntity, 0x9101)
	overflow, overflowErr := field(typeEntity, 0x9102)
	if widthErr != nil || signedErr != nil || overflowErr != nil || width.Unsigned != 64 || signed.Tag != 2 || overflow.Unsigned != 0 {
		return fmt.Errorf("wasm.unsupported_integer_profile")
	}
	return nil
}

func referenced(graph wire.Envelope, entity wire.Entity, fieldID, schema uint64) (wire.Entity, error) {
	value, err := field(entity, fieldID)
	if err != nil {
		return wire.Entity{}, err
	}
	target, ok := graph.Entities[value.Reference]
	if !ok || target.Schema != identity(schema) {
		return wire.Entity{}, fmt.Errorf("wasm.reference_schema:%x", fieldID)
	}
	return target, nil
}

func field(entity wire.Entity, low uint64) (wire.Value, error) {
	value, ok := entity.Fields[identity(low)]
	if !ok {
		return wire.Value{}, fmt.Errorf("wasm.field_missing:%x", low)
	}
	return value, nil
}
func bySchema(graph wire.Envelope, low uint64) []wire.Entity {
	want := identity(low)
	var values []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == want {
			values = append(values, entity)
		}
	}
	return values
}
func identity(low uint64) wire.ID {
	var value wire.ID
	value[14] = byte(low >> 8)
	value[15] = byte(low)
	return value
}
