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

func TestCanonicalVectors(t *testing.T) {
	type value struct {
		Kind string `json:"kind"`
		I64  string `json:"i64"`
	}
	type request struct {
		Arguments []value `json:"arguments"`
	}
	type response struct {
		Value value `json:"value"`
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
		var argument, output int64
		if len(in.Arguments) != 1 {
			t.Fatal("unsupported request")
		}
		if _, err = fmt.Sscan(in.Arguments[0].I64, &argument); err != nil {
			t.Fatal(err)
		}
		if _, err = fmt.Sscan(want.Value.I64, &output); err != nil {
			t.Fatal(err)
		}
		if got := Apply(argument); got != output {
			t.Fatalf("Apply(%d) = %d, want %d", argument, got, output)
		}
	}
	if err = rscan.Err(); err != nil {
		t.Fatal(err)
	}
	if escan.Scan() || escan.Err() != nil {
		t.Fatal("unmatched expected vector")
	}
}
