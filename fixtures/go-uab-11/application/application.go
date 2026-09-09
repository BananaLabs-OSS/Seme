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
	if found {
		if 0 <= command.Index && command.Index <= int64(len(state.Values))-1 {
			index := command.Index
			result := policy.Evaluate(state.Values, command.Scale, command.Delta, command.Amount)
			if result.Ok {
				adjusted := result.Value
				values := append(slices.Replace(slices.Clone(state.Values), int(index), int(index)+1, adjusted), adjusted)
				newCounter := oldCounter + adjusted
				counters := func(input map[int64]int64, key, value int64) map[int64]int64 {
					output := maps.Clone(input)
					output[key] = value
					return output
				}(state.Counters, command.Key, newCounter)
				next := State{Name: state.Name, Values: values, Counters: counters}
				log.Print(true)
				return Outcome{Ok: true, Value: model.Transition[State, int64]{State: next, Result: newCounter}}
			}
			return Outcome{Error: result.Error}
		}
		return Outcome{Error: 2}
	}
	return Outcome{Error: 1}
}
