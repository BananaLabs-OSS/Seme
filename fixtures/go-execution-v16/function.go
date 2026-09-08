package callproof

func decorate(value string) string {
	prefix := "[" + value
	return prefix + "]"
}

func combine(left, right string) string {
	return decorate(left) + decorate(right)
}

func Render(left, right string) string {
	joined := combine(left, right)
	return decorate(joined)
}
