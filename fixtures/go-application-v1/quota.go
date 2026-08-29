// Package quota provides a small admission policy with no runtime dependencies
// or effects. It is ordinary Go and contains no Seme-specific source.
package quota

// Admit reports whether applying delta stays within limit. Addition follows
// Go's fixed-width signed wrapping semantics.
func Admit(current int64, delta int64, limit int64) bool {
	return current+delta <= limit
}
