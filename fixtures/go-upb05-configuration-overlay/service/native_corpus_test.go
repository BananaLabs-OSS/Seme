package service

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/configuration"
)

var nativeCorpusDirectory = flag.String("native-corpus-dir", "", "new directory for deterministic request and observation JSONL")

func TestNativeConfigurationCorpus(t *testing.T) {
	if *nativeCorpusDirectory == "" {
		t.Skip("native corpus output not requested")
	}
	if !filepath.IsAbs(*nativeCorpusDirectory) || filepath.Clean(*nativeCorpusDirectory) != *nativeCorpusDirectory {
		t.Fatal("native corpus directory must be absolute and clean")
	}
	if err := os.Mkdir(*nativeCorpusDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	complete := false
	t.Cleanup(func() {
		if !complete {
			_ = os.RemoveAll(*nativeCorpusDirectory)
		}
	})
	requests, err := os.OpenFile(filepath.Join(*nativeCorpusDirectory, "requests.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	observations, err := os.OpenFile(filepath.Join(*nativeCorpusDirectory, "expected.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = requests.Close()
		t.Fatal(err)
	}
	requestEncoder, observationEncoder := json.NewEncoder(requests), json.NewEncoder(observations)
	log.SetOutput(io.Discard)
	count := 0
	emit := func(input configuration.Input, state application.State, command application.Command) {
		t.Helper()
		got := ApplyConfigured(input, state, command)
		request := map[string]any{"arguments": []any{configValue(input), stateValue(state), commandValue(command)}, "capabilities": []string{"observability.log"}}
		effects := []any{}
		if got.Ok {
			effects = append(effects, map[string]any{"capability": "observability.log", "value": true})
		}
		observation := map[string]any{"value": outcomeValue(got), "effects": effects}
		if err := requestEncoder.Encode(request); err != nil {
			t.Fatal(err)
		}
		if err := observationEncoder.Encode(observation); err != nil {
			t.Fatal(err)
		}
		count++
	}

	random := uint64(0x5eed1234abcdef01)
	for sequence := int64(0); sequence < 128; sequence++ {
		boundary := sequence % 17
		if sequence%11 == 0 {
			boundary = math.MaxInt64
		}
		values := []int64{boundary, sequence % 5, 2}
		if sequence%13 == 0 {
			values = []int64{1, -1, 2}
		}
		state := application.State{Name: "sequence-" + strconv.FormatInt(sequence, 10), Values: values, Counters: map[int64]int64{1: boundary, 2: 0}}
		for step := int64(0); step < 16; step++ {
			random ^= random << 13
			random ^= random >> 7
			random ^= random << 17
			sample := int64(random)
			index := sample % 2
			if index < 0 {
				index = -index
			}
			key := int64(2)
			if step%2 == 0 {
				key = 1
			}
			if step%7 == 1 {
				key = 99
			}
			delta := sample
			if step%8 == 3 {
				delta = math.MaxInt64
			}
			amount := sample >> 3
			if step%8 == 4 {
				amount = math.MinInt64
			}
			command := application.Command{Key: key, Index: index, Delta: delta, Amount: amount, Scale: step%2 == 0}
			emit(configuration.Input{NamePrefix: "sequence", UseDefaultLimit: true}, state, command)
			got := ApplyConfigured(configuration.Input{NamePrefix: "sequence", UseDefaultLimit: true}, state, command)
			base := application.Apply(state, command)
			if !reflect.DeepEqual(got, base) {
				t.Fatalf("configured behavior drift at sequence=%d step=%d", sequence, step)
			}
			if got.Ok {
				state = got.Value.State
			}
		}
	}

	state := application.State{Name: "config", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	command := application.Command{Key: 7, Index: 1, Delta: 4, Amount: 1, Scale: true}
	for _, input := range []configuration.Input{
		{NamePrefix: "世界", UseDefaultLimit: true},
		{NamePrefix: "x", Limit: 1},
		{NamePrefix: "x", Limit: 4096},
		{UseDefaultLimit: true},
		{NamePrefix: "x", Limit: 0},
		{NamePrefix: "x", Limit: -1},
		{NamePrefix: "x", Limit: 4097},
		{NamePrefix: "x", Limit: math.MaxInt64},
		{Limit: 0},
		{NamePrefix: "x", Limit: 2},
	} {
		emit(input, state, command)
	}
	if count != 2058 {
		t.Fatalf("corpus count=%d", count)
	}
	if err := requests.Close(); err != nil {
		t.Fatal(err)
	}
	if err := observations.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*nativeCorpusDirectory, "COMPLETE"), []byte(fmt.Sprintf("seme-go-upb05-native-v1\n%d\n", count)), 0600); err != nil {
		t.Fatal(err)
	}
	complete = true
}

func i64(value int64) any { return map[string]any{"kind": "i64", "i64": strconv.FormatInt(value, 10)} }

func configValue(input configuration.Input) any {
	return map[string]any{"kind": "record", "fields": map[string]any{
		"NamePrefix": i64Text(input.NamePrefix), "Limit": i64(input.Limit), "UseDefaultLimit": map[string]any{"kind": "bool", "bool": input.UseDefaultLimit},
	}}
}

func i64Text(value string) any { return map[string]any{"kind": "text", "text": value} }

func stateValue(state application.State) any {
	values := make([]any, len(state.Values))
	for index, value := range state.Values {
		values[index] = i64(value)
	}
	entries := make([]any, 0, len(state.Counters))
	for _, key := range []int64{1, 2, 7} {
		if value, ok := state.Counters[key]; ok {
			entries = append(entries, map[string]any{"key": i64(key), "value": i64(value)})
		}
	}
	return map[string]any{"kind": "record", "fields": map[string]any{"Name": i64Text(state.Name), "Values": map[string]any{"kind": "slice", "items": values}, "Counters": map[string]any{"kind": "map", "value_type": "i64", "entries": entries}}}
}

func commandValue(command application.Command) any {
	return map[string]any{"kind": "record", "fields": map[string]any{"Key": i64(command.Key), "Index": i64(command.Index), "Delta": i64(command.Delta), "Amount": i64(command.Amount), "Scale": map[string]any{"kind": "bool", "bool": command.Scale}}}
}

func outcomeValue(outcome application.Outcome) any {
	if !outcome.Ok {
		return map[string]any{"kind": "result", "variant": "error", "payload": i64(outcome.Error)}
	}
	return map[string]any{"kind": "result", "variant": "ok", "payload": map[string]any{"kind": "transition", "state": stateValue(outcome.Value.State), "result": i64(outcome.Value.Result)}}
}
