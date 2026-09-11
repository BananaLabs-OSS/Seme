package targetplaninstance

import (
	"fmt"

	"seme.local/reference/wire"
)

type Summary struct {
	Executable, HasInvalidFidelity        bool
	Requirements, Boundaries, Diagnostics int
	Fidelities                            [7]int
}

// Inspect validates against the original authority and model before exposing
// aggregate decisions. It is safe for executable and diagnostic-only plans.
func Inspect(input Inputs) (Summary, error) {
	if err := Validate(input); err != nil {
		return Summary{}, err
	}
	graph, err := wire.Decode(input.Artifact)
	if err != nil {
		return Summary{}, err
	}
	var plan wire.Entity
	count := 0
	for _, entity := range graph.Entities {
		if entity.Schema == id("c014") {
			plan, count = entity, count+1
		}
	}
	if count != 1 {
		return Summary{}, fmt.Errorf("target_plan.summary_plan")
	}
	result := Summary{Executable: plan.Fields[id("c144")].Tag == 2, Requirements: len(plan.Fields[id("c142")].List), Boundaries: len(plan.Fields[id("c143")].List)}
	for _, reference := range plan.Fields[id("c142")].List {
		resolution, exists := graph.Entities[reference.Reference]
		if reference.Tag != 6 || !exists || resolution.Schema != id("c013") {
			return Summary{}, fmt.Errorf("target_plan.summary_resolution")
		}
		fidelity := resolution.Fields[id("c133")].Unsigned
		if fidelity >= uint64(len(result.Fidelities)) {
			result.HasInvalidFidelity = true
		} else {
			result.Fidelities[fidelity]++
		}
		result.Diagnostics += len(resolution.Fields[id("c135")].List)
	}
	if result.HasInvalidFidelity || result.Requirements == 0 || result.Executable == (result.Fidelities[Impossible] != 0) {
		return Summary{}, fmt.Errorf("target_plan.summary_consistency")
	}
	return result, nil
}
