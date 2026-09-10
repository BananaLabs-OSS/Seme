package presentationinstance_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"seme.local/reference/internal/upb07testfixture"
	"seme.local/reference/presentationinstance"
	"seme.local/reference/wire"
)

func TestAuthenticatedPresentationAdversaries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fx, err := upb07testfixture.Build(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	in := fx.Result.Presentation
	if len(in.Model.Aliases) < 2 {
		t.Fatalf("need multiple authenticated aliases: %d", len(in.Model.Aliases))
	}
	a, err := presentationinstance.Emit(in)
	if err != nil || !bytes.Equal(a, in.Artifact) {
		t.Fatalf("reproduction: %v", err)
	}
	reversed := cloneModel(in.Model)
	for i, j := 0, len(reversed.Aliases)-1; i < j; i, j = i+1, j-1 {
		reversed.Aliases[i], reversed.Aliases[j] = reversed.Aliases[j], reversed.Aliases[i]
	}
	x := in
	x.Model = reversed
	b, err := presentationinstance.Emit(x)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("input order changed canonical artifact", err)
	}
	reject := func(name string, mutate func(*presentationinstance.Model)) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			m := cloneModel(in.Model)
			mutate(&m)
			q := in
			q.Model = m
			q.Artifact = nil
			if out, er := presentationinstance.Emit(q); er == nil || len(out) != 0 {
				t.Fatalf("accepted %s: %v", name, er)
			}
		})
	}
	reject("duplicate-owner-name", func(m *presentationinstance.Model) { z := m.Aliases[0]; m.Aliases = append(m.Aliases, z) })
	reject("exported-lowercase", func(m *presentationinstance.Model) { m.Aliases[0].Name = "lower"; m.Aliases[0].Visibility = 1 })
	reject("private-uppercase", func(m *presentationinstance.Model) { m.Aliases[0].Name = "Upper"; m.Aliases[0].Visibility = 0 })
	reject("unknown-visibility", func(m *presentationinstance.Model) { m.Aliases[0].Visibility = 2 })
	reject("foreign-owner", func(m *presentationinstance.Model) { m.Aliases[0].Owner = wire.ID{} })
	reject("unknown-target", func(m *presentationinstance.Model) { m.Aliases[0].Target = wire.ID{} })
	reject("unknown-source", func(m *presentationinstance.Model) { m.Aliases[0].SourceUnit = wire.ID{} })
	reject("source-path", func(m *presentationinstance.Model) { m.Aliases[0].Path += ".foreign" })
	reject("source-digest", func(m *presentationinstance.Model) { m.Aliases[0].Digest[0] ^= 1 })
	reject("empty-span", func(m *presentationinstance.Model) { m.Aliases[0].End = m.Aliases[0].Start })
	reject("span-overrun", func(m *presentationinstance.Model) { m.Aliases[0].End = ^uint64(0) })
	reject("zero-line", func(m *presentationinstance.Model) { m.Aliases[0].StartLine = 0 })
	reject("duplicate-binding", func(m *presentationinstance.Model) {
		i := aliasWithBinding(t, m.Aliases)
		m.Aliases[i].ImportBindings = append(m.Aliases[i].ImportBindings, m.Aliases[i].ImportBindings[0])
	})
	reject("missing-binding", func(m *presentationinstance.Model) { m.Aliases[0].ImportBindings = []wire.ID{{1}} })
	reject("foreign-binding", func(m *presentationinstance.Model) {
		i := aliasWithBinding(t, m.Aliases)
		for j := range m.Aliases {
			if m.Aliases[j].Owner != m.Aliases[i].Owner && len(m.Aliases[j].ImportBindings) != 0 {
				m.Aliases[i].ImportBindings = []wire.ID{m.Aliases[j].ImportBindings[0]}
				return
			}
		}
		t.Fatal("authenticated fixture lacks a foreign-owner binding adversary")
	})
	reject("over-limit", func(m *presentationinstance.Model) {
		base := m.Aliases[0]
		m.Aliases = nil
		for i := 0; i <= presentationinstance.MaxAliases; i++ {
			z := base
			z.Name = fmt.Sprintf("Alias%03d", i)
			z.Visibility = 1
			m.Aliases = append(m.Aliases, z)
		}
	})
	t.Run("artifact-tamper", func(t *testing.T) {
		q := in
		q.Artifact = append([]byte(nil), in.Artifact...)
		q.Artifact[len(q.Artifact)-1] ^= 1
		if presentationinstance.Validate(q) == nil {
			t.Fatal("tamper accepted")
		}
	})
	t.Run("mixed-project", func(t *testing.T) {
		q := in
		q.ProjectV9.Composed = append([]byte(nil), in.ProjectV9.Composed...)
		q.ProjectV9.Composed[len(q.ProjectV9.Composed)-1] ^= 1
		if presentationinstance.Validate(q) == nil {
			t.Fatal("mixed/tampered Project-v9 accepted")
		}
	})
}

func aliasWithBinding(t *testing.T, aliases []presentationinstance.Alias) int {
	t.Helper()
	for i := range aliases {
		if len(aliases[i].ImportBindings) != 0 {
			return i
		}
	}
	t.Fatal("authenticated fixture contains no import binding")
	return 0
}

func cloneModel(in presentationinstance.Model) presentationinstance.Model {
	out := presentationinstance.Model{Aliases: append([]presentationinstance.Alias(nil), in.Aliases...)}
	for i := range out.Aliases {
		out.Aliases[i].ImportBindings = append([]wire.ID(nil), out.Aliases[i].ImportBindings...)
	}
	return out
}
