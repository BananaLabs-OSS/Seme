package mutableclosure

type Counter func(int64) int64

func MakeCounter(start int64) Counter {
	value := start
	return func(delta int64) int64 {
		value = value + delta
		return value
	}
}

func Run(start, first, second int64) int64 {
	counter := MakeCounter(start)
	counter(first)
	return counter(second)
}
