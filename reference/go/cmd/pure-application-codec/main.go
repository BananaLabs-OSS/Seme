// pure-application-codec translates JSON evidence values to and from the
// recursive binary boundary certified from an ordinary canonical graph.
package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

type request struct {
	Arguments []canonicaleval.Value `json:"arguments"`
}

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "encode" && os.Args[1] != "decode" && os.Args[1] != "observe") {
		fatal(fmt.Errorf("usage: pure-application-codec encode|decode|observe PROGRAM.seme"))
	}
	data, err := os.ReadFile(os.Args[2])
	fatal(err)
	graph, err := wire.Decode(data)
	fatal(err)
	boundary, err := wasmtarget.CertifyPureApplicationBoundary(graph)
	fatal(err)
	if os.Args[1] == "encode" {
		decoder := json.NewDecoder(os.Stdin)
		for {
			var input request
			err := decoder.Decode(&input)
			if err == io.EOF {
				return
			}
			fatal(err)
			encoded, err := wasmtarget.EncodePureApplicationRequest(boundary, input.Arguments)
			fatal(err)
			fmt.Println(hex.EncodeToString(encoded))
		}
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), int(wasmtarget.PureValueMaximumHexLineSize))
	for scanner.Scan() {
		parts := strings.Split(strings.TrimSpace(scanner.Text()), "\t")
		data, err := hex.DecodeString(parts[0])
		fatal(err)
		value, err := wasmtarget.DecodePureValue(boundary.Result, data)
		fatal(err)
		if os.Args[1] == "decode" {
			fatal(json.NewEncoder(os.Stdout).Encode(value))
			continue
		}
		if len(parts) != 2 || len(boundary.RequiredCapabilities) > 1 {
			fatal(fmt.Errorf("observe supports zero or one capability and a Boolean trace"))
		}
		var observed []bool
		fatal(json.Unmarshal([]byte(parts[1]), &observed))
		if len(boundary.RequiredCapabilities) == 0 && len(observed) != 0 {
			fatal(fmt.Errorf("observe received an undeclared capability trace"))
		}
		effects := make([]canonicaleval.EffectObservation, len(observed))
		for i, item := range observed {
			effects[i] = canonicaleval.EffectObservation{Capability: boundary.RequiredCapabilities[0], Value: item}
		}
		fatal(json.NewEncoder(os.Stdout).Encode(struct {
			Value   canonicaleval.Value               `json:"value"`
			Effects []canonicaleval.EffectObservation `json:"effects"`
		}{value, effects}))
	}
	fatal(scanner.Err())
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "pure-application-codec:", err)
		os.Exit(65)
	}
}
