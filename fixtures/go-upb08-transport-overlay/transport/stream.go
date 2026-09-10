package transport

import (
	"unicode/utf8"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
)

const (
	MaximumCommands       = 256
	MaximumEvents         = 1024
	MaximumEventsPerReply = 4
	MaximumStreamBytes    = 128
	MaximumKindBytes      = 128
	// Encoded byte limits are enforced by the host codec before Dispatch.
	MaximumEncodedPayloadBytes  = 3072
	MaximumEncodedResponseBytes = 3072
)

type ErrorCode int64

const (
	InvalidState ErrorCode = 80 + iota
	InvalidStream
	InvalidSequence
	InvalidCorrelation
	InvalidKind
	InvalidPayload
	PayloadTooLarge
	SequenceConflict
	OutOfOrder
	CorrelationReused
	CommandLimit
	InvalidDomainOutcome
	DispatchFailed
)

type Correlation [16]byte

type PlannerPayload struct {
	Grants     persistence.Grants
	Loaded     persistence.Loaded
	Initial    state.V2
	NextDigest string
	Key        string
}

type Command struct {
	Stream      string
	Sequence    int64
	Correlation Correlation
	Kind        string
	Payload     PlannerPayload
	Digest      [32]byte
}

type DomainEvent struct {
	Kind    string
	Payload persistence.Plan
}

type DomainOutcome struct {
	Accepted bool
	Code     int64
	Receipt  persistence.Plan
	Events   []DomainEvent
}

type Handler interface {
	Handle(PlannerPayload) (DomainOutcome, error)
}

type Event struct {
	Stream          string
	Sequence        int64
	CommandSequence int64
	Ordinal         int64
	Correlation     Correlation
	Kind            string
	Payload         persistence.Plan
}

type Response struct {
	Correlation Correlation
	Sequence    int64
	Accepted    bool
	Code        int64
	Receipt     persistence.Plan
	Events      []Event
}

type LedgerEntry struct {
	Stream      string
	Sequence    int64
	Correlation Correlation
	Kind        string
	Payload     PlannerPayload
	Digest      [32]byte
	Response    Response
}

type State struct {
	Stream      string
	NextCommand int64
	NextEvent   int64
	Ledger      []LedgerEntry
}

type Result struct {
	OK        bool
	Duplicate bool
	State     State
	Response  Response
	Error     ErrorCode
}

type ReplayResult struct {
	OK        bool
	State     State
	Responses []Response
	Error     ErrorCode
}

func NewState(stream string) State {
	return State{Stream: stream, NextCommand: 1, NextEvent: 1, Ledger: []LedgerEntry{}}
}

func NewCommand(stream string, sequence int64, correlation Correlation, kind string, payload PlannerPayload, digest [32]byte) Command {
	return Command{Stream: stream, Sequence: sequence, Correlation: correlation, Kind: kind, Payload: clonePayload(payload), Digest: digest}
}

