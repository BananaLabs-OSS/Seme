package upb12entry

import (
	"bytes"
	"seme.local/reference/wire"
	"testing"
)

func TestSelectChangesOnlyAuthenticatedEntryReference(t *testing.T) {
	base, run, dispatch := fixture(t)
	result, err := Select(base, "Run", "DispatchControlled")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := wire.Decode(base)
	after, _ := wire.Decode(result)
	programBefore, programAfter := program(t, before), program(t, after)
	if programBefore.Fields[programEntry].Reference != run || programAfter.Fields[programEntry].Reference != dispatch {
		t.Fatal("entry selection")
	}
	programAfter.Fields[programEntry] = programBefore.Fields[programEntry]
	after.Entities[programAfter.ID] = programAfter
	normalized, _ := wire.Encode(after)
	if !bytes.Equal(base, normalized) {
		t.Fatal("entry view changed semantic content beyond program.entry")
	}
}

func TestSelectRejectsUnknownAndIncorrectEntries(t *testing.T) {
	base, _, _ := fixture(t)
	for _, test := range [][2]string{{"Missing", "DispatchControlled"}, {"Run", "Missing"}, {"DispatchControlled", "Run"}} {
		if _, err := Select(base, test[0], test[1]); err == nil {
			t.Fatalf("accepted %q -> %q", test[0], test[1])
		}
	}
}

func fixture(t *testing.T) ([]byte, wire.ID, wire.ID) {
	t.Helper()
	run, dispatch, p := identity(0x2001), identity(0x2002), identity(0x2003)
	blob := func(id wire.ID, name string) wire.Entity {
		return wire.Entity{ID: id, Schema: functionSchema, Version: 1, Fields: map[wire.ID]wire.Value{functionName: {Tag: 5, Bytes: []byte(name)}}}
	}
	e := wire.Envelope{Module: identity(0x9000), Revision: identity(0x9024), Entities: map[wire.ID]wire.Entity{run: blob(run, "Run"), dispatch: blob(dispatch, "DispatchControlled"), p: {ID: p, Schema: programSchema, Version: 1, Fields: map[wire.ID]wire.Value{programFunctions: {Tag: 7, List: []wire.Value{{Tag: 6, Reference: run}, {Tag: 6, Reference: dispatch}}}, programEntry: {Tag: 6, Reference: run}}}}}
	value, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	return value, run, dispatch
}
func program(t *testing.T, e wire.Envelope) wire.Entity {
	t.Helper()
	for _, item := range e.Entities {
		if item.Schema == programSchema {
			return item
		}
	}
	t.Fatal("program missing")
	return wire.Entity{}
}
