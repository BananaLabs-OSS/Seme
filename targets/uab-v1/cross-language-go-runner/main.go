// This adapter is copied beside a Go projection of the shared UAB-v1
// application. It translates only the language-neutral evidence JSON boundary;
// all application behavior remains in the projected native Go package.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"

	application "seme.uab11/application"
)

type value struct {
	Kind      string           `json:"kind"`
	I64       string           `json:"i64,omitempty"`
	Bool      bool             `json:"bool,omitempty"`
	Text      string           `json:"text,omitempty"`
	Items     []value          `json:"items,omitempty"`
	Fields    map[string]value `json:"fields,omitempty"`
	Entries   []entry          `json:"entries,omitempty"`
	ValueType string           `json:"value_type,omitempty"`
	Variant   string           `json:"variant,omitempty"`
	Payload   *value           `json:"payload,omitempty"`
	State     *value           `json:"state,omitempty"`
	Result    *value           `json:"result,omitempty"`
}
type entry struct {
	Key   value `json:"key"`
	Value value `json:"value"`
}
type request struct {
	Arguments []value `json:"arguments"`
}
type effect struct {
	Capability string `json:"capability"`
	Value      bool   `json:"value"`
}
type observation struct {
	Value   value    `json:"value"`
	Effects []effect `json:"effects"`
}
type traceWriter struct{ count int }

func (writer *traceWriter) Write(data []byte) (int, error) {
	writer.count++
	return len(data), nil
}

func main() {
	log.SetFlags(0)
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var input request
		must(json.Unmarshal(scanner.Bytes(), &input))
		if len(input.Arguments) != 2 {
			panic("cross_language_go.arity")
		}
		state := decodeState(input.Arguments[0])
		before := cloneState(state)
		command := decodeCommand(input.Arguments[1])
		writer := &traceWriter{}
		log.SetOutput(writer)
		result := application.Apply(state, command)
		if !reflect.DeepEqual(state, before) {
			panic("cross_language_go.input_mutated")
		}
		out := observation{Effects: []effect{}}
		if result.Ok {
			if writer.count != 1 {
				panic("cross_language_go.success_trace")
			}
			payload := transitionValue(result.Value.State, result.Value.Result)
			out.Value = resultValue("ok", payload)
			out.Effects = append(out.Effects, effect{Capability: "observability.log", Value: true})
		} else {
			if writer.count != 0 {
				panic("cross_language_go.error_trace")
			}
			out.Value = resultValue("error", i64Value(result.Error))
		}
		must(encoder.Encode(out))
	}
	must(scanner.Err())
}

func decodeState(input value) application.State {
	require(input.Kind == "record", "state")
	state := application.State{Name: input.Fields["Name"].Text, Values: []int64{}, Counters: map[int64]int64{}}
	for _, item := range input.Fields["Values"].Items {
		state.Values = append(state.Values, decodeI64(item))
	}
	for _, item := range input.Fields["Counters"].Entries {
		state.Counters[decodeI64(item.Key)] = decodeI64(item.Value)
	}
	return state
}

func decodeCommand(input value) application.Command {
	require(input.Kind == "record", "command")
	return application.Command{
		Key: decodeI64(input.Fields["Key"]), Index: decodeI64(input.Fields["Index"]),
		Delta: decodeI64(input.Fields["Delta"]), Amount: decodeI64(input.Fields["Amount"]),
		Scale: input.Fields["Scale"].Bool,
	}
}

func decodeI64(input value) int64 {
	require(input.Kind == "i64", "i64")
	result, err := strconv.ParseInt(input.I64, 10, 64)
	must(err)
	return result
}

func i64Value(input int64) value { return value{Kind: "i64", I64: strconv.FormatInt(input, 10)} }

func stateValue(state application.State) value {
	items := make([]value, len(state.Values))
	for index, item := range state.Values {
		items[index] = i64Value(item)
	}
	keys := make([]int64, 0, len(state.Counters))
	for key := range state.Counters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	entries := make([]entry, len(keys))
	for index, key := range keys {
		entries[index] = entry{Key: i64Value(key), Value: i64Value(state.Counters[key])}
	}
	return value{Kind: "record", Fields: map[string]value{
		"Name":     {Kind: "text", Text: state.Name},
		"Values":   {Kind: "slice", Items: items},
		"Counters": {Kind: "map", ValueType: "i64", Entries: entries},
	}}
}

func transitionValue(state application.State, result int64) value {
	stateOutput, resultOutput := stateValue(state), i64Value(result)
	return value{Kind: "transition", State: &stateOutput, Result: &resultOutput}
}

func resultValue(variant string, payload value) value {
	return value{Kind: "result", Variant: variant, Payload: &payload}
}

func cloneState(state application.State) application.State {
	copyState := application.State{Name: state.Name, Values: append([]int64(nil), state.Values...), Counters: map[int64]int64{}}
	for key, item := range state.Counters {
		copyState.Counters[key] = item
	}
	return copyState
}

func require(ok bool, name string) {
	if !ok {
		panic(fmt.Sprintf("cross_language_go.%s", name))
	}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
