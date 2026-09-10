// Package goupb09replay verifies an authenticated ControlledReplay by executing
// its exact effect-free transition and comparing every retained authority.
package goupb09replay

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/wire"
)

type Request struct {
	State, CanonicalCommand                           []byte
	CommandSequence, ClockSequence, RandomDrawOrdinal uint64
	UnixMilliseconds, RandomBefore                    int64
}
type Outcome struct {
	Committed                bool
	State, Response, Events  []byte
	RandomAfter, RandomValue int64
	RandomDrawOrdinal        uint64
	EffectValue              bool
	Failure                  string
}
type Executor interface{ Execute(Request) Outcome }
type Result struct {
	FinalState        []byte
	RandomValue       int64
	RandomDrawOrdinal uint64
	Effects           []bool
	Steps             uint64
	TranscriptSHA256  [32]byte
}

func Verify(in controlledeffectsinstance.Inputs, executor Executor) (Result, error) {
	if executor == nil {
		return Result{}, fmt.Errorf("go_upb09_replay.executor")
	}
	if err := controlledeffectsinstance.Validate(in); err != nil {
		return Result{}, fmt.Errorf("go_upb09_replay.authority:%w", err)
	}
	e, err := wire.Decode(in.Artifact)
	if err != nil {
		return Result{}, err
	}
	q, ok := one(e, id("13107"))
	if !ok {
		return Result{}, fmt.Errorf("go_upb09_replay.record")
	}
	transcript, ok := digest(q.Fields[id("13272")].Bytes)
	if !ok {
		return Result{}, fmt.Errorf("go_upb09_replay.transcript")
	}
	initial, ok := digest(q.Fields[id("13276")].Bytes)
	if !ok {
		return Result{}, fmt.Errorf("go_upb09_replay.initial")
	}
	return verify(in.Model.Replay, transcript, initial, executor)
}

func verify(r controlledeffectsinstance.Replay, expectedTranscript, expectedInitial [32]byte, executor Executor) (Result, error) {
	if sha256.Sum256(r.InitialState) != expectedInitial || transcriptDigest(r) != expectedTranscript {
		return Result{}, fmt.Errorf("go_upb09_replay.digest")
	}
	state := bytes.Clone(r.InitialState)
	effects := make([]bool, 0, len(r.Steps))
	var random int64
	var ordinal uint64
	for i, s := range r.Steps {
		want := uint64(i + 1)
		if s.CommandSequence != want || s.ClockSequence != want || s.RandomDrawOrdinal != want || (i > 0 && (s.UnixMilliseconds < r.Steps[i-1].UnixMilliseconds || s.RandomBefore != r.Steps[i-1].RandomAfter)) || (i == 0 && s.RandomBefore != r.InitialSeed) || sha256.Sum256(state) != s.StateBeforeSHA256 {
			return Result{}, fmt.Errorf("go_upb09_replay.order:%d", i)
		}
		req := Request{State: bytes.Clone(state), CanonicalCommand: bytes.Clone(s.CanonicalCommand), CommandSequence: s.CommandSequence, ClockSequence: s.ClockSequence, RandomDrawOrdinal: s.RandomDrawOrdinal, UnixMilliseconds: s.UnixMilliseconds, RandomBefore: s.RandomBefore}
		o := executor.Execute(req)
		if !o.Committed || o.Failure != "" || o.RandomAfter != s.RandomAfter || o.RandomValue != s.RandomValue || o.RandomDrawOrdinal != s.RandomDrawOrdinal || o.EffectValue != s.EffectValue || sha256.Sum256(o.Response) != s.ResponseSHA256 || sha256.Sum256(o.Events) != s.EventsSHA256 || sha256.Sum256(o.State) != s.StateAfterSHA256 {
			return Result{}, fmt.Errorf("go_upb09_replay.step:%d", i)
		}
		state = bytes.Clone(o.State)
		random = o.RandomValue
		ordinal = o.RandomDrawOrdinal
		effects = append(effects, o.EffectValue)
	}
	return Result{FinalState: state, RandomValue: random, RandomDrawOrdinal: ordinal, Effects: effects, Steps: uint64(len(r.Steps)), TranscriptSHA256: expectedTranscript}, nil
}

func transcriptDigest(r controlledeffectsinstance.Replay) [32]byte {
	h := sha256.New()
	h.Write([]byte("seme.controlled-effects.replay.v1\x00"))
	_ = binary.Write(h, binary.BigEndian, r.InitialSeed)
	writeBytes(h, r.InitialState)
	for _, s := range r.Steps {
		_ = binary.Write(h, binary.BigEndian, s.CommandSequence)
		_ = binary.Write(h, binary.BigEndian, s.UnixMilliseconds)
		_ = binary.Write(h, binary.BigEndian, s.ClockSequence)
		_ = binary.Write(h, binary.BigEndian, s.RandomBefore)
		_ = binary.Write(h, binary.BigEndian, s.RandomAfter)
		_ = binary.Write(h, binary.BigEndian, s.RandomValue)
		_ = binary.Write(h, binary.BigEndian, s.RandomDrawOrdinal)
		if s.EffectValue {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		writeBytes(h, s.CanonicalCommand)
		h.Write(s.ResponseSHA256[:])
		h.Write(s.EventsSHA256[:])
		h.Write(s.StateBeforeSHA256[:])
		h.Write(s.StateAfterSHA256[:])
	}
	writeBytes(h, []byte(r.DuplicatePolicy))
	writeBytes(h, []byte(r.RejectionPolicy))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

type writer interface{ Write([]byte) (int, error) }

func writeBytes(w writer, b []byte) {
	_ = binary.Write(w, binary.BigEndian, uint64(len(b)))
	_, _ = w.Write(b)
}
func one(e wire.Envelope, s wire.ID) (wire.Entity, bool) {
	var q wire.Entity
	n := 0
	for _, x := range e.Entities {
		if x.Schema == s {
			q = x
			n++
		}
	}
	return q, n == 1
}
func digest(b []byte) ([32]byte, bool) {
	var x [32]byte
	if len(b) != 32 {
		return x, false
	}
	copy(x[:], b)
	return x, true
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
