package model

func Normalize(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
