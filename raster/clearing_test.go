// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'effacement doit laisser derrière lui — rien du dessin précédent — et les
// deux cas qui cassent un remplissage par doublement : le tampon vide et le tampon
// d'une seule ligne.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestClear vérifie les deux régimes, le transparent passant par une mise à zéro
// en bloc et la couleur par une écriture pixel par pixel.
func TestClear(t *testing.T) {
	cases := []struct {
		name  string
		color render.Color
	}{
		{"transparent", render.Color{}},
		{"couleur opaque", render.Color{R: 10, G: 20, B: 30, A: 255}},
		{"couleur translucide", halfRed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buffer := render.NewBuffer(4, 3)
			for i := range buffer.Pixels {
				buffer.Pixels[i] = white
			}

			Clear(buffer, c.color)

			for i, got := range buffer.Pixels {
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
	Clear(render.NewBuffer(0, 0), white)
}

// TestClearCoversADrawing vérifie qu'une image ne garde rien de la précédente,
// le tampon étant réutilisé d'une image à l'autre.
func TestClearCoversADrawing(t *testing.T) {
	a := atlasOf(t, 1, 1, red)
	buffer := render.NewBuffer(2, 2)

	Draw(buffer, render.DrawList{quadOf(0, 0, 1, 1)}, a)
	Clear(buffer, render.Color{})

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels restants : %d, attendu aucun", n)
	}
}

// TestClearDoesNotAllocate vérifie que la première opération de chaque image reste
// hors du chemin d'allocation, comme le blit qui la suit.
func TestClearDoesNotAllocate(t *testing.T) {
	buffer := render.NewBuffer(64, 64)

	if n := testing.AllocsPerRun(10, func() { Clear(buffer, white) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
