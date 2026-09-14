package predeclaredmethod

func Text(failure error) string {
	return failure.Error()
}
