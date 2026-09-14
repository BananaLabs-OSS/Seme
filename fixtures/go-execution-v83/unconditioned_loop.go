package unconditionedloop

func CountTo(limit int64) int64 {
	count := int64(0)
	for {
		if count >= limit {
			break
		}
		count += 1
	}
	return count
}
