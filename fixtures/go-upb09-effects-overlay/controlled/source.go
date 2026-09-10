package controlled

type ClockSample struct {
	Sequence         int64
	UnixMilliseconds int64
}

type RandomState struct {
	Value int64
	Draws int64
}

type Draw struct {
	Before RandomState
	After  RandomState
	Value  int64
}

func ValidClock(sample ClockSample) bool {
	return 1 <= sample.Sequence && sample.Sequence <= 256 && 0 <= sample.UnixMilliseconds && sample.UnixMilliseconds <= 4102444800000
}

func ValidSeed(state RandomState) bool {
	return 1 <= state.Value && state.Draws == 0
}

func ValidState(state RandomState) bool {
	return 0 <= state.Draws && state.Draws <= 256 && (state.Draws != 0 || 1 <= state.Value)
}

// Next is a versioned fixture algorithm expressed only with ordinary integer
// operations. Its state is explicit; it never reads host entropy.
func Next(state RandomState) Draw {
	next := state.Value*48271 + 1
	return Draw{Before: state, After: RandomState{Value: next, Draws: state.Draws + 1}, Value: next}
}
