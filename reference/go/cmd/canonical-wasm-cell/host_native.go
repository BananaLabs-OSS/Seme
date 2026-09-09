//go:build !wasip1

package main

// hostLogBool keeps native package tests useful; target evidence always uses
// the imported capability implementation in host_wasm.go.
func hostLogBool(value uint32) uint32 {
	_ = value
	return 0
}
