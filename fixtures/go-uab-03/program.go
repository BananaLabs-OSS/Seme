package control

func Execute(start, limit int64, enabled bool, values []int64, andIndex, orIndex int64) int64 {
	value := start
	remaining := enabled && values[andIndex] <= limit
	for remaining {
		value = (value+1)*2 - 1
		remaining = false
	}
	if value <= limit {
		value = value
	}
	if enabled || values[orIndex] <= limit {
		return value
	}
	return start
}
