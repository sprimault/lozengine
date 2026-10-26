// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La prémultiplication offerte au jeu, et la cohérence qu'une couleur doit tenir.
// Les deux se vérifient sur toutes les valeurs possibles plutôt que sur un
// échantillon : il n'y en a que soixante-cinq mille.

package render

import "testing"

// TestPremultiplyOnKnownColors éprouve les cas qu'on sait dire de tête, ceux dont une
// erreur se lirait dans l'image.
func TestPremultiplyOnKnownColors(t *testing.T) {
	cases := []struct {
		name       string
		r, g, b, a uint8
		want       Color
	}{
		{"un blanc opaque reste intact", 255, 255, 255, 255, Color{255, 255, 255, 255}},
		{"un gris à moitié transparent", 128, 128, 128, 128, Color{64, 64, 64, 128}},
		{"ce qui est transparent n'a plus de couleur", 255, 128, 64, 0, Color{0, 0, 0, 0}},
		{"un rouge au quart", 255, 0, 0, 64, Color{64, 0, 0, 64}},
		{"une couleur opaque passe telle quelle", 10, 20, 30, 255, Color{10, 20, 30, 255}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Premultiply(c.r, c.g, c.b, c.a); got != c.want {
				t.Errorf("couleur : %v, attendu %v", got, c.want)
			}
		})
	}
}

// TestPremultiplyIsAlwaysValid vérifie sur toutes les valeurs possibles qu'une couleur
// prémultipliée par cette fonction est cohérente.
//
// C'est la propriété qui compte, et elle se démontre plutôt qu'elle ne s'échantillonne :
// aucune composante ne peut sortir au-dessus de son alpha, puisque la multiplier par un
// alpha ne la fait pas croître. Si l'arrondi la faisait déborder ne serait-ce qu'une
// fois, [Color.Valid] refuserait ce que [Premultiply] vient de produire.
func TestPremultiplyIsAlwaysValid(t *testing.T) {
	for c := 0; c < 256; c++ {
		for a := 0; a < 256; a++ {
			if got := Premultiply(uint8(c), uint8(c), uint8(c), uint8(a)); !got.Valid() {
				t.Fatalf("composante %d et alpha %d donnent %v, qui n'est pas cohérent", c, a, got)
			}
		}
	}
}

// TestPremultiplyRoundsToNearest vérifie l'arrondi sur toutes les valeurs, contre la
// division que le processeur ne sait pas faire gratuitement.
//
// Le décalage double est une astuce, et une astuce se vérifie : elle doit rendre
// exactement ce que rendrait une division arrondie au plus proche, sur les 65 536
// couples et non sur ceux qu'on aurait choisis.
func TestPremultiplyRoundsToNearest(t *testing.T) {
	for c := 0; c < 256; c++ {
		for a := 0; a < 256; a++ {
			want := uint8((c*a + 127) / 255)
			if got := applyAlpha(uint8(c), uint8(a)); got != want {
				t.Fatalf("%d × %d / 255 : %d, attendu %d", c, a, got, want)
			}
		}
	}
}

// TestValidRefusesWhatDescribesNoColor garde la faute que la méthode existe pour
// attraper : une composante écrite sans appliquer son alpha.
func TestValidRefusesWhatDescribesNoColor(t *testing.T) {
	cases := []struct {
		name  string
		color Color
		want  bool
	}{
		{"un blanc opaque", Color{255, 255, 255, 255}, true},
		{"un gris correctement prémultiplié", Color{64, 64, 64, 128}, true},
		{"le transparent total", Color{0, 0, 0, 0}, true},
		{"la borne exacte, où la composante vaut l'alpha", Color{128, 128, 128, 128}, true},
		{"un gris écrit à plat, sans appliquer l'alpha", Color{128, 128, 128, 64}, false},
		{"une seule composante qui dépasse", Color{0, 0, 65, 64}, false},
		{"une couleur posée sur un alpha nul", Color{1, 0, 0, 0}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.color.Valid(); got != c.want {
				t.Errorf("cohérence : %t, attendu %t", got, c.want)
			}
		})
	}
}
