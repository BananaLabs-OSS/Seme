package cumulativestate

type Transition[S, R any] struct {
	State  S
	Result R
}

type Accumulator struct {
	Value int64
}

func (state Accumulator) Add(delta int64) Transition[Accumulator, int64] {
	next := Accumulator{Value: state.Value + delta}
	return Transition[Accumulator, int64]{State: next, Result: next.Value}
}

func Sum(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func Run(state Accumulator, values []int64, enabled bool) Transition[Accumulator, int64] {
	delta := Sum(values)
	if enabled {
		return state.Add(delta)
	}
	return state.Add(0)
}
