package nativefor

func Sum(limit int) int {
	total := 0
	for index := 0; index < limit; index++ {
		total += index
	}
	return total
}
