package guard

func Find(values []int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
