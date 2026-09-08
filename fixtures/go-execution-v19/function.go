package choiceproof

func Choose(original, replacement int64, enabled bool) int64 {
	result := original
	if enabled {
		result = replacement
	}
	return result
}
