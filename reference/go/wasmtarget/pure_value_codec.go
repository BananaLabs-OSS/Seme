package wasmtarget

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"seme.local/reference/canonicaleval"
)

// EncodePureValue serializes a canonical value according to a certified
// recursive layout. Payload allocation follows deterministic semantic order.
func EncodePureValue(layout PureValueLayout, value canonicaleval.Value) ([]byte, error) {
	data := make([]byte, layout.FixedSize)
	if err := encodeValue(layout, value, &data, 0, 32); err != nil {
		return nil, err
	}
	if err := ValidatePureValueBytes(layout, data); err != nil {
		return nil, err
	}
	return data, nil
}

func encodeValue(l PureValueLayout, v canonicaleval.Value, data *[]byte, at uint64, budget int) error {
	if budget == 0 || at+l.FixedSize > uint64(len(*data)) {
		return fmt.Errorf("wasm.pure_value_encode_budget")
	}
	fixed := (*data)[at : at+l.FixedSize]
	switch {
	case l.Type == "i64":
		n, e := strconv.ParseInt(v.I64, 10, 64)
		if v.Kind != "i64" || e != nil {
			return fmt.Errorf("wasm.pure_value_encode_i64")
		}
		binary.LittleEndian.PutUint64(fixed, uint64(n))
	case l.Type == "bool":
		if v.Kind != "bool" {
			return fmt.Errorf("wasm.pure_value_encode_bool")
		}
		if v.Bool {
			fixed[0] = 1
		}
	case l.Type == "text" || l.Type == "bytes":
		var payload []byte
		var e error
		if l.Type == "text" && v.Kind == "text" {
			payload = []byte(v.Text)
		} else if l.Type == "bytes" && v.Kind == "bytes" {
			payload, e = hex.DecodeString(v.Bytes)
		} else {
			return fmt.Errorf("wasm.pure_value_encode_scalar")
		}
		if e != nil || uint64(len(payload)) > l.MaximumPayload {
			return fmt.Errorf("wasm.pure_value_encode_payload")
		}
		start := len(*data)
		*data = append(*data, payload...)
		fixed = (*data)[at : at+l.FixedSize]
		binary.LittleEndian.PutUint32(fixed, uint32(start))
		binary.LittleEndian.PutUint32(fixed[4:], uint32(len(payload)))
	case len(l.Variants) > 0:
		if v.Kind != "result" && v.Kind != "option" {
			return fmt.Errorf("wasm.pure_value_encode_variant")
		}
		var selected *PureValueVariantLayout
		for i := range l.Variants {
			if l.Variants[i].Name == v.Variant {
				selected = &l.Variants[i]
			}
		}
		if selected == nil || (selected.Payload == nil) != (v.Payload == nil) {
			return fmt.Errorf("wasm.pure_value_encode_variant")
		}
		fixed[0] = byte(selected.Tag)
		if selected.Payload != nil {
			return encodeValue(*selected.Payload, *v.Payload, data, at+selected.Offset, budget-1)
		}
	case len(l.Fields) > 0:
		fields := v.Fields
		if strings.HasPrefix(l.Type, "transition<") {
			if v.Kind != "transition" || v.State == nil || v.Result == nil {
				return fmt.Errorf("wasm.pure_value_encode_transition")
			}
			fields = map[string]canonicaleval.Value{"state": *v.State, "result": *v.Result}
		} else if v.Kind != "record" {
			return fmt.Errorf("wasm.pure_value_encode_record")
		}
		if len(fields) != len(l.Fields) {
			return fmt.Errorf("wasm.pure_value_encode_record")
		}
		for _, f := range l.Fields {
			x, ok := fields[f.Name]
			if !ok {
				return fmt.Errorf("wasm.pure_value_encode_record")
			}
			if e := encodeValue(f.Value, x, data, at+f.Offset, budget-1); e != nil {
				return e
			}
		}
	case l.Elements != nil && l.Length > 0:
		if v.Kind != "array" || uint64(len(v.Items)) != l.Length {
			return fmt.Errorf("wasm.pure_value_encode_array")
		}
		for i, x := range v.Items {
			if e := encodeValue(*l.Elements, x, data, at+uint64(i)*l.Elements.FixedSize, budget-1); e != nil {
				return e
			}
		}
	case l.Elements != nil:
		if v.Kind != "slice" || len(v.Items) > 512 {
			return fmt.Errorf("wasm.pure_value_encode_slice")
		}
		start := uint64(len(*data))
		*data = append(*data, make([]byte, uint64(len(v.Items))*l.Elements.FixedSize)...)
		fixed = (*data)[at : at+l.FixedSize]
		binary.LittleEndian.PutUint32(fixed, uint32(start))
		binary.LittleEndian.PutUint32(fixed[4:], uint32(len(v.Items)))
		for i, x := range v.Items {
			if e := encodeValue(*l.Elements, x, data, start+uint64(i)*l.Elements.FixedSize, budget-1); e != nil {
				return e
			}
		}
	case l.Key != nil && l.Value != nil:
		if v.Kind != "map" || len(v.Entries) > 512 {
			return fmt.Errorf("wasm.pure_value_encode_map")
		}
		stride := l.Key.FixedSize + l.Value.FixedSize
		start := uint64(len(*data))
		*data = append(*data, make([]byte, uint64(len(v.Entries))*stride)...)
		fixed = (*data)[at : at+l.FixedSize]
		binary.LittleEndian.PutUint32(fixed, uint32(start))
		binary.LittleEndian.PutUint32(fixed[4:], uint32(len(v.Entries)))
		for i, x := range v.Entries {
			p := start + uint64(i)*stride
			if e := encodeValue(*l.Key, x.Key, data, p, budget-1); e != nil {
				return e
			}
			if e := encodeValue(*l.Value, x.Value, data, p+l.Key.FixedSize, budget-1); e != nil {
				return e
			}
		}
	default:
		return fmt.Errorf("wasm.pure_value_encode_unsupported")
	}
	return nil
}

