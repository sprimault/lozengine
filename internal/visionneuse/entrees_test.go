// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"strings"
	"testing"
)

// TestLireEntrees vérifie qu'un fichier d'entrées s'annote sans décaler les images :
// les lignes vides et les commentaires disparaissent, le reste garde son ordre.
func TestLireEntrees(t *testing.T) {
	fichier := `
# le pion part du bord
pion 0 30

# on le fige au centre, sans voile
pion 60 30 sans-voile
`

	entrees, err := LireEntrees(strings.NewReader(fichier))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}

	veut := []Entrees{"pion 0 30", "pion 60 30 sans-voile"}
	if len(entrees) != len(veut) {
		t.Fatalf("entrées : %q, attendu %q", entrees, veut)
	}
	for i := range veut {
		if entrees[i] != veut[i] {
			t.Errorf("entrée %d : %q, attendu %q", i, entrees[i], veut[i])
		}
	}
}

// TestEntree vérifie qu'un rejeu plus long que son fichier d'entrées continue sur des
// entrées vides, au lieu de déborder.
func TestEntree(t *testing.T) {
	entrees := []Entrees{"une", "deux"}

	cas := []struct {
		numero int
		veut   Entrees
	}{
		{-1, ""},
		{0, "une"},
		{1, "deux"},
		{2, ""},
		{99, ""},
	}
	for _, c := range cas {
		if got := Entree(entrees, c.numero); got != c.veut {
			t.Errorf("image %d : %q, attendu %q", c.numero, got, c.veut)
		}
	}
}
