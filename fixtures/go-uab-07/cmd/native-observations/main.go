package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	transition "example.test/go-uab-07"
)

type vectors struct { Valid []struct { Name, Value, Delta, State, Result string } `json:"valid"` }

func main() {
	if len(os.Args) != 2 { panic("usage: native-observations VECTORS.json") }
	raw, err := os.ReadFile(os.Args[1]); if err != nil { panic(err) }
	var cases vectors; if err := json.Unmarshal(raw, &cases); err != nil { panic(err) }
	observed := map[string]map[string]string{}
	for _, item := range cases.Valid {
		value, err := strconv.ParseInt(item.Value, 10, 64); if err != nil { panic(err) }
		delta, err := strconv.ParseInt(item.Delta, 10, 64); if err != nil { panic(err) }
		original := transition.Counter{Value: value}; got := transition.Step(original, delta)
		state, result := fmt.Sprint(got.State.Value), fmt.Sprint(got.Result)
		if state != item.State || result != item.Result || original.Value != value { panic("native transition mismatch") }
		observed[item.Name] = map[string]string{"state": state, "result": result}
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": observed}); err != nil { panic(err) }
}
