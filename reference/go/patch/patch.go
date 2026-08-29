package patch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"seme.local/reference/foundation"
)

type Rename struct {
	Target      foundation.ID
	Field       foundation.ID
	Expected    []byte
	Replacement []byte
}
type Patch struct {
	ID           foundation.ID
	Author       foundation.ID
	BaseRevision foundation.ID
	Renames      []Rename
}
type Workspace struct {
	Revision foundation.ID
	Module   foundation.Module
	Entities map[foundation.ID]foundation.Entity
}

func Apply(base Workspace, transaction Patch) (Workspace, error) {
	if transaction.ID == "" || transaction.Author == "" {
		return Workspace{}, fmt.Errorf("patch.invalid_candidate")
	}
	if transaction.BaseRevision != base.Revision {
		return Workspace{}, fmt.Errorf("patch.stale_revision")
	}
	writes := map[string]bool{}
	for i, rename := range transaction.Renames {
		entity, ok := base.Entities[rename.Target]
		if !ok {
			return Workspace{}, fmt.Errorf("patch.unknown_target:%d", i)
		}
		value, ok := entity.Fields[rename.Field]
		if !ok {
			return Workspace{}, fmt.Errorf("patch.unknown_field:%d", i)
		}
		if value.Kind != foundation.Bytes || string(value.Bytes) != string(rename.Expected) {
			return Workspace{}, fmt.Errorf("patch.precondition_failed:%d", i)
		}
		key := string(rename.Target) + "\x00" + string(rename.Field)
		if writes[key] {
			return Workspace{}, fmt.Errorf("patch.duplicate_write:%d", i)
		}
		writes[key] = true
	}
	candidate := clone(base)
	for _, rename := range transaction.Renames {
		entity := candidate.Entities[rename.Target]
		value := entity.Fields[rename.Field]
		value.Bytes = append([]byte(nil), rename.Replacement...)
		entity.Fields[rename.Field] = value
		candidate.Entities[rename.Target] = entity
	}
	if _, err := foundation.Validate(candidate.Module, candidate.Entities); err != nil {
		return Workspace{}, fmt.Errorf("patch.invalid_candidate:%w", err)
	}
	payload, _ := json.Marshal(struct {
		Base     foundation.ID
		Patch    Patch
		Entities map[foundation.ID]foundation.Entity
	}{base.Revision, transaction, candidate.Entities})
	sum := sha256.Sum256(append([]byte("seme.patch.revision.v1\x00"), payload...))
	candidate.Revision = foundation.ID(hex.EncodeToString(sum[:16]))
	return candidate, nil
}

func clone(base Workspace) Workspace {
	out := Workspace{Revision: base.Revision, Module: base.Module, Entities: map[foundation.ID]foundation.Entity{}}
	for id, entity := range base.Entities {
		copyEntity := entity
		copyEntity.Fields = map[foundation.ID]foundation.Value{}
		for field, value := range entity.Fields {
			value.Bytes = append([]byte(nil), value.Bytes...)
			copyEntity.Fields[field] = value
		}
		out.Entities[id] = copyEntity
	}
	return out
}
