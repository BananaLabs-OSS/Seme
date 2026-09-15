package call

func Run(base int64) int64 {
	return func(value int64) int64 {
		result := base + value
		return result
	}(2)
}
