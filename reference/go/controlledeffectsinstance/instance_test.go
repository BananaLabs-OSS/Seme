package controlledeffectsinstance

import (
	"bytes"
	"testing"
)

func TestReplayDigestIsDeterministicAndMeaningSensitive(t *testing.T) {
	r := Replay{InitialSeed: 7, Steps: []ReplayStep{{CommandSequence: 1, UnixMilliseconds: 2, ClockSequence: 3, RandomDraw: 4, EffectValue: true}}, DuplicatePolicy: "exact", RejectionPolicy: "atomic"}
	a, b := replayDigest(r), replayDigest(r)
	if len(a) != 32 || !bytes.Equal(a, b) {
		t.Fatal("nondeterministic replay digest")
	}
	r.Steps[0].EffectValue = false
	if bytes.Equal(a, replayDigest(r)) {
		t.Fatal("digest ignored effect value")
	}
	r.Steps[0].EffectValue = true
	r.Steps[0].ClockSequence++
	if bytes.Equal(a, replayDigest(r)) {
		t.Fatal("digest ignored clock sequence")
	}
}

func TestZeroAuthorityAndArtifactAreRejected(t *testing.T) {
	if _, err := Emit(Inputs{}); err == nil {
		t.Fatal("accepted zero authority")
	}
	if err := Validate(Inputs{}); err == nil {
		t.Fatal("accepted zero artifact")
	}
}

func TestBooleanEncodingIsExact(t *testing.T) {
	if boolean(false).Tag != 1 || boolean(true).Tag != 2 {
		t.Fatal("boolean wire tags")
	}
}
