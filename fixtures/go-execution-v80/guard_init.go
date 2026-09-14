package guardinit

func Find(values []int64) int64 {
	for _, value := range values {
		if positive := value > 0; positive {
			return value
		}
	}
	return 0
}
