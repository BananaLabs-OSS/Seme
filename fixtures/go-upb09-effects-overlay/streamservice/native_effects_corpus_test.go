package streamservice

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"example.test/go-uab-11/controlled"
)

var nativeEffectsCorpusDirectory = flag.String("native-effects-corpus-dir", "", "new directory for deterministic controlled-effects JSONL")

func TestNativeControlledEffectsRuntimeCorpus(t *testing.T) {
	if *nativeEffectsCorpusDirectory == "" {
		t.Skip("native controlled-effects corpus output not requested")
	}
	if !filepath.IsAbs(*nativeEffectsCorpusDirectory) || filepath.Clean(*nativeEffectsCorpusDirectory) != *nativeEffectsCorpusDirectory {
		t.Fatal("native corpus directory must be absolute and clean")
	}
	if err := os.Mkdir(*nativeEffectsCorpusDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	complete := false
	t.Cleanup(func() {
		if !complete {
			_ = os.RemoveAll(*nativeEffectsCorpusDirectory)
		}
	})
	requests := createEffectsCorpusFile(t, "requests.jsonl")
	expected := createEffectsCorpusFile(t, "expected.jsonl")
	rq, ex := json.NewEncoder(requests), json.NewEncoder(expected)
	oldWriter, oldFlags := log.Writer(), log.Flags()
	t.Cleanup(func() { log.SetOutput(oldWriter); log.SetFlags(oldFlags) })
	var observed bytes.Buffer
	log.SetOutput(&observed)
	log.SetFlags(0)
	count := int64(0)
	emit := func(before ControlledState, command ControlledCommand) ControlledResult {
		observed.Reset()
		result := DispatchControlled(before, command)
		request := map[string]any{"arguments": []any{transportCorpusValue(reflect.ValueOf(before)), transportCorpusValue(reflect.ValueOf(command))}, "capabilities": []string{"observability.log"}}
		effects := []any{}
		if result.OK && !result.Duplicate {
			want := fmt.Sprintf("%t\n", result.Response.Accepted)
			if observed.String() != want {
				t.Fatalf("native effect %q, want %q", observed.String(), want)
			}
			effects = append(effects, map[string]any{"capability": "observability.log", "value": result.Response.Accepted})
		} else if observed.Len() != 0 {
			t.Fatalf("rejected or duplicate command emitted %q", observed.String())
		}
		observation := map[string]any{"value": transportCorpusValue(reflect.ValueOf(result)), "effects": effects}
		if err := rq.Encode(request); err != nil {
			t.Fatal(err)
		}
		if err := ex.Encode(observation); err != nil {
			t.Fatal(err)
		}
		count++
		return result
	}

	// Pin explicit-input boundaries before the generated stateful corpus.
	initial := NewControlledState("match")
	for _, command := range []ControlledCommand{
		controlledCorpusCommand(1, 1, 0, 1),
		controlledCorpusCommand(1, 2, -1, 1),
		controlledCorpusCommand(1, 3, 4102444800001, 1),
		controlledCorpusCommand(1, 4, 0, 0),
		controlledCorpusCommand(1, 5, 4102444800000, math.MaxInt64),
	} {
		emit(initial, command)
	}

	current := NewControlledState("match")
	random := controlled.RandomState{Value: 1}
	last := ControlledCommand{}
	for index := int64(0); count < 4096; index++ {
		if index%32 == 0 {
			current = NewControlledState("match")
			seeds := []int64{1, math.MaxInt64, 48271, -1}
			random = controlled.RandomState{Value: seeds[(index/32)%int64(len(seeds))]}
			if random.Value < 1 {
				random.Value = 1
			}
			last = ControlledCommand{}
		}
		sequence := int64(len(current.Draws)) + 1
		command := controlledCorpusCommand(sequence, 1000+index, index%4102444800001, random.Value)
		command.Random = random
		switch index % 32 {
		case 1:
			command = last // exact duplicate: no new clock, draw, or effect
		case 2:
			command = last
			command.Clock.UnixMilliseconds++ // cached correlation conflict
		case 3:
			command.Command.Sequence++ // transport gap
			command.Clock.Sequence++
		case 4:
			command.Clock.UnixMilliseconds = -1
		case 5:
			command.Random.Draws++
		case 6:
			command.Random.Value++
		case 7:
			command.Command.Payload.Loaded.Version = 3 // committed domain rejection; false effect
		case 8:
			if len(current.ClockMilliseconds) != 0 {
				command.Clock.UnixMilliseconds = current.ClockMilliseconds[len(current.ClockMilliseconds)-1] // equal clock is valid
			}
		case 9:
			if len(current.ClockMilliseconds) != 0 {
				command.Clock.UnixMilliseconds = current.ClockMilliseconds[len(current.ClockMilliseconds)-1] - 1
			}
		}
		before := current
		result := emit(before, command)
		if result.OK && !result.Duplicate {
			current = result.State
			random = controlled.RandomState{Value: current.RandomAfterValues[len(current.RandomAfterValues)-1], Draws: current.RandomAfterDraws[len(current.RandomAfterDraws)-1]}
			last = command
		}
	}
	if err := requests.Close(); err != nil {
		t.Fatal(err)
	}
	if err := expected.Close(); err != nil {
		t.Fatal(err)
	}
	requestDigest := transportCorpusDigest(t, filepath.Join(*nativeEffectsCorpusDirectory, "requests.jsonl"))
	expectedDigest := transportCorpusDigest(t, filepath.Join(*nativeEffectsCorpusDirectory, "expected.jsonl"))
	marker := fmt.Sprintf("seme-go-upb09-native-v1\ncount=4096\nrequests_sha256=%x\nexpected_sha256=%x\n", requestDigest, expectedDigest)
	if err := os.WriteFile(filepath.Join(*nativeEffectsCorpusDirectory, "COMPLETE"), []byte(marker), 0600); err != nil {
		t.Fatal(err)
	}
	complete = true
}

func controlledCorpusCommand(sequence, identity, milliseconds, seed int64) ControlledCommand {
	command := controlledCommand(sequence, identity)
	command.Clock = controlled.ClockSample{Sequence: sequence, UnixMilliseconds: milliseconds}
	command.Random = controlled.RandomState{Value: seed}
	return command
}

func createEffectsCorpusFile(t *testing.T, name string) *os.File {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(*nativeEffectsCorpusDirectory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
