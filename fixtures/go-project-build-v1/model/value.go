package model

// Normalize is the leaf package's deterministic semantic operation.
func Normalize(value int64) int64 { return value + 1 }

// Enabled and Label make the bounded project exercise the provider's scalar
// boolean and text types as owned program semantics too.
func Enabled(value bool) bool   { return value }
func Label(value string) string { return value }
