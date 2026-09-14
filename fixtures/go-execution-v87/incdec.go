package incdec

func Adjust(value int64) int64 {
	result := value
	result++
	result--
	result++
	return result
}

func NativeAdjust() int {
	value := 0
	value++
	return value
}
