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
	if len(plans) != 1 || len(functions) != 1 || len(effects) != 1 {
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
	layout, err := validateFunction(plan, functions[0])
	if err != nil {
		return nil, err
	}
	return module(layout)
}

type applicationLayout struct {
	requestOffsets [3]uint64
	requestSize    uint64
	responseOffset uint64
	responseSize   uint64
}

func validateFunction(graph wire.Envelope, function wire.Entity) (applicationLayout, error) {
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
	requestFields, err := validateRecord(graph, requestType, "AdmitRequest", []recordFieldProfile{{"Current", 0x9010, 8}, {"Delta", 0x9010, 8}, {"Limit", 0x9010, 8}})
	if err != nil {
		return layout, err
	}
	for index, item := range requestFields {
		layout.requestOffsets[index] = item.offset
	}
	layout.requestSize = requestFields[len(requestFields)-1].offset + requestFields[len(requestFields)-1].size
	resultValue, err := field(function, 0x9112)
	if err != nil {
		return layout, fmt.Errorf("wasm.function_result")
	}
	responseType, ok := graph.Entities[resultValue.Reference]
	if !ok || responseType.Schema != identity(0x9030) {
		return layout, fmt.Errorf("wasm.function_result")
	}
	responseFields, err := validateRecord(graph, responseType, "AdmitResponse", []recordFieldProfile{{"Accepted", 0x9020, 1}})
	if err != nil {
		return layout, err
	}
	layout.responseOffset = responseFields[0].offset
	layout.responseSize = responseFields[0].size
	body, err := referenced(graph, function, 0x9113, 0x9033)
	if err != nil {
		return layout, err
	}
	constructedType, err := field(body, 0x9330)
	if err != nil || constructedType.Reference != responseType.ID {
		return layout, fmt.Errorf("wasm.response_construct_type")
	}
	values, err := field(body, 0x9331)
	if err != nil || len(values.List) != 1 {
		return layout, fmt.Errorf("wasm.response_construct_values")
	}
	comparison, ok := graph.Entities[values.List[0].Reference]
	if !ok || comparison.Schema != identity(0x9021) {
		return layout, fmt.Errorf("wasm.response_value")
	}
	left, err := referenced(graph, comparison, 0x9160, 0x9014)
	if err != nil {
		return layout, err
	}
	right, err := referenced(graph, comparison, 0x9161, 0x9032)
	if err != nil {
		return layout, err
	}
	if err := validateIntegerTypeReference(graph, comparison, 0x9162); err != nil {
		return layout, err
	}
	addLeft, err := referenced(graph, left, 0x9140, 0x9032)
	if err != nil {
		return layout, err
	}
	addRight, err := referenced(graph, left, 0x9141, 0x9032)
	if err != nil {
		return layout, err
	}
	if err := validateIntegerTypeReference(graph, left, 0x9142); err != nil {
		return layout, err
	}
	checks := []struct {
		entity wire.Entity
		field  wire.ID
	}{{addLeft, requestFields[0].entity.ID}, {addRight, requestFields[1].entity.ID}, {right, requestFields[2].entity.ID}}
	for _, check := range checks {
		read, err := referenced(graph, check.entity, 0x9320, 0x9013)
		if err != nil {
			return layout, err
		}
		readParameter, err := field(read, 0x9130)
		selectedField, selectedErr := field(check.entity, 0x9321)
		if err != nil || selectedErr != nil || readParameter.Reference != parameter.ID || selectedField.Reference != check.field {
			return layout, fmt.Errorf("wasm.expression_record_field")
		}
	}
	return layout, nil
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
