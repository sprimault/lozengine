// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// TestEffacer vérifie les deux régimes, le transparent passant par une mise à zéro
// en bloc et la couleur par une écriture pixel par pixel.
func TestEffacer(t *testing.T) {
	cas := []struct {
		nom     string
		couleur rendu.Couleur
	}{
		{"transparent", rendu.Couleur{}},
		{"couleur opaque", rendu.Couleur{R: 10, V: 20, B: 30, A: 255}},
		{"couleur translucide", demiRge},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			tampon := rendu.NouveauTampon(4, 3)
			for i := range tampon.Pixels {
				tampon.Pixels[i] = blanc
			}

			Effacer(tampon, c.couleur)

			for i, got := range tampon.Pixels {
				if got != c.couleur {
					t.Fatalf("pixel %d : %v, attendu %v", i, got, c.couleur)
				}
			}
		})
	}
}

// TestEffacerTamponVide vérifie qu'un tampon sans pixel s'efface sans déborder : le
// doublement part du premier pixel, qui n'existe pas ici.
func TestEffacerTamponVide(t *testing.T) {
	Effacer(rendu.NouveauTampon(0, 0), blanc)
}

// TestEffacerRecouvreUnDessin vérifie qu'une image ne garde rien de la précédente,
// le tampon étant réutilisé d'une image à l'autre.
func TestEffacerRecouvreUnDessin(t *testing.T) {
	a := atlasDe(t, 1, 1, rouge)
	tampon := rendu.NouveauTampon(2, 2)

	Dessiner(tampon, rendu.Rendu{quadDe(0, 0, 1, 1)}, a)
	Effacer(tampon, rendu.Couleur{})

	if n := peints(tampon); n != 0 {
		t.Errorf("pixels restants : %d, attendu aucun", n)
	}
}

// TestEffacerNAlloueRien vérifie que la première opération de chaque image reste
// hors du chemin d'allocation, comme le blit qui la suit.
func TestEffacerNAlloueRien(t *testing.T) {
	tampon := rendu.NouveauTampon(64, 64)

	if n := testing.AllocsPerRun(10, func() { Effacer(tampon, blanc) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
