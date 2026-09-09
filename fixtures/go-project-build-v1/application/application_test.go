package application

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestApply(t *testing.T) {
	for _, test := range []struct{ input, want int64 }{{0, -1}, {4, 7}, {-3, -7}} {
		if got := Apply(test.input); got != test.want {
			t.Fatalf("Apply(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}

// TestCanonicalVectors makes the checked behavior vectors native Go evidence,
// rather than treating their expected responses as an unverified oracle.
func TestCanonicalVectors(t *testing.T) {
	type value struct {
		Kind string `json:"kind"`
		I64  string `json:"i64"`
	}
	type request struct {
		Arguments []value `json:"arguments"`
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
	for line := 1; rscan.Scan(); line++ {
		if !escan.Scan() {
			t.Fatalf("expected vectors ended before request %d", line)
		}
		var in request
		var want response
		if err := json.Unmarshal(rscan.Bytes(), &in); err != nil {
			t.Fatalf("request %d: %v", line, err)
		}
		if err := json.Unmarshal(escan.Bytes(), &want); err != nil {
			t.Fatalf("expected %d: %v", line, err)
		}
		if len(in.Arguments) != 1 || in.Arguments[0].Kind != "i64" || want.Value.Kind != "i64" || len(want.Effects) != 0 {
			t.Fatalf("vector %d has an unsupported shape", line)
		}
		var input, output int64
		if _, err := fmt.Sscan(in.Arguments[0].I64, &input); err != nil {
			t.Fatalf("request %d i64: %v", line, err)
		}
		if _, err := fmt.Sscan(want.Value.I64, &output); err != nil {
			t.Fatalf("expected %d i64: %v", line, err)
		}
		if got := Apply(input); got != output {
			t.Fatalf("vector %d: Apply(%d) = %d, want %d", line, input, got, output)
		}
	}
	if err := rscan.Err(); err != nil {
		t.Fatal(err)
	}
	if escan.Scan() {
		t.Fatal("expected vectors contain an unmatched response")
	}
	if err := escan.Err(); err != nil {
		t.Fatal(err)
	}
}
