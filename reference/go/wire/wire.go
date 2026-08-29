// Package wire provides an untrusted host-side reader for canonical Kernel v1
// envelopes. Frozen canonical validators remain authoritative.
package wire

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

type ID [16]byte

func (id ID) String() string { return hex.EncodeToString(id[:]) }
func ParseID(value string) (ID, error) {
	var id ID
	b, err := hex.DecodeString(value)
	if err != nil || len(b) != 16 {
		return id, errors.New("wire.invalid_identity")
	}
	copy(id[:], b)
	return id, nil
}

type Value struct {
	Tag       byte
	Unsigned  uint64
	Bytes     []byte
	Reference ID
	List      []Value
	Record    map[ID]Value
}
type Entity struct {
	ID, Schema ID
	Version    uint64
	Fields     map[ID]Value
}
type Envelope struct {
	Module, Revision ID
	Parents          []ID
	Entities         map[ID]Entity
}
type reader struct {
	data     []byte
	position int
}

func Read(path string) (Envelope, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Envelope{}, err
	}
	return Decode(b)
}
func Decode(data []byte) (Envelope, error) {
	r := reader{data: data}
	magic, err := r.take(8)
	if err != nil || string(magic) != "SEMEK1\r\n" {
		return Envelope{}, errors.New("wire.magic")
	}
	version, err := r.uleb()
	if err != nil || version != 1 {
		return Envelope{}, errors.New("wire.version")
	}
	module, err := r.id()
	if err != nil {
		return Envelope{}, err
	}
	revision, err := r.id()
	if err != nil {
		return Envelope{}, err
	}
	out := Envelope{Module: module, Revision: revision, Entities: map[ID]Entity{}}
	parents, err := r.uleb()
	if err != nil {
		return Envelope{}, err
	}
	for ; parents > 0; parents-- {
		id, err := r.id()
		if err != nil {
			return Envelope{}, err
		}
		out.Parents = append(out.Parents, id)
	}
	count, err := r.uleb()
	if err != nil {
		return Envelope{}, err
	}
	for ; count > 0; count-- {
		entity, err := r.entity()
		if err != nil {
			return Envelope{}, err
		}
		if _, exists := out.Entities[entity.ID]; exists {
			return Envelope{}, errors.New("wire.duplicate_entity")
		}
		out.Entities[entity.ID] = entity
	}
	if r.position != len(data) {
		return Envelope{}, errors.New("wire.trailing")
	}
	return out, nil
}
func (r *reader) entity() (Entity, error) {
	id, err := r.id()
	if err != nil {
		return Entity{}, err
	}
	schema, err := r.id()
	if err != nil {
		return Entity{}, err
	}
	version, err := r.uleb()
	if err != nil {
		return Entity{}, err
	}
	count, err := r.uleb()
	if err != nil {
		return Entity{}, err
	}
	entity := Entity{ID: id, Schema: schema, Version: version, Fields: map[ID]Value{}}
	for ; count > 0; count-- {
		field, err := r.id()
		if err != nil {
			return Entity{}, err
		}
		value, err := r.value()
		if err != nil {
			return Entity{}, err
		}
		entity.Fields[field] = value
	}
	return entity, nil
}
func (r *reader) value() (Value, error) {
	tagb, err := r.take(1)
	if err != nil {
		return Value{}, err
	}
	v := Value{Tag: tagb[0]}
	switch v.Tag {
	case 0, 1, 2:
	case 3, 4:
		v.Unsigned, err = r.uleb()
	case 5:
		var n uint64
		n, err = r.uleb()
		if err == nil {
			v.Bytes, err = r.take(int(n))
		}
	case 6:
		v.Reference, err = r.id()
	case 7:
		var n uint64
		n, err = r.uleb()
		for ; err == nil && n > 0; n-- {
			var item Value
			item, err = r.value()
			v.List = append(v.List, item)
		}
	case 8:
		var n uint64
		n, err = r.uleb()
		v.Record = map[ID]Value{}
		for ; err == nil && n > 0; n-- {
			var field ID
			field, err = r.id()
			if err != nil {
				break
			}
			var item Value
			item, err = r.value()
			v.Record[field] = item
		}
	case 9:
		_, err = r.id()
	default:
		err = fmt.Errorf("wire.tag:%d", v.Tag)
	}
	return v, err
}
func (r *reader) id() (ID, error) {
	var id ID
	b, err := r.take(16)
	if err != nil {
		return id, err
	}
	copy(id[:], b)
	return id, nil
}
func (r *reader) take(n int) ([]byte, error) {
	if n < 0 || r.position+n > len(r.data) {
		return nil, errors.New("wire.truncated")
	}
	b := r.data[r.position : r.position+n]
	r.position += n
	return b, nil
}
func (r *reader) uleb() (uint64, error) {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		b, err := r.take(1)
		if err != nil {
			return 0, err
		}
		value |= uint64(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			return value, nil
		}
	}
	return 0, errors.New("wire.uleb")
}
