package canonicaleval

import "fmt"

// CumulativeState and CumulativeCommand are the language-neutral boundary
// model for the UAB cumulative application. They contain no ecosystem syntax
// or fixture identity.
type CumulativeState struct {
	Name     string
	Values   []int64
	Counters map[int64]int64
}

type CumulativeCommand struct {
	Key, Index, Delta, Amount int64
	Scale                     bool
}

// EvaluateCumulative applies the frozen UAB application semantics. Every
// rejection is decided before copying/updating state or emitting an effect.
// Integer operations deliberately use Go's defined two's-complement int64
// wrap, matching canonical modular i64.
func EvaluateCumulative(state CumulativeState, command CumulativeCommand, authorized map[string]bool) (Value, []EffectObservation, error) {
	oldCounter, ok := state.Counters[command.Key]
	if !ok {
		return cumulativeError(1), nil, nil
	}
	if command.Index < 0 || command.Index >= int64(len(state.Values)) {
		return cumulativeError(2), nil, nil
	}
	var sum int64 // mutable captured accumulator
	for _, value := range state.Values {
		if value < 0 {
			return cumulativeError(3), nil, nil
		}
		sum += value
	}
	var policy int64
	if command.Scale {
		policy = sum * command.Delta
	} else {
		policy = sum + command.Delta
	}
	adjusted := policy + command.Amount // immutable captured callback
	if !authorized["observability.log"] {
		return Value{}, nil, fmt.Errorf("canonicaleval.effect_denied")
	}
	values := append([]int64(nil), state.Values...)
	values[command.Index] = adjusted
	values = append(values, adjusted)
	counters := make(map[int64]int64, len(state.Counters))
	for key, value := range state.Counters {
		counters[key] = value
	}
	newCounter := oldCounter + adjusted
	counters[command.Key] = newCounter
	next := CumulativeState{Name: state.Name, Values: values, Counters: counters}
	transition := Value{Kind: "transition"}
	stateValue, resultValue := cumulativeStateValue(next), Value{Kind: "i64", I64: fmt.Sprint(newCounter)}
	transition.State, transition.Result = &stateValue, &resultValue
	return Value{Kind: "result", Variant: "ok", Payload: &transition}, []EffectObservation{{Capability: "observability.log", Value: true}}, nil
}

func cumulativeError(code int64) Value {
	payload := Value{Kind: "i64", I64: fmt.Sprint(code)}
	return Value{Kind: "result", Variant: "error", Payload: &payload}
}

func cumulativeStateValue(state CumulativeState) Value {
	values := make([]Value, len(state.Values))
	for i, value := range state.Values {
		values[i] = Value{Kind: "i64", I64: fmt.Sprint(value)}
	}
	keys := make([]int64, 0, len(state.Counters))
	for key := range state.Counters {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	entries := make([]Entry, len(keys))
	for i, key := range keys {
		entries[i] = Entry{Key: Value{Kind: "i64", I64: fmt.Sprint(key)}, Value: Value{Kind: "i64", I64: fmt.Sprint(state.Counters[key])}}
	}
	return Value{Kind: "record", Fields: map[string]Value{"Name": {Kind: "text", Text: state.Name}, "Values": {Kind: "slice", Items: values}, "Counters": {Kind: "map", ValueType: "i64", Entries: entries}}}
}
