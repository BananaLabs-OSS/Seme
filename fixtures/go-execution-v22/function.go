package foldproof

func Sum(values [3]int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func SumEmpty(values [0]int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}
