package goprovider

import "testing"

func TestSnapshotSelectsTargetFiles(t *testing.T) {
	s := DocumentSnapshot{GOOS: "linux", GOARCH: "amd64"}
	for _, test := range []struct {
		name, source string
		want         bool
	}{
		{"main.go", "package platform", true},
		{"value_linux.go", "package platform", true},
		{"value_windows.go", "package platform", false},
		{"value_amd64.go", "package platform", true},
		{"value_linux_arm64.go", "package platform", false},
		{"tagged.go", "//go:build windows\n\npackage platform", false},
		{"tagged_linux.go", "//go:build linux && amd64\n\npackage platform", true},
	} {
		if got := snapshotFileMatches(s, test.name, test.source); got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}
