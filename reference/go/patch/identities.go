package patch

import "seme.local/reference/foundation"

const (
	ModuleID           foundation.ID = "00000000000000000000000000005000"
	SchemaPatch        foundation.ID = "00000000000000000000000000005010"
	SchemaRename       foundation.ID = "00000000000000000000000000005011"
	FieldAuthor        foundation.ID = "00000000000000000000000000005100"
	FieldBaseRevision  foundation.ID = "00000000000000000000000000005101"
	FieldOperations    foundation.ID = "00000000000000000000000000005102"
	FieldTarget        foundation.ID = "00000000000000000000000000005110"
	FieldTargetField   foundation.ID = "00000000000000000000000000005111"
	FieldExpectedValue foundation.ID = "00000000000000000000000000005112"
	FieldReplacement   foundation.ID = "00000000000000000000000000005113"
)

func ModuleV1() foundation.Module {
	return foundation.Module{ID: ModuleID, Schemas: []foundation.Schema{
		{ID: SchemaPatch, Version: 1, Fields: []foundation.Field{
			{ID: FieldAuthor, Shape: foundation.Shape{Kind: foundation.Reference}, Cardinality: foundation.One, Since: 1},
			{ID: FieldBaseRevision, Shape: foundation.Shape{Kind: foundation.Bytes}, Cardinality: foundation.One, Since: 1},
			{ID: FieldOperations, Shape: foundation.Shape{Kind: foundation.Reference, Schema: SchemaRename}, Cardinality: foundation.Many, Since: 1},
		}},
		{ID: SchemaRename, Version: 1, Fields: []foundation.Field{
			{ID: FieldTarget, Shape: foundation.Shape{Kind: foundation.Reference}, Cardinality: foundation.One, Since: 1},
			{ID: FieldTargetField, Shape: foundation.Shape{Kind: foundation.Bytes}, Cardinality: foundation.One, Since: 1},
			{ID: FieldExpectedValue, Shape: foundation.Shape{Kind: foundation.Bytes}, Cardinality: foundation.One, Since: 1},
			{ID: FieldReplacement, Shape: foundation.Shape{Kind: foundation.Bytes}, Cardinality: foundation.One, Since: 1},
		}},
	}}
}