func Dispatch(current State, command Command, handler Handler) Result {
	if !validState(current) {
		return failure(current, InvalidState)
	}
	if command.Stream != current.Stream || !validText(command.Stream, MaximumStreamBytes) {
		return failure(current, InvalidStream)
	}
	if command.Sequence < 1 || command.Sequence > MaximumCommands {
		return failure(current, InvalidSequence)
	}
	if zeroCorrelation(command.Correlation) {
		return failure(current, InvalidCorrelation)
	}
	if !validText(command.Kind, MaximumKindBytes) {
		return failure(current, InvalidKind)
	}
	if command.Digest == [32]byte{} {
		return failure(current, InvalidPayload)
	}
	if command.Sequence < current.NextCommand {
		entry := current.Ledger[command.Sequence-1]
		if !sameCommand(entry, command) {
			return failure(current, SequenceConflict)
		}
		return Result{OK: true, Duplicate: true, State: cloneState(current), Response: cloneResponse(entry.Response)}
	}
	if command.Sequence > current.NextCommand {
		return failure(current, OutOfOrder)
	}
	if len(current.Ledger) >= MaximumCommands {
		return failure(current, CommandLimit)
	}
	for _, entry := range current.Ledger {
		if entry.Correlation == command.Correlation {
			return failure(current, CorrelationReused)
		}
	}
	if handler == nil {
		return failure(current, DispatchFailed)
	}
	outcome, err := handler.Handle(clonePayload(command.Payload))
	if err != nil {
		return failure(current, DispatchFailed)
	}
	if !validOutcome(outcome) || current.NextEvent+int64(len(outcome.Events))-1 > MaximumEvents {
		return failure(current, InvalidDomainOutcome)
	}
	response := Response{Correlation: command.Correlation, Sequence: command.Sequence, Accepted: outcome.Accepted, Code: outcome.Code, Receipt: clonePlan(outcome.Receipt), Events: make([]Event, len(outcome.Events))}
	for index, event := range outcome.Events {
		response.Events[index] = Event{Stream: current.Stream, Sequence: current.NextEvent + int64(index), CommandSequence: command.Sequence, Ordinal: int64(index), Correlation: command.Correlation, Kind: event.Kind, Payload: clonePlan(event.Payload)}
	}
	next := cloneState(current)
	next.NextCommand++
	next.NextEvent += int64(len(response.Events))
	next.Ledger = append(next.Ledger, LedgerEntry{Stream: command.Stream, Sequence: command.Sequence, Correlation: command.Correlation, Kind: command.Kind, Payload: clonePayload(command.Payload), Digest: command.Digest, Response: cloneResponse(response)})
	return Result{OK: true, State: next, Response: response}
}

func Replay(initial State, commands []Command, handler Handler) ReplayResult {
	current := cloneState(initial)
	responses := make([]Response, 0, len(commands))
	for _, command := range commands {
		if command.Sequence != current.NextCommand {
			return ReplayResult{State: current, Responses: responses, Error: OutOfOrder}
		}
		result := Dispatch(current, command, handler)
		if !result.OK || result.Duplicate {
			return ReplayResult{State: current, Responses: responses, Error: result.Error}
		}
		current = result.State
		responses = append(responses, cloneResponse(result.Response))
	}
	return ReplayResult{OK: true, State: current, Responses: responses}
}

func Reject(current State, code ErrorCode) Result { return failure(current, code) }

func validState(value State) bool {
	if !validText(value.Stream, MaximumStreamBytes) || value.NextCommand != int64(len(value.Ledger))+1 || value.NextCommand < 1 || value.NextCommand > MaximumCommands+1 || value.NextEvent < 1 || value.NextEvent > MaximumEvents+1 {
		return false
	}
	nextEvent := int64(1)
	for index, entry := range value.Ledger {
		if entry.Stream != value.Stream || entry.Sequence != int64(index+1) || zeroCorrelation(entry.Correlation) || duplicateCorrelation(value.Ledger, index) || !validText(entry.Kind, MaximumKindBytes) || entry.Digest == [32]byte{} || entry.Response.Correlation != entry.Correlation || entry.Response.Sequence != entry.Sequence || entry.Response.Accepted && entry.Response.Code != 0 || !entry.Response.Accepted && entry.Response.Code == 0 || len(entry.Response.Events) > MaximumEventsPerReply {
			return false
		}
		for ordinal, event := range entry.Response.Events {
			if event.Stream != value.Stream || event.Sequence != nextEvent || event.CommandSequence != entry.Sequence || event.Ordinal != int64(ordinal) || event.Correlation != entry.Correlation || !validText(event.Kind, MaximumKindBytes) {
				return false
			}
			nextEvent++
		}
	}
	return value.NextEvent == nextEvent
}

func validOutcome(value DomainOutcome) bool {
	if len(value.Events) > MaximumEventsPerReply {
		return false
	}
	if value.Accepted && value.Code != 0 || !value.Accepted && value.Code == 0 {
		return false
	}
	for _, event := range value.Events {
		if !validText(event.Kind, MaximumKindBytes) {
			return false
		}
	}
	return true
}

