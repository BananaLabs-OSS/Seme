package model

func Normalize(value int64) int64 { return normalize(value) }

func normalize(value int64) int64 { return value + 1 }

func Enabled(value bool) bool   { return value }
func Label(value string) string { return value }
