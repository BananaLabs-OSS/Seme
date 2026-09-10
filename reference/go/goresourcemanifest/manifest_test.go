package goresourcemanifest

import "testing"

func TestParseStrictDeterministic(t *testing.T) {
	src := []byte(`{"version":"seme.resources/v1","resources":[{"identity":"ui/message","path":"resources/message.txt","destination":"assets/message.txt","media_type":"text/plain; charset=utf-8","size":3,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
	a, err := Parse(src)
	if err != nil || len(a) != 1 || a[0].Size != 3 {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{append(append([]byte(nil), src...), 'x'), []byte(`{"version":"seme.resources/v1","resources":[],"extra":1}`), []byte(`{"version":"wrong","resources":[{}]}`), []byte(`{"version":"seme.resources/v1","resources":[{"identity":"../x","path":"x","destination":"x","media_type":"text/plain","size":0,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`), []byte(`{"version":"seme.resources/v1","resources":[{"identity":"a","path":"../x","destination":"x","media_type":"text/plain","size":0,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`), []byte(`{"version":"seme.resources/v1","resources":[{"identity":"a","path":"x","destination":"x","media_type":"text/plain","size":0,"sha256":"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}]}`)} {
		if got, e := Parse(bad); e == nil || got != nil {
			t.Fatalf("accepted malformed %s", bad)
		}
	}
}
