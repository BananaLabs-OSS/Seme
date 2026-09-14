package nativeindex

func Set(values map[string]int64, key string, value int64) int64 {
	values[key] = value
	return values[key]
}
