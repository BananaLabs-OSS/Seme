package pointer

func Set(target *string, value string) {
	*target = value
}
