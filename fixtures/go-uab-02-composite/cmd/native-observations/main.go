package main

import (
	"encoding/json"
	"os"

	composite "example.test/go-uab-02-composite"
)

func main() {
	type result = composite.Result[[]byte, string]
	type option = composite.Option[result]
	observed := struct {
		None       bool `json:"none"`
		OK         bool `json:"ok"`
		WrongBytes bool `json:"wrong_bytes"`
		Error      bool `json:"error"`
		WrongError bool `json:"wrong_error"`
	}{
		composite.Admit(option{}),
		composite.Admit(option{Some: true, Value: result{Ok: true, Value: []byte("ok")}}),
		composite.Admit(option{Some: true, Value: result{Ok: true, Value: []byte("no")}}),
		composite.Admit(option{Some: true, Value: result{Error: "bad"}}),
		composite.Admit(option{Some: true, Value: result{Error: "no"}}),
	}
	if err := json.NewEncoder(os.Stdout).Encode(observed); err != nil {
		panic(err)
	}
}
