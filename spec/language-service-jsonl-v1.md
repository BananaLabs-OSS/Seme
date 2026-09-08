# Language service JSON-lines transport v1

This process transport connects complete in-memory source snapshots to the
incremental provider session without introducing filesystem authority.

The process reads one JSON object per line from standard input and writes one
deterministic JSON response per non-empty request line to standard output.
Normal end-of-file is a successful shutdown. The default request limit is one
MiB and may be changed with `--max-message-bytes`; an oversized line produces a
bounded error response and is fully drained before the next request.
Responses use the same bound. If canonical evidence would exceed it, the
service returns `transport.response_too_large` with the correlated request ID
instead of emitting a partial frame.

```text
language-service-jsonl --module modules/execution/v12/module.g1
```

The module is loaded once at process start. The transport itself writes no
files. Artifact creation remains an explicit, separately authorized invocation
of the certified canonical build adapter.

## Requests

Every request has an `id` that is a JSON string or number, a `command`, and a
non-empty `session`. Responses preserve the exact request ID.

- `initialize` also requires `package_path` and creates one isolated session.
- `update` requires a positive, strictly newer `revision` and a `files` object
  containing the complete in-memory package snapshot.
- `snapshot` returns the most recent result without changing state.

Unknown fields reject so spelling mistakes cannot silently change meaning.
Duplicate initialization, use before initialization, malformed JSON, invalid
IDs, missing complete snapshots, and unknown commands return stable coded
errors. A protocol error does not end the stream.

An accepted invalid update advances the client revision while retaining the
last valid canonical graph and source mappings. A stale update is rejected and
also reports that retained graph. This is the existing incremental-session
behavior; the transport does not reinterpret it.

With the v14 execution module, an update may contain bounded total-return
branching, string parameters, and a string result. The process transcript gate
exercises UTF-8 string concatenation behind a branch, then proves invalid edit
retention and stale rejection against that last-valid graph.

Responses sort diagnostics by location and code, and mappings by semantic
identity. Identical request transcripts and module bytes therefore produce
byte-identical response transcripts.
