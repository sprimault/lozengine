// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'un fichier d'entrées admet sans décaler les images — lignes vides et
// commentaires — et ce qu'un rejeu fait quand il dépasse la fin du fichier : il
// continue sur des entrées vides, ce qui n'est pas une erreur.

package viewer

import (
	"strings"
	"testing"
)

// TestReadInputLines vérifie qu'un fichier d'entrées s'annote sans décaler les images :
// les lignes vides et les commentaires disparaissent, le reste garde son ordre.
func TestReadInputLines(t *testing.T) {
	content := `
# le pion part du bord
pion 0 30

# on le fige au centre, sans voile
pion 60 30 sans-voile
`

	lines, err := ReadInputLines(strings.NewReader(content))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}

	want := []InputLine{"pion 0 30", "pion 60 30 sans-voile"}
	if len(lines) != len(want) {
		t.Fatalf("entrées : %q, attendu %q", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("entrée %d : %q, attendu %q", i, lines[i], want[i])
		}
	}
}

// TestInputLineAt vérifie qu'un rejeu plus long que son fichier d'entrées continue sur des
// entrées vides, au lieu de déborder.
func TestInputLineAt(t *testing.T) {
	lines := []InputLine{"une", "deux"}

	cases := []struct {
		frame int
		want  InputLine
	}{
		{-1, ""},
		{0, "une"},
		{1, "deux"},
		{2, ""},
		{99, ""},
	}
	for _, c := range cases {
		if got := InputLineAt(lines, c.frame); got != c.want {
			t.Errorf("image %d : %q, attendu %q", c.frame, got, c.want)
		}
	}
}
