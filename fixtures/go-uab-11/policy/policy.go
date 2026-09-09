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
func makeCounter(start int64) func(int64) int64 {
	total := start
	return func(value int64) int64 { total = total + value; return total }
}

func valid(values []int64, index int64) bool {
	if int64(len(values)) <= index {
		return true
	}
	if 0 <= values[index] {
		return valid(values, index+1)
	}
	return false
}

func sum(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func Evaluate(values []int64, scale bool, delta, amount int64) model.Result[int64, int64] {
	if valid(values, 0) {
		total := sum(values)
		if scale {
			return model.Result[int64, int64]{Ok: true, Value: makeAdder(amount)(adjust(Scale{Delta: delta}, total))}
		}
		return model.Result[int64, int64]{Ok: true, Value: makeAdder(amount)(adjust(Offset{Delta: delta}, total))}
	}
	return model.Result[int64, int64]{Error: 3}
}
