// Package orderedtransportruntime realizes the project-neutral, bounded host
// boundary described by Ordered Transport v1. It contains no network protocol.
package orderedtransportruntime

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"seme.local/reference/wasmtarget"
)

const (
	FrameCommand byte = iota
	FrameResponse
)

var frameMagic = [8]byte{'S', 'E', 'M', 'E', 'O', 'T', '0', '1'}

type Frame struct {
	Kind    byte
	Payload []byte
}

// Layouts are the authenticated project's canonical command and response
// envelope layouts. Pure Value validation owns their exact field shape.
type Layouts struct {
	Command  wasmtarget.PureValueLayout
	Response wasmtarget.PureValueLayout
}

func (l Layouts) forKind(kind byte) (wasmtarget.PureValueLayout, error) {
	switch kind {
	case FrameCommand:
		return l.Command, nil
	case FrameResponse:
		return l.Response, nil
	default:
		return wasmtarget.PureValueLayout{}, fmt.Errorf("ordered_transport.frame_kind")
	}
}

// ReadFrame consumes exactly one complete frame. A following coalesced frame
// remains unread; payload validation prevents a second typed value or trailing
// bytes from being confused with part of the first value.
func ReadFrame(profile Profile, layouts Layouts, input io.Reader) (Frame, error) {
	if !validProfile(profile) || input == nil {
		return Frame{}, fmt.Errorf("ordered_transport.profile")
	}
	var prefix [4]byte
	if _, err := io.ReadFull(input, prefix[:]); err != nil {
		return Frame{}, fmt.Errorf("ordered_transport.frame_prefix:%w", err)
	}
	bodySize := uint64(binary.LittleEndian.Uint32(prefix[:]))
	if bodySize < 9 || bodySize+4 > profile.MaximumFrameBytes {
		return Frame{}, fmt.Errorf("ordered_transport.frame_size")
	}
	body := make([]byte, bodySize)
	if _, err := io.ReadFull(input, body); err != nil {
		return Frame{}, fmt.Errorf("ordered_transport.frame_body:%w", err)
	}
	if !bytes.Equal(body[:8], frameMagic[:]) {
		return Frame{}, fmt.Errorf("ordered_transport.frame_magic")
	}
	layout, err := layouts.forKind(body[8])
	if err != nil {
		return Frame{}, err
	}
	payload := body[9:]
	if wasmtarget.ValidatePureValueBytes(layout, payload) != nil {
		return Frame{}, fmt.Errorf("ordered_transport.frame_payload")
	}
	return Frame{Kind: body[8], Payload: bytes.Clone(payload)}, nil
}

// DecodeFrame accepts exactly one frame and rejects coalesced or trailing data.
func DecodeFrame(profile Profile, layouts Layouts, encoded []byte) (Frame, error) {
	r := bytes.NewReader(encoded)
	frame, err := ReadFrame(profile, layouts, r)
	if err != nil {
		return Frame{}, err
	}
	if r.Len() != 0 {
		return Frame{}, fmt.Errorf("ordered_transport.frame_trailing")
	}
	return frame, nil
}

func EncodeFrame(profile Profile, layouts Layouts, frame Frame) ([]byte, error) {
	if !validProfile(profile) {
		return nil, fmt.Errorf("ordered_transport.profile")
	}
	layout, err := layouts.forKind(frame.Kind)
	if err != nil {
		return nil, err
	}
	if wasmtarget.ValidatePureValueBytes(layout, frame.Payload) != nil {
		return nil, fmt.Errorf("ordered_transport.frame_payload")
	}
	total := uint64(4 + len(frameMagic) + 1 + len(frame.Payload))
	if total > profile.MaximumFrameBytes {
		return nil, fmt.Errorf("ordered_transport.frame_size")
	}
	out := make([]byte, total)
	binary.LittleEndian.PutUint32(out[:4], uint32(total-4))
	copy(out[4:12], frameMagic[:])
	out[12] = frame.Kind
	copy(out[13:], frame.Payload)
	return out, nil
}

// WriteFrame retries short writes but never splits semantic values itself.
func WriteFrame(profile Profile, layouts Layouts, output io.Writer, frame Frame) error {
	if output == nil {
		return fmt.Errorf("ordered_transport.writer")
	}
	encoded, err := EncodeFrame(profile, layouts, frame)
	if err != nil {
		return err
	}
	for len(encoded) != 0 {
		n, writeErr := output.Write(encoded)
		if n < 0 || n > len(encoded) || n == 0 && writeErr == nil {
			return fmt.Errorf("ordered_transport.write_progress")
		}
		encoded = encoded[n:]
		if writeErr != nil {
			return fmt.Errorf("ordered_transport.write:%w", writeErr)
		}
	}
	return nil
}
