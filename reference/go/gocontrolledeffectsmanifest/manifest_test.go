package gocontrolledeffectsmanifest

import (
	"strings"
	"testing"
)

const valid = `{"version":"seme.controlled-effects-selection/v1","clock":{"identity":"clock.injected.unix-milliseconds.v1","owner":"example/controlled","sample":{"package":"example/controlled","name":"ClockSample"},"monotonic_policy":"nondecreasing","injection_policy":"explicit-replayable-input"},"random":{"identity":"random.seeded.lcg-48271-plus-1.v1","owner":"example/controlled","state":{"package":"example/controlled","name":"RandomState"},"draw":{"package":"example/controlled","name":"Draw"},"next":{"package":"example/controlled","name":"Next"},"algorithm":"state*48271+1","overflow_policy":"signed-i64-modular"},"effect":{"identity":"observability.log","owner":"example/streamservice","capability":"observability.log","payload":"bool","delivery_policy":"request-not-delivery"},"application":{"command":{"package":"example/streamservice","name":"ControlledCommand"},"state":{"package":"example/streamservice","name":"ControlledState"},"result":{"package":"example/streamservice","name":"ControlledResult"},"dispatch":{"package":"example/streamservice","name":"DispatchControlled"},"replay":{"package":"example/streamservice","name":"ReplayControlled"}},"replay":{"duplicate_policy":"cached-no-new-effects","rejection_policy":"atomic-no-effects"},"bounds":{"maximum_steps":256,"first_clock_sequence":1,"clock_terminal_sentinel":257,"maximum_unix_milliseconds":4102444800000,"minimum_seed":1,"maximum_seed":9223372036854775807,"maximum_draws":256,"maximum_effects":256,"maximum_command_bytes":4096,"maximum_initial_state_bytes":524288,"maximum_transcript_bytes":1048576}}`

func TestParseStrictSelection(t *testing.T) {
	s, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if s.Next.Name != "Next" || s.Replay.Name != "ReplayControlled" || s.Bounds.MaximumSteps != 256 {
		t.Fatalf("%#v", s)
	}
	bad := map[string]string{
		"version":   strings.Replace(valid, Version, "wrong", 1),
		"algorithm": strings.Replace(valid, "state*48271+1", "state*48271", 1),
		"effect":    strings.Replace(valid, `"payload":"bool"`, `"payload":"bytes"`, 1),
		"bound":     strings.Replace(valid, `"maximum_steps":256`, `"maximum_steps":255`, 1),
		"missing":   strings.Replace(valid, `"name":"Next"`, `"name":""`, 1),
		"duplicate": strings.Replace(valid, `"name":"ReplayControlled"`, `"name":"DispatchControlled"`, 1),
		"unknown":   strings.Replace(valid, `"version":`, `"unknown":1,"version":`, 1),
		"trailing":  valid + `{}`,
	}
	for name, text := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(text)); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	if _, err := Parse(nil); err == nil {
		t.Fatal("empty accepted")
	}
	if _, err := Parse(make([]byte, MaxBytes+1)); err == nil {
		t.Fatal("oversize accepted")
	}
}
