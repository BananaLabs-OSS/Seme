# Go UPB-05 configuration overlay

This cumulative fixture is materialized from the byte-pinned Go UPB-04/UAB-11
project and this ordinary Go source delta. The original 2,048-command corpus
remains unchanged. Configuration is explicit typed input: no environment,
filesystem, clock, random source, package initializer, or mutable global is
consulted.

Initialization is observable in ordinary values. `configuration.Initialize`
produces stage 1, `policy.Initialize` accepts only stage 1 and produces stage
2, and `service.Initialize` finalizes stage 3 with `Ready` set. Calls made out
of order reject without changing caller-owned values.
