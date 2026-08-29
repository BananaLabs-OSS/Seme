package foundation

import (
	"bytes"
	"fmt"
	"sort"
)

type ID string
type Kind uint8
type Cardinality uint8
type Disposition string

const (
	Unit Kind = iota
	Bool
	Unsigned
	Signed
	Bytes
	Reference
	List
	Record
	Hole
)

const (
	One Cardinality = iota
	Optional
	Many
)

const (
	Certified        Disposition = "certified"
	PreservedUnknown Disposition = "preserved_unknown"
	Unavailable      Disposition = "unavailable"
)

type Shape struct {
	Kind    Kind
	Schema  ID
	Element *Shape
}
type Field struct {
	ID          ID
	Shape       Shape
	Cardinality Cardinality
	Since       uint64
}
type Schema struct {
	ID      ID
	Version uint64
	Fields  []Field
}
type Module struct {
	ID      ID
	Schemas []Schema
}
type Value struct {
	Kind      Kind
	Bool      bool
	Unsigned  uint64
	Signed    int64
	Bytes     []byte
	Reference ID
	List      []Value
	Record    map[ID]Value
	Hole      ID
}
type Entity struct {
	ID      ID
	Schema  ID
	Version uint64
	Fields  map[ID]Value
}
type Result struct {
	Entity      ID
	Disposition Disposition
}

func Validate(module Module, entities map[ID]Entity) ([]Result, error) {
	if module.ID == "" {
		return nil, fmt.Errorf("foundation.module_identity")
	}
	schemas := map[ID]Schema{}
	var previousSchema ID
	for _, schema := range module.Schemas {
		if schema.ID == "" || schema.Version == 0 {
			return nil, fmt.Errorf("foundation.invalid_schema:%s", schema.ID)
		}
		if previousSchema != "" && schema.ID <= previousSchema {
			return nil, fmt.Errorf("foundation.schema_order:%s", schema.ID)
		}
		previousSchema = schema.ID
		if _, exists := schemas[schema.ID]; exists {
			return nil, fmt.Errorf("foundation.duplicate_schema:%s", schema.ID)
		}
		seen := map[ID]bool{}
		var previousField ID
		for _, field := range schema.Fields {
			if field.ID == "" || field.Since == 0 || field.Since > schema.Version || field.Cardinality > Many {
				return nil, fmt.Errorf("foundation.invalid_field:%s", field.ID)
			}
			if seen[field.ID] {
				return nil, fmt.Errorf("foundation.duplicate_field:%s", field.ID)
			}
			if previousField != "" && field.ID <= previousField {
				return nil, fmt.Errorf("foundation.field_order:%s", field.ID)
			}
			previousField = field.ID
			seen[field.ID] = true
			if err := validateShape(field.Shape); err != nil {
				return nil, fmt.Errorf("foundation.field_shape:%s:%w", field.ID, err)
			}
		}
		schemas[schema.ID] = schema
	}
	ids := make([]ID, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	results := make([]Result, 0, len(ids))
	for _, id := range ids {
		entity := entities[id]
		if entity.ID != id {
			return nil, fmt.Errorf("foundation.entity_identity:%s", id)
		}
		schema, known := schemas[entity.Schema]
		if !known {
			results = append(results, Result{id, Unavailable})
			continue
		}
		if entity.Version > schema.Version {
			results = append(results, Result{id, PreservedUnknown})
			continue
		}
		if entity.Version == 0 {
			return nil, fmt.Errorf("foundation.invalid_entity_version:%s", id)
		}
		declared := map[ID]Field{}
		for _, field := range schema.Fields {
			if field.Since <= entity.Version {
				declared[field.ID] = field
			}
		}
		disposition := Certified
		for fieldID := range entity.Fields {
			if _, known := declared[fieldID]; !known {
				disposition = PreservedUnknown
			}
		}
		for fieldID, field := range declared {
			value, present := entity.Fields[fieldID]
			if !present {
				if field.Cardinality == One {
					return nil, fmt.Errorf("foundation.required_field:%s:%s", id, fieldID)
				}
				continue
			}
			if field.Cardinality == Many {
				if value.Kind != List {
					return nil, fmt.Errorf("foundation.cardinality:%s:%s", id, fieldID)
				}
				for i := range value.List {
					if err := validateValue(field.Shape, value.List[i], entities, schemas); err != nil {
						return nil, fmt.Errorf("foundation.value:%s:%s:%d:%w", id, fieldID, i, err)
					}
				}
				continue
			}
			if err := validateValue(field.Shape, value, entities, schemas); err != nil {
				return nil, fmt.Errorf("foundation.value:%s:%s:%w", id, fieldID, err)
			}
		}
		results = append(results, Result{id, disposition})
	}
	return results, nil
}

func validateShape(shape Shape) error {
	if shape.Kind > Hole {
		return fmt.Errorf("unknown_kind")
	}
	if shape.Kind == List {
		if shape.Element == nil {
			return fmt.Errorf("missing_element")
		}
		return validateShape(*shape.Element)
	}
	if shape.Element != nil {
		return fmt.Errorf("unexpected_element")
	}
	if shape.Kind == Record && shape.Schema == "" {
		return fmt.Errorf("missing_schema")
	}
	return nil
}

func validateValue(shape Shape, value Value, entities map[ID]Entity, schemas map[ID]Schema) error {
	if shape.Kind != value.Kind {
		return fmt.Errorf("kind")
	}
	switch shape.Kind {
	case Reference:
		target, ok := entities[value.Reference]
		if !ok {
			return fmt.Errorf("reference")
		}
		if shape.Schema != "" && target.Schema != shape.Schema {
			return fmt.Errorf("reference_schema")
		}
	case List:
		for i := range value.List {
			if err := validateValue(*shape.Element, value.List[i], entities, schemas); err != nil {
				return fmt.Errorf("list:%d:%w", i, err)
			}
		}
	case Record:
		schema, ok := schemas[shape.Schema]
		if !ok {
			return fmt.Errorf("record_schema")
		}
		for _, field := range schema.Fields {
			value, present := value.Record[field.ID]
			if !present {
				if field.Cardinality == One {
					return fmt.Errorf("record_required:%s", field.ID)
				}
				continue
			}
			if field.Cardinality == Many {
				if value.Kind != List {
					return fmt.Errorf("record_cardinality:%s", field.ID)
				}
				for i := range value.List {
					if err := validateValue(field.Shape, value.List[i], entities, schemas); err != nil {
						return fmt.Errorf("record_value:%s:%d:%w", field.ID, i, err)
					}
				}
				continue
			}
			if err := validateValue(field.Shape, value, entities, schemas); err != nil {
				return fmt.Errorf("record_value:%s:%w", field.ID, err)
			}
		}
	case Bytes:
		value.Bytes = bytes.Clone(value.Bytes)
	}
	return nil
}
