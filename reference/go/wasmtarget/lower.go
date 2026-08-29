// Package wasmtarget implements the first deliberately scoped canonical
// Seme-to-WebAssembly backend. It consumes only a validated semantic plan.
package wasmtarget

import (
	"bytes"
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
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', '\x00', '\x00', '\x00'})
	section(&wasm, 1, []byte{
		6,
		0x60, 1, 0x7f, 1, 0x7f,
		0x60, 3, 0x7e, 0x7e, 0x7e, 1, 0x7f,
		0x60, 1, 0x7f, 1, 0x7f,
		0x60, 2, 0x7f, 0x7f, 1, 0x7f,
		0x60, 2, 0x7f, 0x7f, 1, 0x7f,
		0x60, 0, 1, 0x7f,
	})
	var imports bytes.Buffer
	uleb(&imports, 1)
	name(&imports, "pulp")
	name(&imports, "log_bool")
	imports.Write([]byte{0, 0})
	section(&wasm, 2, imports.Bytes())
	section(&wasm, 3, []byte{5, 1, 2, 3, 4, 5})
	section(&wasm, 5, []byte{1, 0, 1})
	var exports bytes.Buffer
	uleb(&exports, 6)
	name(&exports, "memory")
	exports.Write([]byte{2, 0})
	name(&exports, "admit")
	exports.Write([]byte{0, 1})
	name(&exports, "pulp_alloc")
	exports.Write([]byte{0, 2})
	name(&exports, "pulp_init")
	exports.Write([]byte{0, 3})
	name(&exports, "pulp_step")
	exports.Write([]byte{0, 4})
	name(&exports, "pulp_shutdown")
	exports.Write([]byte{0, 5})
	section(&wasm, 7, exports.Bytes())
	bodies := [][]byte{
		{1, 1, 0x7f, 0x20, 0, 0x20, 1, 0x7c, 0x20, 2, 0x57, 0x22, 3, 0x10, 0, 0x04, 0x40, 0x00, 0x0b, 0x20, 3, 0x0b},
		{0, 0x41, 0x80, 0x08, 0x0b},
		{0, 0x42, 0x28, 0x42, 0x02, 0x42, 0x32, 0x10, 0x01, 0x1a, 0x41, 0, 0x0b},
		{0, 0x41, 0, 0x0b},
		{0, 0x41, 0, 0x0b},
	}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes()
}
func section(output *bytes.Buffer, id byte, payload []byte) {
	output.WriteByte(id)
	uleb(output, uint64(len(payload)))
	output.Write(payload)
}
func name(output *bytes.Buffer, value string) {
	uleb(output, uint64(len(value)))
	output.WriteString(value)
}
func uleb(output *bytes.Buffer, value uint64) {
	for {
		part := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			part |= 0x80
		}
		output.WriteByte(part)
		if value == 0 {
			return
		}
	}
}
