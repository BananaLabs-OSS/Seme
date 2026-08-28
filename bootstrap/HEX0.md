# Seme Hex0 v1

Hex0 is a construction notation, not a programming language or canonical Seme
representation.

## Grammar

Input is ASCII. Outside comments, the only accepted characters are hexadecimal
digits and whitespace (`space`, `tab`, `LF`, `CR`). `#` begins a comment that
continues through LF or end of input.

After removing whitespace and comments, digits are grouped in pairs. Each pair
emits one byte with the first digit as the high nibble. An odd digit count is an
error. Empty input is valid and emits an empty file.

## Invocation

```text
seme-seed-linux-amd64 INPUT.hex OUTPUT
```

Exit status `0` means success, `64` means invalid invocation, `65` means invalid
Hex0, and `74` means an operating-system I/O failure.

The output path is created or truncated with mode `0755`, allowing Hex0 to
construct the next executable bootstrap rung.

