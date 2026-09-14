package nativebinary

func Mix(value uint64, shift uint) uint64 {
	return (value << shift) | (value % 3)
}
