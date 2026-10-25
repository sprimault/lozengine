// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La découpe en grille : ce qu'elle rend quand la case divise la planche, et ce qu'elle
// refuse quand elle ne la divise pas.

package raster

import "testing"

// TestNewSheet vérifie les dimensions et la taille du tableau de pixels,
// dont le rastériseur dépendra sans les revérifier.
func TestNewSheet(t *testing.T) {
	p := NewSheet(32, 16)

	if p.Width != 32 || p.Height != 16 {
		t.Errorf("dimensions : %d×%d", p.Width, p.Height)
	}
	if len(p.Pixels) != 32*16 {
		t.Errorf("pixels : %d, attendu %d", len(p.Pixels), 32*16)
	}
	if p.Rectangles != nil {
		t.Errorf("découpe : %v, attendu aucune", p.Rectangles)
	}
}

// TestCutGrid vérifie l'ordre des cases, de gauche à droite puis de haut en
// bas, sur lequel un fournisseur d'apparence numérote ses directions.
func TestCutGrid(t *testing.T) {
	p := NewSheet(96, 32)
	if err := p.CutGrid(32, 16); err != nil {
		t.Fatalf("découpe : %v", err)
	}

	want := []Rectangle{
		{0, 0, 32, 16}, {32, 0, 32, 16}, {64, 0, 32, 16},
		{0, 16, 32, 16}, {32, 16, 32, 16}, {64, 16, 32, 16},
	}
	if len(p.Rectangles) != len(want) {
		t.Fatalf("nombre de cases : %d, attendu %d", len(p.Rectangles), len(want))
	}
	for i := range want {
		if p.Rectangles[i] != want[i] {
			t.Errorf("case %d : %v, attendu %v", i, p.Rectangles[i], want[i])
		}
	}
}

// TestCutGridRefused vérifie que les découpes qui laisseraient un reste
// échouent, plutôt que de perdre une colonne ou une ligne sans le dire.
func TestCutGridRefused(t *testing.T) {
	cases := []struct {
		name          string
		width, height uint16
	}{
		{"case plus large que la planche", 128, 16},
		{"reste en largeur", 30, 16},
		{"reste en hauteur", 32, 12},
		{"case de largeur nulle", 0, 16},
		{"case de hauteur nulle", 32, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewSheet(96, 32)
			if err := p.CutGrid(c.width, c.height); err == nil {
				t.Errorf("découpe %d×%d acceptée", c.width, c.height)
			}
		})
	}
}
