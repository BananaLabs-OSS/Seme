package reject

func Unsupported(value int64) int64 {
	goto done
done:
	return value
}
