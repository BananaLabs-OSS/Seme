package nativemaprange

func Total(values map[string]int64) int64 {
	total := int64(0)
	for _, value := range values {
		total = total + value
	}
	return total
}
