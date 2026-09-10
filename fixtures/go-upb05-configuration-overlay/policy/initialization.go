package policy

import (
	"example.test/go-uab-11/configuration"
	"example.test/go-uab-11/model"
)

type Initialized struct {
	Limit int64
	Stage int64
}

func Initialize(input configuration.Initialized) model.Result[Initialized, int64] {
	if input.Stage != 1 {
		return model.Result[Initialized, int64]{Error: 20}
	}
	return model.Result[Initialized, int64]{Ok: true, Value: Initialized{Limit: input.Settings.Limit, Stage: 2}}
}
