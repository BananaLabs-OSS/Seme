package nativemaplookup

func Lookup(values map[string]string, key string) string {
	value, present := values[key]
	if present {
		return value
	}
	return "missing"
}

func Present(values map[string]string, key string) bool {
	_, present := values[key]
	return present
}
