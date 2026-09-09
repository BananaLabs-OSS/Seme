package fallible

type Result[T, E any] struct {
	Ok    bool
	Value T
	Error E
}

func CheckPositive(value int64) Result[int64, int64] {
	if value <= 0 {
		return Result[int64, int64]{Ok: false, Error: 99}
	}
	return Result[int64, int64]{Ok: true, Value: value}
}

func PropagateIncrement(checked Result[int64, int64]) Result[int64, int64] {
	if checked.Ok {
		return Result[int64, int64]{Ok: true, Value: checked.Value + 1}
	}
	return Result[int64, int64]{Ok: false, Error: checked.Error}
}

func IncrementPositive(value int64) Result[int64, int64] {
	return PropagateIncrement(CheckPositive(value))
}
