package namedresults

func Resolve(value int64) (result int64, ok bool) {
	result = value + 1
	ok = true
	return result, ok
}
