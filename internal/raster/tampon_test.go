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
			tampon := NouveauTampon(4, 3)
			for i := range tampon.Pixels {
				tampon.Pixels[i] = blanc
			}

			tampon.Effacer(c.couleur)

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
	NouveauTampon(0, 0).Effacer(blanc)
}

// TestEffacerRecouvreUnDessin vérifie qu'une image ne garde rien de la précédente,
// le tampon étant réutilisé d'une image à l'autre.
func TestEffacerRecouvreUnDessin(t *testing.T) {
	a := atlasDe(t, 1, 1, rouge)
	tampon := NouveauTampon(2, 2)

	tampon.Dessiner(rendu.Rendu{quadDe(0, 0, 1, 1)}, a)
	tampon.Effacer(rendu.Couleur{})

	if n := peints(tampon); n != 0 {
		t.Errorf("pixels restants : %d, attendu aucun", n)
	}
}

// TestEffacerNAlloueRien vérifie que la première opération de chaque image reste
// hors du chemin d'allocation, comme le blit qui la suit.
func TestEffacerNAlloueRien(t *testing.T) {
	tampon := NouveauTampon(64, 64)

	if n := testing.AllocsPerRun(10, func() { tampon.Effacer(blanc) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}

// TestNouveauTamponRefuseAuDelaDeLaBorne couvre celle des trois limites du quad que
// rien ne tenait. Les deux autres se voient — une planche trop grande ou trop
// nombreuse est refusée au chargement — alors que celle-ci n'a aucune forme visible :
// la part inatteignable du tampon reçoit les quads lointains repliés.
func TestNouveauTamponRefuseAuDelaDeLaBorne(t *testing.T) {
	cas := []struct {
		nom              string
		largeur, hauteur uint16
	}{
		{"largeur", BorneTampon + 1, 1},
		{"hauteur", 1, BorneTampon + 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("tampon de %d×%d accepté", c.largeur, c.hauteur)
				}
			}()

			NouveauTampon(c.largeur, c.hauteur)
		})
	}
}

// TestNouveauTamponAccepteLaBorne vérifie que la garde s'arrête à la borne au lieu de
// la manger. Un hors-par-un ici retirerait une colonne au plus grand tampon possible,
// et aucun autre cas ne le verrait.
func TestNouveauTamponAccepteLaBorne(t *testing.T) {
	tampon := NouveauTampon(BorneTampon, 1)

	if len(tampon.Pixels) != BorneTampon {
		t.Errorf("pixels : %d, attendu %d", len(tampon.Pixels), BorneTampon)
	}
}
