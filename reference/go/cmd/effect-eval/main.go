package main

import (
	"encoding/json"
	"fmt"
	"os"
	"seme.local/reference/canonicaleval"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage")
	}
	raw, e := os.ReadFile(os.Args[1])
	fatal(e)
	g, e := wire.Decode(raw)
	fatal(e)
	vraw, e := os.ReadFile(os.Args[2])
	fatal(e)
	var vectors struct {
		Valid []struct {
			Name      string `json:"name"`
			Arguments []bool `json:"arguments"`
			Result    bool   `json:"result"`
			Trace     []bool `json:"trace"`
		} `json:"valid"`
	}
	fatal(json.Unmarshal(vraw, &vectors))
	out := map[string]any{}
	for _, x := range vectors.Valid {
		args := make([]canonicaleval.Value, len(x.Arguments))
		for i, v := range x.Arguments {
			args[i] = canonicaleval.Value{Kind: "bool", Bool: v}
		}
		result, trace, e := canonicaleval.EvaluateObserved(g, args, map[string]bool{"observability.log": true})
		fatal(e)
		seen := []bool{}
		for _, event := range trace {
			if event.Capability != "observability.log" {
				fatal(fmt.Errorf("capability"))
			}
			seen = append(seen, event.Value)
		}
		if result.Kind != "bool" || result.Bool != x.Result || fmt.Sprint(seen) != fmt.Sprint(x.Trace) {
			fatal(fmt.Errorf("mismatch:%s", x.Name))
		}
		out[x.Name] = map[string]any{"result": result.Bool, "trace": seen}
	}
	_, trace, e := canonicaleval.EvaluateObserved(g, []canonicaleval.Value{{Kind: "bool", Bool: true}, {Kind: "bool"}}, map[string]bool{})
	if e == nil || len(trace) != 0 {
		fatal(fmt.Errorf("denial"))
	}
	fatal(json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": out, "denied": "no-events"}))
}
func fatal(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(65)
	}
}
