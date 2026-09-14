package mixedshort

func Pair(value int64, failure error) (int64, error) {
	return value, failure
}

func Read(value int64, failure error) int64 {
	err := failure
	first, err := Pair(value, err)
	if err != nil {
		return 0
	}
	return first
}
