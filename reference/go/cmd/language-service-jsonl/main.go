// Command language-service-jsonl exposes the incremental provider session over
// a bounded JSON-lines stdin/stdout transport. It performs no filesystem writes.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"seme.local/reference/goprovider"
)

const defaultMaxMessage = 1 << 20

type request struct {
	ID          json.RawMessage   `json:"id"`
	Command     string            `json:"command"`
	Session     string            `json:"session"`
	PackagePath string            `json:"package_path,omitempty"`
	Revision    uint64            `json:"revision,omitempty"`
	Files       map[string]string `json:"files,omitempty"`
}

type response struct {
	ID      json.RawMessage `json:"id"`
	OK      bool            `json:"ok"`
	Session string          `json:"session,omitempty"`
	State   *state          `json:"state,omitempty"`
	Error   *protocolError  `json:"error,omitempty"`
}

type protocolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type state struct {
	Initialized       bool                             `json:"initialized"`
	Revision          uint64                           `json:"revision"`
	Accepted          bool                             `json:"accepted"`
	Valid             bool                             `json:"valid"`
	LastValidRevision uint64                           `json:"last_valid_revision"`
	ContentDigest     string                           `json:"content_digest,omitempty"`
	Disposition       string                           `json:"disposition,omitempty"`
	CanonicalG1       string                           `json:"canonical_g1,omitempty"`
	Diagnostics       []goprovider.SessionDiagnostic   `json:"diagnostics"`
	Sources           []goprovider.SourceIdentity      `json:"sources"`
	References        []goprovider.ReferenceOccurrence `json:"references"`
}

type liveSession struct {
	provider    *goprovider.IncrementalSession
	packagePath string
	latest      *goprovider.SessionResult
}

func main() {
	modulePath := flag.String("module", "", "Core Execution module G1")
	maxMessage := flag.Int("max-message-bytes", defaultMaxMessage, "maximum JSON request line size")
	flag.Parse()
	if *modulePath == "" || *maxMessage < 256 {
		fmt.Fprintln(os.Stderr, "language-service-jsonl: requires --module and --max-message-bytes >= 256")
		os.Exit(64)
	}
	module, err := os.ReadFile(*modulePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "language-service-jsonl:", err)
		os.Exit(66)
	}
	if err := serve(os.Stdin, os.Stdout, module, *maxMessage); err != nil {
		fmt.Fprintln(os.Stderr, "language-service-jsonl:", err)
		os.Exit(74)
	}
}

