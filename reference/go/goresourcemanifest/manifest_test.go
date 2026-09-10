package goresourcemanifest

import (
	"bytes"
	"testing"
)

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

func TestEncodeIsCanonicalSortedAndRoundTrips(t *testing.T) {
	var a, b [32]byte
	a[0], b[0] = 1, 2
	input := []Selection{
		{Identity: "z/item", Path: "resources/z", Destination: "assets/z", MediaType: "application/octet-stream", Size: 2, SHA256: b},
		{Identity: "a/item", Path: "resources/a", Destination: "assets/a", MediaType: "text/plain", Size: 1, SHA256: a},
	}
	one, err := Encode(input)
	two, err2 := Encode([]Selection{input[1], input[0]})
	if err != nil || err2 != nil || !bytes.Equal(one, two) {
		t.Fatalf("encode: %v %v", err, err2)
	}
	got, err := Parse(one)
	if err != nil || len(got) != 2 || got[0].Identity != "a/item" || got[1].Identity != "z/item" {
		t.Fatalf("round trip: %#v %v", got, err)
	}
	if _, err = Encode(append(input, input[0])); err == nil {
		t.Fatal("accepted duplicate resource")
	}
}
