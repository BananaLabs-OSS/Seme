package canonicalclosure

import (
	"os"
	"testing"

	"seme.local/reference/wire"
)

func TestAuthenticatedModuleIsClosedAndTamperRejects(t *testing.T) {
	source, err := os.ReadFile("../../../modules/target/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(source); err != nil {
		t.Fatal(err)
	}
	graph, _ := wire.Decode(source)
	for identity, entity := range graph.Entities {
		for field, value := range entity.Fields {
			if replaceReference(&value) {
				entity.Fields[field] = value
				graph.Entities[identity] = entity
				forged, encodeErr := wire.Encode(graph)
				if encodeErr != nil || Validate(forged) == nil {
					t.Fatal("accepted missing reference", encodeErr)
				}
				return
			}
		}
	}
	t.Fatal("fixture has no reference")
}

func replaceReference(value *wire.Value) bool {
	if value.Tag == 6 {
		value.Reference = wire.ID{0xff}
		return true
	}
	for index := range value.List {
		if replaceReference(&value.List[index]) {
			return true
		}
	}
	for key, item := range value.Record {
		if replaceReference(&item) {
			value.Record[key] = item
			return true
		}
	}
	return false
}
