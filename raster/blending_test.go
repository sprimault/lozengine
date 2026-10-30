// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les trois arrondis dont tout le rastériseur dépend : un blanc modulé par un blanc
// doit rendre 255 et non 254, et une teinte prémultipliée doit moduler un blanc opaque
// en elle-même.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
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

// TestProductMatchesThePublishedPremultiplication vérifie que l'arrondi du rastériseur
// et celui que le paquet des structures publie ne divergent jamais.
//
// Les deux sont écrits séparément, et c'est voulu : `product` multiplie deux
// composantes quelconques sur le chemin de rendu, [render.Premultiply] applique un
// alpha au chargement. Les déplacer dans un seul endroit demanderait de sortir du
// rastériseur une fonction que le compilateur y met en ligne, ce que la règle du
// rastériseur encadre et qu'aucune charge observée ne justifie.
//
// Ce qui reste, c'est le risque que l'un dérive de l'autre, et ce cas le ferme : une
// même couleur décrite donnerait sinon deux teintes selon qu'elle passe par le
// chargement ou par la modulation. La vérification est exhaustive, les deux fonctions
// n'ayant que 65 536 couples d'entrées.
func TestProductMatchesThePublishedPremultiplication(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			attendu := render.Premultiply(uint8(a), 0, 0, uint8(b)).R
			if got := product(uint8(a), uint8(b)); got != attendu {
				t.Fatalf("produit(%d, %d) = %d, la prémultiplication publiée rend %d",
					a, b, got, attendu)
			}
		}
	}
}

// TestModulate vérifie qu'une teinte prémultipliée module un blanc opaque en
// elle-même, ce qui est la propriété qui caractérise la convention : son alpha est
// déjà dans ses composantes, donc le réappliquer les atténuerait deux fois.
//
// Le cas translucide est celui qui rougit sous l'ancien calcul, et le cas gris est
// celui qui refuse d'oublier le filtre : avec une source blanche seule, multiplier
// par la composante de la teinte ne se distingue pas de la recopier.
func TestModulate(t *testing.T) {
	grey := render.Color{R: 128, G: 128, B: 128, A: 255}
	half := render.Color{R: 128, G: 128, B: 128, A: 128}
	cream := render.Premultiply(252, 246, 226, 110)

	cases := []struct {
		name      string
		src, tint render.Color
		want      render.Color
	}{
		{"teinte neutre", white, white, white},
		{"teinte rouge", white, render.Color{R: 255, A: 255}, render.Color{R: 255, A: 255}},
		{"demi-opacité", white, half, half},
		{"teinte translucide rendue telle quelle", white, cream, cream},
		{"la teinte filtre aussi la source", grey, half, render.Color{R: 64, G: 64, B: 64, A: 128}},
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
	dst := render.Color{R: 255, G: 255, B: 255, A: 255}
	blend(&dst, render.Color{R: 128, A: 128})

	if want := (render.Color{R: 255, G: 127, B: 127, A: 255}); dst != want {
		t.Errorf("mélange : %v, attendu %v", dst, want)
	}
}

// TestAddSaturates vérifie que l'additif plafonne au lieu de reboucler.
//
// Sans plafond la somme sort de son octet et la composante repart du sombre : un
// éclair noircirait ce qu'il éclaire, et c'est un défaut qui ne se voit qu'aux endroits
// les plus clairs de l'image.
func TestAddSaturates(t *testing.T) {
	cases := []struct {
		name     string
		dst, src render.Color
		want     render.Color
	}{
		{
			"une somme qui tient dans l'octet",
			render.Color{R: 30, G: 30, B: 45, A: 255},
			render.Color{R: 90, G: 90, B: 110, A: 255},
			render.Color{R: 120, G: 120, B: 155, A: 255},
		},
		{
			"une somme qui déborde, plafonnée",
			render.Color{R: 210, G: 210, B: 210, A: 255},
			render.Color{R: 90, G: 90, B: 110, A: 255},
			render.Color{R: 255, G: 255, B: 255, A: 255},
		},
		{
			"la borne exacte, qui ne doit pas basculer",
			render.Color{R: 200, G: 100, B: 0, A: 255},
			render.Color{R: 55, G: 155, B: 255, A: 0},
			render.Color{R: 255, G: 255, B: 255, A: 255},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dst := c.dst
			add(&dst, c.src)

			if dst != c.want {
				t.Errorf("addition : %v, attendu %v", dst, c.want)
			}
		})
	}
}
