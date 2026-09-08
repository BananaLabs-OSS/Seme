package localproof

func Scale(left, right int64) int64 {
	total := left + right
	scaled := total * 2
	return scaled - left
}
