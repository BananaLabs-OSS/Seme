package cumulativetext

func Sum(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func Describe(prefix string, values []int64) string {
	total := Sum(values)
	if total <= 0 {
		return prefix + ":non-positive"
	}
	return prefix + ":positive"
}
