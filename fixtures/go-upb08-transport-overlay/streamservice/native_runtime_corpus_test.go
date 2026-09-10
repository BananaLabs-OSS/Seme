package streamservice

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"example.test/go-uab-11/transport"
)

var nativeTransportCorpusDirectory = flag.String("native-transport-corpus-dir", "", "new directory for deterministic ordered transport JSONL")

func TestNativeTransportRuntimeCorpus(t *testing.T) {
	if *nativeTransportCorpusDirectory == "" {
		t.Skip("native transport corpus output not requested")
	}
	if !filepath.IsAbs(*nativeTransportCorpusDirectory) || filepath.Clean(*nativeTransportCorpusDirectory) != *nativeTransportCorpusDirectory {
		t.Fatal("native corpus directory must be absolute and clean")
	}
	if err := os.Mkdir(*nativeTransportCorpusDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	complete := false
	t.Cleanup(func() {
		if !complete {
			_ = os.RemoveAll(*nativeTransportCorpusDirectory)
		}
	})
	requests := createTransportCorpusFile(t, "requests.jsonl")
	expected := createTransportCorpusFile(t, "expected.jsonl")
	rq, ex := json.NewEncoder(requests), json.NewEncoder(expected)
	count := int64(0)
	emit := func(before transport.State, value transport.Command) transport.Result {
		result := Dispatch(before, value)
		request := map[string]any{"arguments": []any{transportCorpusValue(reflect.ValueOf(before)), transportCorpusValue(reflect.ValueOf(value))}, "capabilities": []string{}}
		observation := map[string]any{"value": transportCorpusValue(reflect.ValueOf(result)), "effects": []any{}}
		if err := rq.Encode(request); err != nil {
			t.Fatal(err)
		}
		if err := ex.Encode(observation); err != nil {
			t.Fatal(err)
		}
		count++
		return result
	}
	initial := transport.NewState("match")
	invalid := []transport.Command{corpusCommand(0, 1), corpusCommand(257, 2), corpusCommand(2, 3)}
	zeroCorrelation := corpusCommand(1, 4)
	zeroCorrelation.Correlation = transport.Correlation{}
	zeroDigest := corpusCommand(1, 5)
	zeroDigest.Digest = transport.Digest{}
	wrongKind := corpusCommand(1, 6)
	wrongKind.Kind = 2
	emptyPayload := corpusCommand(1, 7)
	emptyPayload.PayloadWords, emptyPayload.PayloadByteLength = []int64{}, 0
	invalid = append(invalid, zeroCorrelation, zeroDigest, wrongKind, emptyPayload)
	for _, value := range invalid {
		emit(initial, value)
	}
	oversize := corpusCommand(1, 8000)
	oversize.PayloadWords = make([]int64, 384)
	oversize.PayloadWords[0], oversize.PayloadByteLength = 8000, 3073
	emit(initial, oversize)
	largeFirst := corpusCommand(1, 8100)
	largeFirst.PayloadWords = make([]int64, 384)
	largeFirst.PayloadWords[0], largeFirst.PayloadByteLength = 8100, 3072
	largeFirstResult := emit(initial, largeFirst)
	largeSecond := corpusCommand(2, 8200)
	largeSecond.PayloadWords = make([]int64, 128)
	largeSecond.PayloadWords[0], largeSecond.PayloadByteLength = 8200, 1024
	largeSecondResult := emit(largeFirstResult.State, largeSecond)
	emit(largeSecondResult.State, largeFirst)
	largeThird := corpusCommand(3, 8300)
	largeThird.PayloadWords, largeThird.PayloadByteLength = []int64{8300}, 1
	emit(largeSecondResult.State, largeThird)
	terminal := initial
	for sequence := int64(1); sequence <= 256; sequence++ {
		value := corpusCommand(sequence, 10000+sequence)
		terminal = transport.Commit(terminal, value, true, 0, sequence, 4).State
	}
	emit(terminal, corpusCommand(1, 10001))
	emit(terminal, corpusCommand(257, 11000))
	malformed := transport.CloneState(terminal)
	malformed.PayloadStarts[0] = 1
	emit(malformed, corpusCommand(1, 10001))
	current := transport.NewState("match")
	last := corpusCommand(1, 1)
	for index := int64(0); count < 4096; index++ {
		if index%32 == 0 {
			current = transport.NewState("match")
			last = corpusCommand(1, index+1)
		}
		value := corpusCommand(current.NextCommand, index+1)
		switch index % 32 {
		case 1:
			value = last
		case 2:
			value = last
			value.PayloadWords = append([]int64(nil), last.PayloadWords...)
			value.PayloadWords[0]++
		case 3:
			value.Sequence++
		case 4:
			value.Correlation = last.Correlation
		}
		if index%7 == 0 {
			value.Payload.Loaded.Version = 3
		}
		before := current
		result := emit(before, value)
		if result.OK && !result.Duplicate {
			current = result.State
			last = value
		}
	}
	if err := requests.Close(); err != nil {
		t.Fatal(err)
	}
	if err := expected.Close(); err != nil {
		t.Fatal(err)
	}
	requestDigest := transportCorpusDigest(t, filepath.Join(*nativeTransportCorpusDirectory, "requests.jsonl"))
	expectedDigest := transportCorpusDigest(t, filepath.Join(*nativeTransportCorpusDirectory, "expected.jsonl"))
	marker := fmt.Sprintf("seme-go-upb08-native-v1\ncount=4096\nrequests_sha256=%x\nexpected_sha256=%x\n", requestDigest, expectedDigest)
	if err := os.WriteFile(filepath.Join(*nativeTransportCorpusDirectory, "COMPLETE"), []byte(marker), 0600); err != nil {
		t.Fatal(err)
	}
	complete = true
}

func transportCorpusDigest(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

func corpusCommand(sequence, identity int64) transport.Command {
	value := command(sequence, identity)
	return value
}

func createTransportCorpusFile(t *testing.T, name string) *os.File {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(*nativeTransportCorpusDirectory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func transportCorpusValue(value reflect.Value) any {
	if value.Kind() == reflect.Pointer {
		return transportCorpusValue(value.Elem())
	}
	switch value.Kind() {
	case reflect.Bool:
		out := map[string]any{"kind": "bool"}
		if value.Bool() {
			out["bool"] = true
		}
		return out
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"kind": "i64", "i64": strconv.FormatInt(value.Int(), 10)}
	case reflect.String:
		out := map[string]any{"kind": "text"}
		if value.String() != "" {
			out["text"] = value.String()
		}
		return out
	case reflect.Struct:
		fields := map[string]any{}
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			if typeOf.Field(index).IsExported() {
				fields[typeOf.Field(index).Name] = transportCorpusValue(value.Field(index))
			}
		}
		return map[string]any{"kind": "record", "fields": fields}
	case reflect.Slice:
		items := make([]any, value.Len())
		for index := range items {
			items[index] = transportCorpusValue(value.Index(index))
		}
		out := map[string]any{"kind": "slice"}
		if len(items) != 0 {
			out["items"] = items
		}
		return out
	case reflect.Map:
		keys := value.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].Int() < keys[j].Int() })
		entries := make([]any, len(keys))
		for index, key := range keys {
			entries[index] = map[string]any{"key": transportCorpusValue(key), "value": transportCorpusValue(value.MapIndex(key))}
		}
		out := map[string]any{"kind": "map", "value_type": "i64"}
		if len(entries) != 0 {
			out["entries"] = entries
		}
		return out
	default:
		panic(fmt.Sprintf("unsupported corpus kind %s", value.Kind()))
	}
}
