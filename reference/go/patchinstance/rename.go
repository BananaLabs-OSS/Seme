// Package patchinstance emits canonical instances of the neutral Patch-v1
// contract. It knows nothing about Go source, editors, or project layouts.
package patchinstance

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

type Rename struct {
	Base                  wire.Envelope
	Target, Field         wire.ID
	Expected, Replacement []byte
}

func EmitRename(contract contractcatalog.Contract, in Rename) ([]byte, error) {
	if contract.Pin() != (contractcatalog.Pin{Module: id("5000"), Revision: id("5001")}) ||
		in.Base.Revision == (wire.ID{}) || in.Target == (wire.ID{}) || in.Field == (wire.ID{}) ||
		len(in.Expected) == 0 || len(in.Replacement) == 0 || bytes.Equal(in.Expected, in.Replacement) {
		return nil, fmt.Errorf("patch_instance.authority")
	}
	target, ok := in.Base.Entities[in.Target]
	if !ok {
		return nil, fmt.Errorf("patch_instance.target")
	}
	value, ok := target.Fields[in.Field]
	if !ok || value.Tag != 5 || !bytes.Equal(value.Bytes, in.Expected) {
		return nil, fmt.Errorf("patch_instance.precondition")
	}
	out := contract.Envelope()
	out.Entities = cloneEntities(out.Entities)
	seed := sha256.New()
	seed.Write([]byte("seme.patch-instance.rename.v1\x00"))
	seed.Write(in.Base.Revision[:])
	seed.Write(in.Target[:])
	seed.Write(in.Field[:])
	seed.Write(in.Expected)
	seed.Write([]byte{0})
	seed.Write(in.Replacement)
	stable := func(label string) wire.ID {
		h := sha256.New()
		h.Write(seed.Sum(nil))
		h.Write([]byte(label))
		var result wire.ID
		copy(result[:], h.Sum(nil))
		return result
	}
	operation, transaction := stable("operation"), stable("transaction")
	out.Entities[operation] = entity(operation, "5011", map[string]wire.Value{
		"5110": blob(in.Target[:]), "5111": blob(in.Field[:]),
		"5112": blob(in.Expected), "5113": blob(in.Replacement),
	})
	// Patch-v1 intentionally leaves actor identity open. The transaction owns
	// its operation as deterministic local authorship evidence.
	out.Entities[transaction] = entity(transaction, "5010", map[string]wire.Value{
		"5100": ref(operation), "5101": blob(in.Base.Revision[:]), "5102": refs(operation),
	})
	out.Revision = artifactRevision(out)
	return wire.Encode(out)
}

func cloneEntities(in map[wire.ID]wire.Entity) map[wire.ID]wire.Entity {
	out := make(map[wire.ID]wire.Entity, len(in)+2)
	for key, value := range in {
		fields := make(map[wire.ID]wire.Value, len(value.Fields))
		for field, item := range value.Fields {
			fields[field] = cloneValue(item)
		}
		value.Fields = fields
		out[key] = value
	}
	return out
}
func cloneValue(in wire.Value) wire.Value {
	in.Bytes = bytes.Clone(in.Bytes)
	for i := range in.List {
		in.List[i] = cloneValue(in.List[i])
	}
	for i := range in.Record {
		in.Record[i] = cloneValue(in.Record[i])
	}
	return in
}
func artifactRevision(in wire.Envelope) wire.ID {
	in.Revision = wire.ID{}
	b, _ := wire.Encode(in)
	h := sha256.Sum256(append([]byte("seme.patch-instance.artifact.v1\x00"), b...))
	var result wire.ID
	copy(result[:], h[:16])
	return result
}
func entity(identity wire.ID, schema string, fields map[string]wire.Value) wire.Entity {
	result := wire.Entity{ID: identity, Schema: id(schema), Version: 1, Fields: map[wire.ID]wire.Value{}}
	for field, value := range fields {
		result.Fields[id(field)] = value
	}
	return result
}
func blob(value []byte) wire.Value { return wire.Value{Tag: 5, Bytes: bytes.Clone(value)} }
func ref(value wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: value} }
func refs(values ...wire.ID) wire.Value {
	result := wire.Value{Tag: 7}
	for _, value := range values {
		result.List = append(result.List, ref(value))
	}
	return result
}
func id(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	result, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return result
}
