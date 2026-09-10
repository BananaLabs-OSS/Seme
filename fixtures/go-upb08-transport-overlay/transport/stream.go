package transport

import (
	"bytes"
	"slices"

	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
)

const (
	MaximumCommands       = 256
	MaximumEvents         = 1024
	MaximumEventsPerReply = 4
	PlannerCommand        = 1
	AcceptedEvent         = 1
	RejectedEvent         = 2
	ClassificationNew     = 1
	ClassificationCached  = 2
	// The host frame codec enforces byte and UTF-8 limits before this typed entry.
	MaximumStreamBytes         = 128
	MaximumEncodedPayloadBytes = 3072
	MaximumEncodedFrameBytes   = 4096
)

type ErrorCode = int64

const (
	InvalidState ErrorCode = 80 + iota
	InvalidStream
	InvalidSequence
	InvalidCorrelation
	InvalidKind
	InvalidPayload
	SequenceConflict
	OutOfOrder
	CorrelationReused
	InvalidDomainOutcome
)

// Correlation and Digest are exact host-authenticated words. The frame adapter
// maps their 16 and 32 bytes respectively without interpreting them.
type Correlation struct{ High, Low int64 }
type Digest struct{ A, B, C, D int64 }

type PlannerPayload struct {
	Grants     persistence.Grants
	Loaded     persistence.Loaded
	Initial    state.V2
	NextDigest string
	Key        string
}

type Command struct {
	Stream           string
	Sequence         int64
	Correlation      Correlation
	Kind             int64
	Payload          PlannerPayload
	CanonicalPayload []byte
	Digest           Digest
}

type Event struct {
	Sequence        int64
	CommandSequence int64
	Ordinal         int64
	CorrelationHigh int64
	CorrelationLow  int64
	Kind            int64
	Code            int64
	Revision        int64
}

type Response struct {
	CorrelationHigh int64
	CorrelationLow  int64
	Sequence        int64
	Accepted        bool
	Code            int64
	Revision        int64
	EventCount      int64
	Event0          Event
	Event1          Event
	Event2          Event
	Event3          Event
}

// State is a bounded columnar ledger. Every column has exactly one item per
// accepted unique command, so cached replies require no domain re-execution.
type State struct {
	Stream          string
	NextCommand     int64
	NextEvent       int64
	CorrelationHigh []int64
	CorrelationLow  []int64
	DigestA         []int64
	DigestB         []int64
	DigestC         []int64
	DigestD         []int64
	PayloadBytes    []byte
	PayloadStarts   []int64
	PayloadLengths  []int64
	Kinds           []int64
	Accepted        []int64
	Codes           []int64
	Revisions       []int64
	EventCounts     []int64
	EventStarts     []int64
}

type Classification struct {
	Kind     int64
	Index    int64
	Response Response
	Error    int64
}

type Result struct {
	OK        bool
	Duplicate bool
	State     State
	Response  Response
	Error     int64
}

func NewState(stream string) State {
	return State{Stream: stream, NextCommand: 1, NextEvent: 1, CorrelationHigh: []int64{}, CorrelationLow: []int64{}, DigestA: []int64{}, DigestB: []int64{}, DigestC: []int64{}, DigestD: []int64{}, PayloadBytes: []byte{}, PayloadStarts: []int64{}, PayloadLengths: []int64{}, Kinds: []int64{}, Accepted: []int64{}, Codes: []int64{}, Revisions: []int64{}, EventCounts: []int64{}, EventStarts: []int64{}}
}

func Classify(current State, command Command) Classification {
	if !ValidState(current) {
		return Classification{Error: 80}
	}
	if command.Stream == "" || command.Stream != current.Stream {
		return Classification{Error: 81}
	}
	if command.Sequence < 1 || 256 < command.Sequence {
		return Classification{Error: 82}
	}
	if EqualI64(command.Correlation.High, 0) && EqualI64(command.Correlation.Low, 0) {
		return Classification{Error: 83}
	}
	if !EqualI64(command.Kind, 1) {
		return Classification{Error: 84}
	}
	if ZeroDigest(command.Digest) || len(command.CanonicalPayload) == 0 || 3072 < len(command.CanonicalPayload) {
		return Classification{Error: 85}
	}
	if command.Sequence < current.NextCommand {
		index := command.Sequence - 1
		start := current.PayloadStarts[index]
		length := current.PayloadLengths[index]
		if !EqualI64(current.CorrelationHigh[index], command.Correlation.High) || !EqualI64(current.CorrelationLow[index], command.Correlation.Low) || !EqualI64(current.Kinds[index], command.Kind) || !EqualDigestAt(current, index, command.Digest) || !EqualI64(length, int64(len(command.CanonicalPayload))) || !bytes.Equal(current.PayloadBytes[start:start+length], command.CanonicalPayload) {
			return Classification{Error: 86}
		}
		return Classification{Kind: 2, Index: index, Response: ResponseAt(current, index)}
	}
	if current.NextCommand < command.Sequence {
		return Classification{Error: 87}
	}
	index := int64(0)
	reused := false
	for index < int64(len(current.CorrelationHigh)) {
		if EqualI64(current.CorrelationHigh[index], command.Correlation.High) && EqualI64(current.CorrelationLow[index], command.Correlation.Low) {
			reused = true
		}
		index = index + 1
	}
	if reused {
		return Classification{Error: 88}
	}
	return Classification{Kind: 1, Index: current.NextCommand - 1}
}

