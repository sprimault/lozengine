// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le choix d'une icône et la conversion de ses pixels, éprouvés sans fenêtre : les
// deux sont des calculs, et c'est pour cela qu'ils vivent hors du backend Windows.

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// square rend une icône carrée d'une seule couleur, pour n'avoir à lire que sa taille.
func square(side uint16, tint render.Color) *render.Buffer {
	icon := render.NewBuffer(side, side)
	for i := range icon.Pixels {
		icon.Pixels[i] = tint
	}
	return icon
}

// TestPickIconTakesTheClosestSize vérifie le choix de taille, qui est tout ce que le
// moteur s'autorise : il ne redimensionne pas, donc il choisit.
func TestPickIconTakesTheClosestSize(t *testing.T) {
	small, medium, large := square(16, render.White()), square(24, render.White()), square(32, render.White())
	icons := []*render.Buffer{large, small, medium}

	cases := []struct {
		name string
		want *render.Buffer
		side uint16
	}{
		{"la taille exacte quand elle est là", small, 16},
		{"la plus proche en dessous", medium, 26},
		{"la plus proche au-dessus", medium, 22},
		{"la plus grande quand on demande plus que tout", large, 256},
		{"la plus petite quand on demande moins que tout", small, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pickIcon(icons, c.side); got != c.want {
				t.Errorf("icône de %d, attendu %d", got.Width, c.want.Width)
			}
		})
	}
}

// TestPickIconPrefersTheLargerOnATie garde la règle du départage : à écart égal, on
// réduit plutôt qu'on agrandit, parce que réduire garde les contours là qu'agrandir
// les invente.
func TestPickIconPrefersTheLargerOnATie(t *testing.T) {
	small, large := square(16, render.White()), square(32, render.White())

	if got := pickIcon([]*render.Buffer{small, large}, 24); got != large {
		t.Errorf("icône de %d retenue, attendu la plus grande", got.Width)
	}
}

// TestPickIconIgnoresWhatCannotBeDrawn vérifie qu'une entrée vide ne devient pas le
// choix par défaut. Le paquet racine les écarte déjà, et un backend ne doit pas
// dépendre de ce qu'un autre a nettoyé.
func TestPickIconIgnoresWhatCannotBeDrawn(t *testing.T) {
	good := square(32, render.White())

	if got := pickIcon([]*render.Buffer{nil, {}, good}, 16); got != good {
		t.Error("une icône vide a été retenue")
	}
	if got := pickIcon([]*render.Buffer{nil, {}}, 16); got != nil {
		t.Error("une icône a été rendue là où aucune n'était dessinable")
	}
	if got := pickIcon(nil, 16); got != nil {
		t.Error("une icône a été rendue sans qu'on en fournisse")
	}
}

// TestStraightBGRAUndoesThePremultiplication vérifie les deux conversions d'une passe :
// l'alpha qu'on rend à sa valeur pleine, et l'ordre des octets.
//
// C'est le calcul dont une erreur ne se verrait qu'à l'œil, sur une icône posée dans
// une barre des tâches : un rouge à demi transparent sortirait sombre, ou bleu.
func TestStraightBGRAUndoesThePremultiplication(t *testing.T) {
	icon := render.NewBuffer(2, 2)
	icon.Pixels[0] = render.Color{R: 255, G: 0, B: 0, A: 255} // rouge opaque
	icon.Pixels[1] = render.Color{R: 128, G: 0, B: 0, A: 128} // rouge à demi, prémultiplié
	icon.Pixels[2] = render.Color{R: 0, G: 0, B: 0, A: 0}     // rien
	icon.Pixels[3] = render.Color{R: 0, G: 64, B: 32, A: 64}  // vert sur fond de bleu, à un quart

	got := straightBGRA(icon)

	cases := []struct {
		name string
		want uint32
	}{
		{"un rouge opaque passe tel quel, bleu en poids faible", 0xFF_FF0000},
		{"un rouge à demi retrouve sa pleine valeur", 0x80_FF0000},
		{"ce qui est transparent n'a pas de couleur à retrouver", 0x00_000000},
		{"un vert au quart, son bleu rendu à sa valeur", 0x40_00FF80},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got[i] != c.want {
				t.Errorf("pixel %d : %#08x, attendu %#08x", i, got[i], c.want)
			}
		})
	}
}

// TestStraightBGRAKeepsABadPixelInItsOwnBytes garde le plafond de la conversion.
//
// Une planche qu'un jeu remplit lui-même peut porter une composante plus grande que
// son alpha, ce qu'un prémultiplié correct ne fait jamais. Le quotient dépasse alors
// 255 et, sans plafond, débordait de son octet pour écraser les voisins : un pixel
// faux en abîmait trois autres.
func TestStraightBGRAKeepsABadPixelInItsOwnBytes(t *testing.T) {
	icon := render.NewBuffer(1, 1)
	icon.Pixels[0] = render.Color{R: 255, G: 0, B: 0, A: 1}

	if got := straightBGRA(icon)[0]; got != 0x01_FF0000 {
		t.Errorf("pixel : %#08x, attendu %#08x", got, 0x01_FF0000)
	}
}
