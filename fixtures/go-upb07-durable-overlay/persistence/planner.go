package persistence

import (
	"example.test/go-uab-11/model"
	"example.test/go-uab-11/state"
)

type Grants struct {
	Read            bool
	CompareExchange bool
}
type Loaded struct {
	Found   bool
	Version int64
	Token   string
	Digest  string
	V1      state.V1
	V2      state.V2
}
type Operation struct {
	Sequence int64
	Kind     int64
	Status   bool
	Token    string
	Digest   string
	Version  int64
	Value    state.V2
}
type Decision struct {
	Value           state.V2
	Load            Operation
	CompareExchange Operation
}
type Plan = model.Result[Decision, int64]

func Resolve(loaded Loaded, initial state.V2) model.Result[state.V2, int64] {
	if !loaded.Found {
		return model.Result[state.V2, int64]{Ok: true, Value: initial}
	}
	if loaded.Version <= 1 && 1 <= loaded.Version {
		return state.MigrateV1ToV2(loaded.V1)
	}
	if loaded.Version <= 2 && 2 <= loaded.Version {
		return state.ValidateV2(loaded.V2)
	}
	return model.Result[state.V2, int64]{Error: 71}
}

func BuildPlan(grants Grants, loaded Loaded, initial state.V2, nextDigest string, key string) Plan {
	if !grants.Read || !grants.CompareExchange {
		return Plan{Error: 74}
	}
	if key == "" {
		return Plan{Error: 75}
	}
	load := Operation{Sequence: 0, Kind: 0, Status: loaded.Found, Token: loaded.Token, Digest: loaded.Digest, Version: loaded.Version, Value: loaded.V2}
	resolved := Resolve(loaded, initial)
	if !resolved.Ok {
		return Plan{Error: resolved.Error}
	}
	updated := state.Update(resolved.Value)
	if !updated.Ok {
		return Plan{Error: updated.Error}
	}
	next := updated.Value
	cas := Operation{Sequence: 1, Kind: 1, Status: false, Token: loaded.Token, Digest: nextDigest, Version: 2, Value: next}
	return Plan{Ok: true, Value: Decision{Value: next, Load: load, CompareExchange: cas}}
}
