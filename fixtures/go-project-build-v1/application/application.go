package application

import "example.test/go-project-build-v1/policy"

// Apply is the root executable entry.
func Apply(value int64) int64 { return policy.Apply(value) - 3 }
