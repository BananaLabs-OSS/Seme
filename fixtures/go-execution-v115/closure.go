package recursiveclosure

func Run(n int64) int64 {
	total := int64(0)
	var walk func(int64) int64
	walk = func(value int64) int64 {
		total += value
		if value > 0 {
			return walk(value - 1)
		}
		return total
	}
	return walk(n)
}
