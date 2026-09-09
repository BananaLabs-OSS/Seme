package composite

import "bytes"

type Option[T any] struct {
	Some  bool
	Value T
}

type Result[T, E any] struct {
	Ok    bool
	Value T
	Error E
}

func Admit(request Option[Result[[]byte, string]]) bool {
	if request.Some {
		if request.Value.Ok {
			return bytes.Equal(request.Value.Value, []byte{'o', 'k'})
		}
		return request.Value.Error == "bad"
	}
	return false
}
