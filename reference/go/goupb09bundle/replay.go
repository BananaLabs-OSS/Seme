package goupb09bundle

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"seme.local/reference/controlledeffectsinstance"
)

const replayVersion = "seme.controlled-replay/v1"

type replayStepJSON struct {
	CommandSequence   uint64 `json:"command_sequence"`
	ClockSequence     uint64 `json:"clock_sequence"`
	UnixMilliseconds  int64  `json:"unix_milliseconds"`
	RandomBefore      int64  `json:"random_before"`
	RandomAfter       int64  `json:"random_after"`
	RandomValue       int64  `json:"random_value"`
	RandomDrawOrdinal uint64 `json:"random_draw_ordinal"`
	EffectValue       bool   `json:"effect_value"`
	CanonicalCommand  string `json:"canonical_command"`
	ResponseSHA256    string `json:"response_sha256"`
	EventsSHA256      string `json:"events_sha256"`
	StateBeforeSHA256 string `json:"state_before_sha256"`
	StateAfterSHA256  string `json:"state_after_sha256"`
}
type replayJSON struct {
	Version         string           `json:"version"`
	InitialSeed     int64            `json:"initial_seed"`
	InitialState    string           `json:"initial_state"`
	DuplicatePolicy string           `json:"duplicate_policy"`
	RejectionPolicy string           `json:"rejection_policy"`
	Steps           []replayStepJSON `json:"steps"`
}

// EncodeReplayAuthority emits the sole canonical JSON representation accepted
// as static project authority. Dynamic session transcripts are separate.
func EncodeReplayAuthority(r controlledeffectsinstance.Replay, z controlledeffectsinstance.Bounds) ([]byte, error) {
	x := replayJSON{Version: replayVersion, InitialSeed: r.InitialSeed, InitialState: hex.EncodeToString(r.InitialState), DuplicatePolicy: r.DuplicatePolicy, RejectionPolicy: r.RejectionPolicy, Steps: make([]replayStepJSON, 0, len(r.Steps))}
	for _, v := range r.Steps {
		x.Steps = append(x.Steps, replayStepJSON{CommandSequence: v.CommandSequence, ClockSequence: v.ClockSequence, UnixMilliseconds: v.UnixMilliseconds, RandomBefore: v.RandomBefore, RandomAfter: v.RandomAfter, RandomValue: v.RandomValue, RandomDrawOrdinal: v.RandomDrawOrdinal, EffectValue: v.EffectValue, CanonicalCommand: hex.EncodeToString(v.CanonicalCommand), ResponseSHA256: hex.EncodeToString(v.ResponseSHA256[:]), EventsSHA256: hex.EncodeToString(v.EventsSHA256[:]), StateBeforeSHA256: hex.EncodeToString(v.StateBeforeSHA256[:]), StateAfterSHA256: hex.EncodeToString(v.StateAfterSHA256[:])})
	}
	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	if _, err = parseReplayUnchecked(b, z); err != nil {
		return nil, err
	}
	return b, nil
}

// ParseReplayAuthority accepts only the canonical encoding produced by
// EncodeReplayAuthority.
func ParseReplayAuthority(data []byte, z controlledeffectsinstance.Bounds) (controlledeffectsinstance.Replay, error) {
	r, err := parseReplayUnchecked(data, z)
	if err != nil {
		return r, err
	}
	canonical, err := EncodeReplayAuthority(r, z)
	if err != nil || !bytes.Equal(data, canonical) {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.replay_canonical")
	}
	return r, nil
}

func parseReplay(data []byte, z controlledeffectsinstance.Bounds) (controlledeffectsinstance.Replay, error) {
	return ParseReplayAuthority(data, z)
}

func parseReplayUnchecked(data []byte, z controlledeffectsinstance.Bounds) (controlledeffectsinstance.Replay, error) {
	if len(data) == 0 || uint64(len(data)) > z.MaximumTranscriptBytes {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.replay_size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x replayJSON
	if err := d.Decode(&x); err != nil {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.replay_json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.replay_trailing")
	}
	if x.Version != replayVersion || x.DuplicatePolicy != "cached-no-new-effects" || x.RejectionPolicy != "atomic-no-effects" || uint64(len(x.Steps)) > z.MaximumSteps {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.replay_policy")
	}
	decode := func(s string, max uint64) ([]byte, error) {
		if len(s)%2 != 0 || uint64(len(s)/2) > max {
			return nil, fmt.Errorf("size")
		}
		b, e := hex.DecodeString(s)
		if e != nil || hex.EncodeToString(b) != s {
			return nil, fmt.Errorf("hex")
		}
		return b, nil
	}
	initial, err := decode(x.InitialState, z.MaximumInitialStateBytes)
	if err != nil {
		return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.initial_state")
	}
	r := controlledeffectsinstance.Replay{InitialSeed: x.InitialSeed, InitialState: initial, DuplicatePolicy: x.DuplicatePolicy, RejectionPolicy: x.RejectionPolicy, Steps: make([]controlledeffectsinstance.ReplayStep, 0, len(x.Steps))}
	total := uint64(len(initial))
	digest := func(s string) ([32]byte, error) {
		var out [32]byte
		b, e := decode(s, 32)
		if e != nil || len(b) != 32 {
			return out, fmt.Errorf("digest")
		}
		copy(out[:], b)
		return out, nil
	}
	for i, v := range x.Steps {
		command, e := decode(v.CanonicalCommand, z.MaximumCommandBytes)
		if e != nil {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.command:%d", i)
		}
		total += uint64(len(command)) + 128
		if total > z.MaximumTranscriptBytes {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.transcript")
		}
		response, e := digest(v.ResponseSHA256)
		if e != nil {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.response:%d", i)
		}
		events, e := digest(v.EventsSHA256)
		if e != nil {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.events:%d", i)
		}
		before, e := digest(v.StateBeforeSHA256)
		if e != nil {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.before:%d", i)
		}
		after, e := digest(v.StateAfterSHA256)
		if e != nil {
			return controlledeffectsinstance.Replay{}, fmt.Errorf("go_upb09_bundle.after:%d", i)
		}
		r.Steps = append(r.Steps, controlledeffectsinstance.ReplayStep{CommandSequence: v.CommandSequence, ClockSequence: v.ClockSequence, UnixMilliseconds: v.UnixMilliseconds, RandomBefore: v.RandomBefore, RandomAfter: v.RandomAfter, RandomValue: v.RandomValue, RandomDrawOrdinal: v.RandomDrawOrdinal, EffectValue: v.EffectValue, CanonicalCommand: command, ResponseSHA256: response, EventsSHA256: events, StateBeforeSHA256: before, StateAfterSHA256: after})
	}
	return r, nil
}
