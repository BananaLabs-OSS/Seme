// Package executionprofile binds a canonical construction's reachable schema
// vocabulary to one exact authenticated Execution contract revision.
package executionprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

var trusted = map[wire.ID][sha256.Size]byte{
	id("9023"): digest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a"),
	id("9024"): digest("2315477d7c0d248ce167aeada313654d25c056be1d8c851e43bd9c225ff07450"),
}

// ValidateConstruction requires every entity reachable from the one canonical
// Program to use a schema exported by the exact pinned Execution revision.
// Foundation Effect and Capability are the only non-Execution runtime schemas
// admitted into an Execution construction.
func ValidateConstruction(contract contractcatalog.Contract, graph wire.Envelope) error {
	pin := contract.Pin()
	wantDigest, known := trusted[pin.Revision]
	if !contract.Validated() || pin.Module != id("9000") || !known || contract.Digest() != wantDigest {
		return fmt.Errorf("execution_profile.contract")
	}
	allowed := map[wire.ID]bool{id("15"): true, id("16"): true}
	for _, schema := range contract.Exports() {
		allowed[schema] = true
	}
	var program wire.ID
	for x, entity := range graph.Entities {
		if entity.Schema != id("9015") {
			continue
		}
		if program != (wire.ID{}) {
			return fmt.Errorf("execution_profile.program_count")
		}
		program = x
	}
	if program == (wire.ID{}) {
		return fmt.Errorf("execution_profile.program_count")
	}
	seen := map[wire.ID]bool{}
	queue := []wire.ID{program}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if seen[x] {
			continue
		}
		entity, exists := graph.Entities[x]
		if !exists {
			return fmt.Errorf("execution_profile.reference_missing:%s", x)
		}
		seen[x] = true
		if !allowed[entity.Schema] {
			return fmt.Errorf("execution_profile.schema_not_exported:%s:%s", x, entity.Schema)
		}
		for _, value := range entity.Fields {
			collect(value, &queue)
		}
	}
	return nil
}

func collect(v wire.Value, out *[]wire.ID) {
	if v.Tag == 6 {
		*out = append(*out, v.Reference)
	}
	if v.Tag == 7 {
		for _, item := range v.List {
			collect(item, out)
		}
	}
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
func digest(s string) [sha256.Size]byte {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != sha256.Size {
		panic("invalid trusted digest")
	}
	var out [sha256.Size]byte
	copy(out[:], b)
	return out
}
