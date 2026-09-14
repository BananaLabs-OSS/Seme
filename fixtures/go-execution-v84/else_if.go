package elseif

func Classify(value int64) int64 {
	result := int64(0)
	if value < 0 {
		result = 2
	} else if value > 0 {
		result = 1
	}
	return result
}
