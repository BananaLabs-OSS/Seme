package transition

type Transition[S, R any] struct {
	State  S
	Result R
}

type Counter struct {
	Value int64
}

func Step(counter Counter, delta int64) Transition[Counter, int64] {
	return Transition[Counter, int64]{
		State:  Counter{Value: counter.Value + delta},
		Result: counter.Value,
	}
}
