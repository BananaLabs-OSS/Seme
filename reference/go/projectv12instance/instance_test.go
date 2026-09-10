package projectv12instance

import (
	"bytes"
	"testing"

	"seme.local/reference/wire"
)

func TestRejectsUnauthenticatedWithoutPartialArtifact(t *testing.T) {
	if artifact, err := Emit(Inputs{}); err == nil || len(artifact) != 0 {
		t.Fatal("accepted unauthenticated")
	}
	if err := Validate(Inputs{Composed: []byte("forged")}); err == nil {
		t.Fatal("accepted forged artifact")
	}
}

func TestSnapshotDigestBindsBothExactRoots(t *testing.T) {
	base, effects, snapshot := v12id("aa01"), v12id("aa02"), v12id("aa03")
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		base:     v12entity(base, "e026", map[string]wire.Value{"e262": v12blob(bytes.Repeat([]byte{1}, 32))}),
		effects:  v12entity(effects, "13100", map[string]wire.Value{"13207": v12blob(bytes.Repeat([]byte{2}, 32))}),
		snapshot: v12entity(snapshot, "e029", map[string]wire.Value{"e290": v12ref(base), "e291": v12ref(effects), "e292": v12blob(make([]byte, 32))}),
	}}
	first := v12ContentRevision(e, snapshot)
	second := v12ContentRevision(e, snapshot)
	if len(first) != 32 || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic")
	}
	q := e.Entities[effects]
	q.Fields[v12id("13207")] = v12blob(bytes.Repeat([]byte{3}, 32))
	e.Entities[effects] = q
	if bytes.Equal(first, v12ContentRevision(e, snapshot)) {
		t.Fatal("effects tamper not bound")
	}
	q = e.Entities[base]
	q.Fields[v12id("e262")] = v12blob(bytes.Repeat([]byte{4}, 32))
	e.Entities[base] = q
	if bytes.Equal(first, v12ContentRevision(e, snapshot)) {
		t.Fatal("project tamper not bound")
	}
}

func TestEntityCollisionAndArtifactTamperAreMeaningSensitive(t *testing.T) {
	x := v12id("bb01")
	a := v12entity(x, "e026", map[string]wire.Value{"e262": v12blob([]byte("a"))})
	b := v12entity(x, "e026", map[string]wire.Value{"e262": v12blob([]byte("b"))})
	if v12same(a, b) || !v12same(a, a) {
		t.Fatal("collision comparison")
	}
	e := wire.Envelope{Module: x, Revision: v12id("1"), Entities: map[wire.ID]wire.Entity{x: a}}
	first := v12ArtifactRevision(e)
	e.Entities[x] = b
	if first == v12ArtifactRevision(e) {
		t.Fatal("artifact revision ignored tamper")
	}
}
