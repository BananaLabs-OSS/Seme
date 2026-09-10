package streamservice

import (
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/transport"
)

const PlannerCommand = "planner.update.v1"

type PlannerAdapter struct{}

func (PlannerAdapter) Handle(payload transport.PlannerPayload) (transport.DomainOutcome, error) {
	plan := persistence.BuildPlan(payload.Grants, payload.Loaded, payload.Initial, payload.NextDigest, payload.Key)
	if !plan.Ok {
		return transport.DomainOutcome{Code: plan.Error, Receipt: plan, Events: []transport.DomainEvent{{Kind: "planner.rejected.v1", Payload: plan}}}, nil
	}
	return transport.DomainOutcome{Accepted: true, Receipt: plan, Events: []transport.DomainEvent{{Kind: "planner.accepted.v1", Payload: plan}}}, nil
}

func Dispatch(current transport.State, command transport.Command) transport.Result {
	if command.Kind != PlannerCommand {
		return transport.Reject(current, transport.InvalidKind)
	}
	return transport.Dispatch(current, command, PlannerAdapter{})
}
