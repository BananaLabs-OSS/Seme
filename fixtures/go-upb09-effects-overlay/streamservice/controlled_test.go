package streamservice

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/controlled"
	"example.test/go-uab-11/persistence"
	"example.test/go-uab-11/state"
	"example.test/go-uab-11/transport"
)

func controlledCommand(sequence int64, identity int64) ControlledCommand {
	base := application.State{Name: "pilot", Counters: map[int64]int64{}}
	command := transport.Command{Stream: "match", Sequence: sequence, Correlation: transport.Correlation{Low: identity}, Kind: transport.PlannerCommand, Digest: transport.Digest{A: identity}, PayloadWords: []int64{identity}, PayloadByteLength: 1, Payload: transport.PlannerPayload{Grants: persistence.Grants{Read: true, CompareExchange: true}, Loaded: persistence.Loaded{Found: true, Version: 2, Token: "opaque", V2: state.V2{State: base, Revision: 4}}, Initial: state.V2{State: base, Revision: 1}, NextDigest: "sha256:next", Key: "slot"}}
	return ControlledCommand{Command: command, Clock: controlled.ClockSample{UnixMilliseconds: 1000 + sequence}, Random: controlled.RandomState{Value: identity}}
}

func TestOnlyNewCommandsConsumeInputsAndLog(t *testing.T) {
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	defer log.SetOutput(oldWriter)
	defer log.SetFlags(oldFlags)
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetFlags(0)
	initial := NewControlledState("match")
	command := controlledCommand(1, 7)
	first := DispatchControlled(initial, command)
	if !first.OK || len(first.State.Draws) != 1 || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("first=%#v log=%q", first, output.String())
	}
	duplicate := DispatchControlled(first.State, command)
	if !duplicate.OK || !duplicate.Duplicate || !reflect.DeepEqual(duplicate.State, first.State) || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("duplicate=%#v log=%q", duplicate, output.String())
	}
	for name, changed := range map[string]ControlledCommand{"gap": controlledCommand(3, 9), "conflict-clock": command, "conflict-seed": command} {
		if name == "conflict-clock" {
			changed.Clock.UnixMilliseconds++
		}
		if name == "conflict-seed" {
			changed.Random.Value++
		}
		got := DispatchControlled(first.State, changed)
		if got.OK || !reflect.DeepEqual(got.State, first.State) || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
			t.Fatalf("%s consumed: %#v %q", name, got, output.String())
		}
	}
}

func TestInvalidExplicitInputsAreAtomic(t *testing.T) {
	initial := NewControlledState("match")
	for _, command := range []ControlledCommand{{Command: controlledCommand(1, 1).Command, Clock: controlled.ClockSample{UnixMilliseconds: -1}, Random: controlled.RandomState{Value: 1}}, {Command: controlledCommand(1, 1).Command, Clock: controlled.ClockSample{}, Random: controlled.RandomState{}}} {
		got := DispatchControlled(initial, command)
		if got.OK || got.Error != InvalidControlledInput || !reflect.DeepEqual(got.State, initial) {
			t.Fatal(got)
		}
	}
}

func TestExplicitInputsReplayDeterministically(t *testing.T) {
	run := func() ControlledState {
		current := NewControlledState("match")
		random := controlled.RandomState{Value: 11}
		for sequence := int64(1); sequence <= 32; sequence++ {
			command := controlledCommand(sequence, sequence+20)
			command.Clock.UnixMilliseconds = 5000 + sequence
			command.Random = random
			result := DispatchControlled(current, command)
			if !result.OK {
				t.Fatal(result.Error)
			}
			current = result.State
			random.Value = current.RandomAfter[len(current.RandomAfter)-1]
		}
		return current
	}
	oldWriter := log.Writer()
	defer log.SetOutput(oldWriter)
	log.SetOutput(&bytes.Buffer{})
	if first, second := run(), run(); !reflect.DeepEqual(first, second) {
		t.Fatal("explicit replay diverged")
	}
}
