package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"seme.local/reference/canonicaleval"
)

func TestRequestRequiresCanonicalJSONShape(t *testing.T) {
	var input request
	if err := json.Unmarshal([]byte(`{"arguments":[],"capabilities":["observability.log"]}`), &input); err != nil || len(input.Capabilities) != 1 {
		t.Fatalf("input=%#v err=%v", input, err)
	}
}

func TestResponseEncodesNoEffectsAsOrderedEmptyArray(t *testing.T) {
	var output bytes.Buffer
	value := canonicaleval.Value{Kind: "result", Variant: "error"}
	if err := json.NewEncoder(&output).Encode(response{Value: value, Effects: []canonicaleval.EffectObservation{}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`"effects":[]`)) {
		t.Fatalf("response = %s", output.Bytes())
	}
}
