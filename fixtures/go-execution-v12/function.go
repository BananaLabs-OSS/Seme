package purefunction

func Allowed(enabled bool, current, limit int64) bool {
	return enabled && current <= limit
}
