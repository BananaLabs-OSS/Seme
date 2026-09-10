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
	Transport          transport.State
	ClockSequences     []int64
	ClockMilliseconds  []int64
	RandomBeforeValues []int64
	RandomBeforeDraws  []int64
	RandomAfterValues  []int64
	RandomAfterDraws   []int64
	Draws              []int64
}

type ControlledResult struct {
	OK        bool
	Duplicate bool
	State     ControlledState
	Response  transport.Response
	Error     int64
}

func NewControlledState(stream string) ControlledState {
	return ControlledState{Transport: transport.NewState(stream), ClockSequences: []int64{}, ClockMilliseconds: []int64{}, RandomBeforeValues: []int64{}, RandomBeforeDraws: []int64{}, RandomAfterValues: []int64{}, RandomAfterDraws: []int64{}, Draws: []int64{}}
}

func DispatchControlled(current ControlledState, command ControlledCommand) ControlledResult {
	result := transitionControlled(current, command)
	if result.OK && !result.Duplicate {
		log.Print(result.Response.Accepted)
	}
	return result
}

// ReplayControlled is the effect-free replay transition. A replay driver folds
// it over the retained ordered commands and their explicit clock/random input;
// unlike live dispatch it never requests physical effect delivery.
func ReplayControlled(current ControlledState, command ControlledCommand) ControlledResult {
	return transitionControlled(current, command)
}

func transitionControlled(current ControlledState, command ControlledCommand) ControlledResult {
	classification := transport.Classify(current.Transport, command.Command)
	if classification.Error != 0 {
		return controlledFailure(current, classification.Error)
	}
	if classification.Kind == transport.ClassificationCached {
		index := classification.Index
		if index < 0 || int64(len(current.ClockMilliseconds)) <= index || current.ClockSequences[index] != command.Clock.Sequence || current.ClockMilliseconds[index] != command.Clock.UnixMilliseconds || current.RandomBeforeValues[index] != command.Random.Value || current.RandomBeforeDraws[index] != command.Random.Draws {
			return controlledFailure(current, transport.SequenceConflict)
		}
		return ControlledResult{OK: true, Duplicate: true, State: cloneControlled(current), Response: classification.Response}
	}
	// Every condition that could prevent the transport commit is checked before
	// consuming the explicit clock/random values or requesting the log effect.
	count := len(current.Draws)
	validRandom := controlled.ValidState(command.Random)
	if count == 0 {
		validRandom = controlled.ValidSeed(command.Random)
	}
	if !validControlledState(current) || !controlled.ValidClock(command.Clock) || !validRandom || command.Clock.Sequence != command.Command.Sequence || command.Clock.Sequence != int64(count)+1 || command.Random.Draws != int64(count) || command.Random.Draws == 256 || 512 < len(current.Transport.PayloadWords)+len(command.Command.PayloadWords) || 1024 < current.Transport.NextEvent {
		return controlledFailure(current, InvalidControlledInput)
	}
	if 0 < count && (command.Clock.UnixMilliseconds < current.ClockMilliseconds[count-1] || command.Random.Value != current.RandomAfterValues[count-1] || command.Random.Draws != current.RandomAfterDraws[count-1]) {
		return controlledFailure(current, InvalidControlledInput)
	}
	draw := controlled.Next(command.Random)
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
	next.ClockSequences = append(next.ClockSequences, command.Clock.Sequence)
	next.ClockMilliseconds = append(next.ClockMilliseconds, command.Clock.UnixMilliseconds)
	next.RandomBeforeValues = append(next.RandomBeforeValues, draw.Before.Value)
	next.RandomBeforeDraws = append(next.RandomBeforeDraws, draw.Before.Draws)
	next.RandomAfterValues = append(next.RandomAfterValues, draw.After.Value)
	next.RandomAfterDraws = append(next.RandomAfterDraws, draw.After.Draws)
	next.Draws = append(next.Draws, draw.Value)
	return ControlledResult{OK: true, State: next, Response: committed.Response}
}

func validControlledState(value ControlledState) bool {
	count := len(value.Transport.CorrelationHigh)
	if !transport.ValidState(value.Transport) || len(value.ClockSequences) != count || len(value.ClockMilliseconds) != count || len(value.RandomBeforeValues) != count || len(value.RandomBeforeDraws) != count || len(value.RandomAfterValues) != count || len(value.RandomAfterDraws) != count || len(value.Draws) != count {
		return false
	}
	for index := 0; index < count; index++ {
		if value.ClockSequences[index] != int64(index)+1 || value.RandomBeforeDraws[index] != int64(index) || value.RandomAfterDraws[index] != int64(index)+1 || value.RandomAfterValues[index] != value.Draws[index] || 0 < index && (value.ClockMilliseconds[index] < value.ClockMilliseconds[index-1] || value.RandomBeforeValues[index] != value.RandomAfterValues[index-1] || value.RandomBeforeDraws[index] != value.RandomAfterDraws[index-1]) {
			return false
		}
	}
	return true
}
func cloneControlled(value ControlledState) ControlledState {
	return ControlledState{Transport: transport.CloneState(value.Transport), ClockSequences: slices.Clone(value.ClockSequences), ClockMilliseconds: slices.Clone(value.ClockMilliseconds), RandomBeforeValues: slices.Clone(value.RandomBeforeValues), RandomBeforeDraws: slices.Clone(value.RandomBeforeDraws), RandomAfterValues: slices.Clone(value.RandomAfterValues), RandomAfterDraws: slices.Clone(value.RandomAfterDraws), Draws: slices.Clone(value.Draws)}
}
func controlledFailure(current ControlledState, code int64) ControlledResult {
	return ControlledResult{State: cloneControlled(current), Error: code}
}
