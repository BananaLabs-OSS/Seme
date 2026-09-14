package parallelfields

type Pair struct {
	Left  string
	Right string
}

func Swap(pair *Pair) {
	pair.Left, pair.Right = pair.Right, pair.Left
}
