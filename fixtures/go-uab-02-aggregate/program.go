package aggregate

type Bounds struct{ Bias int64 }

func Observe(bounds Bounds, fixed [2]int64, dynamic []int64, counts map[int64]int64, key int64) int64 {
	return bounds.Bias + fixed[1] + int64(len(dynamic)) + counts[key]
}
