package main

import (
	"seme.local/reference/canonicaleval"
	"testing"
)

func TestVectorNamesAreUniqueAcrossValidAndMalformed(t *testing.T) {
	var input vectorFile
	input.Valid = append(input.Valid, struct {
		Name      string                `json:"name"`
		Arguments []canonicaleval.Value `json:"arguments"`
		Result    canonicaleval.Value   `json:"result"`
	}{Name: "same"})
	input.Malformed = append(input.Malformed, struct {
		Name      string                `json:"name"`
		Arguments []canonicaleval.Value `json:"arguments"`
	}{Name: "same"})
	if validateNames(input) == nil {
		t.Fatal("duplicate vector name accepted")
	}
}
