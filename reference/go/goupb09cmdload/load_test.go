package goupb09cmdload

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUsesRegularOpenFileAuthority(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "selection.json")
	if err := os.WriteFile(p, []byte("authority"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := read(p)
	if err != nil || !bytes.Equal(b, []byte("authority")) {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link.json")
	if err = os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if _, err = read(link); err == nil {
		t.Fatal("symlink accepted")
	}
}
