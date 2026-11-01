// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la relecture en octets doit tenir : qu'elle alias, qu'elle couvre tout le
// tampon, et qu'un tampon vide ne la fasse pas paniquer.

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestBytesOfAliasesThePixels écrit par la tranche d'octets et relit le pixel.
//
// **C'est le seul cas de ce fichier qu'aucune autre couche n'attrape**, et le mode de panne
// qu'il garde est muet : une relecture qui copierait au lieu d'aliaser rendrait une
// présentation correcte à la première image et figée ensuite, sans erreur ni message. Rien
// dans la suite ne verrait la différence — les octets sont les bons, ils ne sont simplement
// plus ceux du tampon que le moteur repeint.
//
// L'ordre des composantes, lui, est gardé ailleurs et par le même chemin : `TestColorLayout`
// du paquet `render` fige les quatre décalages, donc ce cas n'a pas à les revérifier.
func TestBytesOfAliasesThePixels(t *testing.T) {
	pixels := []render.Color{{R: 1, G: 2, B: 3, A: 4}, {R: 5, G: 6, B: 7, A: 8}}

	raw := bytesOf(pixels)
	raw[4] = 50

	if got, want := pixels[1].R, uint8(50); got != want {
		t.Errorf("le rouge du second pixel vaut %d, attendu %d — la tranche copie au lieu d'aliaser", got, want)
	}
}

// TestBytesOfCoversTheWholeBuffer vérifie la longueur, quatre octets par pixel.
//
// Il rougit sur un facteur faux — trois, ou une longueur en pixels — qui laisserait le
// dernier quart du tampon hors de ce qu'on présente, donc une bande non peinte en bas de
// l'image. Le défaut serait visible, mais seulement une fois un canvas monté : ici il se
// voit sur l'hôte.
func TestBytesOfCoversTheWholeBuffer(t *testing.T) {
	for _, pixels := range []int{1, 2, 480 * 270} {
		got := len(bytesOf(make([]render.Color, pixels)))
		if want := pixels * 4; got != want {
			t.Errorf("%d pixels donnent %d octets, attendu %d", pixels, got, want)
		}
	}
}

// TestBytesOfAcceptsAnEmptyBuffer vérifie qu'un tampon vide ne panique pas.
//
// Sans la garde, `&pixels[0]` indexe un élément qui n'existe pas. Le cas n'est pas
// défensif pour rien : une fenêtre réduite rend une taille nulle, ce que le backend
// Windows documente déjà comme un régime à traverser sans casser.
func TestBytesOfAcceptsAnEmptyBuffer(t *testing.T) {
	if got := bytesOf(nil); len(got) != 0 {
		t.Errorf("un tampon vide rend %d octets, attendu 0", len(got))
	}
}
