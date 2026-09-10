# Go UPB-05 runtime readiness gap

Status: observed 2026-09-10; UPB-05 remains unclaimed.

The committed ordinary Go fixture and its deterministic 2,058-case native
corpus pass. The current production Go provider does not yet produce a
canonical graph for the three-argument `service.ApplyConfigured` entry.

The direct command is:

```sh
go-session-proof \
  --module modules/execution/v35/module.g1 \
  --project MATERIALIZED_UPB05_PROJECT \
  --package example.test/go-uab-11/service \
  --entry ApplyConfigured \
  --revision 1 \
  --out program.g1
```

It returns an `accepted-invalid` snapshot with `session.entry_missing` and
diagnostics for natural configuration/lifecycle bodies, including
`expression.unsupported_operator:<`, `expression.unsupported_operator:!=`, and
`expression.unsupported_node` in configuration, policy, and service functions.

This is the first runtime blocker. `canonical-observe`,
`pure-application-codec`, standalone Wasm, and pinned Pulp all consume a
canonical graph, so none can provide UPB-05 parity evidence until lifting
succeeds. Their existing recursive record/result ABI is not yet implicated by
this failure.

`reference/js/go-upb05-native-corpus-check.mjs` independently verifies that
the future target corpus contains exactly 2,058 paired observations, that each
request has the intended `ConfigInput`, `State`, and `Command` arguments, and
that the ten bounded configuration cases remain present.
