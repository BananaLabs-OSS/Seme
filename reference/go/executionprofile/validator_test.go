package executionprofile

import (
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func TestRevisionAuthorityRejectsV36SchemaUnderV35(t *testing.T) {
	v35 := resolve(t, "v35", "9023", 35, "54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")
	v36 := resolve(t, "v36", "9024", 36, "2315477d7c0d248ce167aeada313654d25c056be1d8c851e43bd9c225ff07450")
	g := booleanNotGraph()
	if err := ValidateConstruction(v35, g); err == nil {
		t.Fatal("v35 accepted BooleanNot")
	}
	if err := ValidateConstruction(v36, g); err != nil {
		t.Fatal(err)
	}
	if err := ValidateConstruction(contractcatalog.Contract{}, g); err == nil {
		t.Fatal("untrusted contract accepted")
	}
	bad := clone(g)
	delete(bad.Entities, id("5"))
	if err := ValidateConstruction(v36, bad); err == nil {
		t.Fatal("missing reachable entity accepted")
	}
	bad = clone(g)
	bad.Entities[id("9")] = wire.Entity{ID: id("9"), Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	if err := ValidateConstruction(v36, bad); err == nil {
		t.Fatal("multiple Programs accepted")
	}
	unreachable := clone(g)
	unreachable.Entities[id("a")] = wire.Entity{ID: id("a"), Schema: id("ffffffff"), Version: 1}
	if err := ValidateConstruction(v36, unreachable); err != nil {
		t.Fatalf("unreachable metadata rejected: %v", err)
	}
}

func resolve(t *testing.T, version, revision string, moduleVersion uint64, digestHex string) contractcatalog.Contract {
	t.Helper()
	raw, err := os.ReadFile("../../../modules/execution/" + version + "/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contractcatalog.Resolve(raw, contractcatalog.Expectation{Pin: contractcatalog.Pin{Module: id("9000"), Revision: id(revision)}, ModuleVersion: moduleVersion, Digest: digest(digestHex)})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func booleanNotGraph() wire.Envelope {
	e := map[wire.ID]wire.Entity{}
	put := func(x, s string, fields map[wire.ID]wire.Value) {
		idv := id(x)
		e[idv] = wire.Entity{ID: idv, Schema: id(s), Version: 1, Fields: fields}
	}
	put("1", "9020", nil)
	put("2", "90b0", map[wire.ID]wire.Value{id("9b00"): {Tag: 2}})
	put("3", "a069", map[wire.ID]wire.Value{id("a0690"): ref(id("2"))})
	put("4", "9081", map[wire.ID]wire.Value{id("9810"): list(id("3"))})
	put("5", "9080", map[wire.ID]wire.Value{id("9800"): list(id("4"))})
	put("6", "9011", map[wire.ID]wire.Value{id("9110"): {Tag: 5, Bytes: []byte("Apply")}, id("9111"): {Tag: 7}, id("9112"): ref(id("1")), id("9113"): ref(id("5"))})
	put("7", "9015", map[wire.ID]wire.Value{id("9150"): list(id("6")), id("9151"): ref(id("6"))})
	return wire.Envelope{Entities: e}
}
func clone(in wire.Envelope) wire.Envelope {
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range in.Entities {
		fields := map[wire.ID]wire.Value{}
		for k, v := range q.Fields {
			fields[k] = v
		}
		q.Fields = fields
		out.Entities[x] = q
	}
	return out
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
}
