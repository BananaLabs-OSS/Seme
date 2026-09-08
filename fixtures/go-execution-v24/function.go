package collectionquery

func LastOr(values []int64, fallback int64) int64 {
	if len(values) <= 0 {
		return fallback
	}
	return values[len(values)-1]
}
