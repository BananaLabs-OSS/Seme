// Package quota is an ordinary Go package with an observable logging effect.
package quota

import "log"

// Admit evaluates the quota policy and records its decision.
func Admit(current int64, delta int64, limit int64) bool {
	accepted := current+delta <= limit
	log.Printf("quota.accepted=%t", accepted)
	return accepted
}
