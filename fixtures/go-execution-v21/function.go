package arrayproof

func Pick(first, second, third, index int64) int64 {
	return [3]int64{first, second, third}[index]
}
