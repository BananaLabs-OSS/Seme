package resource

import (
	"bytes"

	"example.test/go-uab-11/model"
)

type Set struct {
	Notice string
	Marker []byte
}

func Validate(resources Set) model.Result[bool, int64] {
	if resources.Notice != "Seme resources — 世界\n" {
		return model.Result[bool, int64]{Error: 40}
	}
	if !bytes.Equal(resources.Marker, []byte{0x00, 0xff, 'S', 'E', 'M', 'E', '\n'}) {
		return model.Result[bool, int64]{Error: 41}
	}
	return model.Result[bool, int64]{Ok: true, Value: true}
}
