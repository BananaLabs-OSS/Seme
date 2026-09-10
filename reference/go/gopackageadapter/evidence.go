package gopackageadapter

import (
	"crypto/sha256"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

var (
	unitSchema     = mustID("e015")
	classSchema    = mustID("e013")
	pathField      = mustID("e150")
	digestField    = mustID("e151")
	sizeField      = mustID("e152")
	classField     = mustID("e153")
	classCodeField = mustID("e130")
)

// EvidenceFrom validates the detached inventory against its semantic project,
// then reconciles its tracked units with the complete in-memory provider
// snapshot. Raw source bytes are used only to verify digest and size.
func EvidenceFrom(snapshot goprovider.DocumentSnapshot, resolution goprovider.ResolutionManifest, contract contractcatalog.Contract, project, inventory []byte) (Evidence, error) {
	if err := sourceinventory.Validate(contract, project, inventory); err != nil {
		return Evidence{}, fmt.Errorf("go_package_adapter.inventory:%w", err)
	}
	envelope, err := wire.Decode(inventory)
	if err != nil {
		return Evidence{}, err
	}
	tracked := map[string]packagedetail.Source{}
	for id, entity := range envelope.Entities {
		if entity.Schema != unitSchema {
			continue
		}
		cv := entity.Fields[classField]
		class := envelope.Entities[cv.Reference]
		if cv.Tag != 6 || class.Schema != classSchema || class.Fields[classCodeField].Tag != 3 {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_class")
		}
		if class.Fields[classCodeField].Unsigned != 0 {
			continue
		}
		pv, dv, sv := entity.Fields[pathField], entity.Fields[digestField], entity.Fields[sizeField]
		if pv.Tag != 5 || dv.Tag != 5 || len(dv.Bytes) != sha256.Size || sv.Tag != 3 {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_unit")
		}
		path := string(pv.Bytes)
		var digest [32]byte
		copy(digest[:], dv.Bytes)
		if _, ok := tracked[path]; ok {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_path_duplicate")
		}
		tracked[path] = packagedetail.Source{Identity: id.String(), Path: path, ContentDigest: digest, ByteSize: sv.Unsigned}
	}
	wanted := map[string]bool{}
	for _, p := range resolution.Packages {
		for _, file := range p.Files {
			if wanted[file] {
				return Evidence{}, fmt.Errorf("go_package_adapter.resolution_source_duplicate")
			}
			wanted[file] = true
		}
	}
	if len(wanted) != len(snapshot.Files) || len(tracked) != len(snapshot.Files) {
		return Evidence{}, fmt.Errorf("go_package_adapter.source_set")
	}
	e := Evidence{Sources: map[string]packagedetail.Source{}, Origins: map[Key]packagedetail.Origin{}}
	for path, data := range snapshot.Files {
		if !wanted[path] {
			return Evidence{}, fmt.Errorf("go_package_adapter.snapshot_extra")
		}
		source, ok := tracked[path]
		if !ok {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_missing")
		}
		digest := sha256.Sum256([]byte(data))
		if digest != source.ContentDigest || uint64(len(data)) != source.ByteSize {
			return Evidence{}, fmt.Errorf("go_package_adapter.source_drift")
		}
		e.Sources[path] = source
	}
	for _, p := range resolution.Packages {
		for _, d := range p.Declarations {
			if err := addOrigin(e, d.Location); err != nil {
				return Evidence{}, err
			}
		}
		for _, im := range p.Imports {
			if err := addOrigin(e, im.Location); err != nil {
				return Evidence{}, err
			}
		}
	}
	return e, nil
}
func addOrigin(e Evidence, l goprovider.ProjectLocation) error {
	s, ok := e.Sources[l.File]
	if !ok {
		return fmt.Errorf("go_package_adapter.origin_source")
	}
	if l.ByteStart < 0 || l.ByteEnd <= l.ByteStart || l.EndLine <= 0 || l.EndColumn <= 0 || uint64(l.ByteEnd) > s.ByteSize {
		return fmt.Errorf("go_package_adapter.origin_span")
	}
	k := key(l)
	if _, ok = e.Origins[k]; ok {
		return fmt.Errorf("go_package_adapter.origin_duplicate")
	}
	e.Origins[k] = packagedetail.Origin{SourceIdentity: s.Identity, Path: s.Path, ContentDigest: s.ContentDigest, ByteStart: uint64(l.ByteStart), ByteEnd: uint64(l.ByteEnd), StartLine: uint32(l.Line), StartColumn: uint32(l.Column), EndLine: uint32(l.EndLine), EndColumn: uint32(l.EndColumn)}
	return nil
}
func mustID(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	x, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return x
}
