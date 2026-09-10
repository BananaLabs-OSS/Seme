package controlled

type ClockSample struct {
	UnixMilliseconds int64
}

type RandomState struct {
	Value int64
}

type Draw struct {
	Before RandomState
	After  RandomState
	Value  int64
}

func ValidClock(sample ClockSample) bool {
	return 0 <= sample.UnixMilliseconds
}

func ValidRandom(state RandomState) bool {
	return state.Value != 0
}

// Next is a versioned fixture algorithm expressed only with ordinary integer
// operations. Its state is explicit; it never reads host entropy.
func Next(state RandomState) Draw {
	next := state.Value*6364136223846793005 + 1442695040888963407
	return Draw{Before: state, After: RandomState{Value: next}, Value: next}
}