func Commit(current State, command Command, accepted bool, code int64, revision int64, eventCount int64) Result {
	classification := Classify(current, command)
	if !EqualI64(classification.Error, 0) {
		return Failure(current, classification.Error)
	}
	if EqualI64(classification.Kind, 2) {
		return Cached(current, classification.Response)
	}
	if eventCount < 0 || 4 < eventCount || 1024 < current.NextEvent+eventCount-1 || accepted && !EqualI64(code, 0) || !accepted && EqualI64(code, 0) {
		return Failure(current, 89)
	}
	if 4096 < len(current.PayloadBytes)+len(command.CanonicalPayload) {
		return Failure(current, 85)
	}
	acceptedValue := int64(0)
	if accepted {
		acceptedValue = 1
	}
	next := State{
		Stream: current.Stream, NextCommand: current.NextCommand + 1, NextEvent: current.NextEvent + eventCount,
		CorrelationHigh: append(slices.Clone(current.CorrelationHigh), command.Correlation.High), CorrelationLow: append(slices.Clone(current.CorrelationLow), command.Correlation.Low),
		DigestA: append(slices.Clone(current.DigestA), command.Digest.A), DigestB: append(slices.Clone(current.DigestB), command.Digest.B), DigestC: append(slices.Clone(current.DigestC), command.Digest.C), DigestD: append(slices.Clone(current.DigestD), command.Digest.D),
		PayloadBytes: append(slices.Clone(current.PayloadBytes), command.CanonicalPayload...), PayloadStarts: append(slices.Clone(current.PayloadStarts), int64(len(current.PayloadBytes))), PayloadLengths: append(slices.Clone(current.PayloadLengths), int64(len(command.CanonicalPayload))),
		Kinds: append(slices.Clone(current.Kinds), command.Kind), Accepted: append(slices.Clone(current.Accepted), acceptedValue), Codes: append(slices.Clone(current.Codes), code), Revisions: append(slices.Clone(current.Revisions), revision),
		EventCounts: append(slices.Clone(current.EventCounts), eventCount), EventStarts: append(slices.Clone(current.EventStarts), current.NextEvent),
	}
	return Result{OK: true, State: next, Response: ResponseAt(next, command.Sequence-1)}
}

func Failure(current State, code int64) Result {
	return Result{State: CloneState(current), Error: code}
}
func Cached(current State, response Response) Result {
	return Result{OK: true, Duplicate: true, State: CloneState(current), Response: response}
}

func ValidState(value State) bool {
	count := int64(len(value.CorrelationHigh))
	if value.Stream == "" || 4096 < len(value.PayloadBytes) || !EqualI64(value.NextCommand, count+1) || value.NextCommand < 1 || 257 < value.NextCommand || value.NextEvent < 1 || 1025 < value.NextEvent || !EqualI64(int64(len(value.CorrelationLow)), count) || !EqualI64(int64(len(value.DigestA)), count) || !EqualI64(int64(len(value.DigestB)), count) || !EqualI64(int64(len(value.DigestC)), count) || !EqualI64(int64(len(value.DigestD)), count) || !EqualI64(int64(len(value.PayloadStarts)), count) || !EqualI64(int64(len(value.PayloadLengths)), count) || !EqualI64(int64(len(value.Kinds)), count) || !EqualI64(int64(len(value.Accepted)), count) || !EqualI64(int64(len(value.Codes)), count) || !EqualI64(int64(len(value.Revisions)), count) || !EqualI64(int64(len(value.EventCounts)), count) || !EqualI64(int64(len(value.EventStarts)), count) {
		return false
	}
	nextEvent := int64(1)
	payloadEnd := int64(0)
	index := int64(0)
	valid := true
	for index < count {
		if !EqualI64(value.PayloadStarts[index], payloadEnd) || value.PayloadLengths[index] < 1 || 3072 < value.PayloadLengths[index] || int64(len(value.PayloadBytes)) < value.PayloadStarts[index]+value.PayloadLengths[index] || !EqualI64(value.EventStarts[index], nextEvent) || value.EventCounts[index] < 0 || 4 < value.EventCounts[index] || value.Accepted[index] < 0 || 1 < value.Accepted[index] || EqualI64(value.Accepted[index], 1) && !EqualI64(value.Codes[index], 0) || EqualI64(value.Accepted[index], 0) && EqualI64(value.Codes[index], 0) {
			valid = false
		}
		prior := int64(0)
		for prior < index {
			if EqualI64(value.CorrelationHigh[prior], value.CorrelationHigh[index]) && EqualI64(value.CorrelationLow[prior], value.CorrelationLow[index]) {
				valid = false
			}
			prior = prior + 1
		}
		nextEvent = nextEvent + value.EventCounts[index]
		payloadEnd = payloadEnd + value.PayloadLengths[index]
		index = index + 1
	}
	return valid && EqualI64(value.NextEvent, nextEvent) && EqualI64(payloadEnd, int64(len(value.PayloadBytes)))
}

