package localconstants

func Sum(value int64) int64 {
	const (
		one = 1
		two = 2
	)
	const label = "ready"
	if label == "ready" {
		return value + one + two
	}
	return 0
}
