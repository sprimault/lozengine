// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// TestConstantesRepubliees vérifie que chaque constante republiée vaut celle du
// noyau. Un bloc const Go répète l'expression précédente quand une ligne n'en
// porte pas : l'oubli ne se compile pas en erreur, il se compile en valeur
// fausse.
func TestConstantesRepubliees(t *testing.T) {
	cas := []struct {
		nom           string
		racine, noyau Drapeaux
	}{
		{"MiroirX", MiroirX, rendu.MiroirX},
		{"MiroirY", MiroirY, rendu.MiroirY},
		{"Aplat", Aplat, rendu.Aplat},
	}
	for _, c := range cas {
		if c.racine != c.noyau {
			t.Errorf("%s republié : %#b, attendu %#b", c.nom, c.racine, c.noyau)
		}
	}
}