func ResponseAt(value State, index int64) Response {
	accepted := EqualI64(value.Accepted[index], 1)
	count := value.EventCounts[index]
	start := value.EventStarts[index]
	kind := int64(2)
	if accepted {
		kind = 1
	}
	event0 := Event{Sequence: 0}
	event1 := Event{Sequence: 0}
	event2 := Event{Sequence: 0}
	event3 := Event{Sequence: 0}
	if 0 < count {
		event0 = MakeEvent(start, index+1, 0, value.CorrelationHigh[index], value.CorrelationLow[index], kind, value.Codes[index], value.Revisions[index])
	}
	if 1 < count {
		event1 = MakeEvent(start+1, index+1, 1, value.CorrelationHigh[index], value.CorrelationLow[index], kind, value.Codes[index], value.Revisions[index])
	}
	if 2 < count {
		event2 = MakeEvent(start+2, index+1, 2, value.CorrelationHigh[index], value.CorrelationLow[index], kind, value.Codes[index], value.Revisions[index])
	}
	if 3 < count {
		event3 = MakeEvent(start+3, index+1, 3, value.CorrelationHigh[index], value.CorrelationLow[index], kind, value.Codes[index], value.Revisions[index])
	}
	return Response{CorrelationHigh: value.CorrelationHigh[index], CorrelationLow: value.CorrelationLow[index], Sequence: index + 1, Accepted: accepted, Code: value.Codes[index], Revision: value.Revisions[index], EventCount: count, Event0: event0, Event1: event1, Event2: event2, Event3: event3}
}

func MakeEvent(sequence int64, commandSequence int64, ordinal int64, correlationHigh int64, correlationLow int64, kind int64, code int64, revision int64) Event {
	return Event{Sequence: sequence, CommandSequence: commandSequence, Ordinal: ordinal, CorrelationHigh: correlationHigh, CorrelationLow: correlationLow, Kind: kind, Code: code, Revision: revision}
}

func CloneState(value State) State {
	return State{Stream: value.Stream, NextCommand: value.NextCommand, NextEvent: value.NextEvent, CorrelationHigh: slices.Clone(value.CorrelationHigh), CorrelationLow: slices.Clone(value.CorrelationLow), DigestA: slices.Clone(value.DigestA), DigestB: slices.Clone(value.DigestB), DigestC: slices.Clone(value.DigestC), DigestD: slices.Clone(value.DigestD), PayloadBytes: slices.Clone(value.PayloadBytes), PayloadStarts: slices.Clone(value.PayloadStarts), PayloadLengths: slices.Clone(value.PayloadLengths), Kinds: slices.Clone(value.Kinds), Accepted: slices.Clone(value.Accepted), Codes: slices.Clone(value.Codes), Revisions: slices.Clone(value.Revisions), EventCounts: slices.Clone(value.EventCounts), EventStarts: slices.Clone(value.EventStarts)}
}

func EqualI64(left int64, right int64) bool { return left <= right && right <= left }
func ZeroDigest(value Digest) bool {
	return EqualI64(value.A, 0) && EqualI64(value.B, 0) && EqualI64(value.C, 0) && EqualI64(value.D, 0)
}
func EqualDigestAt(value State, index int64, digest Digest) bool {
	return EqualI64(value.DigestA[index], digest.A) && EqualI64(value.DigestB[index], digest.B) && EqualI64(value.DigestC[index], digest.C) && EqualI64(value.DigestD[index], digest.D)
}
