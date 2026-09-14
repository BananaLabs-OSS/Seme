package contextualnil

type item struct {
	value int
}

func consume(pointer *item, values []string, index map[string]int, signal chan int, callback func(int) int, opaque any) int64 {
	if pointer == nil && values == nil && index == nil && signal == nil && callback == nil && opaque == nil {
		return 1
	}
	return 0
}

func acceptMap(values map[string]any) int64 {
	return 1
}

func ContextualNil() int64 {
	var pointer *item = nil
	pointer = nil
	return consume(pointer, nil, nil, nil, nil, nil) + acceptMap(map[string]any{"value": nil})
}
