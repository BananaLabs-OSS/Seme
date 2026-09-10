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
	return ControlledCommand{Command: command, Clock: controlled.ClockSample{Sequence: sequence, UnixMilliseconds: 1000 + sequence}, Random: controlled.RandomState{Value: identity}}
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
	if output.String() != "true\n" {
		t.Fatalf("accepted effect=%q", output.String())
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
	changedSequence := command
	changedSequence.Clock.Sequence++
	changedDraw := command
	changedDraw.Random.Draws++
	for name, changed := range map[string]ControlledCommand{"clock-sequence": changedSequence, "draw-count": changedDraw} {
		got := DispatchControlled(first.State, changed)
		if got.OK || !reflect.DeepEqual(got.State, first.State) || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
			t.Fatalf("%s consumed: %#v", name, got)
		}
	}
	rejectedCommand := controlledCommand(2, 9)
	rejectedCommand.Clock = controlled.ClockSample{Sequence: 2, UnixMilliseconds: 2000}
	rejectedCommand.Random = controlled.RandomState{Value: first.State.RandomAfterValues[0], Draws: 1}
	rejectedCommand.Command.Payload.Loaded.Version = 3
	rejected := DispatchControlled(first.State, rejectedCommand)
	if !rejected.OK || rejected.Response.Accepted || output.String() != "true\nfalse\n" {
		t.Fatalf("domain rejection effect: %#v %q", rejected, output.String())
	}
	over := controlledCommand(2, 10)
	over.Clock = controlled.ClockSample{Sequence: 2, UnixMilliseconds: 2001}
	over.Random = controlled.RandomState{Value: first.State.RandomAfterValues[0], Draws: 1}
	over.Command.PayloadWords = make([]int64, 512)
	over.Command.PayloadByteLength = 3072
	failed := DispatchControlled(first.State, over)
	if failed.OK || !reflect.DeepEqual(failed.State, first.State) || output.String() != "true\nfalse\n" {
		t.Fatalf("failed commit boundary logged: %#v %q", failed, output.String())
	}
}

func TestInvalidExplicitInputsAreAtomic(t *testing.T) {
	initial := NewControlledState("match")
	for _, command := range []ControlledCommand{{Command: controlledCommand(1, 1).Command, Clock: controlled.ClockSample{Sequence: 1, UnixMilliseconds: -1}, Random: controlled.RandomState{Value: 1}}, {Command: controlledCommand(1, 1).Command, Clock: controlled.ClockSample{Sequence: 1, UnixMilliseconds: 4102444800001}, Random: controlled.RandomState{Value: 1}}, {Command: controlledCommand(1, 1).Command, Clock: controlled.ClockSample{}, Random: controlled.RandomState{}}} {
		got := DispatchControlled(initial, command)
		if got.OK || got.Error != InvalidControlledInput || !reflect.DeepEqual(got.State, initial) {
			t.Fatal(got)
		}
	}
}

func TestCommitBoundFailureDoesNotLog(t *testing.T) {
	oldWriter := log.Writer()
	defer log.SetOutput(oldWriter)
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetFlags(0)
	firstCommand := controlledCommand(1, 31)
	firstCommand.Command.PayloadWords = make([]int64, 384)
	firstCommand.Command.PayloadWords[0] = 31
	firstCommand.Command.PayloadByteLength = 3072
	first := DispatchControlled(NewControlledState("match"), firstCommand)
	if !first.OK {
		t.Fatal(first)
	}
	output.Reset()
	second := controlledCommand(2, 32)
	second.Clock = controlled.ClockSample{Sequence: 2, UnixMilliseconds: 2000}
	second.Random = controlled.RandomState{Value: first.State.RandomAfterValues[0], Draws: 1}
	second.Command.PayloadWords = make([]int64, 129)
	second.Command.PayloadWords[0] = 32
	second.Command.PayloadByteLength = 1032
	got := DispatchControlled(first.State, second)
	if got.OK || !reflect.DeepEqual(got.State, first.State) || output.Len() != 0 {
		t.Fatalf("failed commit boundary logged: %#v %q", got, output.String())
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
			random.Value = current.RandomAfterValues[len(current.RandomAfterValues)-1]
			random.Draws = current.RandomAfterDraws[len(current.RandomAfterDraws)-1]
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

func TestReplayControlledMatchesDispatchWithoutPhysicalEffects(t *testing.T) {
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	defer log.SetOutput(oldWriter)
	defer log.SetFlags(oldFlags)
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetFlags(0)
	command := controlledCommand(1, 41)
	live := DispatchControlled(NewControlledState("match"), command)
	if output.String() != "true\n" {
		t.Fatalf("live effect=%q", output.String())
	}
	output.Reset()
	replayed := ReplayControlled(NewControlledState("match"), command)
	if !reflect.DeepEqual(replayed, live) || output.Len() != 0 {
		t.Fatalf("replay=%#v live=%#v effect=%q", replayed, live, output.String())
	}
	duplicate := ReplayControlled(replayed.State, command)
	if !duplicate.OK || !duplicate.Duplicate || output.Len() != 0 {
		t.Fatalf("duplicate replay=%#v effect=%q", duplicate, output.String())
	}
}

func TestDraw256IsTerminal(t *testing.T) {
	oldWriter := log.Writer()
	defer log.SetOutput(oldWriter)
	log.SetOutput(&bytes.Buffer{})
	current := NewControlledState("match")
	random := controlled.RandomState{Value: 3}
	for sequence := int64(1); sequence <= 256; sequence++ {
		command := controlledCommand(sequence, sequence+1000)
		command.Clock = controlled.ClockSample{Sequence: sequence, UnixMilliseconds: sequence}
		command.Random = random
		result := DispatchControlled(current, command)
		if !result.OK {
			t.Fatalf("draw %d: %d", sequence, result.Error)
		}
		current = result.State
		random = controlled.RandomState{Value: current.RandomAfterValues[sequence-1], Draws: current.RandomAfterDraws[sequence-1]}
	}
	if random.Draws != 256 || !validControlledState(current) {
		t.Fatal("terminal state")
	}
	terminal := controlledCommand(257, 2000)
	terminal.Clock = controlled.ClockSample{Sequence: 256, UnixMilliseconds: 257}
	terminal.Random = random
	if got := DispatchControlled(current, terminal); got.OK || got.Error != transport.InvalidSequence || !reflect.DeepEqual(got.State, current) {
		t.Fatal("terminal draw consumed", got)
	}
}
