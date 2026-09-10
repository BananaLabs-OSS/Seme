package service

import (
	"example.test/go-uab-11/application"
	"example.test/go-uab-11/configuration"
	"example.test/go-uab-11/model"
	"example.test/go-uab-11/policy"
)

type Runtime struct {
	Ready    bool
	Stage    int64
	Settings configuration.Settings
	Policy   policy.Initialized
	State    application.State
}

func Initialize(input configuration.Input, state application.State) model.Result[Runtime, int64] {
	configured := configuration.Initialize(input)
	if !configured.Ok {
		return model.Result[Runtime, int64]{Error: configured.Error}
	}
	prepared := policy.Initialize(configured.Value)
	if !prepared.Ok {
		return model.Result[Runtime, int64]{Error: prepared.Error}
	}
	return Assemble(configured.Value, prepared.Value, state)
}

func Assemble(configured configuration.Initialized, prepared policy.Initialized, state application.State) model.Result[Runtime, int64] {
	if configured.Stage != 1 || prepared.Stage != 2 {
		return model.Result[Runtime, int64]{Error: 21}
	}
	if configured.Settings.Limit != prepared.Limit {
		return model.Result[Runtime, int64]{Error: 22}
	}
	return model.Result[Runtime, int64]{Ok: true, Value: Runtime{Ready: true, Stage: 3, Settings: configured.Settings, Policy: prepared, State: state}}
}

func Apply(runtime Runtime, command application.Command) application.Outcome {
	if !runtime.Ready || runtime.Stage != 3 {
		return application.Outcome{Error: 30}
	}
	if runtime.Policy.Limit < int64(len(runtime.State.Values)) {
		return application.Outcome{Error: 31}
	}
	return application.Apply(runtime.State, command)
}

func ApplyConfigured(input configuration.Input, state application.State, command application.Command) application.Outcome {
	runtime := Initialize(input, state)
	if !runtime.Ok {
		return application.Outcome{Error: runtime.Error}
	}
	return Apply(runtime.Value, command)
}
