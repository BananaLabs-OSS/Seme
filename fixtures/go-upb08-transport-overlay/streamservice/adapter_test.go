package streamservice

import (
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
	"example.test/go-uab-11/transport"
)

func id(value byte) transport.Correlation {
	var result transport.Correlation
	result[15] = value
	return result
}

func plannerPayload() transport.PlannerPayload {
	base := application.State{Name: "pilot", Counters: map[int64]int64{}}
	return transport.PlannerPayload{Grants: persistence.Grants{Read: true, CompareExchange: true}, Loaded: persistence.Loaded{Found: true, Version: 2, Token: "opaque", V2: state.V2{State: base, Revision: 4}}, Initial: state.V2{State: base, Revision: 1}, NextDigest: "sha256:next", Key: "slot"}
}

func TestAdapterUsesDurablePlannerForAcceptedAndRejectedCommands(t *testing.T) {
	var firstDigest [32]byte
	firstDigest[0] = 1
	acceptedCommand := transport.NewCommand("match", 1, id(1), PlannerCommand, plannerPayload(), firstDigest)
	accepted := Dispatch(transport.NewState("match"), acceptedCommand)
	if !accepted.OK || !accepted.Response.Accepted || accepted.Response.Code != 0 || len(accepted.Response.Events) != 1 || accepted.Response.Events[0].Kind != "planner.accepted.v1" {
		t.Fatalf("accepted=%#v", accepted)
	}
	plan := accepted.Response.Receipt
	if !plan.Ok || plan.Value.Value.Revision != 5 {
		t.Fatalf("plan=%#v", plan)
	}
	rejectedPayload := plannerPayload()
	rejectedPayload.Loaded.Version = 3
	var secondDigest [32]byte
	secondDigest[0] = 2
	rejectedCommand := transport.NewCommand("match", 2, id(2), PlannerCommand, rejectedPayload, secondDigest)
	rejected := Dispatch(accepted.State, rejectedCommand)
	if !rejected.OK || rejected.Response.Accepted || rejected.Response.Code != 71 || rejected.State.NextCommand != 3 || rejected.Response.Events[0].Kind != "planner.rejected.v1" {
		t.Fatalf("rejected=%#v", rejected)
	}
}

func TestAdapterRejectsUnknownKindBeforePlanner(t *testing.T) {
	var digest [32]byte
	digest[0] = 1
	command := transport.NewCommand("match", 1, id(1), "unknown", plannerPayload(), digest)
	result := Dispatch(transport.NewState("match"), command)
	if result.OK || result.Error != transport.InvalidKind || result.State.NextCommand != 1 {
		t.Fatalf("result=%#v", result)
	}
}
