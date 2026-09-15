package emptyrecord

type Entry struct {
	Count int64
	Name  string
}

func Empty() Entry {
	return Entry{}
}
