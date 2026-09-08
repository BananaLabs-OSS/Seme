package immutableclosure

type Unary func(int64) int64

func MakeAdder(base int64) Unary {
	return func(value int64) int64 {
		return base + value
	}
}

func Apply(fn Unary, value int64) int64 {
	return fn(value)
}

func Run(base, value int64) int64 {
	return Apply(MakeAdder(base), value)
}
