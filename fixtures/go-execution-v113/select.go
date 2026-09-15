package selectreceive

func Run(ch <-chan int64) int64 {
	result := int64(0)
	select {
	case value, ok := <-ch:
		if ok {
			result = value
		}
	default:
		result = 1
	}
	return result
}
