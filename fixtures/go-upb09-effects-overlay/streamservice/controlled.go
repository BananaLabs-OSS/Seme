package streamservice

import (
	"log"
	"slices"

	"example.test/go-uab-11/controlled"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/transport"
)

const InvalidControlledInput int64 = 90

type ControlledCommand struct {
	Command transport.Command
	Clock   controlled.ClockSample
	Random  controlled.RandomState
}

type ControlledState struct {
	Transport         transport.State
	ClockMilliseconds []int64
	RandomBefore      []int64
	RandomAfter       []int64
	Draws             []int64
}

type ControlledResult struct {
	OK        bool
	Duplicate bool
	State     ControlledState
	Response  transport.Response
	Error     int64
}

func NewControlledState(stream string) ControlledState {
	return ControlledState{Transport: transport.NewState(stream), ClockMilliseconds: []int64{}, RandomBefore: []int64{}, RandomAfter: []int64{}, Draws: []int64{}}
}

func DispatchControlled(current ControlledState, command ControlledCommand) ControlledResult {
	classification := transport.Classify(current.Transport, command.Command)
	if classification.Error != 0 {
		return controlledFailure(current, classification.Error)
	}
	if classification.Kind == transport.ClassificationCached {
		index := classification.Index
		if index < 0 || int64(len(current.ClockMilliseconds)) <= index || current.ClockMilliseconds[index] != command.Clock.UnixMilliseconds || current.RandomBefore[index] != command.Random.Value {
			return controlledFailure(current, transport.SequenceConflict)
		}
		return ControlledResult{OK: true, Duplicate: true, State: cloneControlled(current), Response: classification.Response}
	}
	// Every condition that could prevent the transport commit is checked before
	// consuming the explicit clock/random values or requesting the log effect.
	if !validControlledState(current) || !controlled.ValidClock(command.Clock) || !controlled.ValidRandom(command.Random) || 512 < len(current.Transport.PayloadWords)+len(command.Command.PayloadWords) || 1024 < current.Transport.NextEvent {
		return controlledFailure(current, InvalidControlledInput)
	}
	count := len(current.Draws)
	if 0 < count && (command.Clock.UnixMilliseconds < current.ClockMilliseconds[count-1] || command.Random.Value != current.RandomAfter[count-1]) {
		return controlledFailure(current, InvalidControlledInput)
	}
	draw := controlled.Next(command.Random)
	log.Print(draw.Value >= 0)
	plan := persistence.BuildPlan(command.Command.Payload.Grants, command.Command.Payload.Loaded, command.Command.Payload.Initial, command.Command.Payload.NextDigest, command.Command.Payload.Key)
	accepted, code, revision := true, int64(0), int64(0)
	if plan.Ok {
		revision = plan.Value.Value.Revision
	} else {
		accepted, code = false, plan.Error
	}
	committed := transport.Commit(current.Transport, command.Command, accepted, code, revision, 1)
	if !committed.OK || committed.Duplicate {
		return controlledFailure(current, committed.Error)
	}
	next := cloneControlled(current)
	next.Transport = committed.State
	next.ClockMilliseconds = append(next.ClockMilliseconds, command.Clock.UnixMilliseconds)
	next.RandomBefore = append(next.RandomBefore, draw.Before.Value)
	next.RandomAfter = append(next.RandomAfter, draw.After.Value)
	next.Draws = append(next.Draws, draw.Value)
	return ControlledResult{OK: true, State: next, Response: committed.Response}
}

func validControlledState(value ControlledState) bool {
	count := len(value.Transport.CorrelationHigh)
	if !transport.ValidState(value.Transport) || len(value.ClockMilliseconds) != count || len(value.RandomBefore) != count || len(value.RandomAfter) != count || len(value.Draws) != count {
		return false
	}
	for index := 0; index < count; index++ {
		if value.RandomAfter[index] != value.Draws[index] || 0 < index && (value.ClockMilliseconds[index] < value.ClockMilliseconds[index-1] || value.RandomBefore[index] != value.RandomAfter[index-1]) {
			return false
		}
	}
	return true
}
func cloneControlled(value ControlledState) ControlledState {
	return ControlledState{Transport: transport.CloneState(value.Transport), ClockMilliseconds: slices.Clone(value.ClockMilliseconds), RandomBefore: slices.Clone(value.RandomBefore), RandomAfter: slices.Clone(value.RandomAfter), Draws: slices.Clone(value.Draws)}
}
func controlledFailure(current ControlledState, code int64) ControlledResult {
	return ControlledResult{State: cloneControlled(current), Error: code}
}
