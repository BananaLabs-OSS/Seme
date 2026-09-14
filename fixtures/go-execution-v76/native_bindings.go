package nativebindings

func Pair(value int64) (int64, *int64) {
	return value, &value
}

func Read(value int64) int64 {
	first, pointer := Pair(value)
	if current := pointer; current == pointer {
		return first
	}
	return 0
}
