package cumulativestate

// Sum intentionally lives in a second source file. The cumulative proof uses
// it to verify package-wide, type-resolved lifting rather than single-file
// syntax recognition.
func Sum(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}
