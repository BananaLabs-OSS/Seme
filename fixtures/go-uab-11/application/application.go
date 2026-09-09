package application

import (
	"log"
	"maps"
	"slices"

	"example.test/go-uab-11/model"
	"example.test/go-uab-11/policy"
)

type State struct {
	Name     string
	Values   []int64
	Counters map[int64]int64
}
type Command struct {
	Key, Index, Delta, Amount int64
	Scale                     bool
}
type Outcome = model.Result[model.Transition[State, int64], int64]

func Apply(state State, command Command) Outcome {
	oldCounter, found := state.Counters[command.Key]
	if !found {
		return model.Failure[model.Transition[State, int64]](int64(1))
	}
	if command.Index < 0 || command.Index >= int64(len(state.Values)) {
		return model.Failure[model.Transition[State, int64]](int64(2))
	}
	result := policy.Evaluate(state.Values, command.Scale, command.Delta, command.Amount)
	if !result.Ok {
		return model.Failure[model.Transition[State, int64]](result.Error)
	}
	adjusted := result.Value
	values := slices.Clone(state.Values)
	values[command.Index] = adjusted
	values = append(values, adjusted)
	counters := maps.Clone(state.Counters)
	newCounter := oldCounter + adjusted
	counters[command.Key] = newCounter
	next := State{Name: state.Name, Values: values, Counters: counters}
	log.Print(true)
	return model.Success[model.Transition[State, int64], int64](model.Transition[State, int64]{State: next, Result: newCounter})
}
