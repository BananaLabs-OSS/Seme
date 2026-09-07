package boolean

func WithinAndEnabled(current, limit int64) bool {
	return true && current <= limit
}
