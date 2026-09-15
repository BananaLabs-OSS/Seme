package closure

func Run(base int64) func(int64) int64 {
	return func(value int64) int64 {
		total := base + value
		return total
	}
}
