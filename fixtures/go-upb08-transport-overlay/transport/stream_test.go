package transport

import (
	"errors"
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
)

type handlerFunc func(PlannerPayload) (DomainOutcome, error)

func (f handlerFunc) Handle(payload PlannerPayload) (DomainOutcome, error) { return f(payload) }

func correlation(value byte) Correlation {
	var result Correlation
	result[15] = value
	return result
}

func payload() PlannerPayload {
	base := application.State{Name: "pilot", Values: []int64{1, 2}, Counters: map[int64]int64{1: 2}}
	return PlannerPayload{Grants: persistence.Grants{Read: true, CompareExchange: true}, Loaded: persistence.Loaded{Found: true, Version: 2, Token: "opaque", V2: state.V2{State: base, Revision: 2}}, Initial: state.V2{State: base, Revision: 1}, NextDigest: "sha256:next", Key: "slot"}
}

func command(t *testing.T, sequence int64, id byte) Command {
	t.Helper()
	var digest [32]byte
	digest[0], digest[31] = id, byte(sequence)
	return NewCommand("match", sequence, correlation(id), "planner.update.v1", payload(), digest)
}

func accepting(events int, calls *int) handlerFunc {
	return func(got PlannerPayload) (DomainOutcome, error) {
		*calls++
		if !equalPayload(got, payload()) {
			return DomainOutcome{}, errors.New("payload changed")
		}
		out := DomainOutcome{Accepted: true, Receipt: persistence.Plan{Ok: true}, Events: make([]DomainEvent, events)}
		for index := range out.Events {
			out.Events[index] = DomainEvent{Kind: "changed", Payload: persistence.Plan{Ok: true}}
		}
		return out, nil
	}
}

func TestOrderingDuplicateConflictGapAndCorrelation(t *testing.T) {
	calls := 0
	handler := accepting(2, &calls)
	initial := NewState("match")
	firstCommand := command(t, 1, 1)
	first := Dispatch(initial, firstCommand, handler)
	if !first.OK || first.Duplicate || calls != 1 || first.State.NextCommand != 2 || first.State.NextEvent != 3 || len(first.Response.Events) != 2 || first.Response.Events[0].Sequence != 1 || first.Response.Events[1].Ordinal != 1 {
		t.Fatalf("first=%#v calls=%d", first, calls)
	}
	beforeDuplicate := cloneState(first.State)
	duplicate := Dispatch(first.State, firstCommand, handler)
	if !duplicate.OK || !duplicate.Duplicate || calls != 1 || !reflect.DeepEqual(duplicate.State, beforeDuplicate) || !reflect.DeepEqual(duplicate.Response, first.Response) {
		t.Fatalf("duplicate=%#v calls=%d", duplicate, calls)
	}
	conflict := firstCommand
	conflict.Kind = "other"
	if got := Dispatch(first.State, conflict, handler); got.OK || got.Error != SequenceConflict || !reflect.DeepEqual(got.State, first.State) || calls != 1 {
		t.Fatalf("conflict=%#v", got)
	}
	payloadConflict := firstCommand
	payloadConflict.Payload.Key = "different"
	if got := Dispatch(first.State, payloadConflict, handler); got.OK || got.Error != SequenceConflict || calls != 1 {
		t.Fatalf("payload conflict=%#v", got)
	}
	if got := Dispatch(first.State, command(t, 3, 3), handler); got.OK || got.Error != OutOfOrder || got.State.NextCommand != 2 || calls != 1 {
		t.Fatalf("gap=%#v", got)
	}
	reused := command(t, 2, 1)
	if got := Dispatch(first.State, reused, handler); got.OK || got.Error != CorrelationReused || got.State.NextCommand != 2 || calls != 1 {
		t.Fatalf("reuse=%#v", got)
	}
	second := Dispatch(first.State, command(t, 2, 2), handler)
	if !second.OK || calls != 2 || second.Response.Events[0].Sequence != 3 || second.State.NextCommand != 3 {
		t.Fatalf("second=%#v calls=%d", second, calls)
	}
}

func TestDomainRejectionIsRecordedAndConsumesSequence(t *testing.T) {
	calls := 0
	handler := handlerFunc(func(PlannerPayload) (DomainOutcome, error) {
		calls++
		plan := persistence.Plan{Error: 71}
		return DomainOutcome{Code: 71, Receipt: plan, Events: []DomainEvent{{Kind: "rejected", Payload: plan}}}, nil
	})
	cmd := command(t, 1, 1)
	result := Dispatch(NewState("match"), cmd, handler)
	if !result.OK || result.Response.Accepted || result.Response.Code != 71 || result.State.NextCommand != 2 || len(result.State.Ledger) != 1 || calls != 1 {
		t.Fatalf("result=%#v", result)
	}
	again := Dispatch(result.State, cmd, handler)
	if !again.OK || !again.Duplicate || !reflect.DeepEqual(again.Response, result.Response) || calls != 1 {
		t.Fatalf("again=%#v calls=%d", again, calls)
	}
}

