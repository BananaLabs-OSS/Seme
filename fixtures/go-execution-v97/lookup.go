package maplookup

func Lookup(values map[string]string, key string) string {
	if value, ok := values[key]; ok {
		return value
	}
	return "missing"
}
