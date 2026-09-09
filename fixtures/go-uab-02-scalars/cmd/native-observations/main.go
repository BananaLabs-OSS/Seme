package main

import (
	"encoding/json"
	"os"

	scalars "example.test/go-uab-02-scalars"
)

func main() {
	observed := map[string]any{
		"i64_zero": scalars.ObserveI64(0), "i64_wrap": scalars.ObserveI64(9223372036854775807),
		"bool_false": scalars.ObserveBoolean(false), "bool_true": scalars.ObserveBoolean(true),
		"text_empty": scalars.ObserveText(""), "text_unicode": scalars.ObserveText("世界🚀"),
	}
	if err := json.NewEncoder(os.Stdout).Encode(observed); err != nil {
		panic(err)
	}
}
