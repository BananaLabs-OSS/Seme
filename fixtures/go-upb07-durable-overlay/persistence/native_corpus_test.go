package persistence

import (
	"encoding/json"
	"example.test/go-uab-11/application"
	"example.test/go-uab-11/state"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

var nativeCorpusDirectory = flag.String("native-corpus-dir", "", "new directory for deterministic durable planner JSONL")

func TestNativeDurableCorpus(t *testing.T) {
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
	requests := createCorpusFile(t, "requests.jsonl")
	expected := createCorpusFile(t, "expected.jsonl")
	rq, ex := json.NewEncoder(requests), json.NewEncoder(expected)
	count := 0
	emit := func(grants Grants, loaded Loaded, initial state.V2, digest, key string) {
		t.Helper()
		plan := BuildPlan(grants, loaded, initial, digest, key)
		request := map[string]any{"arguments": []any{grantsCorpusValue(grants), loadedCorpusValue(loaded), stateV2CorpusValue(initial), textCorpusValue(digest), textCorpusValue(key)}, "capabilities": []string{}}
		observation := map[string]any{"value": planCorpusValue(plan), "effects": []any{}}
		if err := rq.Encode(request); err != nil {
			t.Fatal(err)
		}
		if err := ex.Encode(observation); err != nil {
			t.Fatal(err)
		}
		count++
	}
	base := application.State{Name: "pilot", Values: []int64{1, -1, 2}, Counters: map[int64]int64{1: 2, 7: 9}}
	initial := state.V2{State: base, Revision: 1}
	for index := 0; index < 32; index++ {
		loaded := Loaded{Found: true, Version: 2, Token: "boundary-token-" + strconv.Itoa(index), Digest: "sha256:old", V2: state.V2{State: base, Revision: int64(index + 1)}}
		grants, key := Grants{true, true}, "slot"
		switch index {
		case 0:
			loaded = Loaded{}
		case 1:
			loaded = Loaded{Found: true, Version: 1, Token: "migration", V1: state.V1{State: base}}
		case 2:
			loaded.Version = 3
		case 3:
			loaded.V2.Revision = 0
		case 4:
			loaded.V2.Revision = math.MaxInt64
		case 5:
			loaded.V1 = state.V1{}
			loaded.Version = 1
		case 6:
			grants.Read = false
		case 7:
			grants.CompareExchange = false
		case 8:
			key = ""
		}
		emit(grants, loaded, initial, "sha256:next-"+strconv.Itoa(index), key)
	}
	seed := uint64(0x5eed1234abcdef01)
	for sequence := 0; sequence < 2048; sequence++ {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		revision := int64(seed & 0x3fffffff)
		if revision == 0 {
			revision = 1
		}
		generated := application.State{Name: "generated-" + strconv.Itoa(sequence), Values: []int64{int64(seed), int64(seed >> 1)}, Counters: map[int64]int64{int64(sequence % 11): int64(seed >> 3)}}
		loaded := Loaded{Found: true, Version: 2, Token: "opaque-" + strconv.Itoa(sequence), Digest: "sha256:prior-" + strconv.Itoa(sequence), V2: state.V2{State: generated, Revision: revision}}
		emit(Grants{true, true}, loaded, initial, "sha256:next-"+strconv.Itoa(sequence), "slot-"+strconv.Itoa(sequence))
	}
	if count != 2080 {
		t.Fatalf("count=%d", count)
	}
	if err := requests.Close(); err != nil {
		t.Fatal(err)
	}
	if err := expected.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*nativeCorpusDirectory, "COMPLETE"), []byte(fmt.Sprintf("seme-go-upb07-native-v1\n%d\n", count)), 0600); err != nil {
		t.Fatal(err)
	}
	complete = true
}

func createCorpusFile(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(*nativeCorpusDirectory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func i64CorpusValue(v int64) any {
	return map[string]any{"kind": "i64", "i64": strconv.FormatInt(v, 10)}
}
func boolCorpusValue(v bool) any   { return map[string]any{"kind": "bool", "bool": v} }
func textCorpusValue(v string) any { return map[string]any{"kind": "text", "text": v} }
func recordCorpusValue(fields map[string]any) any {
	return map[string]any{"kind": "record", "fields": fields}
}
func grantsCorpusValue(v Grants) any {
	return recordCorpusValue(map[string]any{"Read": boolCorpusValue(v.Read), "CompareExchange": boolCorpusValue(v.CompareExchange)})
}
func stateCorpusValue(v application.State) any {
	items := make([]any, len(v.Values))
	for i, item := range v.Values {
		items[i] = i64CorpusValue(item)
	}
	keys := make([]int64, 0, len(v.Counters))
	for key := range v.Counters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	entries := make([]any, len(keys))
	for i, key := range keys {
		entries[i] = map[string]any{"key": i64CorpusValue(key), "value": i64CorpusValue(v.Counters[key])}
	}
	return recordCorpusValue(map[string]any{"Name": textCorpusValue(v.Name), "Values": map[string]any{"kind": "slice", "items": items}, "Counters": map[string]any{"kind": "map", "value_type": "i64", "entries": entries}})
}
func stateV1CorpusValue(v state.V1) any {
	return recordCorpusValue(map[string]any{"State": stateCorpusValue(v.State)})
}
func stateV2CorpusValue(v state.V2) any {
	return recordCorpusValue(map[string]any{"State": stateCorpusValue(v.State), "Revision": i64CorpusValue(v.Revision)})
}
func loadedCorpusValue(v Loaded) any {
	return recordCorpusValue(map[string]any{"Found": boolCorpusValue(v.Found), "Version": i64CorpusValue(v.Version), "Token": textCorpusValue(v.Token), "Digest": textCorpusValue(v.Digest), "V1": stateV1CorpusValue(v.V1), "V2": stateV2CorpusValue(v.V2)})
}
func operationCorpusValue(v Operation) any {
	return recordCorpusValue(map[string]any{"Sequence": i64CorpusValue(v.Sequence), "Kind": i64CorpusValue(v.Kind), "Status": boolCorpusValue(v.Status), "Token": textCorpusValue(v.Token), "Digest": textCorpusValue(v.Digest), "Version": i64CorpusValue(v.Version), "Value": stateV2CorpusValue(v.Value)})
}
func planCorpusValue(v Plan) any {
	if !v.Ok {
		return map[string]any{"kind": "result", "variant": "error", "payload": i64CorpusValue(v.Error)}
	}
	decision := recordCorpusValue(map[string]any{"Value": stateV2CorpusValue(v.Value.Value), "Load": operationCorpusValue(v.Value.Load), "CompareExchange": operationCorpusValue(v.Value.CompareExchange)})
	return map[string]any{"kind": "result", "variant": "ok", "payload": decision}
}
