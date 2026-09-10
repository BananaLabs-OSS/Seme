package application

import (
	"example.test/go-upb03-dependency-v1/model"
	"example.test/go-upb03-dependency-v1/policy"
)

func Apply(value int64) int64 {
	return policy.Score(model.Normalize(value))
}
