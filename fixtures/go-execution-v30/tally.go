package runtimemap

func Tally(values []int64, key int64) int64 {
	counts := map[int64]int64{}
	for _, value := range values {
		counts[value] = counts[value] + 1
	}
	return counts[key]
}
