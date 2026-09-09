package main

import (
	"encoding/json"
	"fmt"
	"os"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wire"
)

type vectorFile struct {
	Valid []struct {
		Name      string                `json:"name"`
		Arguments []canonicaleval.Value `json:"arguments"`
		Result    canonicaleval.Value   `json:"result"`
	} `json:"valid"`
	Malformed []struct {
		Name      string                `json:"name"`
		Arguments []canonicaleval.Value `json:"arguments"`
	} `json:"malformed"`
}

func main() {
	if len(os.Args) != 3 {
		fatal(fmt.Errorf("usage: canonical-eval PROGRAM.seme VECTORS.json"))
	}
	program, err := os.ReadFile(os.Args[1])
	fatal(err)
	graph, err := wire.Decode(program)
	fatal(err)
	raw, err := os.ReadFile(os.Args[2])
	fatal(err)
	var vectors vectorFile
	fatal(json.Unmarshal(raw, &vectors))
	if len(vectors.Valid) == 0 {
		fatal(fmt.Errorf("canonicaleval.requires_vectors"))
	}
	fatal(validateNames(vectors))
	observed := map[string]canonicaleval.Value{}
	for _, item := range vectors.Valid {
		if _, exists := observed[item.Name]; exists || item.Name == "" {
			fatal(fmt.Errorf("canonicaleval.vector_name"))
		}
		value, err := canonicaleval.Evaluate(graph, item.Arguments)
		fatal(err)
		if !same(value, item.Result) {
			fatal(fmt.Errorf("canonicaleval.mismatch:%s", item.Name))
		}
		observed[item.Name] = value
	}
	for _, item := range vectors.Malformed {
		if item.Name == "" {
			fatal(fmt.Errorf("canonicaleval.vector_name"))
		}
		if _, err := canonicaleval.Evaluate(graph, item.Arguments); err == nil {
			fatal(fmt.Errorf("canonicaleval.malformed_accepted:%s", item.Name))
		}
	}
	fatal(json.NewEncoder(os.Stdout).Encode(struct {
		Valid     map[string]canonicaleval.Value `json:"valid"`
		Malformed int                            `json:"malformed"`
	}{observed, len(vectors.Malformed)}))
}

func validateNames(v vectorFile) error {
	seen := map[string]bool{}
	for _, item := range v.Valid {
		if item.Name == "" || seen[item.Name] {
			return fmt.Errorf("canonicaleval.vector_name")
		}
		seen[item.Name] = true
	}
	for _, item := range v.Malformed {
		if item.Name == "" || seen[item.Name] {
			return fmt.Errorf("canonicaleval.vector_name")
		}
		seen[item.Name] = true
	}
	return nil
}
func same(a, b canonicaleval.Value) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "canonical-eval:", err)
		os.Exit(65)
	}
}
