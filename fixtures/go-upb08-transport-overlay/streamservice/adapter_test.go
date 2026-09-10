package streamservice

import (
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
	"example.test/go-uab-11/transport"
)

func command(sequence int64, identity int64) transport.Command {
	base := application.State{Name: "pilot", Counters: map[int64]int64{}}
	return transport.Command{Stream: "match", Sequence: sequence, Correlation: transport.Correlation{Low: identity}, Kind: transport.PlannerCommand, Digest: transport.Digest{A: identity}, CanonicalPayload: []byte{byte(identity)}, Payload: transport.PlannerPayload{Grants: persistence.Grants{Read: true, CompareExchange: true}, Loaded: persistence.Loaded{Found: true, Version: 2, Token: "opaque", V2: state.V2{State: base, Revision: 4}}, Initial: state.V2{State: base, Revision: 1}, NextDigest: "sha256:next", Key: "slot"}}
}

func TestDispatchUsesPlannerOnlyForNewCommands(t *testing.T) {
	firstCommand := command(1, 1)
	first := Dispatch(transport.NewState("match"), firstCommand)
	if !first.OK || !first.Response.Accepted || first.Response.Revision != 5 || first.Response.Event0.Kind != transport.AcceptedEvent {
		t.Fatalf("first=%#v", first)
	}
	duplicate := Dispatch(first.State, firstCommand)
	if !duplicate.OK || !duplicate.Duplicate || duplicate.Response.Revision != 5 || !reflect.DeepEqual(duplicate.State, first.State) {
		t.Fatalf("duplicate=%#v", duplicate)
	}
	rejectedCommand := command(2, 2)
	rejectedCommand.Payload.Loaded.Version = 3
	rejected := Dispatch(first.State, rejectedCommand)
	if !rejected.OK || rejected.Response.Accepted || rejected.Response.Code != 71 || rejected.State.NextCommand != 3 || rejected.Response.Event0.Kind != transport.RejectedEvent {
		t.Fatalf("rejected=%#v", rejected)
	}
}

func TestTransportRejectionDoesNotConsumeSequence(t *testing.T) {
	initial := transport.NewState("match")
	gap := Dispatch(initial, command(2, 2))
	if gap.OK || gap.Error != transport.OutOfOrder || !reflect.DeepEqual(gap.State, initial) {
		t.Fatalf("gap=%#v", gap)
	}
	valid := Dispatch(gap.State, command(1, 1))
	if !valid.OK || valid.State.NextCommand != 2 {
		t.Fatalf("valid=%#v", valid)
	}
}
