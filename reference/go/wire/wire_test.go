package wire

import "testing"

func testID(last byte) ID {
	var value ID
	value[len(value)-1] = last
	return value
}

func TestEncodeCanonicalRoundTrip(t *testing.T) {
	first, second := testID(1), testID(2)
	envelope := Envelope{
		Module: first, Revision: second, Parents: []ID{second, first},
		Entities: map[ID]Entity{
			second: {ID: second, Schema: first, Version: 1, Fields: map[ID]Value{
				second: {Tag: 8, Record: map[ID]Value{second: {Tag: 5, Bytes: []byte("value")}}},
				first:  {Tag: 7, List: []Value{{Tag: 6, Reference: second}, {Tag: 9, Hole: first}}},
			}},
			first: {ID: first, Schema: second, Version: 1, Fields: map[ID]Value{}},
		},
	}
	encoded, err := Encode(envelope)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := Encode(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(reencoded) {
		t.Fatal("canonical wire did not reproduce byte-identically")
	}
}

func TestEncodeRejectsInvalidConstruction(t *testing.T) {
	first, second := testID(1), testID(2)
	if _, err := Encode(Envelope{Parents: []ID{first, first}}); err == nil {
		t.Fatal("accepted duplicate parents")
	}
	if _, err := Encode(Envelope{Entities: map[ID]Entity{first: {ID: second}}}); err == nil {
		t.Fatal("accepted entity map key mismatch")
	}
	if _, err := Encode(Envelope{Entities: map[ID]Entity{first: {ID: first, Fields: map[ID]Value{first: {Tag: 255}}}}}); err == nil {
		t.Fatal("accepted unknown value tag")
	}
}

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
