package methodtransition

type Transition[S, R any] struct {
	State  S
	Result R
}

type Counter struct {
	Value int64
}

func (counter Counter) Add(delta int64) Transition[Counter, int64] {
	next := Counter{Value: counter.Value + delta}
	return Transition[Counter, int64]{State: next, Result: next.Value}
}

func Step(counter Counter, delta int64) Transition[Counter, int64] {
	return counter.Add(delta)
}
