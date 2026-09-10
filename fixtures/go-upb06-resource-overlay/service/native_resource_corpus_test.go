package service

import (
	"encoding/hex"
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
	"example.test/go-uab-11/resource"
)

var nativeResourceCorpusDirectory = flag.String("native-resource-corpus-dir", "", "new directory for deterministic resource request and observation JSONL")

func TestNativeResourceCorpus(t *testing.T) {
	if *nativeResourceCorpusDirectory == "" {
		t.Skip("native resource corpus output not requested")
	}
	if !filepath.IsAbs(*nativeResourceCorpusDirectory) || filepath.Clean(*nativeResourceCorpusDirectory) != *nativeResourceCorpusDirectory {
		t.Fatal("native resource corpus directory must be absolute and clean")
	}
	if err := os.Mkdir(*nativeResourceCorpusDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	complete := false
	t.Cleanup(func() {
		if !complete {
			_ = os.RemoveAll(*nativeResourceCorpusDirectory)
		}
	})
	requests, err := os.OpenFile(filepath.Join(*nativeResourceCorpusDirectory, "requests.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	observations, err := os.OpenFile(filepath.Join(*nativeResourceCorpusDirectory, "expected.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = requests.Close()
		t.Fatal(err)
	}
	requestEncoder, observationEncoder := json.NewEncoder(requests), json.NewEncoder(observations)
	log.SetOutput(io.Discard)
	notice, err := os.ReadFile("../resources/notice.txt")
	if err != nil {
		t.Fatal(err)
	}
	marker, err := os.ReadFile("../resources/marker.bin")
	if err != nil {
		t.Fatal(err)
	}
	actual := resource.Set{Notice: string(notice), Marker: marker}
	count := 0
	emit := func(input configuration.Input, state application.State, command application.Command, resources resource.Set) {
		t.Helper()
		got := ApplyConfiguredResource(input, state, command, resources)
		request := map[string]any{"arguments": []any{configValue(input), stateValue(state), commandValue(command), resourceValue(resources)}, "capabilities": []string{"observability.log"}}
		effects := []any{}
		if got.Ok {
			effects = append(effects, map[string]any{"capability": "observability.log", "value": true})
		}
		if err := requestEncoder.Encode(request); err != nil {
			t.Fatal(err)
		}
		if err := observationEncoder.Encode(map[string]any{"value": outcomeValue(got), "effects": effects}); err != nil {
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
			input := configuration.Input{NamePrefix: "sequence", UseDefaultLimit: true}
			emit(input, state, command, actual)
			got, base := ApplyConfiguredResource(input, state, command, actual), application.Apply(state, command)
			if !reflect.DeepEqual(got, base) {
				t.Fatalf("resource behavior drift at sequence=%d step=%d", sequence, step)
			}
			if got.Ok {
				state = got.Value.State
			}
		}
	}

	state := application.State{Name: "config", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	command := application.Command{Key: 7, Index: 1, Delta: 4, Amount: 1, Scale: true}
	for _, input := range []configuration.Input{{NamePrefix: "世界", UseDefaultLimit: true}, {NamePrefix: "x", Limit: 1}, {NamePrefix: "x", Limit: 4096}, {UseDefaultLimit: true}, {NamePrefix: "x", Limit: 0}, {NamePrefix: "x", Limit: -1}, {NamePrefix: "x", Limit: 4097}, {NamePrefix: "x", Limit: math.MaxInt64}, {Limit: 0}, {NamePrefix: "x", Limit: 2}} {
		emit(input, state, command, actual)
	}

	variants := []resource.Set{
		actual,
		{Marker: marker},
		{Notice: "Seme resources - world\n", Marker: marker},
		{Notice: "Seme resources — 世界", Marker: marker},
		{Notice: actual.Notice},
		{Notice: actual.Notice, Marker: []byte{}},
		{Notice: actual.Notice, Marker: []byte{0x00, 0xfe, 'S', 'E', 'M', 'E', '\n'}},
		{Notice: actual.Notice, Marker: append(append([]byte{}, marker...), 0x00)},
	}
	for index, resources := range variants {
		got := ApplyConfiguredResource(configuration.Input{NamePrefix: "x", UseDefaultLimit: true}, state, command, resources)
		if index == 0 && !got.Ok {
			t.Fatalf("valid resource case: %+v", got)
		}
		if index >= 1 && index <= 3 && (got.Ok || got.Error != 40) {
			t.Fatalf("notice case %d: %+v", index, got)
		}
		if index >= 4 && (got.Ok || got.Error != 41) {
			t.Fatalf("marker case %d: %+v", index, got)
		}
		emit(configuration.Input{NamePrefix: "x", UseDefaultLimit: true}, state, command, resources)
	}
	if count != 2066 {
		t.Fatalf("corpus count=%d", count)
	}
	if err := requests.Close(); err != nil {
		t.Fatal(err)
	}
	if err := observations.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*nativeResourceCorpusDirectory, "COMPLETE"), []byte(fmt.Sprintf("seme-go-upb06-native-v1\n%d\n", count)), 0600); err != nil {
		t.Fatal(err)
	}
	complete = true
}

func resourceValue(resources resource.Set) any {
	return map[string]any{"kind": "record", "fields": map[string]any{"Notice": i64Text(resources.Notice), "Marker": map[string]any{"kind": "bytes", "bytes_hex": hex.EncodeToString(resources.Marker)}}}
}
