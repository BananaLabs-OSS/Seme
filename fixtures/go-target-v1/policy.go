package quota

// WithinLimit is kept in a separate ordinary Go file so Seme must resolve the
// package call graph rather than recognizing one monolithic source file.
func WithinLimit(current, delta, limit int64) bool {
	return current+delta <= limit
}