func validText(value string, limit int) bool {
	return value != "" && len(value) <= limit && utf8.ValidString(value)
}

func zeroCorrelation(value Correlation) bool {
	return value == Correlation{}
}

func sameCommand(entry LedgerEntry, command Command) bool {
	return entry.Stream == command.Stream && entry.Sequence == command.Sequence && entry.Correlation == command.Correlation && entry.Kind == command.Kind && entry.Digest == command.Digest && equalPayload(entry.Payload, command.Payload)
}

func failure(current State, code ErrorCode) Result {
	return Result{State: cloneState(current), Error: code}
}

func cloneState(value State) State {
	out := State{Stream: value.Stream, NextCommand: value.NextCommand, NextEvent: value.NextEvent, Ledger: make([]LedgerEntry, len(value.Ledger))}
	for index, entry := range value.Ledger {
		out.Ledger[index] = entry
		out.Ledger[index].Payload = clonePayload(entry.Payload)
		out.Ledger[index].Response = cloneResponse(entry.Response)
	}
	return out
}

func cloneResponse(value Response) Response {
	out := value
	out.Receipt = clonePlan(value.Receipt)
	out.Events = make([]Event, len(value.Events))
	for index, event := range value.Events {
		out.Events[index] = event
		out.Events[index].Payload = clonePlan(event.Payload)
	}
	return out
}

func duplicateCorrelation(entries []LedgerEntry, index int) bool {
	for prior := 0; prior < index; prior++ {
		if entries[prior].Correlation == entries[index].Correlation {
			return true
		}
	}
	return false
}

func equalPayload(left, right PlannerPayload) bool {
	if left.Grants != right.Grants || left.Loaded.Found != right.Loaded.Found || left.Loaded.Version != right.Loaded.Version || left.Loaded.Token != right.Loaded.Token || left.Loaded.Digest != right.Loaded.Digest || left.NextDigest != right.NextDigest || left.Key != right.Key || !equalV1(left.Loaded.V1, right.Loaded.V1) || !equalV2(left.Loaded.V2, right.Loaded.V2) || !equalV2(left.Initial, right.Initial) {
		return false
	}
	return true
}

func equalV1(left, right state.V1) bool { return equalApplication(left.State, right.State) }
func equalV2(left, right state.V2) bool {
	return left.Revision == right.Revision && equalApplication(left.State, right.State)
}

func equalApplication(left, right application.State) bool {
	if left.Name != right.Name || len(left.Values) != len(right.Values) || len(left.Counters) != len(right.Counters) {
		return false
	}
	for index := range left.Values {
		if left.Values[index] != right.Values[index] {
			return false
		}
	}
	for key, value := range left.Counters {
		other, ok := right.Counters[key]
		if !ok || other != value {
			return false
		}
	}
	return true
}

func clonePayload(value PlannerPayload) PlannerPayload {
	out := value
	out.Loaded.V1.State = cloneApplication(value.Loaded.V1.State)
	out.Loaded.V2.State = cloneApplication(value.Loaded.V2.State)
	out.Initial.State = cloneApplication(value.Initial.State)
	return out
}

func cloneApplication(value application.State) application.State {
	out := value
	out.Values = append([]int64(nil), value.Values...)
	if value.Counters != nil {
		out.Counters = map[int64]int64{}
		for key, item := range value.Counters {
			out.Counters[key] = item
		}
	}
	return out
}

func clonePlan(value persistence.Plan) persistence.Plan {
	out := value
	out.Value.Value.State = cloneApplication(value.Value.Value.State)
	out.Value.Load.Value.State = cloneApplication(value.Value.Load.Value.State)
	out.Value.CompareExchange.Value.State = cloneApplication(value.Value.CompareExchange.Value.State)
	return out
}
