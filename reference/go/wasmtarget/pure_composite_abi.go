package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

// PureCompositeABI is the type-certified boundary contract reserved for
// functions containing neutral aggregate values. It is separate from legacy
// PureABI so existing JSON and artifacts remain byte-for-byte unchanged.
type PureCompositeABI struct {
	Contract          string            `json:"contract"`
	Provider          string            `json:"provider"`
	Target            string            `json:"target"`
	Fidelity          string            `json:"fidelity"`
	CanonicalModule   string            `json:"canonical_module"`
	CanonicalRevision string            `json:"canonical_revision"`
	CanonicalProgram  string            `json:"canonical_program"`
	Function          string            `json:"function"`
	ProgramSHA256     string            `json:"program_sha256,omitempty"`
	ArtifactSHA256    string            `json:"artifact_sha256,omitempty"`
	MaximumMessage    uint64            `json:"maximum_message_size"`
	RequestFixedSize  uint64            `json:"request_fixed_size"`
	ResponseFixedSize uint64            `json:"response_fixed_size"`
	Parameters        []PureValueLayout `json:"parameters"`
	Result            PureValueLayout   `json:"result"`
}

// CertifyPureCompositeABI derives the complete boundary from the sole
// executable program and its canonical function signature.
func CertifyPureCompositeABI(graph wire.Envelope) (PureCompositeABI, error) {
	programs := bySchema(graph, 0x9015)
	if len(programs) != 1 {
		return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_program")
	}
	functions, fErr := field(programs[0], 0x9150)
	entry, eErr := field(programs[0], 0x9151)
	if fErr != nil || eErr != nil || functions.Tag != 7 || len(functions.List) != 1 || entry.Tag != 6 {
		return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_program")
	}
	member := functions.List[0].Tag == 6 && functions.List[0].Reference == entry.Reference
	function, ok := graph.Entities[entry.Reference]
	if !member || !ok || function.Schema != identity(0x9011) {
		return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_entry")
	}
	parameters, pErr := field(function, 0x9111)
	result, rErr := field(function, 0x9112)
	if pErr != nil || rErr != nil || parameters.Tag != 7 || result.Tag != 6 || len(parameters.List) > 32 {
		return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_signature")
	}
	abi := PureCompositeABI{Contract: "seme.pure-composite-abi/v1", Provider: "seme.function-composite-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: programs[0].ID.String(), Function: function.ID.String(), MaximumMessage: pureValueMaximumMessage}
	seen := map[wire.ID]bool{}
	for index, item := range parameters.List {
		if item.Tag != 6 || seen[item.Reference] {
			return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_parameter")
		}
		seen[item.Reference] = true
		parameter, exists := graph.Entities[item.Reference]
		position, posErr := field(parameter, 0x9122)
		typeValue, typeErr := field(parameter, 0x9121)
		if !exists || parameter.Schema != identity(0x9012) || posErr != nil || typeErr != nil || position.Tag != 3 || position.Unsigned != uint64(index) || typeValue.Tag != 6 {
			return PureCompositeABI{}, fmt.Errorf("wasm.pure_composite_parameter")
		}
		layout, err := CertifyPureValueLayout(graph, typeValue.Reference)
		if err != nil {
			return PureCompositeABI{}, err
		}
		abi.Parameters = append(abi.Parameters, layout)
		abi.RequestFixedSize += layout.FixedSize
	}
	resultLayout, err := CertifyPureValueLayout(graph, result.Reference)
	if err != nil {
		return PureCompositeABI{}, err
	}
	abi.Result, abi.ResponseFixedSize = resultLayout, resultLayout.FixedSize
	return abi, nil
}
