package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"

	effects "example.test/go-uab-09"
)

type vectors struct {
	Valid []struct {
		Name                  string
		First, Second, Result bool
		Trace                 []bool
	} `json:"valid"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: native-observations VECTORS.json")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var cases vectors
	if err := json.Unmarshal(raw, &cases); err != nil {
		panic(err)
	}
	observed := map[string]any{}
	previousWriter, previousFlags := log.Writer(), log.Flags()
	defer func() { log.SetOutput(previousWriter); log.SetFlags(previousFlags) }()
	log.SetFlags(0)
	for _, item := range cases.Valid {
		var output bytes.Buffer
		log.SetOutput(&output)
		result := effects.Observe(item.First, item.Second)
		lines := strings.Fields(output.String())
		trace := make([]bool, len(lines))
		for i, line := range lines {
			trace[i] = line == "true"
			if line != "true" && line != "false" {
				panic("unexpected log")
			}
		}
		if result != item.Result || len(trace) != len(item.Trace) || trace[0] != item.Trace[0] || trace[1] != item.Trace[1] {
			panic("native effect mismatch")
		}
		observed[item.Name] = map[string]any{"result": result, "trace": trace}
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": observed}); err != nil {
		panic(err)
	}
}
