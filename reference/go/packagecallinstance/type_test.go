package packagecallinstance

import (
	"testing"

	"seme.local/reference/wire"
)

func TestExpressionTypeReusesRichCanonicalDerivationAndFailsClosed(t *testing.T) {
	typ, param, pread := id("7001"), id("7002"), id("7003")
	record, field, construct, read := id("7010"), id("7011"), id("7012"), id("7013")
	binding, local := id("7020"), id("7021")
	mapType, collection, update := id("702f"), id("7030"), id("7031")
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		typ: {ID: typ, Schema: id("9010")}, param: {ID: param, Schema: id("9012"), Fields: map[wire.ID]wire.Value{id("9121"): ref(typ)}}, pread: {ID: pread, Schema: id("9013"), Fields: map[wire.ID]wire.Value{id("9130"): ref(param)}},
		record: {ID: record, Schema: id("9030"), Fields: map[wire.ID]wire.Value{id("9301"): {Tag: 7, List: []wire.Value{ref(field)}}}}, field: {ID: field, Schema: id("9031"), Fields: map[wire.ID]wire.Value{id("9311"): ref(typ)}}, construct: {ID: construct, Schema: id("9033"), Fields: map[wire.ID]wire.Value{id("9330"): ref(record)}}, read: {ID: read, Schema: id("9032"), Fields: map[wire.ID]wire.Value{id("9320"): ref(construct), id("9321"): ref(field)}},
		binding: {ID: binding, Schema: id("90d0"), Fields: map[wire.ID]wire.Value{id("9d01"): ref(typ)}}, local: {ID: local, Schema: id("90d2"), Fields: map[wire.ID]wire.Value{id("9d20"): ref(binding)}},
		mapType: {ID: mapType, Schema: id("a040")}, collection: {ID: collection, Schema: id("a041"), Fields: map[wire.ID]wire.Value{id("a0410"): ref(mapType)}}, update: {ID: update, Schema: id("a043"), Fields: map[wire.ID]wire.Value{id("a0430"): ref(collection)}},
	}}
	for name, tc := range map[string]struct{ x, want wire.ID }{"parameter": {pread, typ}, "record": {construct, record}, "field": {read, typ}, "local": {local, typ}, "collection": {update, mapType}} {
		got, err := expressionType(e, tc.x)
		if err != nil || got != tc.want {
			t.Fatalf("%s got=%s err=%v", name, got, err)
		}
	}
	unknown := id("7fff")
	e.Entities[unknown] = wire.Entity{ID: unknown, Schema: id("90f9")}
	if _, err := expressionType(e, unknown); err == nil {
		t.Fatal("unsupported expression type accepted")
	}
}
