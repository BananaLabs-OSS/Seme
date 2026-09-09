package wasmtarget

import (
	"fmt"
	"sort"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wire"
)

// PureApplicationBoundary is the recursive, language-neutral boundary of one
// selected canonical entry. It is derived from type and effect entities only;
// names of applications, fixtures, and source functions never select it.
type PureApplicationBoundary struct {
	Program              string            `json:"program"`
	Function             string            `json:"function"`
	Parameters           []PureValueLayout `json:"parameters"`
	Request              PureValueLayout   `json:"request"`
	Result               PureValueLayout   `json:"result"`
	RequiredCapabilities []string          `json:"required_capabilities,omitempty"`
}

// CertifyPureApplicationBoundary validates program membership and recursively
// derives every parameter/result representation before target code generation.
// Effect declarations are also certified here so authorization can happen
// before request decoding or execution emits any observation.
func CertifyPureApplicationBoundary(graph wire.Envelope) (PureApplicationBoundary, error) {
	programs := bySchema(graph, 0x9015)
	if len(programs) != 1 {
		return PureApplicationBoundary{}, fmt.Errorf("wasm.application_program_cardinality")
	}
	listed, le := field(programs[0], 0x9150)
	entry, ee := field(programs[0], 0x9151)
	if le != nil || ee != nil || listed.Tag != 7 || entry.Tag != 6 {
		return PureApplicationBoundary{}, fmt.Errorf("wasm.application_program")
	}
	member := false
	for _, item := range listed.List {
		if item.Tag != 6 {
			return PureApplicationBoundary{}, fmt.Errorf("wasm.application_membership")
		}
		member = member || item.Reference == entry.Reference
	}
	function, ok := graph.Entities[entry.Reference]
	if !member || !ok || function.Schema != identity(0x9011) {
		return PureApplicationBoundary{}, fmt.Errorf("wasm.application_entry")
	}
	parameters, pe := field(function, 0x9111)
	result, re := field(function, 0x9112)
	if pe != nil || re != nil || parameters.Tag != 7 || result.Tag != 6 || len(parameters.List) > 32 {
		return PureApplicationBoundary{}, fmt.Errorf("wasm.application_signature")
	}
	out := PureApplicationBoundary{Program: programs[0].ID.String(), Function: function.ID.String()}
	seen := map[wire.ID]bool{}
	for index, item := range parameters.List {
		parameter, exists := graph.Entities[item.Reference]
		position, xe := field(parameter, 0x9122)
		typeRef, te := field(parameter, 0x9121)
		if item.Tag != 6 || seen[item.Reference] || !exists || parameter.Schema != identity(0x9012) || xe != nil || te != nil || position.Tag != 3 || position.Unsigned != uint64(index) || typeRef.Tag != 6 {
			return PureApplicationBoundary{}, fmt.Errorf("wasm.application_parameter")
		}
		seen[item.Reference] = true
		layout, err := CertifyPureValueLayout(graph, typeRef.Reference)
		if err != nil {
			return PureApplicationBoundary{}, fmt.Errorf("wasm.application_parameter_layout:%w", err)
		}
		out.Parameters = append(out.Parameters, layout)
	}
	out.Request = aggregateApplicationRequest(out.Parameters)
	resultLayout, err := CertifyPureValueLayout(graph, result.Reference)
	if err != nil {
		return PureApplicationBoundary{}, fmt.Errorf("wasm.application_result_layout:%w", err)
	}
	out.Result = resultLayout
	capabilities := map[string]bool{}
	for _, effect := range bySchema(graph, 0x15) {
		nameValue, ne := field(effect, 0x150)
		capabilityRef, ce := field(effect, 0x151)
		capability, exists := graph.Entities[capabilityRef.Reference]
		capabilityName, cne := field(capability, 0x160)
		if ne != nil || ce != nil || cne != nil || nameValue.Tag != 5 || capabilityRef.Tag != 6 || !exists || capability.Schema != identity(0x16) || capabilityName.Tag != 5 || string(nameValue.Bytes) != string(capabilityName.Bytes) {
			return PureApplicationBoundary{}, fmt.Errorf("wasm.application_effect_authority")
		}
		capabilities[string(nameValue.Bytes)] = true
	}
	for capability := range capabilities {
		out.RequiredCapabilities = append(out.RequiredCapabilities, capability)
	}
	sort.Strings(out.RequiredCapabilities)
	return out, nil
}

func aggregateApplicationRequest(parameters []PureValueLayout) PureValueLayout {
	request := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "application-request", Encoding: "ordered-inline-parameters"}
	for index, layout := range parameters {
		request.Fields = append(request.Fields, PureValueFieldLayout{Name: fmt.Sprint(index), Offset: request.FixedSize, Value: layout})
		request.FixedSize += layout.FixedSize
		request.VariablePayload = request.VariablePayload || layout.VariablePayload
		request.MaximumPayload += layout.MaximumPayload
	}
	return request
}

// EncodePureApplicationRequest encodes all parameters as one canonical message.
// This is essential for nested descriptors: their offsets are absolute within
// the complete request, never relative to a separately encoded parameter.
func EncodePureApplicationRequest(boundary PureApplicationBoundary, parameters []canonicaleval.Value) ([]byte, error) {
	if len(parameters) != len(boundary.Parameters) || len(boundary.Request.Fields) != len(parameters) {
		return nil, fmt.Errorf("wasm.application_request_arity")
	}
	fields := make(map[string]canonicaleval.Value, len(parameters))
	for index, value := range parameters {
		fields[fmt.Sprint(index)] = value
	}
	return EncodePureValue(boundary.Request, canonicaleval.Value{Kind: "record", Fields: fields})
}

// DecodePureApplicationRequest performs whole-message validation before
// returning parameters in their declared order.
func DecodePureApplicationRequest(boundary PureApplicationBoundary, data []byte) ([]canonicaleval.Value, error) {
	value, err := DecodePureValue(boundary.Request, data)
	if err != nil {
		return nil, err
	}
	result := make([]canonicaleval.Value, len(boundary.Parameters))
	for index := range result {
		item, ok := value.Fields[fmt.Sprint(index)]
		if !ok {
			return nil, fmt.Errorf("wasm.application_request_field")
		}
		result[index] = item
	}
	return result, nil
}
