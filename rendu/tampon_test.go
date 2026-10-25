// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

import "testing"

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

// TestNouveauTamponTransparent vérifie que l'image part transparente : un hôte qui
// n'écrit rien reçoit une valeur zéro, et c'est sur elle que le rastériseur efface.
func TestNouveauTamponTransparent(t *testing.T) {
	tampon := NouveauTampon(2, 2)

	if len(tampon.Pixels) != 4 {
		t.Fatalf("pixels : %d, attendu 4", len(tampon.Pixels))
	}
	for i, got := range tampon.Pixels {
		if got != (Couleur{}) {
			t.Errorf("pixel %d : %v, attendu transparent", i, got)
		}
	}
}
