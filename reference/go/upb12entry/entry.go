// Package upb12entry derives an executable entry view from an authenticated
// UPB12 compiled project graph. It changes only program.entry; every semantic
// entity, identity, revision, and function remains byte-for-byte unchanged.
package upb12entry

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

var functionSchema = identity(0x9011)
var programSchema = identity(0x9015)
var functionName = identity(0x9110)
var programFunctions = identity(0x9150)
var programEntry = identity(0x9151)

func Select(graph []byte, current, next string) ([]byte, error) {
	envelope, err := wire.Decode(graph)
	if err != nil {
		return nil, fmt.Errorf("upb12_entry.decode:%w", err)
	}
	encoded, err := wire.Encode(envelope)
	if err != nil || !bytes.Equal(encoded, graph) {
		return nil, fmt.Errorf("upb12_entry.noncanonical")
	}
	var programID wire.ID
	for id, entity := range envelope.Entities {
		if entity.Schema == programSchema {
			if programID != (wire.ID{}) {
				return nil, fmt.Errorf("upb12_entry.program_count")
			}
			programID = id
		}
	}
	if programID == (wire.ID{}) {
		return nil, fmt.Errorf("upb12_entry.program_count")
	}
	program := envelope.Entities[programID]
	functions, ok := program.Fields[programFunctions]
	if !ok || functions.Tag != 7 {
		return nil, fmt.Errorf("upb12_entry.functions")
	}
	currentID, nextID := wire.ID{}, wire.ID{}
	for _, value := range functions.List {
		if value.Tag != 6 {
			return nil, fmt.Errorf("upb12_entry.function_reference")
		}
		function, exists := envelope.Entities[value.Reference]
		name := function.Fields[functionName]
		if !exists || function.Schema != functionSchema || name.Tag != 5 {
			return nil, fmt.Errorf("upb12_entry.function")
		}
		switch string(name.Bytes) {
		case current:
			if currentID != (wire.ID{}) {
				return nil, fmt.Errorf("upb12_entry.current_ambiguous")
			}
			currentID = value.Reference
		case next:
			if nextID != (wire.ID{}) {
				return nil, fmt.Errorf("upb12_entry.next_ambiguous")
			}
			nextID = value.Reference
		}
	}
	entry, ok := program.Fields[programEntry]
	if !ok || entry.Tag != 6 || entry.Reference != currentID || currentID == (wire.ID{}) {
		return nil, fmt.Errorf("upb12_entry.current")
	}
	if nextID == (wire.ID{}) {
		return nil, fmt.Errorf("upb12_entry.next")
	}
	program.Fields[programEntry] = wire.Value{Tag: 6, Reference: nextID}
	envelope.Entities[programID] = program
	result, err := wire.Encode(envelope)
	if err != nil {
		return nil, fmt.Errorf("upb12_entry.encode:%w", err)
	}
	return result, nil
}

func identity(value uint64) (out wire.ID) {
	for index := 15; index >= 0 && value > 0; index-- {
		out[index] = byte(value)
		value >>= 8
	}
	return
}
