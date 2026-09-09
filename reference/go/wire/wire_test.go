package wire

import "testing"

func base(entityFields byte) []byte {
	b := append([]byte("SEMEK1\r\n"), 1)
	b = append(b, make([]byte, 32)...)
	b = append(b, 0, 1)
	b = append(b, make([]byte, 32)...)
	b = append(b, 1, entityFields)
	return b
}
func TestDecodeRejectsDuplicateEntityFields(t *testing.T) {
	b := base(2)
	field := make([]byte, 16)
	field[15] = 1
	b = append(b, field...)
	b = append(b, 0)
	b = append(b, field...)
	b = append(b, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted duplicate field")
	}
}
func TestDecodeRejectsDuplicateRecordFields(t *testing.T) {
	b := base(1)
	field := make([]byte, 16)
	field[15] = 1
	b = append(b, field...)
	b = append(b, 8, 2)
	member := make([]byte, 16)
	member[15] = 2
	b = append(b, member...)
	b = append(b, 0)
	b = append(b, member...)
	b = append(b, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted duplicate record field")
	}
}
func TestDecodeRejectsNoncanonicalULEB(t *testing.T) {
	b := append([]byte("SEMEK1\r\n"), 0x81, 0)
	b = append(b, make([]byte, 34)...)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted long ULEB")
	}
	b = append([]byte("SEMEK1\r\n"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 2}...)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted overflow ULEB")
	}
}
func TestDecodeRejectsOrdering(t *testing.T) {
	header := append([]byte("SEMEK1\r\n"), 1)
	header = append(header, make([]byte, 32)...)
	header = append(header, 2)
	hi, lo := make([]byte, 16), make([]byte, 16)
	hi[15] = 2
	lo[15] = 1
	b := append(append(append([]byte{}, header...), hi...), lo...)
	b = append(b, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted unsorted parents")
	}
	b = base(2)
	f2, f1 := make([]byte, 16), make([]byte, 16)
	f2[15] = 2
	f1[15] = 1
	b = append(b, f2...)
	b = append(b, 0)
	b = append(b, f1...)
	b = append(b, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted unsorted fields")
	}
	header = append([]byte("SEMEK1\r\n"), 1)
	header = append(header, make([]byte, 32)...)
	header = append(header, 0, 2)
	b = append(append([]byte{}, header...), hi...)
	b = append(b, make([]byte, 16)...)
	b = append(b, 1, 0)
	b = append(b, lo...)
	b = append(b, make([]byte, 16)...)
	b = append(b, 1, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted unsorted entities")
	}
	b = base(1)
	b = append(b, f1...)
	b = append(b, 8, 2)
	b = append(b, f2...)
	b = append(b, 0)
	b = append(b, f1...)
	b = append(b, 0)
	if _, e := Decode(b); e == nil {
		t.Fatal("accepted unsorted record fields")
	}
}
