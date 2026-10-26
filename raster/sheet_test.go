// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La découpe en grille : ce qu'elle rend quand la case divise la planche, et ce qu'elle
// refuse quand elle ne la divise pas.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

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

// TestNewSheetFromStraightAppliesTheAlpha vérifie que la porte des octets à plat
// applique l'alpha, et qu'elle le fait comme le reste du moteur.
//
// Le cas qui compte est le pixel à alpha partiel : c'est le seul où l'oubli de la
// conversion se voit, et il se verrait comme un pixel trop dense plutôt que comme une
// erreur de chargement.
func TestNewSheetFromStraightAppliesTheAlpha(t *testing.T) {
	// Deux pixels : un rouge à moitié transparent, puis un blanc opaque.
	p, err := NewSheetFromStraight(2, 1, []uint8{
		255, 0, 0, 128,
		255, 255, 255, 255,
	})
	if err != nil {
		t.Fatalf("planche : %v", err)
	}

	if want := (render.Color{R: 128, G: 0, B: 0, A: 128}); p.Pixels[0] != want {
		t.Errorf("le pixel à demi : %v, attendu %v", p.Pixels[0], want)
	}
	if want := render.White(); p.Pixels[1] != want {
		t.Errorf("le pixel opaque : %v, attendu %v", p.Pixels[1], want)
	}
	for i, pixel := range p.Pixels {
		if !pixel.Valid() {
			t.Errorf("le pixel %d, %v, n'est pas un prémultiplié cohérent", i, pixel)
		}
	}
}

// TestNewSheetFromStraightRefusesTheWrongLength garde les deux fautes de dimension,
// qui se taisent toutes les deux à l'image.
//
// Une tranche trop courte laisserait une bande transparente au bas de la planche, une
// trop longue dit qu'on s'est trompé de largeur ou de hauteur. Ni l'une ni l'autre ne
// ressemble à une erreur de chargement quand on la voit.
func TestNewSheetFromStraightRefusesTheWrongLength(t *testing.T) {
	cases := []struct {
		name          string
		width, height uint16
		pixels        int
	}{
		{"une tranche trop courte d'un pixel", 2, 2, 12},
		{"une tranche trop longue d'un pixel", 2, 2, 20},
		{"une tranche vide là où des pixels sont attendus", 1, 1, 0},
		{"des octets qui ne tombent pas sur un pixel entier", 2, 1, 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewSheetFromStraight(c.width, c.height, make([]uint8, c.pixels)); err == nil {
				t.Error("la planche a été acceptée")
			}
		})
	}
}

// TestNewSheetFromStraightAcceptsAnEmptySheet vérifie la borne basse : une planche
// sans pixel s'accorde avec une tranche vide, et il n'y a rien à refuser.
func TestNewSheetFromStraightAcceptsAnEmptySheet(t *testing.T) {
	if _, err := NewSheetFromStraight(0, 0, nil); err != nil {
		t.Errorf("une planche vide a été refusée : %v", err)
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
