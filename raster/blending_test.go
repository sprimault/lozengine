// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// TestProduct vérifie l'arrondi de la multiplication par un huitième de tour : la
// division entière tronquerait, et un blanc modulé par un blanc rendrait 254.
func TestProduct(t *testing.T) {
	cases := []struct {
		a, b, want uint8
	}{
		{255, 255, 255},
		{255, 0, 0},
		{0, 128, 0},
		{255, 128, 128},
		{128, 128, 64},
		{255, 127, 127},
		{200, 51, 40},
	}
	for _, c := range cases {
		if got := product(c.a, c.b); got != c.want {
			t.Errorf("produit(%d, %d) = %d, attendu %d", c.a, c.b, got, c.want)
		}
	}
}

// TestModulate vérifie que l'alpha de la teinte agit aussi sur les composantes de
// couleur : prémultipliées, les laisser en place rendrait un pixel plus sombre au
// lieu d'un pixel translucide.
func TestModulate(t *testing.T) {
	blanc := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

	cases := []struct {
		name      string
		src, tint rendu.Couleur
		want      rendu.Couleur
	}{
		{"teinte neutre", blanc, blanc, blanc},
		{"teinte rouge", blanc, rendu.Couleur{R: 255, A: 255}, rendu.Couleur{R: 255, A: 255}},
		{"demi-opacité", blanc, rendu.Couleur{R: 255, V: 255, B: 255, A: 128}, rendu.Couleur{R: 128, V: 128, B: 128, A: 128}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := modulate(c.src, c.tint); got != c.want {
				t.Errorf("moduler : %v, attendu %v", got, c.want)
			}
		})
	}
}

// TestBlend vérifie la composition d'un pixel à demi transparent sur un fond
// opaque, celle dont tout le reste du rastériseur dépend.
func TestBlend(t *testing.T) {
	dst := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	blend(&dst, rendu.Couleur{R: 128, A: 128})

	if want := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); dst != want {
		t.Errorf("mélange : %v, attendu %v", dst, want)
	}
}