func TestInvalidInputsAndHandlerFailureAreAtomic(t *testing.T) {
	initial := NewState("match")
	valid := command(t, 1, 1)
	badDigest := valid
	badDigest.Digest = [32]byte{}
	for name, cmd := range map[string]Command{
		"zero-sequence":    {Stream: "match", Correlation: correlation(1), Kind: "planner.update.v1", Payload: payload()},
		"zero-correlation": func() Command { x := valid; x.Correlation = Correlation{}; return x }(),
		"bad-digest":       badDigest,
		"gap":              command(t, 2, 2),
		"invalid-utf8": func() Command {
			x := valid
			x.Stream = string([]byte{0xff})
			return x
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			got := Dispatch(initial, cmd, accepting(1, &calls))
			if got.OK || !reflect.DeepEqual(got.State, initial) || calls != 0 {
				t.Fatalf("got=%#v calls=%d", got, calls)
			}
		})
	}
	calls := 0
	failed := Dispatch(initial, valid, handlerFunc(func(PlannerPayload) (DomainOutcome, error) { calls++; return DomainOutcome{}, errors.New("failure") }))
	if failed.OK || failed.Error != DispatchFailed || !reflect.DeepEqual(failed.State, initial) || calls != 1 {
		t.Fatalf("failed=%#v calls=%d", failed, calls)
	}
	invalidOutcome := Dispatch(initial, valid, handlerFunc(func(PlannerPayload) (DomainOutcome, error) { return DomainOutcome{Accepted: true, Code: 1}, nil }))
	if invalidOutcome.OK || invalidOutcome.Error != InvalidDomainOutcome || !reflect.DeepEqual(invalidOutcome.State, initial) {
		t.Fatalf("invalid outcome=%#v", invalidOutcome)
	}
}

func TestReplayReproducesStateAndResponses(t *testing.T) {
	commands := []Command{command(t, 1, 1), command(t, 2, 2), command(t, 3, 3)}
	firstCalls, secondCalls := 0, 0
	first := Replay(NewState("match"), commands, accepting(1, &firstCalls))
	second := Replay(NewState("match"), commands, accepting(1, &secondCalls))
	if !first.OK || !second.OK || firstCalls != 3 || secondCalls != 3 || !reflect.DeepEqual(first, second) {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	bad := []Command{commands[0], commands[2]}
	if got := Replay(NewState("match"), bad, accepting(1, &firstCalls)); got.OK || got.Error != OutOfOrder || len(got.Responses) != 1 {
		t.Fatalf("bad replay=%#v", got)
	}
}

func TestCommandAndEventBounds(t *testing.T) {
	current := NewState("match")
	calls := 0
	for index := 1; index <= MaximumCommands; index++ {
		var id Correlation
		id[14], id[15] = byte(index>>8), byte(index)
		var digest [32]byte
		digest[30], digest[31] = id[14], id[15]
		result := Dispatch(current, NewCommand("match", int64(index), id, "planner.update.v1", payload(), digest), accepting(0, &calls))
		if !result.OK {
			t.Fatalf("command %d: %#v", index, result)
		}
		current = result.State
	}
	if current.NextCommand != MaximumCommands+1 || len(current.Ledger) != MaximumCommands || calls != MaximumCommands {
		t.Fatalf("terminal state=%#v calls=%d", current, calls)
	}
	first := NewCommand("match", 1, current.Ledger[0].Correlation, current.Ledger[0].Kind, current.Ledger[0].Payload, current.Ledger[0].Digest)
	if duplicate := Dispatch(current, first, accepting(0, &calls)); !duplicate.OK || !duplicate.Duplicate || calls != MaximumCommands {
		t.Fatalf("terminal duplicate=%#v calls=%d", duplicate, calls)
	}
	var nextID Correlation
	nextID[0] = 1
	var nextDigest [32]byte
	nextDigest[0] = 1
	if over := Dispatch(current, NewCommand("match", MaximumCommands+1, nextID, "planner.update.v1", payload(), nextDigest), accepting(0, &calls)); over.OK || over.Error != InvalidSequence || calls != MaximumCommands {
		t.Fatalf("over=%#v calls=%d", over, calls)
	}
	tooMany := Dispatch(NewState("match"), command(t, 1, 1), accepting(MaximumEventsPerReply+1, new(int)))
	if tooMany.OK || tooMany.Error != InvalidDomainOutcome {
		t.Fatalf("too many events=%#v", tooMany)
	}
}

func TestPayloadMapKeyPresenceParticipatesInDuplicateIdentity(t *testing.T) {
	first := command(t, 1, 1)
	first.Payload.Initial.State.Counters = map[int64]int64{1: 0}
	calls := 0
	handler := handlerFunc(func(PlannerPayload) (DomainOutcome, error) {
		calls++
		return DomainOutcome{Accepted: true, Receipt: persistence.Plan{Ok: true}}, nil
	})
	accepted := Dispatch(NewState("match"), first, handler)
	if !accepted.OK {
		t.Fatal(accepted.Error)
	}
	changed := first
	changed.Payload = clonePayload(first.Payload)
	changed.Payload.Initial.State.Counters = map[int64]int64{2: 0}
	if result := Dispatch(accepted.State, changed, handler); result.OK || result.Error != SequenceConflict || calls != 1 {
		t.Fatalf("result=%#v calls=%d", result, calls)
	}
}

func TestDispatchOwnsTypedPayloadCopies(t *testing.T) {
	cmd := command(t, 1, 1)
	result := Dispatch(NewState("match"), cmd, accepting(1, new(int)))
	cmd.Payload.Initial.State.Values[0] = 99
	cmd.Payload.Initial.State.Counters[1] = 99
	if result.State.Ledger[0].Payload.Initial.State.Values[0] != 1 || result.State.Ledger[0].Payload.Initial.State.Counters[1] != 2 {
		t.Fatal("ledger retained caller aliases")
	}
}
