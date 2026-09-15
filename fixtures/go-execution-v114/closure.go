package nestedclosure

func Run(value int64) int64 {
	decorate := func(fn func(int64) int64) func(int64) int64 {
		return func(input int64) int64 {
			return fn(input) + value
		}
	}
	addOne := func(input int64) int64 { return input + 1 }
	return decorate(addOne)(2)
}
