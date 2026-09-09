//go:build wasip1

package main

//go:wasmimport pulp log_bool
func hostLogBool(value uint32) uint32
