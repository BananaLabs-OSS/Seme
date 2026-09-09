// Command canonical-observe executes one canonical program request and emits
// its value plus ordered effect trace. It is language-independent harness
// plumbing; program identity and behavior are discovered from the graph.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wire"
)

type request struct {
	Arguments    []canonicaleval.Value `json:"arguments"`
	Capabilities []string              `json:"capabilities"`
}
type response struct {
	Value   canonicaleval.Value               `json:"value"`
	Effects []canonicaleval.EffectObservation `json:"effects"`
}

func main() {
	if len(os.Args) != 2 {
		fatal(fmt.Errorf("usage: canonical-observe PROGRAM.seme"))
	}
	data, err := os.ReadFile(os.Args[1])
	fatal(err)
	graph, err := wire.Decode(data)
	fatal(err)
	decoder, encoder := json.NewDecoder(os.Stdin), json.NewEncoder(os.Stdout)
	count := 0
	for {
		var input request
		err := decoder.Decode(&input)
		if err == io.EOF {
			break
		}
		fatal(err)
		count++
		authorized := map[string]bool{}
		for _, capability := range input.Capabilities {
			if capability == "" || authorized[capability] {
				fatal(fmt.Errorf("canonicaleval.capability_list"))
			}
			authorized[capability] = true
		}
		value, effects, err := canonicaleval.EvaluateAuthorized(graph, input.Arguments, authorized)
		fatal(err)
		if effects == nil {
			effects = []canonicaleval.EffectObservation{}
		}
		fatal(encoder.Encode(response{Value: value, Effects: effects}))
	}
	if count == 0 {
		fatal(fmt.Errorf("canonicaleval.requires_request"))
	}
}
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "canonical-observe:", err)
		os.Exit(65)
	}
}
