package model

type Result[T any, E any] struct {
	Ok    bool
	Value T
	Error E
}

type Transition[S any, R any] struct {
	State  S
	Result R
}

func Success[T any, E any](value T) Result[T, E]   { return Result[T, E]{Ok: true, Value: value} }
func Failure[T any, E any](failure E) Result[T, E] { return Result[T, E]{Error: failure} }
