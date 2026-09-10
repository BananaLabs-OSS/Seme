package transport

import (
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
)

func correlation(value int64) Correlation { return Correlation{High: value / 251, Low: value + 1} }
func digest(value int64) Digest {
	return Digest{A: value + 1, B: value + 2, C: value + 3, D: value + 4}
}

func payload() PlannerPayload {
	base := application.State{Name: "pilot", Values: []int64{1, 2}, Counters: map[int64]int64{1: 2}}
	return PlannerPayload{Grants: persistence.Grants{Read: true, CompareExchange: true}, Loaded: persistence.Loaded{Found: true, Version: 2, Token: "opaque", V2: state.V2{State: base, Revision: 2}}, Initial: state.V2{State: base, Revision: 1}, NextDigest: "sha256:next", Key: "slot"}
}

func command(sequence int64, identity int64) Command {
	canonical := make([]byte, 16)
	for index := range canonical {
		canonical[index] = byte(identity + int64(index))
	}
	return Command{Stream: "match", Sequence: sequence, Correlation: correlation(identity), Kind: PlannerCommand, Payload: payload(), CanonicalPayload: canonical, Digest: digest(identity)}
}

func TestClassifyCommitCachedConflictGapAndCorrelation(t *testing.T) {
	initial := NewState("match")
	firstCommand := command(1, 1)
	if classified := Classify(initial, firstCommand); classified.Kind != ClassificationNew || classified.Error != 0 {
		t.Fatalf("classification=%#v", classified)
	}
	first := Commit(initial, firstCommand, true, 0, 3, 2)
	if !first.OK || first.State.NextCommand != 2 || first.State.NextEvent != 3 || first.Response.EventCount != 2 || first.Response.Event0.Sequence != 1 || first.Response.Event1.Ordinal != 1 {
		t.Fatalf("first=%#v", first)
	}
	cachedClass := Classify(first.State, firstCommand)
	if cachedClass.Kind != ClassificationCached || !reflect.DeepEqual(cachedClass.Response, first.Response) {
		t.Fatalf("cached classification=%#v", cachedClass)
	}
	cached := Commit(first.State, firstCommand, true, 0, 99, 4)
	if !cached.OK || !cached.Duplicate || !reflect.DeepEqual(cached.State, first.State) || !reflect.DeepEqual(cached.Response, first.Response) {
		t.Fatalf("cached=%#v", cached)
	}
	changed := firstCommand
	changed.Digest.A++
	if got := Classify(first.State, changed); got.Error != SequenceConflict {
		t.Fatalf("conflict=%#v", got)
	}
	collision := firstCommand
	collision.CanonicalPayload = append([]byte{}, firstCommand.CanonicalPayload...)
	collision.CanonicalPayload[0]++
	if got := Classify(first.State, collision); got.Error != SequenceConflict {
		t.Fatalf("payload collision=%#v", got)
	}
	if got := Classify(first.State, command(3, 3)); got.Error != OutOfOrder {
		t.Fatalf("gap=%#v", got)
	}
	reused := command(2, 1)
	if got := Classify(first.State, reused); got.Error != CorrelationReused {
		t.Fatalf("reuse=%#v", got)
	}
	second := Commit(first.State, command(2, 2), false, 71, 0, 1)
	if !second.OK || second.Response.Accepted || second.Response.Code != 71 || second.Response.Event0.Sequence != 3 || second.Response.Event0.Kind != RejectedEvent || second.State.NextCommand != 3 {
		t.Fatalf("second=%#v", second)
	}
}

func TestInvalidInputsAndOutcomesAreAtomic(t *testing.T) {
	initial := NewState("match")
	zeroCorrelation := command(1, 1)
	zeroCorrelation.Correlation = Correlation{}
	zeroDigest := command(1, 1)
	zeroDigest.Digest = Digest{}
	wrongKind := command(1, 1)
	wrongKind.Kind = 2
	wrongStream := command(1, 1)
	wrongStream.Stream = "other"
	for name, value := range map[string]Command{"sequence": command(0, 1), "correlation": zeroCorrelation, "digest": zeroDigest, "kind": wrongKind, "stream": wrongStream, "gap": command(2, 2)} {
		t.Run(name, func(t *testing.T) {
			classified := Classify(initial, value)
			if classified.Error == 0 {
				t.Fatalf("accepted=%#v", classified)
			}
		})
	}
	for _, input := range []struct {
		accepted   bool
		code       int64
		eventCount int64
	}{{true, 1, 1}, {false, 0, 1}, {true, 0, 5}, {true, 0, -1}} {
		got := Commit(initial, command(1, 1), input.accepted, input.code, 1, input.eventCount)
		if got.OK || got.Error != InvalidDomainOutcome || !reflect.DeepEqual(got.State, initial) {
			t.Fatalf("outcome=%#v", got)
		}
	}
	tampered := NewState("match")
	tampered.Codes = []int64{1}
	if ValidState(tampered) || Classify(tampered, command(1, 1)).Error != InvalidState {
		t.Fatal("accepted malformed column ledger")
	}
	malformed := Commit(initial, command(1, 1), true, 0, 1, 0).State
	malformed.PayloadStarts[0] = 1
	if ValidState(malformed) {
		t.Fatal("accepted malformed payload segment")
	}
}

