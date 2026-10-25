// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// TestClear vérifie les deux régimes, le transparent passant par une mise à zéro
// en bloc et la couleur par une écriture pixel par pixel.
func TestClear(t *testing.T) {
	cases := []struct {
		name  string
		color rendu.Couleur
	}{
		{"transparent", rendu.Couleur{}},
		{"couleur opaque", rendu.Couleur{R: 10, V: 20, B: 30, A: 255}},
		{"couleur translucide", demiRge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tampon := rendu.NouveauTampon(4, 3)
			for i := range tampon.Pixels {
				tampon.Pixels[i] = blanc
			}

			Clear(tampon, c.color)

			for i, got := range tampon.Pixels {
				if got != c.color {
					t.Fatalf("pixel %d : %v, attendu %v", i, got, c.color)
				}
			}
		})
	}
}

// TestClearEmptyBuffer vérifie qu'un tampon sans pixel s'efface sans déborder : le
// doublement part du premier pixel, qui n'existe pas ici.
func TestClearEmptyBuffer(t *testing.T) {
	Clear(rendu.NouveauTampon(0, 0), blanc)
}

// TestClearCoversADrawing vérifie qu'une image ne garde rien de la précédente,
// le tampon étant réutilisé d'une image à l'autre.
func TestClearCoversADrawing(t *testing.T) {
	a := atlasOf(t, 1, 1, rouge)
	buffer := rendu.NouveauTampon(2, 2)

	Draw(buffer, rendu.Rendu{quadOf(0, 0, 1, 1)}, a)
	Clear(buffer, rendu.Couleur{})

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels restants : %d, attendu aucun", n)
	}
}

// TestClearDoesNotAllocate vérifie que la première opération de chaque image reste
// hors du chemin d'allocation, comme le blit qui la suit.
func TestClearDoesNotAllocate(t *testing.T) {
	buffer := rendu.NouveauTampon(64, 64)

	if n := testing.AllocsPerRun(10, func() { Clear(buffer, blanc) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
