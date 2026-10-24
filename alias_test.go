// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"testing"

	"github.com/sprimault/lozengine/internal/geometrie"
	"github.com/sprimault/lozengine/rendu"
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

// TestConstructeursRepublies vérifie que chaque enveloppe rend la matrice du noyau.
//
// Ce qu'elle est seule à pouvoir attraper : une enveloppe qui appellerait le mauvais
// constructeur, ou qui échangerait ses deux paramètres, se compilerait sans broncher
// et ne se verrait qu'à l'image. La comparaison elle-même prouve au passage que
// Matrice est bien un alias et non un type distinct — deux types différents ne se
// compareraient pas.
func TestConstructeursRepublies(t *testing.T) {
	cas := []struct {
		nom           string
		racine, noyau Matrice
	}{
		{"Dimetrique", Dimetrique(32, 16), geometrie.Dimetrique(32, 16)},
		{"Isometrique", Isometrique(32), geometrie.Isometrique(32)},
		{"Grille", Grille(32), geometrie.Grille(32)},
		{"Brute", Brute(2, 3, 5, 7), geometrie.Brute(2, 3, 5, 7)},
	}
	for _, c := range cas {
		if c.racine != c.noyau {
			t.Errorf("%s republié ne rend pas la matrice du noyau", c.nom)
		}
	}
}

// TestMatriceSUtiliseDepuisLaRacine vérifie qu'une matrice obtenue de la racine
// s'emploie sans rien importer d'autre : c'est tout ce qu'un consommateur peut en
// faire au sortir de ce jalon, et le seul cas qui le démontre.
func TestMatriceSUtiliseDepuisLaRacine(t *testing.T) {
	m := Dimetrique(32, 16).AvecEtage(8)

	ex, ey, profondeur, hauteur := m.Projeter(1, 1, 1)
	if ex != 0 || ey != 8 || profondeur != 8 || hauteur != 4 {
		t.Errorf("projeté (%d, %d), profondeur %d, hauteur %d — attendu (0, 8), 8 et 4", ex, ey, profondeur, hauteur)
	}

	if x, y := m.Inverser(ex, ey); x != 0.5 || y != 0.5 {
		t.Errorf("inversé (%g, %g), attendu (0.5, 0.5)", x, y)
	}
	if n := m.CellulesMax(480, 270); n == 0 {
		t.Error("majorant de cellules nul")
	}
}
