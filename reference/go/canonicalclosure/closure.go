// Package canonicalclosure performs a bounded linear-time closure audit over
// canonical Kernel wire. Frozen K0 validators remain the independent authority
// on bounded fixtures; this checker makes the same local-reference invariant
// practical for large composed project envelopes.
package canonicalclosure

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

const MaximumEnvelopeBytes = 1 << 20

func Validate(source []byte) error {
	if len(source) == 0 || len(source) > MaximumEnvelopeBytes {
		return fmt.Errorf("canonical_closure.bounds")
	}
	graph, err := wire.Decode(source)
	if err != nil {
		return fmt.Errorf("canonical_closure.wire:%w", err)
	}
	canonical, err := wire.Encode(graph)
	if err != nil || !bytes.Equal(canonical, source) {
		return fmt.Errorf("canonical_closure.noncanonical")
	}
	if graph.Module == (wire.ID{}) || graph.Revision == (wire.ID{}) {
		return fmt.Errorf("canonical_closure.metadata")
	}
	if _, exists := graph.Entities[graph.Module]; !exists {
		return fmt.Errorf("canonical_closure.module")
	}
	for _, entity := range graph.Entities {
		for _, value := range entity.Fields {
			if err = validateValue(graph.Entities, value); err != nil {
				return fmt.Errorf("canonical_closure.entity:%s:%w", entity.ID, err)
			}
		}
	}
	return nil
}

func validateValue(entities map[wire.ID]wire.Entity, value wire.Value) error {
	if value.Tag == 6 {
		if _, exists := entities[value.Reference]; !exists {
			return fmt.Errorf("missing_reference:%s", value.Reference)
		}
	}
	for _, item := range value.List {
		if err := validateValue(entities, item); err != nil {
			return err
		}
	}
	for _, item := range value.Record {
		if err := validateValue(entities, item); err != nil {
			return err
		}
	}
	return nil
}
