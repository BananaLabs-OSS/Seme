package resourceinstance

import "testing"

func TestPortablePathIdentityAndMediaProfiles(t *testing.T) {
	for _, value := range []string{"resources/a.txt", "a", "dir/a-b_2.bin"} {
		if !path(value) {
			t.Fatalf("safe path rejected: %q", value)
		}
	}
	for _, value := range []string{"", "/a", "a/", "a//b", "a/./b", "a/../b", `a\b`, "a\x00b"} {
		if path(value) {
			t.Fatalf("unsafe path accepted: %q", value)
		}
	}
	if !logical("example.test/game/resource.notice-v1") || logical("Example Test") {
		t.Fatal("logical identity profile")
	}
	if !media("text/plain; charset=utf-8") || !media("application/octet-stream") || media("Text/Plain") || media("text") {
		t.Fatal("media profile")
	}
}

func TestDestinationPrefixCollisions(t *testing.T) {
	for _, pair := range [][2]string{{"a", "a"}, {"a", "a/b"}, {"a/b", "a"}} {
		if !collide(pair[0], pair[1]) {
			t.Fatalf("collision missed: %q", pair)
		}
	}
	if collide("a/b", "a/c") || collide("a", "ab") {
		t.Fatal("false collision")
	}
}
