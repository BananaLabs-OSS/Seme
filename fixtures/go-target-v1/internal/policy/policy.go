// Package policy is the imported-package boundary used by the next lifting
// stage. Provider v1 already revision-tracks it before projecting its symbols.
package policy

func Allows(current, delta, limit int64) bool {
	return limit >= delta+current
}