func serve(input io.Reader, output io.Writer, module []byte, maxMessage int) error {
	reader := bufio.NewReaderSize(input, min(maxMessage+1, 64<<10))
	sessions := map[string]*liveSession{}
	for {
		line, tooLarge, err := readBoundedLine(reader, maxMessage)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if tooLarge {
			if encodeErr := writeResponse(output, errorResponse(nil, "transport.message_too_large", "request exceeds configured byte limit"), maxMessage); encodeErr != nil {
				return encodeErr
			}
		} else if len(bytes.TrimSpace(line)) != 0 {
			if encodeErr := writeResponse(output, dispatch(line, module, sessions), maxMessage); encodeErr != nil {
				return encodeErr
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}

func writeResponse(output io.Writer, item response, limit int) error {
	encoded, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if len(encoded)+1 > limit {
		encoded, err = json.Marshal(errorResponse(item.ID, "transport.response_too_large", "response exceeds configured byte limit"))
		if err != nil {
			return err
		}
	}
	encoded = append(encoded, '\n')
	written, err := output.Write(encoded)
	if err == nil && written != len(encoded) {
		return io.ErrShortWrite
	}
	return err
}

func readBoundedLine(reader *bufio.Reader, limit int) ([]byte, bool, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(line)+len(part) > limit {
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = reader.ReadSlice('\n')
			}
			return nil, true, err
		}
		line = append(line, part...)
		if !errors.Is(err, bufio.ErrBufferFull) {
			return bytes.TrimSuffix(line, []byte{'\n'}), false, err
		}
	}
}

func dispatch(line, module []byte, sessions map[string]*liveSession) response {
	var req request
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return errorResponse(nil, "transport.invalid_json", err.Error())
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errorResponse(req.ID, "transport.invalid_json", "request must contain exactly one JSON value")
	}
	if !validID(req.ID) {
		return errorResponse(nil, "transport.invalid_id", "id must be a JSON string or number")
	}
	if req.Session == "" {
		return errorResponse(req.ID, "transport.session_required", "session is required")
	}
	switch req.Command {
	case "initialize":
		if req.PackagePath == "" {
			return errorResponse(req.ID, "initialize.package_path_required", "package_path is required")
		}
		if _, exists := sessions[req.Session]; exists {
			return errorResponse(req.ID, "initialize.session_exists", "session already exists")
		}
		provider, err := goprovider.NewIncrementalSession(module)
		if err != nil {
			return errorResponse(req.ID, "initialize.module_invalid", err.Error())
		}
		sessions[req.Session] = &liveSession{provider: provider, packagePath: req.PackagePath}
		return response{ID: req.ID, OK: true, Session: req.Session, State: emptyState()}
	case "update":
		current, exists := sessions[req.Session]
		if !exists {
			return errorResponse(req.ID, "session.not_initialized", "session is not initialized")
		}
		if req.Files == nil {
			return errorResponse(req.ID, "update.files_required", "files must contain a complete in-memory snapshot")
		}
		result := current.provider.Apply(goprovider.DocumentSnapshot{Revision: req.Revision, PackagePath: current.packagePath, Files: req.Files})
		current.latest = &result
		view := stateFromResult(result)
		return response{ID: req.ID, OK: true, Session: req.Session, State: &view}
	case "snapshot":
		current, exists := sessions[req.Session]
		if !exists {
			return errorResponse(req.ID, "session.not_initialized", "session is not initialized")
		}
		if current.latest == nil {
			return response{ID: req.ID, OK: true, Session: req.Session, State: emptyState()}
		}
		view := stateFromResult(*current.latest)
		return response{ID: req.ID, OK: true, Session: req.Session, State: &view}
	default:
		return errorResponse(req.ID, "transport.unknown_command", "unknown command")
	}
}

func emptyState() *state {
	return &state{Initialized: true, Diagnostics: []goprovider.SessionDiagnostic{}, Sources: []goprovider.SourceIdentity{}, References: []goprovider.ReferenceOccurrence{}}
}

func stateFromResult(result goprovider.SessionResult) state {
	diagnostics := append([]goprovider.SessionDiagnostic(nil), result.Diagnostics...)
	sources := append([]goprovider.SourceIdentity(nil), result.Sources...)
	references := append([]goprovider.ReferenceOccurrence(nil), result.References...)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].File != diagnostics[j].File {
			return diagnostics[i].File < diagnostics[j].File
		}
		if diagnostics[i].Line != diagnostics[j].Line {
			return diagnostics[i].Line < diagnostics[j].Line
		}
		if diagnostics[i].Column != diagnostics[j].Column {
			return diagnostics[i].Column < diagnostics[j].Column
		}
		return diagnostics[i].Code < diagnostics[j].Code
	})
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
	sort.SliceStable(references, func(i, j int) bool {
		if references[i].Document != references[j].Document {
			return references[i].Document < references[j].Document
		}
		return references[i].Start < references[j].Start
	})
	return state{Initialized: true, Revision: result.Revision, Accepted: result.Accepted, Valid: result.Valid, LastValidRevision: result.LastValidRevision, ContentDigest: result.ContentDigest, Disposition: result.Disposition, CanonicalG1: result.CanonicalG1, Diagnostics: diagnostics, Sources: sources, References: references}
}

func validID(id json.RawMessage) bool {
	trimmed := bytes.TrimSpace(id)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	switch value.(type) {
	case string, json.Number:
		return true
	default:
		return false
	}
}

func errorResponse(id json.RawMessage, code, message string) response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return response{ID: id, OK: false, Error: &protocolError{Code: code, Message: message}}
}
