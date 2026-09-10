// Package goconfigurationmanifest parses the explicit, fixture-facing JSON
// boundary into project-neutral Configuration v2 selections.
package goconfigurationmanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/wire"
)

const MaxBytes = 1 << 20

type document struct {
	Fields       []field   `json:"fields"`
	Runtime      []runtime `json:"runtime_inputs"`
	Initializers []unit    `json:"initializers"`
}
type typ struct {
	Package string `json:"package,omitempty"`
	Name    string `json:"name,omitempty"`
	ID      string `json:"id,omitempty"`
}
type function struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}
type field struct {
	Key             string     `json:"key"`
	OwnerPackage    string     `json:"owner_package"`
	Type            typ        `json:"type"`
	Origin          typ        `json:"origin"`
	Default         string     `json:"default,omitempty"`
	DefaultProvider *function  `json:"default_provider,omitempty"`
	Required        bool       `json:"required"`
	Resolution      resolution `json:"resolution"`
	Validator       *function  `json:"validator,omitempty"`
	ValidationOrder uint64     `json:"validation_order"`
}
type resolution struct {
	Kind       string `json:"kind"`
	Value      string `json:"value,omitempty"`
	Capability string `json:"capability,omitempty"`
}
type runtime struct {
	Identity   string `json:"identity"`
	Type       typ    `json:"type"`
	Capability string `json:"capability,omitempty"`
}
type unit struct {
	Key          string   `json:"key"`
	Callable     function `json:"callable"`
	Dependencies []string `json:"dependencies"`
	Arguments    []source `json:"arguments"`
}
type source struct {
	Kind           string    `json:"kind"`
	Field          string    `json:"field,omitempty"`
	Runtime        string    `json:"runtime,omitempty"`
	Predecessor    string    `json:"predecessor,omitempty"`
	Static         string    `json:"static,omitempty"`
	StaticProvider *function `json:"static_provider,omitempty"`
	Record         *record   `json:"record,omitempty"`
}
type record struct {
	Type    typ      `json:"type"`
	Members []member `json:"members"`
}
type member struct {
	Name   string `json:"name"`
	Source source `json:"source"`
}

func Parse(data []byte) (goconfigurationadapter.Selection, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return goconfigurationadapter.Selection{}, fmt.Errorf("configuration_manifest.size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x document
	if err := d.Decode(&x); err != nil {
		return goconfigurationadapter.Selection{}, fmt.Errorf("configuration_manifest.json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return goconfigurationadapter.Selection{}, fmt.Errorf("configuration_manifest.trailing")
	}
	out := goconfigurationadapter.Selection{}
	for _, f := range x.Fields {
		def, err := optionalID(f.Default)
		if err != nil {
			return out, err
		}
		value, err := optionalID(f.Resolution.Value)
		if err != nil {
			return out, err
		}
		capability, err := optionalID(f.Resolution.Capability)
		if err != nil {
			return out, err
		}
		kind, ok := map[string]goconfigurationadapter.ResolutionKind{"explicit": goconfigurationadapter.ExplicitValue, "default": goconfigurationadapter.DefaultValue, "capability": goconfigurationadapter.CapabilityValue}[f.Resolution.Kind]
		if !ok {
			return out, fmt.Errorf("configuration_manifest.resolution_kind")
		}
		var validator *goconfigurationadapter.FunctionSelection
		if f.Validator != nil {
			validator = &goconfigurationadapter.FunctionSelection{Package: f.Validator.Package, Name: f.Validator.Name}
		}
		var defaultProvider *goconfigurationadapter.FunctionSelection
		if f.DefaultProvider != nil {
			defaultProvider = &goconfigurationadapter.FunctionSelection{Package: f.DefaultProvider.Package, Name: f.DefaultProvider.Name}
		}
		out.Fields = append(out.Fields, goconfigurationadapter.FieldSelection{Key: f.Key, OwnerPackage: f.OwnerPackage, Type: convertType(f.Type), Origin: convertType(f.Origin), Default: def, DefaultProvider: defaultProvider, Required: f.Required, Resolution: goconfigurationadapter.ResolutionSelection{Kind: kind, Value: value, Capability: capability}, Validator: validator, ValidationOrder: f.ValidationOrder})
	}
	for _, r := range x.Runtime {
		capability, err := optionalID(r.Capability)
		if err != nil {
			return out, err
		}
		out.Runtime = append(out.Runtime, goconfigurationadapter.RuntimeInputSelection{Identity: r.Identity, Type: convertType(r.Type), Capability: capability})
	}
	for _, u := range x.Initializers {
		item := goconfigurationadapter.UnitSelection{Key: u.Key, Callable: goconfigurationadapter.FunctionSelection{Package: u.Callable.Package, Name: u.Callable.Name}, Dependencies: append([]string(nil), u.Dependencies...)}
		for _, a := range u.Arguments {
			s, err := convertSource(a, 0)
			if err != nil {
				return out, err
			}
			item.Arguments = append(item.Arguments, s)
		}
		out.Units = append(out.Units, item)
	}
	if len(out.Units) == 0 {
		return out, fmt.Errorf("configuration_manifest.initializers")
	}
	return out, nil
}
func convertSource(x source, depth int) (goconfigurationadapter.SourceSelection, error) {
	if depth > 64 {
		return goconfigurationadapter.SourceSelection{}, fmt.Errorf("configuration_manifest.depth")
	}
	k, ok := map[string]goconfigurationadapter.SourceKind{"resolved-field": goconfigurationadapter.ResolvedField, "runtime-input": goconfigurationadapter.RuntimeInputSource, "predecessor-ok": goconfigurationadapter.PredecessorOK, "static": goconfigurationadapter.StaticCanonical, "record": goconfigurationadapter.RecordConstruction}[x.Kind]
	if !ok {
		return goconfigurationadapter.SourceSelection{}, fmt.Errorf("configuration_manifest.source_kind")
	}
	static, err := optionalID(x.Static)
	if err != nil {
		return goconfigurationadapter.SourceSelection{}, err
	}
	out := goconfigurationadapter.SourceSelection{Kind: k, Field: x.Field, Runtime: x.Runtime, Predecessor: x.Predecessor, Static: static}
	if x.StaticProvider != nil {
		out.StaticProvider = &goconfigurationadapter.FunctionSelection{Package: x.StaticProvider.Package, Name: x.StaticProvider.Name}
	}
	if x.Record != nil {
		r := &goconfigurationadapter.RecordSelection{Type: convertType(x.Record.Type)}
		for _, m := range x.Record.Members {
			s, er := convertSource(m.Source, depth+1)
			if er != nil {
				return out, er
			}
			r.Members = append(r.Members, goconfigurationadapter.MemberSelection{Name: m.Name, Source: s})
		}
		out.Record = r
	}
	return out, nil
}
func convertType(x typ) goconfigurationadapter.TypeSelection {
	return goconfigurationadapter.TypeSelection{Package: x.Package, Name: x.Name, ID: x.ID}
}
func optionalID(s string) (*wire.ID, error) {
	if s == "" {
		return nil, nil
	}
	x, err := wire.ParseID(s)
	if err != nil {
		return nil, fmt.Errorf("configuration_manifest.id")
	}
	return &x, nil
}
