// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'exemple doit tenir sans qu'on ouvre sa fenêtre : ses planches se montent, et
// sa police écrit vraiment ce qu'il affiche.

package main

import "testing"

// TestBuildMountsTheAtlas vérifie que les quatre planches entrent dans l'atlas.
//
// Un exemple ne se joue pas en intégration continue, donc ce qui s'y casse ne se voit
// qu'à la main. Ce cas couvre la part qui se vérifie sans écran : un montage qui
// paniquerait — une planche de taille nulle, un atlas plein — tomberait ici plutôt
// qu'au lancement.
func TestBuildMountsTheAtlas(t *testing.T) {
	atlas, font := build()

	if atlas == nil {
		t.Fatal("atlas absent")
	}
	if font.Width == 0 || font.Height == 0 {
		t.Fatalf("police de %d×%d, attendue non vide", font.Width, font.Height)
	}
}

// TestTheFontCarriesWhatTheExampleWrites garde l'accord entre la planche de glyphes et
// les deux lignes que l'exemple affiche.
//
// **C'est le seul cas à pouvoir attraper un signe manquant**, et c'est un défaut qui
// ne se voit pas autrement : un caractère absent ne pose rien et garde sa place, donc
// la ligne s'affiche avec un trou au lieu de se plaindre. Il parcourt les deux textes
// que [game.writeOverlay] peut écrire, chiffres compris.
func TestTheFontCarriesWhatTheExampleWrites(t *testing.T) {
	_, font := build()

	for _, line := range []string{"DESIGNE AUCUNE CASE", "CASE 0123456789"} {
		for _, r := range line {
			if r == ' ' {
				continue
			}
			if _, ok := font.Glyph(r); !ok {
				t.Errorf("%q : la planche ne porte pas %q", line, r)
			}
		}
	}
}

// TestTheOverlayFitsTheScreen vérifie que la ligne la plus large tient dans la
// résolution interne.
//
// L'exemple centre son texte sur 480 pixels ; une ligne plus large déborderait des
// deux côtés, et le découpage la rognerait sans rien dire.
func TestTheOverlayFitsTheScreen(t *testing.T) {
	_, font := build()

	if width, _ := font.Measure("DESIGNE AUCUNE CASE"); width > 480 {
		t.Errorf("ligne de %d pixels, la résolution interne en fait 480", width)
	}
}
