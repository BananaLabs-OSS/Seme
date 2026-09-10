package application

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestApplyVectors(t *testing.T) {
	type value struct {
		Kind string `json:"kind"`
		I64  string `json:"i64"`
	}
	type request struct {
		Arguments    []value  `json:"arguments"`
		Capabilities []string `json:"capabilities"`
	}
	type response struct {
		Value   value `json:"value"`
		Effects []any `json:"effects"`
	}
	requests, err := os.Open("../vectors/requests.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer requests.Close()
	expected, err := os.Open("../vectors/expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer expected.Close()
	rscan, escan := bufio.NewScanner(requests), bufio.NewScanner(expected)
	count := 0
	for rscan.Scan() {
		if !escan.Scan() {
			t.Fatal("missing expected vector")
		}
		var in request
		var want response
		if err = json.Unmarshal(rscan.Bytes(), &in); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(escan.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		if len(in.Arguments) != 1 || len(in.Capabilities) != 0 || in.Arguments[0].Kind != "i64" || want.Value.Kind != "i64" || len(want.Effects) != 0 {
			t.Fatal("vector is outside the pure signed-i64 profile")
		}
		input, err := strconv.ParseInt(in.Arguments[0].I64, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		output, err := strconv.ParseInt(want.Value.I64, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if got := Apply(input); got != output {
			t.Fatalf("Apply(%d) = %d, want %d", input, got, output)
		}
		count++
	}
	if err = rscan.Err(); err != nil {
		t.Fatal(err)
	}
	if escan.Scan() || escan.Err() != nil {
		t.Fatal("unmatched expected vector")
	}
	if count == 0 {
		t.Fatal("empty vector corpus")
	}
}
