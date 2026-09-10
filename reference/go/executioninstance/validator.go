// Package executioninstance validates a compiled Core Execution program
// against an independently resolved, immutable contract.
package executioninstance

import (
	"bytes"
	"fmt"
	"reflect"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/executionmodule"
	"seme.local/reference/wire"
)

var executionModule = id(0x9000)

func Validate(contract contractcatalog.Contract, input wire.Envelope) error {
	return validate(contract, input, 35, id(0x9023))
}

// ValidateV36 is an additive authority boundary. It does not permit a v35
// contract to authorize v36-only schemas.
func ValidateV36(contract contractcatalog.Contract, input wire.Envelope) error {
	return validate(contract, input, 36, id(0x9024))
}

func validate(contract contractcatalog.Contract, input wire.Envelope, version int, revision wire.ID) error {
	if !contract.Validated() || contract.Pin() != (contractcatalog.Pin{Module: executionModule, Revision: revision}) {
		return fmt.Errorf("execution_instance.contract")
	}
	if input.Module != executionModule {
		return fmt.Errorf("execution_instance.module")
	}
	declarations, err := executionmodule.Declarations(version)
	if err != nil {
		return err
	}
	schemas := map[wire.ID]executionmodule.Schema{}
	declarationIDs := map[wire.ID]bool{executionModule: true}
	for _, s := range declarations {
		schemas[id(s.ID)] = s
		declarationIDs[id(s.ID)] = true
		for _, f := range s.Fields {
			declarationIDs[id(f.ID)] = true
		}
	}
	// Execution may directly reference the two runtime-boundary Foundation
	// value schemas imported by the authenticated v35 contract. Keep this list
	// deliberately closed; no other Foundation schema is admitted here.
	schemas[id(0x15)] = executionmodule.Schema{ID: 0x15, Name: "Effect", Fields: []executionmodule.Field{{ID: 0x150, Name: "effect.name", Kind: 4}, {ID: 0x151, Name: "effect.capability", Kind: 5, Schema: 0x16}}}
	schemas[id(0x16)] = executionmodule.Schema{ID: 0x16, Name: "Capability", Fields: []executionmodule.Field{{ID: 0x160, Name: "capability.name", Kind: 4}}}
	var program wire.ID
	for eid, e := range input.Entities {
		if e.Schema == id(0x9015) {
			if program != (wire.ID{}) {
				return fmt.Errorf("execution_instance.program_count")
			}
			program = eid
		}
	}
	if program == (wire.ID{}) {
		return fmt.Errorf("execution_instance.program_count")
	}
	reachable := map[wire.ID]bool{}
	queue := []wire.ID{program}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if reachable[x] {
			continue
		}
		e, ok := input.Entities[x]
		if !ok {
			return fmt.Errorf("execution_instance.reference_missing:%s", x)
		}
		schema, ok := schemas[e.Schema]
		if !ok {
			return fmt.Errorf("execution_instance.schema_unknown:%s", e.Schema)
		}
		if err := validateEntity(e, schema, input.Entities, &queue); err != nil {
			return fmt.Errorf("execution_instance.entity:%s:%w", x, err)
		}
		reachable[x] = true
	}
	contractGraph := contract.Envelope()
	for eid, e := range input.Entities {
		if reachable[eid] {
			continue
		}
		if !declarationIDs[eid] || !reflect.DeepEqual(e, contractGraph.Entities[eid]) {
			return fmt.Errorf("execution_instance.unreachable:%s", eid)
		}
	}
	return validateProgram(input.Entities[program], input.Entities)
}

func validateEntity(e wire.Entity, s executionmodule.Schema, all map[wire.ID]wire.Entity, queue *[]wire.ID) error {
	if e.Version != 1 {
		return fmt.Errorf("version")
	}
	fields := map[wire.ID]executionmodule.Field{}
	for _, f := range s.Fields {
		fields[id(f.ID)] = f
	}
	if len(e.Fields) != len(fields) {
		return fmt.Errorf("field_count")
	}
	for fid, f := range fields {
		v, ok := e.Fields[fid]
		if !ok {
			return fmt.Errorf("field_missing:%s", fid)
		}
		if err := validateValue(v, f, all, queue); err != nil {
			return fmt.Errorf("field:%s:%w", fid, err)
		}
	}
	for fid := range e.Fields {
		if _, ok := fields[fid]; !ok {
			return fmt.Errorf("field_unknown:%s", fid)
		}
	}
	return nil
}

func validateValue(v wire.Value, f executionmodule.Field, all map[wire.ID]wire.Entity, queue *[]wire.ID) error {
	if f.Card == 2 {
		if f.Kind != 5 || v.Tag != 7 {
			return fmt.Errorf("list_kind")
		}
		for _, x := range v.List {
			if x.Tag != 6 {
				return fmt.Errorf("list_item_kind")
			}
			if err := reference(x.Reference, f.Schema, all, queue); err != nil {
				return err
			}
		}
		return nil
	}
	switch f.Kind {
	case 1:
		if v.Tag != 1 && v.Tag != 2 {
			return fmt.Errorf("truth_kind")
		}
	case 2:
		if v.Tag != 3 {
			return fmt.Errorf("unsigned_kind")
		}
	case 4:
		if v.Tag != 5 {
			return fmt.Errorf("bytes_kind")
		}
	case 5:
		if v.Tag != 6 {
			return fmt.Errorf("reference_kind")
		}
		return reference(v.Reference, f.Schema, all, queue)
	default:
		return fmt.Errorf("declaration_kind:%d", f.Kind)
	}
	return nil
}
func reference(target wire.ID, constraint uint64, all map[wire.ID]wire.Entity, queue *[]wire.ID) error {
	e, ok := all[target]
	if !ok {
		return fmt.Errorf("reference_missing:%s", target)
	}
	if constraint != 0 && e.Schema != id(constraint) {
		return fmt.Errorf("reference_schema:%s", target)
	}
	*queue = append(*queue, target)
	return nil
}

func validateProgram(program wire.Entity, all map[wire.ID]wire.Entity) error {
	functions := program.Fields[id(0x9150)].List
	entry := program.Fields[id(0x9151)].Reference
	found := false
	seen := map[wire.ID]bool{}
	var previous wire.ID
	for i, v := range functions {
		if seen[v.Reference] {
			return fmt.Errorf("execution_instance.function_duplicate")
		}
		if i > 0 && bytes.Compare(previous[:], v.Reference[:]) >= 0 {
			return fmt.Errorf("execution_instance.function_order")
		}
		seen[v.Reference] = true
		previous = v.Reference
		if v.Reference == entry {
			found = true
		}
		f := all[v.Reference]
		params := f.Fields[id(0x9111)].List
		for i, p := range params {
			parameter := all[p.Reference]
			index := parameter.Fields[id(0x9122)]
			if index.Tag != 3 || index.Unsigned != uint64(i) {
				return fmt.Errorf("execution_instance.parameter_index:%s", p.Reference)
			}
		}
	}
	if !found {
		return fmt.Errorf("execution_instance.entry_membership")
	}
	return nil
}
func id(n uint64) wire.ID {
	var out wire.ID
	for i := 15; i >= 0 && n > 0; i-- {
		out[i] = byte(n)
		n >>= 8
	}
	return out
}
