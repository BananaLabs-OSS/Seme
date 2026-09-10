package state

import (
	"example.test/go-uab-11/application"
	"example.test/go-uab-11/model"
)

type V1 struct{ State application.State }
type V2 struct {
	State    application.State
	Revision int64
}

func ValidateV1(value V1) model.Result[V1, int64] {
	if value.State.Name == "" {
		return model.Result[V1, int64]{Error: 60}
	}
	return model.Result[V1, int64]{Ok: true, Value: value}
}
func ValidateV2(value V2) model.Result[V2, int64] {
	if value.State.Name == "" || value.Revision < 1 {
		return model.Result[V2, int64]{Error: 61}
	}
	return model.Result[V2, int64]{Ok: true, Value: value}
}
func MigrateV1ToV2(value V1) model.Result[V2, int64] {
	valid := ValidateV1(value)
	if !valid.Ok {
		return model.Result[V2, int64]{Error: valid.Error}
	}
	return model.Result[V2, int64]{Ok: true, Value: V2{State: valid.Value.State, Revision: 1}}
}
func Update(value V2) model.Result[V2, int64] {
	valid := ValidateV2(value)
	if !valid.Ok {
		return valid
	}
	if 9223372036854775806 < value.Revision {
		return model.Result[V2, int64]{Error: 62}
	}
	return model.Result[V2, int64]{Ok: true, Value: V2{State: value.State, Revision: value.Revision + 1}}
}
