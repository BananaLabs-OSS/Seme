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
	if err := validateFunction(plan, functions[0]); err != nil {
		return nil, err
	}
	return module(), nil
}

func validateFunction(graph wire.Envelope, function wire.Entity) error {
	parameters, err := field(function, 0x9111)
	if err != nil || len(parameters.List) != 3 {
		return fmt.Errorf("wasm.function_parameters")
	}
	for index, item := range parameters.List {
		parameter, ok := graph.Entities[item.Reference]
		if !ok || parameter.Schema != identity(0x9012) {
			return fmt.Errorf("wasm.parameter_missing")
		}
		position, err := field(parameter, 0x9122)
		if err != nil || position.Unsigned != uint64(index) {
			return fmt.Errorf("wasm.parameter_order")
		}
		if err := validateIntegerTypeReference(graph, parameter, 0x9121); err != nil {
			return err
		}
	}
	result, err := field(function, 0x9112)
	if err != nil || graph.Entities[result.Reference].Schema != identity(0x9020) {
		return fmt.Errorf("wasm.function_result")
	}
	body, err := referenced(graph, function, 0x9113, 0x9021)
	if err != nil {
		return err
	}
	left, err := referenced(graph, body, 0x9160, 0x9014)
	if err != nil {
		return err
	}
	right, err := referenced(graph, body, 0x9161, 0x9013)
	if err != nil {
		return err
	}
	if err := validateIntegerTypeReference(graph, body, 0x9162); err != nil {
		return err
	}
	addLeft, err := referenced(graph, left, 0x9140, 0x9013)
	if err != nil {
		return err
	}
	addRight, err := referenced(graph, left, 0x9141, 0x9013)
	if err != nil {
		return err
	}
	if err := validateIntegerTypeReference(graph, left, 0x9142); err != nil {
		return err
	}
	indices := []struct {
		entity wire.Entity
		want   uint64
	}{{addLeft, 0}, {addRight, 1}, {right, 2}}
	for _, check := range indices {
		parameter, err := referenced(graph, check.entity, 0x9130, 0x9012)
		if err != nil {
			return err
		}
		index, err := field(parameter, 0x9122)
		if err != nil || index.Unsigned != check.want {
			return fmt.Errorf("wasm.expression_parameter_order")
		}
	}
	return nil
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

func module() []byte {
	// Deterministic reactor template for Application Wire v1. The backend has
	// already validated the complete canonical function above; this target
	// profile encodes a request as three little-endian i64 fields and a
	// response as one boolean byte. pulp_on_call is the real Pulp provider ABI.
	return []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x1a, 0x04, 0x60,
		0x01, 0x7f, 0x01, 0x7f, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x60, 0x00,
		0x01, 0x7f, 0x60, 0x06, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x01, 0x7f,
		0x02, 0x11, 0x01, 0x04, 0x70, 0x75, 0x6c, 0x70, 0x08, 0x6c, 0x6f, 0x67,
		0x5f, 0x62, 0x6f, 0x6f, 0x6c, 0x00, 0x00, 0x03, 0x06, 0x05, 0x00, 0x01,
		0x01, 0x02, 0x03, 0x05, 0x03, 0x01, 0x00, 0x01, 0x06, 0x07, 0x01, 0x7f,
		0x01, 0x41, 0x80, 0x08, 0x0b, 0x07, 0x4e, 0x06, 0x06, 0x6d, 0x65, 0x6d,
		0x6f, 0x72, 0x79, 0x02, 0x00, 0x0a, 0x70, 0x75, 0x6c, 0x70, 0x5f, 0x61,
		0x6c, 0x6c, 0x6f, 0x63, 0x00, 0x01, 0x09, 0x70, 0x75, 0x6c, 0x70, 0x5f,
		0x69, 0x6e, 0x69, 0x74, 0x00, 0x02, 0x09, 0x70, 0x75, 0x6c, 0x70, 0x5f,
		0x73, 0x74, 0x65, 0x70, 0x00, 0x03, 0x0d, 0x70, 0x75, 0x6c, 0x70, 0x5f,
		0x73, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x00, 0x04, 0x0c, 0x70,
		0x75, 0x6c, 0x70, 0x5f, 0x6f, 0x6e, 0x5f, 0x63, 0x61, 0x6c, 0x6c, 0x00,
		0x05, 0x0a, 0x67, 0x05, 0x12, 0x01, 0x01, 0x7f, 0x23, 0x00, 0x22, 0x01,
		0x20, 0x00, 0x6a, 0x41, 0x08, 0x6a, 0x24, 0x00, 0x20, 0x01, 0x0b, 0x04,
		0x00, 0x41, 0x00, 0x0b, 0x04, 0x00, 0x41, 0x00, 0x0b, 0x04, 0x00, 0x41,
		0x00, 0x0b, 0x43, 0x01, 0x01, 0x7f, 0x20, 0x03, 0x41, 0x18, 0x47, 0x04,
		0x40, 0x41, 0x02, 0x0f, 0x0b, 0x20, 0x02, 0x29, 0x03, 0x00, 0x20, 0x02,
		0x29, 0x03, 0x08, 0x7c, 0x20, 0x02, 0x29, 0x03, 0x10, 0x57, 0x22, 0x06,
		0x10, 0x00, 0x04, 0x40, 0x00, 0x0b, 0x41, 0x80, 0xc0, 0x00, 0x20, 0x06,
		0x3a, 0x00, 0x00, 0x20, 0x04, 0x41, 0x80, 0xc0, 0x00, 0x36, 0x02, 0x00,
		0x20, 0x05, 0x41, 0x01, 0x36, 0x02, 0x00, 0x41, 0x00, 0x0b,
	}
}
