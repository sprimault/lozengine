// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// TestProduit vérifie l'arrondi de la multiplication par un huitième de tour : la
// division entière tronquerait, et un blanc modulé par un blanc rendrait 254.
func TestProduit(t *testing.T) {
	cas := []struct {
		a, b, veut uint8
	}{
		{255, 255, 255},
		{255, 0, 0},
		{0, 128, 0},
		{255, 128, 128},
		{128, 128, 64},
		{255, 127, 127},
		{200, 51, 40},
	}
	for _, c := range cas {
		if got := produit(c.a, c.b); got != c.veut {
			t.Errorf("produit(%d, %d) = %d, attendu %d", c.a, c.b, got, c.veut)
		}
	}
}

// TestModuler vérifie que l'alpha de la teinte agit aussi sur les composantes de
// couleur : prémultipliées, les laisser en place rendrait un pixel plus sombre au
// lieu d'un pixel translucide.
func TestModuler(t *testing.T) {
	blanc := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

	cas := []struct {
		nom         string
		src, teinte rendu.Couleur
		veut        rendu.Couleur
	}{
		{"teinte neutre", blanc, blanc, blanc},
		{"teinte rouge", blanc, rendu.Couleur{R: 255, A: 255}, rendu.Couleur{R: 255, A: 255}},
		{"demi-opacité", blanc, rendu.Couleur{R: 255, V: 255, B: 255, A: 128}, rendu.Couleur{R: 128, V: 128, B: 128, A: 128}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := moduler(c.src, c.teinte); got != c.veut {
				t.Errorf("moduler : %v, attendu %v", got, c.veut)
			}
		})
	}
}

// TestMelanger vérifie la composition d'un pixel à demi transparent sur un fond
// opaque, celle dont tout le reste du rastériseur dépend.
func TestMelanger(t *testing.T) {
	dst := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	melanger(&dst, rendu.Couleur{R: 128, A: 128})

	if veut := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); dst != veut {
		t.Errorf("mélange : %v, attendu %v", dst, veut)
	}
}
