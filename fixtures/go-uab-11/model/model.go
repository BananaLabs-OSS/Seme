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
