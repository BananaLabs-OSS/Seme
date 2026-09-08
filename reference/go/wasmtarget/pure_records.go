package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

// normalizePureRecords validates internal immutable record construction and
// field selection, then scalar-replaces it for this target realization.
func normalizePureRecords(graph wire.Envelope, body wire.ID) (wire.Envelope, error) {
	if len(bySchema(graph, 0x9032))+len(bySchema(graph, 0x9033)) == 0 {
		return graph, nil
	}
	graph = cloneWireGraph(graph)
	block, ok := graph.Entities[body]
	statements, err := field(block, 0x9800)
	if !ok || err != nil || statements.Tag != 7 || len(statements.List) != 1 {
		return wire.Envelope{}, fmt.Errorf("wasm.pure_record_block")
	}
	statement := graph.Entities[statements.List[0].Reference]
	values, err := field(statement, 0x9810)
	if statement.Schema != identity(0x9081) || err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return wire.Envelope{}, fmt.Errorf("wasm.pure_record_return")
	}
	budget := 4096
	if err := normalizeRecordExpression(&graph, values.List[0].Reference, map[wire.ID]bool{}, &budget); err != nil {
		return wire.Envelope{}, err
	}
	return graph, nil
}

func normalizeRecordExpression(graph *wire.Envelope, id wire.ID, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.pure_record_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	entity, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.pure_record_expression_missing")
	}
	for _, child := range expressionChildren(entity) {
		if err := normalizeRecordExpression(graph, child, visiting, budget); err != nil {
			return err
		}
	}
	if entity.Schema != identity(0x9032) {
		return nil
	}
	recordValue, a := field(entity, 0x9320)
	fieldValue, b := field(entity, 0x9321)
	record, exists := graph.Entities[recordValue.Reference]
	if a != nil || b != nil || recordValue.Tag != 6 || fieldValue.Tag != 6 || !exists || record.Schema != identity(0x9033) {
		return fmt.Errorf("wasm.pure_field_read")
	}
	typeValue, c := field(record, 0x9330)
	values, d := field(record, 0x9331)
	typeEntity, typeExists := graph.Entities[typeValue.Reference]
	fields, e := field(typeEntity, 0x9301)
	if c != nil || d != nil || e != nil || typeValue.Tag != 6 || values.Tag != 7 || !typeExists || typeEntity.Schema != identity(0x9030) || fields.Tag != 7 || len(fields.List) != len(values.List) {
		return fmt.Errorf("wasm.pure_record_construct")
	}
	selected := -1
	for index, candidate := range fields.List {
		if candidate.Tag != 6 || values.List[index].Tag != 6 {
			return fmt.Errorf("wasm.pure_record_members")
		}
		fieldEntity, fieldExists := graph.Entities[candidate.Reference]
		position, positionErr := field(fieldEntity, 0x9312)
		if !fieldExists || fieldEntity.Schema != identity(0x9031) || positionErr != nil || position.Tag != 3 || position.Unsigned != uint64(index) {
			return fmt.Errorf("wasm.pure_record_field_order")
		}
		if candidate.Reference == fieldValue.Reference {
			selected = index
		}
	}
	if selected < 0 {
		return fmt.Errorf("wasm.pure_record_field_membership")
	}
	replacement := cloneWireEntity(graph.Entities[values.List[selected].Reference])
	replacement.ID = id
	graph.Entities[id] = replacement
	return nil
}
