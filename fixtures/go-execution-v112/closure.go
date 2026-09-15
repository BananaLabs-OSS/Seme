package closure

func Apply(read func() int64) int64 { return read() }

func Run(value int64) int64 {
	total := int64(0)
	total = value
	return Apply(func() int64 { return total })
}
