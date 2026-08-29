package greeting

// Add is deliberately ordinary Go. Core Execution v1 uses it as its first
// exact, independently executable provider lift.
func Add(a int64, b int64) int64 {
	return a + b
}
