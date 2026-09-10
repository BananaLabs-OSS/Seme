package streamservice

import (
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/transport"
)

func Dispatch(current transport.State, command transport.Command) transport.Result {
	classification := transport.Classify(current, command)
	if !transport.EqualI64(classification.Error, 0) {
		return transport.Failure(current, classification.Error)
	}
	if transport.EqualI64(classification.Kind, 2) {
		return transport.Cached(current, classification.Response)
	}
	plan := persistence.BuildPlan(command.Payload.Grants, command.Payload.Loaded, command.Payload.Initial, command.Payload.NextDigest, command.Payload.Key)
	if !plan.Ok {
		return transport.Commit(current, command, false, plan.Error, 0, 1)
	}
	return transport.Commit(current, command, true, 0, plan.Value.Value.Revision, 1)
}
