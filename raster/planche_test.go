// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "testing"

// TestNouvellePlanche vérifie les dimensions et la taille du tableau de pixels,
// dont le rastériseur dépendra sans les revérifier.
func TestNouvellePlanche(t *testing.T) {
	p := NouvellePlanche(32, 16)

	if p.Largeur != 32 || p.Hauteur != 16 {
		t.Errorf("dimensions : %d×%d", p.Largeur, p.Hauteur)
	}
	if len(p.Pixels) != 32*16 {
		t.Errorf("pixels : %d, attendu %d", len(p.Pixels), 32*16)
	}
	if p.Rectangles != nil {
		t.Errorf("découpe : %v, attendu aucune", p.Rectangles)
	}
}

// TestDecouperGrille vérifie l'ordre des cases, de gauche à droite puis de haut en
// bas, sur lequel un fournisseur d'apparence numérote ses directions.
func TestDecouperGrille(t *testing.T) {
	p := NouvellePlanche(96, 32)
	if err := p.DecouperGrille(32, 16); err != nil {
		t.Fatalf("découpe : %v", err)
	}

	attendu := []Rectangle{
		{0, 0, 32, 16}, {32, 0, 32, 16}, {64, 0, 32, 16},
		{0, 16, 32, 16}, {32, 16, 32, 16}, {64, 16, 32, 16},
	}
	if len(p.Rectangles) != len(attendu) {
		t.Fatalf("nombre de cases : %d, attendu %d", len(p.Rectangles), len(attendu))
	}
	for i, veut := range attendu {
		if p.Rectangles[i] != veut {
			t.Errorf("case %d : %v, attendu %v", i, p.Rectangles[i], veut)
		}
	}
}

// TestDecouperGrilleRefusee vérifie que les découpes qui laisseraient un reste
// échouent, plutôt que de perdre une colonne ou une ligne sans le dire.
func TestDecouperGrilleRefusee(t *testing.T) {
	cas := []struct {
		nom              string
		largeur, hauteur uint16
	}{
		{"case plus large que la planche", 128, 16},
		{"reste en largeur", 30, 16},
		{"reste en hauteur", 32, 12},
		{"case de largeur nulle", 0, 16},
		{"case de hauteur nulle", 32, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := NouvellePlanche(96, 32)
			if err := p.DecouperGrille(c.largeur, c.hauteur); err == nil {
				t.Errorf("découpe %d×%d acceptée", c.largeur, c.hauteur)
			}
		})
	}
}
