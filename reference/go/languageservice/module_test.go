package languageservice

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestModuleDeterministicAndIdentitiesUnique(t *testing.T) {
	var first, second bytes.Buffer
	Emit(&first)
	Emit(&second)
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("module emission is not deterministic")
	}
	seen := map[uint64]bool{}
	for _, schema := range Declarations() {
		if seen[schema.ID] {
			t.Fatal("duplicate schema identity")
		}
		seen[schema.ID] = true
		for _, field := range schema.Fields {
			if seen[field.ID] {
				t.Fatal("duplicate field identity")
			}
			seen[field.ID] = true
		}
	}
}

func TestApplyUpdatePreservesLastValidAndRejectsStale(t *testing.T) {
	baseRevision := bytes.Repeat([]byte{1}, 16)
	current := DocumentState{Document: "stable-document", ClientRevision: 4, LastValidRevision: baseRevision}
	staleContent := []byte("stale")
	called := false
	next, result, err := ApplyUpdate(current, UpdateRequest{Document: current.Document, ClientRevision: 4, Content: staleContent, ContentDigest: sha256.Sum256(staleContent)}, func([]byte) LiftEvidence { called = true; return LiftEvidence{} })
	if err != nil || called || result.Disposition != LiftRejectedStale || next.ClientRevision != 4 {
		t.Fatalf("stale transition = %#v, %#v, %v", next, result, err)
	}

	invalid := []byte("invalid")
	next, result, err = ApplyUpdate(current, UpdateRequest{Document: current.Document, ClientRevision: 5, Content: invalid, ContentDigest: sha256.Sum256(invalid)}, func([]byte) LiftEvidence { return LiftEvidence{Diagnostics: []Diagnostic{{Code: "syntax.invalid"}}} })
	if err != nil || result.Disposition != LiftAcceptedInvalid || next.ClientRevision != 5 || !bytes.Equal(next.LastValidRevision, baseRevision) {
		t.Fatalf("invalid transition = %#v, %#v, %v", next, result, err)
	}

	valid := []byte("valid")
	newRevision := bytes.Repeat([]byte{2}, 16)
	next, result, err = ApplyUpdate(next, UpdateRequest{Document: current.Document, ClientRevision: 6, Content: valid, ContentDigest: sha256.Sum256(valid)}, func([]byte) LiftEvidence {
		return LiftEvidence{CanonicalRevision: newRevision, SourceMappings: []SourceMapping{{Start: 0, End: 5}}}
	})
	if err != nil || result.Disposition != LiftAcceptedValid || !bytes.Equal(next.LastValidRevision, newRevision) || len(result.SourceMappings) != 1 {
		t.Fatalf("valid transition = %#v, %#v, %v", next, result, err)
	}
}

func TestApplyUpdateRejectsDigestMismatchWithoutMutation(t *testing.T) {
	current := DocumentState{Document: "stable-document", ClientRevision: 2}
	next, _, err := ApplyUpdate(current, UpdateRequest{Document: current.Document, ClientRevision: 3, Content: []byte("changed")}, func([]byte) LiftEvidence { t.Fatal("lift called"); return LiftEvidence{} })
	if !errorsIs(err, ErrDigestMismatch) || next.ClientRevision != current.ClientRevision {
		t.Fatalf("digest transition = %#v, %v", next, err)
	}
}

func TestApplyUpdateRejectsMalformedLiftEvidenceWithoutMutation(t *testing.T) {
	content := []byte("value")
	current := DocumentState{Document: "stable-document", ClientRevision: 2}
	next, _, err := ApplyUpdate(current, UpdateRequest{Document: current.Document, ClientRevision: 3, Content: content, ContentDigest: sha256.Sum256(content)}, func([]byte) LiftEvidence {
		return LiftEvidence{CanonicalRevision: []byte{1}, SourceMappings: []SourceMapping{{Start: 4, End: 6}}}
	})
	if !errorsIs(err, ErrInvalidLiftEvidence) || next.ClientRevision != current.ClientRevision {
		t.Fatalf("malformed evidence transition = %#v, %v", next, err)
	}
}

func errorsIs(got, want error) bool { return got == want }
