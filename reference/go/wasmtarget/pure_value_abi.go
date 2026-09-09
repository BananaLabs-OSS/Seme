package wasmtarget

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/wire"
)

const pureValueMaximumMessage = 7160

// PureValueLayout is the recursively certified boundary representation used
// by the next compositional pure ABI. It describes bytes only; it does not
// select a source-language representation or authorize execution.
type PureValueLayout struct {
	Contract        string                   `json:"contract"`
	Type            string                   `json:"type"`
	FixedSize       uint64                   `json:"fixed_size"`
	VariablePayload bool                     `json:"variable_payload,omitempty"`
	MaximumPayload  uint64                   `json:"maximum_payload,omitempty"`
	Encoding        string                   `json:"encoding"`
	Variants        []PureValueVariantLayout `json:"variants,omitempty"`
}

type PureValueVariantLayout struct {
	Tag     uint64           `json:"tag"`
	Name    string           `json:"name"`
	Offset  uint64           `json:"offset"`
	Payload *PureValueLayout `json:"payload,omitempty"`
}

// CertifyPureValueLayout derives a layout exclusively from canonical type
// entities. Recursive types and unsupported schemas fail closed.
func CertifyPureValueLayout(graph wire.Envelope, typeID wire.ID) (PureValueLayout, error) {
	return certifyPureValueLayout(graph, typeID, map[wire.ID]bool{}, 32)
}

