package localtype

func Read(value int64) int64 {
	type local struct {
		Value int64
	}
	item := local{Value: value}
	return item.Value
}
