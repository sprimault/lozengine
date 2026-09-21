// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// tamponDe rend un tampon rempli des pixels donnés, ligne par ligne.
func tamponDe(t *testing.T, largeur, hauteur uint16, pixels ...rendu.Couleur) *Tampon {
	t.Helper()

	tampon := NouveauTampon(largeur, hauteur)
	if len(pixels) != len(tampon.Pixels) {
		t.Fatalf("tampon de %d×%d : %d pixels donnés", largeur, hauteur, len(pixels))
	}
	copy(tampon.Pixels, pixels)
	return tampon
}

// TestFacteur vérifie le facteur retenu, y compris quand la sortie ne peut pas
// contenir la résolution interne.
func TestFacteur(t *testing.T) {
	cas := []struct {
		nom                          string
		largeur, hauteur             uint16
		largeurSortie, hauteurSortie uint16
		veut                         int
	}{
		{"tombe juste", 480, 270, 1920, 1080, 4},
		{"limité par la hauteur", 480, 270, 1920, 700, 2},
		{"limité par la largeur", 480, 270, 1000, 1080, 2},
		{"sortie trop petite", 480, 270, 100, 100, 1},
		{"sortie identique", 480, 270, 480, 270, 1},
		{"tampon vide", 0, 0, 800, 600, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			interne := NouveauTampon(c.largeur, c.hauteur)
			sortie := NouveauTampon(c.largeurSortie, c.hauteurSortie)

			if got := interne.Facteur(sortie); got != c.veut {
				t.Errorf("facteur : %d, attendu %d", got, c.veut)
			}
		})
	}
}

// TestAgrandirSansBande vérifie qu'au facteur deux chaque pixel devient un carré de
// quatre, et qu'une sortie qui tombe juste ne garde rien du fond.
func TestAgrandirSansBande(t *testing.T) {
	interne := tamponDe(t, 2, 2, rouge, vert, bleu, blanc)
	sortie := NouveauTampon(4, 4)

	interne.Agrandir(sortie, 2, demiRge)

	attendu := [16]rendu.Couleur{
		rouge, rouge, vert, vert,
		rouge, rouge, vert, vert,
		bleu, bleu, blanc, blanc,
		bleu, bleu, blanc, blanc,
	}
	for i, veut := range attendu {
		if sortie.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, sortie.Pixels[i], veut)
		}
	}
}

// TestAgrandirBandes vérifie que ce qui reste autour de l'image est peint en fond, et
// que l'image est bien centrée dedans.
func TestAgrandirBandes(t *testing.T) {
	interne := tamponDe(t, 1, 1, rouge)
	sortie := NouveauTampon(3, 3)

	interne.Agrandir(sortie, 1, bleu)

	for i, got := range sortie.Pixels {
		veut := bleu
		if i == 4 {
			veut = rouge
		}
		if got != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, got, veut)
		}
	}
}

// TestAgrandirCentrageImpair fige le partage du reste : à droite et en bas. Une image
// de référence qui basculerait d'un pixel d'une version à l'autre ne vaudrait rien.
func TestAgrandirCentrageImpair(t *testing.T) {
	interne := tamponDe(t, 1, 1, rouge)
	sortie := NouveauTampon(5, 2)

	interne.Agrandir(sortie, 2, bleu)

	// Deux colonnes de reste : une à gauche, deux à droite.
	for x := 0; x < 5; x++ {
		for y := 0; y < 2; y++ {
			veut := bleu
			if x == 1 || x == 2 {
				veut = rouge
			}
			if got := sortie.Pixels[y*5+x]; got != veut {
				t.Errorf("pixel (%d, %d) : %v, attendu %v", x, y, got, veut)
			}
		}
	}
}

// TestAgrandirSortiePlusPetite vérifie le découpage centré quand la fenêtre est plus
// petite que la résolution interne, cas d'un redimensionnement et non d'une erreur.
func TestAgrandirSortiePlusPetite(t *testing.T) {
	interne := tamponDe(t, 4, 2,
		rouge, vert, bleu, blanc,
		blanc, bleu, vert, rouge,
	)
	sortie := NouveauTampon(2, 2)

	interne.Agrandir(sortie, 1, demiRge)

	attendu := [4]rendu.Couleur{vert, bleu, bleu, vert}
	for i, veut := range attendu {
		if sortie.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, sortie.Pixels[i], veut)
		}
	}
}

// TestAgrandirBandesEtDecoupage éprouve le croisement des deux : la sortie est plus
// étroite que l'image et plus haute qu'elle, donc il faut découper sur un axe et
// peindre des bandes sur l'autre.
func TestAgrandirBandesEtDecoupage(t *testing.T) {
	interne := tamponDe(t, 4, 2,
		rouge, vert, bleu, blanc,
		blanc, bleu, vert, rouge,
	)
	sortie := NouveauTampon(2, 4)

	interne.Agrandir(sortie, 1, demiRge)

	attendu := [8]rendu.Couleur{
		demiRge, demiRge,
		vert, bleu,
		bleu, vert,
		demiRge, demiRge,
	}
	for i, veut := range attendu {
		if sortie.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, sortie.Pixels[i], veut)
		}
	}
}

// TestAgrandirRienATenir vérifie qu'une sortie dans laquelle l'image ne tient pas du
// tout n'est pas laissée à moitié peinte : tout y est fond.
func TestAgrandirRienATenir(t *testing.T) {
	sortie := NouveauTampon(2, 2)

	NouveauTampon(0, 0).Agrandir(sortie, 4, bleu)

	for i, got := range sortie.Pixels {
		if got != bleu {
			t.Errorf("pixel %d : %v, attendu %v", i, got, bleu)
		}
	}
}

// TestAgrandirMemeTaille vérifie le chemin rapide : une sortie de même taille au
// facteur un est une recopie.
func TestAgrandirMemeTaille(t *testing.T) {
	interne := tamponDe(t, 2, 1, rouge, vert)
	sortie := NouveauTampon(2, 1)

	interne.Agrandir(sortie, 1, bleu)

	if sortie.Pixels[0] != rouge || sortie.Pixels[1] != vert {
		t.Errorf("recopie : %v", sortie.Pixels)
	}
}

// TestAgrandirFacteurInvalide vérifie qu'un facteur nul ou négatif vaut un, plutôt
// que de rendre une sortie vide sans rien dire.
func TestAgrandirFacteurInvalide(t *testing.T) {
	for _, facteur := range []int{0, -3} {
		interne := tamponDe(t, 1, 1, rouge)
		sortie := NouveauTampon(1, 1)

		interne.Agrandir(sortie, facteur, bleu)

		if sortie.Pixels[0] != rouge {
			t.Errorf("facteur %d : %v, attendu %v", facteur, sortie.Pixels[0], rouge)
		}
	}
}

// TestAgrandirNAlloueRien vérifie que la dernière opération de l'image reste elle
// aussi hors du chemin d'allocation, bandes comprises.
func TestAgrandirNAlloueRien(t *testing.T) {
	interne := NouveauTampon(32, 16)
	sortie := NouveauTampon(100, 60)

	if n := testing.AllocsPerRun(10, func() { interne.Agrandir(sortie, 3, bleu) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
