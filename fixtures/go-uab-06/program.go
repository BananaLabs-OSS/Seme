package closures

type Unary func(int64) int64

func MakeAdder(base int64) Unary           { return func(value int64) int64 { return base + value } }
func ImmutableRun(base, value int64) int64 { return MakeAdder(base)(value) }

type Counter func(int64) int64

func MakeCounter(start int64) Counter {
	value := start
	return func(delta int64) int64 { value = value + delta; return value }
}
func MutableRun(start, first, second int64) int64 {
	counter := MakeCounter(start)
	counter(first)
	return counter(second)
}
