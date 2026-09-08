package mutationproof

func AppendOnce(value, suffix string, enabled bool) string {
	result := value
	remaining := enabled
	for remaining {
		result = result + suffix
		remaining = false
	}
	return result
}