func TestZeroAndFourEventResponses(t *testing.T) {
	zero := Commit(NewState("match"), command(1, 1), true, 0, 3, 0)
	if !zero.OK || zero.Response.EventCount != 0 || zero.State.NextEvent != 1 {
		t.Fatalf("zero=%#v", zero)
	}
	four := Commit(zero.State, command(2, 2), true, 0, 4, 4)
	if !four.OK || four.Response.EventCount != 4 || four.Response.Event0.Sequence != 1 || four.Response.Event3.Sequence != 4 || four.Response.Event3.Ordinal != 3 || four.State.NextEvent != 5 {
		t.Fatalf("four=%#v", four)
	}
}

func TestBoundedLedgerAndOwnedColumns(t *testing.T) {
	current := NewState("match")
	for index := int64(1); index <= MaximumCommands; index++ {
		result := Commit(current, command(index, index), true, 0, index, 0)
		if !result.OK {
			t.Fatalf("command %d: %#v", index, result)
		}
		current = result.State
	}
	if current.NextCommand != MaximumCommands+1 || len(current.Codes) != MaximumCommands {
		t.Fatalf("terminal=%#v", current)
	}
	duplicate := Commit(current, command(1, 1), true, 0, 999, 4)
	if !duplicate.OK || !duplicate.Duplicate || duplicate.Response.Revision != 1 {
		t.Fatalf("duplicate=%#v", duplicate)
	}
	over := command(MaximumCommands+1, 300)
	if got := Classify(current, over); got.Error != InvalidSequence {
		t.Fatalf("over=%#v", got)
	}
	copy := CloneState(current)
	copy.Codes[0] = 99
	copy.PayloadBytes[0]++
	if current.Codes[0] != 0 {
		t.Fatal("clone retained ledger aliases")
	}
	if copy.PayloadBytes[0] == current.PayloadBytes[0] {
		t.Fatal("clone retained payload alias")
	}
	largeFirst := command(1, 1001)
	largeFirst.CanonicalPayload = make([]byte, 3072)
	largeFirst.CanonicalPayload[0] = 1
	largeState := Commit(NewState("match"), largeFirst, true, 0, 1, 0).State
	largeSecond := command(2, 1002)
	largeSecond.CanonicalPayload = make([]byte, 1024)
	largeSecond.CanonicalPayload[0] = 2
	exact := Commit(largeState, largeSecond, true, 0, 2, 0)
	if !exact.OK || len(exact.State.PayloadBytes) != MaximumEncodedFrameBytes {
		t.Fatalf("exact payload bound=%#v", exact)
	}
	largeThird := command(3, 1003)
	largeThird.CanonicalPayload = []byte{3}
	got := Commit(exact.State, largeThird, true, 0, 3, 0)
	if got.OK || got.Error != InvalidPayload || !reflect.DeepEqual(got.State, exact.State) {
		t.Fatalf("payload overflow=%#v", got)
	}
}

func TestReplayByAcceptedCommandsIsDeterministic(t *testing.T) {
	replay := func() (State, []Response) {
		current := NewState("match")
		responses := []Response{}
		for index := int64(1); index <= 32; index++ {
			accepted := index%3 != 0
			code := int64(0)
			if !accepted {
				code = 71
			}
			result := Commit(current, command(index, index), accepted, code, index, index%5)
			if !result.OK {
				t.Fatal(result.Error)
			}
			current = result.State
			responses = append(responses, result.Response)
		}
		return current, responses
	}
	firstState, firstResponses := replay()
	secondState, secondResponses := replay()
	if !reflect.DeepEqual(firstState, secondState) || !reflect.DeepEqual(firstResponses, secondResponses) {
		t.Fatal("accepted transcript did not replay exactly")
	}
}
