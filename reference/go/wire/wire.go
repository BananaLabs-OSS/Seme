// Package wire provides an untrusted host-side reader for canonical Kernel v1
// envelopes. Frozen canonical validators remain authoritative.
package wire

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
)

// Encode writes the unique canonical Kernel v1 wire representation.
func Encode(e Envelope) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("SEMEK1\r\n")
	putU(&b, 1)
	b.Write(e.Module[:])
	b.Write(e.Revision[:])
	parents := append([]ID(nil), e.Parents...)
	sort.Slice(parents, func(i, j int) bool { return idLess(parents[i], parents[j]) })
	for i := 1; i < len(parents); i++ {
		if parents[i] == parents[i-1] {
			return nil, errors.New("wire.duplicate_parent")
		}
	}
	putU(&b, uint64(len(parents)))
	for _, parent := range parents {
		b.Write(parent[:])
	}
	ids := make([]ID, 0, len(e.Entities))
	for entityID := range e.Entities {
		ids = append(ids, entityID)
	}
	sort.Slice(ids, func(i, j int) bool { return idLess(ids[i], ids[j]) })
	putU(&b, uint64(len(ids)))
	for _, entityID := range ids {
		entity := e.Entities[entityID]
		if entity.ID != entityID {
			return nil, errors.New("wire.entity_key_mismatch")
		}
		b.Write(entity.ID[:])
		b.Write(entity.Schema[:])
		putU(&b, entity.Version)
		keys := make([]ID, 0, len(entity.Fields))
		for fieldID := range entity.Fields {
			keys = append(keys, fieldID)
		}
		sort.Slice(keys, func(i, j int) bool { return idLess(keys[i], keys[j]) })
		putU(&b, uint64(len(keys)))
		for _, fieldID := range keys {
			b.Write(fieldID[:])
			if err := putValue(&b, entity.Fields[fieldID]); err != nil {
				return nil, err
			}
		}
	}
	return b.Bytes(), nil
}

func putU(b *bytes.Buffer, n uint64) {
	for n >= 128 {
		b.WriteByte(byte(n) | 128)
		n >>= 7
	}
	b.WriteByte(byte(n))
}

func putValue(b *bytes.Buffer, value Value) error {
	b.WriteByte(value.Tag)
	switch value.Tag {
	case 0, 1, 2:
	case 3, 4:
		putU(b, value.Unsigned)
	case 5:
		putU(b, uint64(len(value.Bytes)))
		b.Write(value.Bytes)
	case 6:
		b.Write(value.Reference[:])
	case 7:
		putU(b, uint64(len(value.List)))
		for _, item := range value.List {
			if err := putValue(b, item); err != nil {
				return err
			}
		}
	case 8:
		keys := make([]ID, 0, len(value.Record))
		for fieldID := range value.Record {
			keys = append(keys, fieldID)
		}
		sort.Slice(keys, func(i, j int) bool { return idLess(keys[i], keys[j]) })
		putU(b, uint64(len(keys)))
		for _, fieldID := range keys {
			b.Write(fieldID[:])
			if err := putValue(b, value.Record[fieldID]); err != nil {
				return err
			}
		}
	case 9:
		b.Write(value.Hole[:])
	default:
		return fmt.Errorf("wire.tag:%d", value.Tag)
	}
	return nil
}

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
	Hole      ID
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
		if len(out.Parents) > 0 && !idLess(out.Parents[len(out.Parents)-1], id) {
			return Envelope{}, errors.New("wire.parents_order")
		}
		out.Parents = append(out.Parents, id)
	}
	count, err := r.uleb()
	if err != nil {
		return Envelope{}, err
	}
	var previous ID
	first := true
	for ; count > 0; count-- {
		entity, err := r.entity()
		if err != nil {
			return Envelope{}, err
		}
		if _, exists := out.Entities[entity.ID]; exists {
			return Envelope{}, errors.New("wire.duplicate_entity")
		}
		if !first && !idLess(previous, entity.ID) {
			return Envelope{}, errors.New("wire.entities_order")
		}
		previous, first = entity.ID, false
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
	var previous ID
	first := true
	for ; count > 0; count-- {
		field, err := r.id()
		if err != nil {
			return Entity{}, err
		}
		value, err := r.value()
		if err != nil {
			return Entity{}, err
		}
		if _, exists := entity.Fields[field]; exists {
			return Entity{}, errors.New("wire.duplicate_field")
		}
		if !first && !idLess(previous, field) {
			return Entity{}, errors.New("wire.fields_order")
		}
		previous, first = field, false
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
		var previous ID
		first := true
		for ; err == nil && n > 0; n-- {
			var field ID
			field, err = r.id()
			if err != nil {
				break
			}
			var item Value
			item, err = r.value()
			if _, exists := v.Record[field]; exists {
				return Value{}, errors.New("wire.duplicate_record_field")
			}
			if !first && !idLess(previous, field) {
				return Value{}, errors.New("wire.record_order")
			}
			previous, first = field, false
			v.Record[field] = item
		}
	case 9:
		v.Hole, err = r.id()
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
	count := 0
	for shift := uint(0); shift < 64; shift += 7 {
		b, err := r.take(1)
		if err != nil {
			return 0, err
		}
		count++
		if shift == 63 && b[0]&0xfe != 0 {
			return 0, errors.New("wire.uleb_overflow")
		}
		value |= uint64(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			if count > 1 && b[0] == 0 {
				return 0, errors.New("wire.uleb_noncanonical")
			}
			return value, nil
		}
	}
	return 0, errors.New("wire.uleb")
}
func idLess(a, b ID) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
