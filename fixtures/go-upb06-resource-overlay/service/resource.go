package service

import (
	"example.test/go-uab-11/application"
	"example.test/go-uab-11/configuration"
	"example.test/go-uab-11/resource"
)

func ApplyConfiguredResource(input configuration.Input, state application.State, command application.Command, resources resource.Set) application.Outcome {
	validated := resource.Validate(resources)
	if !validated.Ok {
		return application.Outcome{Error: validated.Error}
	}
	return ApplyConfigured(input, state, command)
}
