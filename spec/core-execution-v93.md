# Core Execution v93

Core Execution v93 lifts Go character literals to their language-neutral
Unicode scalar integer meaning.

Go's type checker still determines the literal's context. When the value enters
a native call, the existing typed boundary records the required native target
type. Go projection may render the exact scalar as an integer token; the value
and call behavior remain exact without introducing a Go-specific character
concept into neutral Core.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
