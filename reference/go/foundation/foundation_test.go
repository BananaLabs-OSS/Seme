package foundation

import "testing"

func testModule() Module {
	return Module{ID: "module:test", Schemas: []Schema{
		{ID: "schema:item", Version: 2, Fields: []Field{
			{ID: "field:name", Shape: Shape{Kind: Bytes}, Cardinality: One, Since: 1},
			{ID: "field:tags", Shape: Shape{Kind: Bytes}, Cardinality: Many, Since: 1},
			{ID: "field:target", Shape: Shape{Kind: Reference, Schema: "schema:item"}, Cardinality: Optional, Since: 2},
		}},
	}}
}

func TestValidatesCardinalityAndReferences(t *testing.T) {
	entities := map[ID]Entity{
		"item:a": {ID: "item:a", Schema: "schema:item", Version: 2, Fields: map[ID]Value{
			"field:name":   {Kind: Bytes, Bytes: []byte("A")},
			"field:tags":   {Kind: List, List: []Value{{Kind: Bytes, Bytes: []byte("one")}, {Kind: Bytes, Bytes: []byte("two")}}},
			"field:target": {Kind: Reference, Reference: "item:b"},
		}},
		"item:b": {ID: "item:b", Schema: "schema:item", Version: 1, Fields: map[ID]Value{"field:name": {Kind: Bytes, Bytes: []byte("B")}, "field:tags": {Kind: List}}},
	}
	results, err := Validate(testModule(), entities)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Disposition != Certified {
			t.Fatalf("result = %#v", result)
		}
	}
}

func TestPreservesUnknownFieldAndFutureVersion(t *testing.T) {
	entities := map[ID]Entity{
		"item:a": {ID: "item:a", Schema: "schema:item", Version: 2, Fields: map[ID]Value{"field:name": {Kind: Bytes}, "field:tags": {Kind: List}, "field:future": {Kind: Unsigned, Unsigned: 42}}},
		"item:b": {ID: "item:b", Schema: "schema:item", Version: 3, Fields: map[ID]Value{}},
	}
	results, err := Validate(testModule(), entities)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Disposition != PreservedUnknown {
			t.Fatalf("result = %#v", result)
		}
	}
}

func TestUnavailableSchemaIsNotCertified(t *testing.T) {
	results, err := Validate(testModule(), map[ID]Entity{"foreign:a": {ID: "foreign:a", Schema: "schema:foreign", Version: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Disposition != Unavailable {
		t.Fatalf("results = %#v", results)
	}
}

func TestRejectsRequiredFieldAndWrongReferenceSchema(t *testing.T) {
	_, err := Validate(testModule(), map[ID]Entity{"item:a": {ID: "item:a", Schema: "schema:item", Version: 1, Fields: map[ID]Value{"field:tags": {Kind: List}}}})
	if err == nil {
		t.Fatal("missing required field accepted")
	}
	entities := map[ID]Entity{
		"item:a":    {ID: "item:a", Schema: "schema:item", Version: 2, Fields: map[ID]Value{"field:name": {Kind: Bytes}, "field:tags": {Kind: List}, "field:target": {Kind: Reference, Reference: "foreign:a"}}},
		"foreign:a": {ID: "foreign:a", Schema: "schema:foreign", Version: 1},
	}
	if _, err := Validate(testModule(), entities); err == nil {
		t.Fatal("wrong reference schema accepted")
	}
}

func TestValidatesRecursiveRecordShape(t *testing.T) {
	module := Module{ID: "module:record", Schemas: []Schema{
		{ID: "schema:inner", Version: 1, Fields: []Field{{ID: "field:value", Shape: Shape{Kind: Unsigned}, Cardinality: One, Since: 1}}},
		{ID: "schema:outer", Version: 1, Fields: []Field{{ID: "field:inner", Shape: Shape{Kind: Record, Schema: "schema:inner"}, Cardinality: One, Since: 1}}},
	}}
	valid := map[ID]Entity{"outer:a": {ID: "outer:a", Schema: "schema:outer", Version: 1, Fields: map[ID]Value{"field:inner": {Kind: Record, Record: map[ID]Value{"field:value": {Kind: Unsigned, Unsigned: 42}}}}}}
	if _, err := Validate(module, valid); err != nil {
		t.Fatal(err)
	}
	invalid := map[ID]Entity{"outer:a": {ID: "outer:a", Schema: "schema:outer", Version: 1, Fields: map[ID]Value{"field:inner": {Kind: Record, Record: map[ID]Value{}}}}}
	if _, err := Validate(module, invalid); err == nil {
		t.Fatal("record missing required field accepted")
	}
}
