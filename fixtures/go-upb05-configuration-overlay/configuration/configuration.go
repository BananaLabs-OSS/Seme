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

func DefaultLimit() int64 { return 64 }

func DefaultNamePrefix() string { return "unit-" }

func InternalDefaultDisabled() bool { return false }

func ValidateNamePrefix(value string) model.Result[string, int64] {
	if value == "" {
		return model.Result[string, int64]{Error: 10}
	}
	return model.Result[string, int64]{Ok: true, Value: value}
}

func ValidateLimit(value int64) model.Result[int64, int64] {
	if value < 1 {
		return model.Result[int64, int64]{Error: 11}
	}
	if 4096 < value {
		return model.Result[int64, int64]{Error: 12}
	}
	return model.Result[int64, int64]{Ok: true, Value: value}
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
