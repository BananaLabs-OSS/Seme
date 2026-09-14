package nativeslice

func Window(values []int64, low int, high int) []int64 {
	return values[low:high]
}
