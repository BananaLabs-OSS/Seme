package configuration

import "example.test/go-uab-11/model"

type Input struct {
	NamePrefix      string
	Limit           int64
	UseDefaultLimit bool
}

type Settings struct {
	NamePrefix string
	Limit      int64
}

type Initialized struct {
	Settings Settings
	Stage    int64
}

func Resolve(input Input) model.Result[Settings, int64] {
	if input.NamePrefix == "" {
		return model.Result[Settings, int64]{Error: 10}
	}
	limit := input.Limit
	if input.UseDefaultLimit {
		limit = 64
	}
	if limit < 1 {
		return model.Result[Settings, int64]{Error: 11}
	}
	if 4096 < limit {
		return model.Result[Settings, int64]{Error: 12}
	}
	return model.Result[Settings, int64]{Ok: true, Value: Settings{NamePrefix: input.NamePrefix, Limit: limit}}
}

func Initialize(input Input) model.Result[Initialized, int64] {
	resolved := Resolve(input)
	if !resolved.Ok {
		return model.Result[Initialized, int64]{Error: resolved.Error}
	}
	return model.Result[Initialized, int64]{Ok: true, Value: Initialized{Settings: resolved.Value, Stage: 1}}
}
