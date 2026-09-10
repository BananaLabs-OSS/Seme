// Package checksum is the single pinned ecosystem dependency of the bounded
// UPB-03 fixture.
package checksum

func Mix(value int64) int64 {
	return value*3 + 7
}
