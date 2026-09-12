// Package upb12revision verifies that two compiled semantic revisions differ
// by exactly one identity-bound field transition.
package upb12revision

import (
	"bytes"
	"fmt"
	"seme.local/reference/wire"
)

func Verify(priorBytes, resultBytes []byte, target, field wire.ID, expected, replacement []byte) error {
	prior, err := wire.Decode(priorBytes)
	if err != nil {
		return fmt.Errorf("upb12_revision.prior:%w", err)
	}
	result, err := wire.Decode(resultBytes)
	if err != nil {
		return fmt.Errorf("upb12_revision.result:%w", err)
	}
	if prior.Revision == (wire.ID{}) || result.Revision == (wire.ID{}) || prior.Revision == result.Revision {
		return fmt.Errorf("upb12_revision.revision")
	}
	before, bok := prior.Entities[target]
	after, aok := result.Entities[target]
	bv, bvok := before.Fields[field]
	av, avok := after.Fields[field]
	if !bok || !aok || !bvok || !avok || bv.Tag != 5 || av.Tag != 5 || !bytes.Equal(bv.Bytes, expected) || !bytes.Equal(av.Bytes, replacement) {
		return fmt.Errorf("upb12_revision.transition")
	}
	after.Fields[field] = bv
	result.Entities[target] = after
	result.Revision = prior.Revision
	normalized, err := wire.Encode(result)
	if err != nil || !bytes.Equal(normalized, priorBytes) {
		return fmt.Errorf("upb12_revision.additional_change")
	}
	return nil
}
