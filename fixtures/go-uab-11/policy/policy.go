package policy

import "example.test/go-uab-11/model"

type Adjuster interface{ Adjust(int64) int64 }
type Offset struct{ Delta int64 }
type Scale struct{ Delta int64 }

func (value Offset) Adjust(input int64) int64  { return input + value.Delta }
func (value Scale) Adjust(input int64) int64   { return input * value.Delta }
func adjust(value Adjuster, input int64) int64 { return value.Adjust(input) }

func makeAdder(amount int64) func(int64) int64 {
	return func(value int64) int64 { return value + amount }
}
func makeCounter() func(int64) int64 {
	var total int64
	return func(value int64) int64 { total += value; return total }
}

func Evaluate(values []int64, scale bool, delta, amount int64) model.Result[int64, int64] {
	accumulate := makeCounter()
	var total int64
	for _, value := range values {
		if value < 0 {
			return model.Failure[int64](int64(3))
		}
		total = accumulate(value)
	}
	var selected Adjuster = Offset{Delta: delta}
	if scale {
		selected = Scale{Delta: delta}
	}
	return model.Success[int64, int64](makeAdder(amount)(adjust(selected, total)))
}