// ValidatePureValueBytes rejects every noncanonical encoding described by a
// certified layout. Variable offsets are absolute within data.
func ValidatePureValueBytes(layout PureValueLayout, data []byte) error {
	if layout.Contract != "seme.pure-value-abi/v1" || uint64(len(data)) < layout.FixedSize || len(data) > pureValueMaximumMessage {
		return fmt.Errorf("wasm.pure_value_size")
	}
	ranges := make([]pureValueRange, 0, 4)
	if err := validatePureValueAt(layout, data, 0, layout.FixedSize, &ranges, 32); err != nil {
		return err
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	next := layout.FixedSize
	for _, item := range ranges {
		if item.start != next || item.end < item.start || item.end > uint64(len(data)) {
			return fmt.Errorf("wasm.pure_value_payload_layout")
		}
		next = item.end
	}
	if next != uint64(len(data)) {
		return fmt.Errorf("wasm.pure_value_payload_layout")
	}
	return nil
}

type pureValueRange struct{ start, end uint64 }

func validatePureValueAt(layout PureValueLayout, data []byte, offset, rootFixed uint64, ranges *[]pureValueRange, budget int) error {
	if budget == 0 || offset+layout.FixedSize > uint64(len(data)) {
		return fmt.Errorf("wasm.pure_value_nested_size")
	}
	fixed := data[offset : offset+layout.FixedSize]
	switch {
	case layout.Type == "bool":
		if fixed[0] > 1 {
			return fmt.Errorf("wasm.pure_value_boolean")
		}
	case layout.Type == "text" || layout.Type == "bytes":
		start, length := uint64(binary.LittleEndian.Uint32(fixed[:4])), uint64(binary.LittleEndian.Uint32(fixed[4:8]))
		if length > layout.MaximumPayload || start < rootFixed || start+length < start || start+length > uint64(len(data)) {
			return fmt.Errorf("wasm.pure_value_descriptor")
		}
		*ranges = append(*ranges, pureValueRange{start, start + length})
		if layout.Type == "text" && !utf8.Valid(data[start:start+length]) {
			return fmt.Errorf("wasm.pure_value_text")
		}
	case len(layout.Variants) > 0:
		tag := uint64(fixed[0])
		var selected *PureValueVariantLayout
		for index := range layout.Variants {
			if layout.Variants[index].Tag == tag {
				selected = &layout.Variants[index]
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("wasm.pure_value_tag")
		}
		payloadStart := selected.Offset
		if selected.Payload == nil {
			for _, value := range fixed[payloadStart:] {
				if value != 0 {
					return fmt.Errorf("wasm.pure_value_inactive_payload")
				}
			}
			return nil
		}
		if err := validatePureValueAt(*selected.Payload, data, offset+payloadStart, rootFixed, ranges, budget-1); err != nil {
			return err
		}
		used := payloadStart + selected.Payload.FixedSize
		for _, value := range fixed[used:] {
			if value != 0 {
				return fmt.Errorf("wasm.pure_value_inactive_payload")
			}
		}
	}
	return nil
}

func certifyPureValueLayout(graph wire.Envelope, typeID wire.ID, visiting map[wire.ID]bool, budget int) (PureValueLayout, error) {
	if budget == 0 || visiting[typeID] {
		return PureValueLayout{}, fmt.Errorf("wasm.pure_value_type_cycle_or_size")
	}
	entity, ok := graph.Entities[typeID]
	if !ok {
		return PureValueLayout{}, fmt.Errorf("wasm.pure_value_type_missing")
	}
	visiting[typeID] = true
	defer delete(visiting, typeID)
	base := PureValueLayout{Contract: "seme.pure-value-abi/v1"}
	switch entity.Schema {
	case identity(0x9010):
		base.Type, base.FixedSize, base.Encoding = "i64", 8, "little-endian-twos-complement-i64-modular"
	case identity(0x9020):
		base.Type, base.FixedSize, base.Encoding = "bool", 1, "canonical-u8-0-or-1"
	case identity(0x9040):
		base.Type, base.FixedSize, base.VariablePayload, base.MaximumPayload = "text", 8, true, 4096
		base.Encoding = "u32le-offset-u32le-byte-length/utf8-scalar-exact"
	case identity(0x9041):
		base.Type, base.FixedSize, base.VariablePayload, base.MaximumPayload = "bytes", 8, true, 4096
		base.Encoding = "u32le-offset-u32le-byte-length/opaque"
	case identity(0x9042):
		okType, okErr := requiredTypeReference(entity, 0x9400)
		errorType, errorErr := requiredTypeReference(entity, 0x9401)
		if okErr != nil || errorErr != nil {
			return PureValueLayout{}, fmt.Errorf("wasm.pure_result_type_fields")
		}
		okLayout, err := certifyPureValueLayout(graph, okType, visiting, budget-1)
		if err != nil {
			return PureValueLayout{}, err
		}
		errorLayout, err := certifyPureValueLayout(graph, errorType, visiting, budget-1)
		if err != nil {
			return PureValueLayout{}, err
		}
		base.Type, base.FixedSize = "result<"+okLayout.Type+","+errorLayout.Type+">", 1+max64(okLayout.FixedSize, errorLayout.FixedSize)
		base.VariablePayload = okLayout.VariablePayload || errorLayout.VariablePayload
		base.MaximumPayload = max64(okLayout.MaximumPayload, errorLayout.MaximumPayload)
		base.Encoding = "u8-tag(0=ok,1=error)/zeroed-inactive-union-payload"
		base.Variants = []PureValueVariantLayout{{Tag: 0, Name: "ok", Offset: 1, Payload: &okLayout}, {Tag: 1, Name: "error", Offset: 1, Payload: &errorLayout}}
	case identity(0xa050):
		valueType, err := requiredTypeReference(entity, 0xa0500)
		if err != nil {
			return PureValueLayout{}, fmt.Errorf("wasm.pure_option_type_fields")
		}
		valueLayout, err := certifyPureValueLayout(graph, valueType, visiting, budget-1)
		if err != nil {
			return PureValueLayout{}, err
		}
		base.Type, base.FixedSize = "option<"+valueLayout.Type+">", 1+valueLayout.FixedSize
		base.VariablePayload, base.MaximumPayload = valueLayout.VariablePayload, valueLayout.MaximumPayload
		base.Encoding = "u8-tag(0=none,1=some)/zeroed-absent-payload"
		base.Variants = []PureValueVariantLayout{{Tag: 0, Name: "none", Offset: 1}, {Tag: 1, Name: "some", Offset: 1, Payload: &valueLayout}}
	default:
		return PureValueLayout{}, fmt.Errorf("wasm.pure_value_type_unsupported")
	}
	return base, nil
}

func requiredTypeReference(entity wire.Entity, fieldID uint64) (wire.ID, error) {
	value, err := field(entity, fieldID)
	if err != nil || value.Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.pure_value_type_reference")
	}
	return value.Reference, nil
}

func max64(left, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}