// DecodePureValue validates the entire message before materializing its
// canonical value, so malformed offsets, tags, padding, and map order never
// become partially trusted state.
func DecodePureValue(layout PureValueLayout, data []byte) (canonicaleval.Value, error) {
	if err := ValidatePureValueBytes(layout, data); err != nil {
		return canonicaleval.Value{}, err
	}
	return decodeValue(layout, data, 0, 32)
}

func decodeValue(l PureValueLayout, data []byte, at uint64, budget int) (canonicaleval.Value, error) {
	if budget == 0 {
		return canonicaleval.Value{}, fmt.Errorf("wasm.pure_value_decode_budget")
	}
	fixed := data[at : at+l.FixedSize]
	switch {
	case l.Type == "i64":
		return canonicaleval.Value{Kind: "i64", I64: strconv.FormatInt(int64(binary.LittleEndian.Uint64(fixed)), 10)}, nil
	case l.Type == "bool":
		return canonicaleval.Value{Kind: "bool", Bool: fixed[0] == 1}, nil
	case l.Type == "text" || l.Type == "bytes":
		start, n := uint64(binary.LittleEndian.Uint32(fixed)), uint64(binary.LittleEndian.Uint32(fixed[4:]))
		if l.Type == "text" {
			return canonicaleval.Value{Kind: "text", Text: string(data[start : start+n])}, nil
		}
		return canonicaleval.Value{Kind: "bytes", Bytes: hex.EncodeToString(data[start : start+n])}, nil
	case len(l.Variants) > 0:
		var selected *PureValueVariantLayout
		for i := range l.Variants {
			if l.Variants[i].Tag == uint64(fixed[0]) {
				selected = &l.Variants[i]
			}
		}
		out := canonicaleval.Value{Kind: "option", Variant: selected.Name}
		if strings.HasPrefix(l.Type, "result<") {
			out.Kind = "result"
		}
		if selected.Payload != nil {
			x, e := decodeValue(*selected.Payload, data, at+selected.Offset, budget-1)
			out.Payload = &x
			return out, e
		}
		return out, nil
	case len(l.Fields) > 0:
		fields := map[string]canonicaleval.Value{}
		for _, f := range l.Fields {
			x, e := decodeValue(f.Value, data, at+f.Offset, budget-1)
			if e != nil {
				return canonicaleval.Value{}, e
			}
			fields[f.Name] = x
		}
		if strings.HasPrefix(l.Type, "transition<") {
			state, result := fields["state"], fields["result"]
			return canonicaleval.Value{Kind: "transition", State: &state, Result: &result}, nil
		}
		return canonicaleval.Value{Kind: "record", Fields: fields}, nil
	case l.Elements != nil && l.Length > 0:
		out := canonicaleval.Value{Kind: "array"}
		for i := uint64(0); i < l.Length; i++ {
			x, e := decodeValue(*l.Elements, data, at+i*l.Elements.FixedSize, budget-1)
			if e != nil {
				return out, e
			}
			out.Items = append(out.Items, x)
		}
		return out, nil
	case l.Elements != nil:
		start, n := uint64(binary.LittleEndian.Uint32(fixed)), uint64(binary.LittleEndian.Uint32(fixed[4:]))
		out := canonicaleval.Value{Kind: "slice"}
		for i := uint64(0); i < n; i++ {
			x, e := decodeValue(*l.Elements, data, start+i*l.Elements.FixedSize, budget-1)
			if e != nil {
				return out, e
			}
			out.Items = append(out.Items, x)
		}
		return out, nil
	case l.Key != nil && l.Value != nil:
		start, n := uint64(binary.LittleEndian.Uint32(fixed)), uint64(binary.LittleEndian.Uint32(fixed[4:]))
		stride := l.Key.FixedSize + l.Value.FixedSize
		out := canonicaleval.Value{Kind: "map", ValueType: l.Value.Type}
		for i := uint64(0); i < n; i++ {
			key, e := decodeValue(*l.Key, data, start+i*stride, budget-1)
			if e != nil {
				return out, e
			}
			value, e := decodeValue(*l.Value, data, start+i*stride+l.Key.FixedSize, budget-1)
			if e != nil {
				return out, e
			}
			out.Entries = append(out.Entries, canonicaleval.Entry{Key: key, Value: value})
		}
		return out, nil
	}
	return canonicaleval.Value{}, fmt.Errorf("wasm.pure_value_decode_unsupported")
}
