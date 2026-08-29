package patch

import (
	"reflect"
	"testing"

	"seme.local/reference/foundation"
)

const nameField foundation.ID = "field:name"

func workspace() Workspace {
	module := foundation.Module{ID: "module:test", Schemas: []foundation.Schema{{ID: "schema:function", Version: 1, Fields: []foundation.Field{{ID: nameField, Shape: foundation.Shape{Kind: foundation.Bytes}, Cardinality: foundation.One, Since: 1}}}}}
	return Workspace{Revision: "revision:one", Module: module, Entities: map[foundation.ID]foundation.Entity{"function:greeting": {ID: "function:greeting", Schema: "schema:function", Version: 1, Fields: map[foundation.ID]foundation.Value{nameField: {Kind: foundation.Bytes, Bytes: []byte("Greeting")}}}}}
}

func TestApplyRename(t *testing.T) {
	base := workspace()
	transaction := Patch{ID: "patch:one", Author: "actor:test", BaseRevision: base.Revision, Renames: []Rename{{Target: "function:greeting", Field: nameField, Expected: []byte("Greeting"), Replacement: []byte("Welcome")}}}
	got, err := Apply(base, transaction)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision == base.Revision || string(got.Entities["function:greeting"].Fields[nameField].Bytes) != "Welcome" {
		t.Fatalf("unexpected candidate: %#v", got)
	}
	if string(base.Entities["function:greeting"].Fields[nameField].Bytes) != "Greeting" {
		t.Fatal("base workspace mutated")
	}
	again, err := Apply(base, transaction)
	if err != nil {
		t.Fatal(err)
	}
	if again.Revision != got.Revision {
		t.Fatalf("revision is nondeterministic: %s != %s", again.Revision, got.Revision)
	}
}

func TestRejectsStaleRevision(t *testing.T) {
	base := workspace()
	_, err := Apply(base, Patch{ID: "patch:one", Author: "actor:test", BaseRevision: "stale"})
	if err == nil || err.Error() != "patch.stale_revision" {
		t.Fatalf("error = %v", err)
	}
}
func TestRejectsPreconditionAtomically(t *testing.T) {
	base := workspace()
	before := clone(base)
	_, err := Apply(base, Patch{ID: "patch:one", Author: "actor:test", BaseRevision: base.Revision, Renames: []Rename{{Target: "function:greeting", Field: nameField, Expected: []byte("wrong"), Replacement: []byte("Welcome")}}})
	if err == nil {
		t.Fatal("precondition accepted")
	}
	if !reflect.DeepEqual(base, before) {
		t.Fatal("failed patch mutated base")
	}
}
func TestRejectsDuplicateWriteAtomically(t *testing.T) {
	base := workspace()
	before := clone(base)
	rename := Rename{Target: "function:greeting", Field: nameField, Expected: []byte("Greeting"), Replacement: []byte("Welcome")}
	_, err := Apply(base, Patch{ID: "patch:one", Author: "actor:test", BaseRevision: base.Revision, Renames: []Rename{rename, rename}})
	if err == nil {
		t.Fatal("duplicate write accepted")
	}
	if !reflect.DeepEqual(base, before) {
		t.Fatal("failed patch mutated base")
	}
}
