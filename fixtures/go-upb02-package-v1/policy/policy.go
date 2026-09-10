package policy

import "example.test/go-project-build-v1/model"

func Apply(value int64) int64 { return model.Normalize(value) * 2 }
