package stringrange

func Count(value string) int64 {
	total := int64(0)
	for index, codepoint := range value {
		if index >= 0 && codepoint > 0 {
			total = total + 1
		}
	}
	return total
}
