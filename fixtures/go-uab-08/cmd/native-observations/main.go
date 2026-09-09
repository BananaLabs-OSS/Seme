package main

import (
	"encoding/json"
	"os"
	"strconv"

	fallible "example.test/go-uab-08"
)

type vectors struct { Valid []struct { Name, Value, Variant, Payload string } `json:"valid"` }
func main() {
	if len(os.Args) != 2 { panic("usage: native-observations VECTORS.json") }
	raw, err := os.ReadFile(os.Args[1]); if err != nil { panic(err) }; var cases vectors; if err := json.Unmarshal(raw, &cases); err != nil { panic(err) }
	observed := map[string]map[string]string{}
	for _, item := range cases.Valid {
		value, err := strconv.ParseInt(item.Value, 10, 64); if err != nil { panic(err) }; got := fallible.IncrementPositive(value)
		variant, payload := "error", strconv.FormatInt(got.Error, 10); if got.Ok { variant, payload = "ok", strconv.FormatInt(got.Value, 10) }
		if variant != item.Variant || payload != item.Payload { panic("native result mismatch") }; observed[item.Name] = map[string]string{"variant": variant, "payload": payload}
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": observed}); err != nil { panic(err) }
}
