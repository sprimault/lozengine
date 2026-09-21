// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

var (
	rouge   = rendu.Couleur{R: 255, A: 255}
	vert    = rendu.Couleur{V: 255, A: 255}
	bleu    = rendu.Couleur{B: 255, A: 255}
	blanc   = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	demiRge = rendu.Couleur{R: 128, A: 128} // rouge à demi transparent, prémultiplié
	rien    = rendu.Couleur{}
)

// atlasDe range une planche remplie des pixels donnés, ligne par ligne, et rend
// l'atlas dans lequel un quad la désigne par l'index 0.
func atlasDe(t *testing.T, largeur, hauteur uint16, pixels ...rendu.Couleur) *Atlas {
	t.Helper()

	p := NouvellePlanche(largeur, hauteur)
	if len(pixels) != len(p.Pixels) {
		t.Fatalf("planche de %d×%d : %d pixels donnés", largeur, hauteur, len(pixels))
	}
	copy(p.Pixels, pixels)

	var a Atlas
	if _, err := a.Ajouter(p); err != nil {
		t.Fatalf("ajout : %v", err)
	}
	return &a
}

// quadDe rend un quad à teinte neutre, celle qui laisse la source intacte.
func quadDe(x, y int16, largeur, hauteur uint16) rendu.Quad {
	return rendu.Quad{X: x, Y: y, Largeur: largeur, Hauteur: hauteur, Teinte: rendu.Blanc()}
}

// pixel rend le pixel du tampon, pour une lecture qui se lit en coordonnées.
func pixel(t *Tampon, x, y int) rendu.Couleur {
	return t.Pixels[y*int(t.Largeur)+x]
}

// peints compte les pixels non transparents, ce qui suffit à dire qu'un découpage
// n'a pas dessiné plus que sa part.
func peints(t *Tampon) int {
	n := 0
	for _, c := range t.Pixels {
		if c.A != 0 {
			n++
		}
	}
	return n
}

// TestDessinerPose vérifie qu'un quad atterrit à sa position et que ses pixels
// gardent l'ordre de la planche.
func TestDessinerPose(t *testing.T) {
	a := atlasDe(t, 2, 2, rouge, vert, bleu, blanc)
	tampon := NouveauTampon(4, 4)

	tampon.Dessiner(rendu.Rendu{quadDe(1, 1, 2, 2)}, a)

	attendu := map[[2]int]rendu.Couleur{
		{1, 1}: rouge, {2, 1}: vert,
		{1, 2}: bleu, {2, 2}: blanc,
	}
	for position, veut := range attendu {
		if got := pixel(tampon, position[0], position[1]); got != veut {
			t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, veut)
		}
	}
	if n := peints(tampon); n != 4 {
		t.Errorf("pixels peints : %d, attendu 4", n)
	}
}

// TestDessinerDecoupage éprouve le découpage sur les quatre bords : ce qui sort du
// tampon ne se dessine pas, et ce qui reste ne se décale pas.
func TestDessinerDecoupage(t *testing.T) {
	cas := []struct {
		nom     string
		x, y    int16
		attendu map[[2]int]rendu.Couleur
	}{
		{"coin haut-gauche", -1, -1, map[[2]int]rendu.Couleur{{0, 0}: blanc}},
		{"coin bas-droit", 3, 3, map[[2]int]rendu.Couleur{{3, 3}: rouge}},
		{"bord gauche", -1, 0, map[[2]int]rendu.Couleur{{0, 0}: vert, {0, 1}: blanc}},
		{"bord bas", 0, 3, map[[2]int]rendu.Couleur{{0, 3}: rouge, {1, 3}: vert}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := atlasDe(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := NouveauTampon(4, 4)

			tampon.Dessiner(rendu.Rendu{quadDe(c.x, c.y, 2, 2)}, a)

			for position, veut := range c.attendu {
				if got := pixel(tampon, position[0], position[1]); got != veut {
					t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, veut)
				}
			}
			if n := peints(tampon); n != len(c.attendu) {
				t.Errorf("pixels peints : %d, attendu %d", n, len(c.attendu))
			}
		})
	}
}

// TestDessinerHorsChamp vérifie qu'un quad entièrement dehors ne coûte et ne peint
// rien, y compris quand il touche le bord sans le franchir.
func TestDessinerHorsChamp(t *testing.T) {
	cas := []struct {
		nom  string
		x, y int16
	}{
		{"à droite", 4, 0},
		{"à gauche", -2, 0},
		{"en bas", 0, 4},
		{"en haut", 0, -2},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := atlasDe(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := NouveauTampon(4, 4)

			tampon.Dessiner(rendu.Rendu{quadDe(c.x, c.y, 2, 2)}, a)

			if n := peints(tampon); n != 0 {
				t.Errorf("pixels peints : %d, attendu aucun", n)
			}
		})
	}
}

// TestDessinerOrdre vérifie que le dernier quad recouvre le premier : l'ordre du
// tableau est la seule vérité du résultat.
func TestDessinerOrdre(t *testing.T) {
	a := atlasDe(t, 1, 1, blanc)
	tampon := NouveauTampon(2, 1)

	premier, second := quadDe(0, 0, 1, 1), quadDe(0, 0, 1, 1)
	premier.Teinte, second.Teinte = rouge, vert

	tampon.Dessiner(rendu.Rendu{premier, second}, a)

	if got := pixel(tampon, 0, 0); got != vert {
		t.Errorf("pixel recouvert : %v, attendu %v", got, vert)
	}
}

// TestDessinerSaut vérifie qu'un pixel transparent de la planche laisse le fond en
// place : c'est ce que le pré-encodage promet au blit.
func TestDessinerSaut(t *testing.T) {
	a := atlasDe(t, 2, 1, rien, rouge)
	tampon := NouveauTampon(2, 1)
	tampon.Pixels[0], tampon.Pixels[1] = bleu, bleu

	tampon.Dessiner(rendu.Rendu{quadDe(0, 0, 2, 1)}, a)

	if got := pixel(tampon, 0, 0); got != bleu {
		t.Errorf("fond sous un pixel transparent : %v, attendu %v", got, bleu)
	}
	if got := pixel(tampon, 1, 0); got != rouge {
		t.Errorf("pixel opaque : %v, attendu %v", got, rouge)
	}
}

// TestDessinerMelange vérifie la composition d'une source à demi transparente sur
// un fond opaque.
func TestDessinerMelange(t *testing.T) {
	a := atlasDe(t, 1, 1, demiRge)
	tampon := NouveauTampon(1, 1)
	tampon.Pixels[0] = blanc

	tampon.Dessiner(rendu.Rendu{quadDe(0, 0, 1, 1)}, a)

	if veut := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); pixel(tampon, 0, 0) != veut {
		t.Errorf("mélange : %v, attendu %v", pixel(tampon, 0, 0), veut)
	}
}

// TestDessinerTeinte vérifie les trois régimes de modulation, dont celui où la
// teinte rend translucide un pixel que la planche donnait opaque.
func TestDessinerTeinte(t *testing.T) {
	cas := []struct {
		nom    string
		teinte rendu.Couleur
		veut   rendu.Couleur
	}{
		{"neutre", blanc, blanc},
		{"couleur, opaque", rouge, rouge},
		{"demi-opacité", rendu.Couleur{R: 255, V: 255, B: 255, A: 128}, rendu.Couleur{R: 128, V: 128, B: 128, A: 128}},
		{"teinte nulle", rien, rien},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := atlasDe(t, 1, 1, blanc)
			tampon := NouveauTampon(1, 1)

			q := quadDe(0, 0, 1, 1)
			q.Teinte = c.teinte
			tampon.Dessiner(rendu.Rendu{q}, a)

			if got := pixel(tampon, 0, 0); got != c.veut {
				t.Errorf("pixel : %v, attendu %v", got, c.veut)
			}
		})
	}
}

// TestDessinerMiroir vérifie les deux miroirs et leur combinaison, par lesquels un
// fournisseur d'apparence tire l'est de l'ouest sans doubler son atlas.
func TestDessinerMiroir(t *testing.T) {
	cas := []struct {
		nom      string
		drapeaux rendu.Drapeaux
		attendu  [4]rendu.Couleur // rouge, vert, bleu, blanc dans l'ordre du tampon
	}{
		{"sans miroir", 0, [4]rendu.Couleur{rouge, vert, bleu, blanc}},
		{"miroir horizontal", rendu.MiroirX, [4]rendu.Couleur{vert, rouge, blanc, bleu}},
		{"miroir vertical", rendu.MiroirY, [4]rendu.Couleur{bleu, blanc, rouge, vert}},
		{"les deux", rendu.MiroirX | rendu.MiroirY, [4]rendu.Couleur{blanc, bleu, vert, rouge}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := atlasDe(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := NouveauTampon(2, 2)

			q := quadDe(0, 0, 2, 2)
			q.Drapeaux = c.drapeaux
			tampon.Dessiner(rendu.Rendu{q}, a)

			for i, veut := range c.attendu {
				if tampon.Pixels[i] != veut {
					t.Errorf("pixel %d : %v, attendu %v", i, tampon.Pixels[i], veut)
				}
			}
		})
	}
}

// TestDessinerMiroirDecoupe éprouve le croisement des deux : le bord gauche du
// tampon découpe la droite de la source quand le miroir est posé.
func TestDessinerMiroirDecoupe(t *testing.T) {
	a := atlasDe(t, 2, 1, rouge, vert)
	tampon := NouveauTampon(2, 1)

	q := quadDe(-1, 0, 2, 1)
	q.Drapeaux = rendu.MiroirX
	tampon.Dessiner(rendu.Rendu{q}, a)

	if got := pixel(tampon, 0, 0); got != rouge {
		t.Errorf("pixel visible : %v, attendu %v", got, rouge)
	}
	if n := peints(tampon); n != 1 {
		t.Errorf("pixels peints : %d, attendu 1", n)
	}
}

// TestDessinerSource vérifie qu'un quad ne dessine que son rectangle de source,
// celui qu'une découpe en grille lui a donné.
func TestDessinerSource(t *testing.T) {
	a := atlasDe(t, 2, 2, rouge, vert, bleu, blanc)
	tampon := NouveauTampon(1, 1)

	q := quadDe(0, 0, 1, 1)
	q.SourceX, q.SourceY = 1, 1
	tampon.Dessiner(rendu.Rendu{q}, a)

	if got := pixel(tampon, 0, 0); got != blanc {
		t.Errorf("pixel : %v, attendu %v", got, blanc)
	}
}

// TestDessinerAplat vérifie qu'un aplat se peint sans lire l'atlas — l'atlas est
// vide ici — et qu'il se découpe comme les autres.
func TestDessinerAplat(t *testing.T) {
	var a Atlas
	tampon := NouveauTampon(3, 2)

	q := quadDe(1, 0, 5, 1)
	q.Teinte = vert
	q.Drapeaux = rendu.Aplat
	tampon.Dessiner(rendu.Rendu{q}, &a)

	for x := 1; x < 3; x++ {
		if got := pixel(tampon, x, 0); got != vert {
			t.Errorf("pixel (%d, 0) : %v, attendu %v", x, got, vert)
		}
	}
	if n := peints(tampon); n != 2 {
		t.Errorf("pixels peints : %d, attendu 2", n)
	}
}

// TestDessinerAplatTranslucide vérifie qu'un aplat à demi transparent se mélange au
// fond, ce qu'un fondu au noir demande.
func TestDessinerAplatTranslucide(t *testing.T) {
	var a Atlas
	tampon := NouveauTampon(1, 1)
	tampon.Pixels[0] = blanc

	q := quadDe(0, 0, 1, 1)
	q.Teinte = demiRge
	q.Drapeaux = rendu.Aplat
	tampon.Dessiner(rendu.Rendu{q}, &a)

	if veut := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); pixel(tampon, 0, 0) != veut {
		t.Errorf("aplat mélangé : %v, attendu %v", pixel(tampon, 0, 0), veut)
	}
}

// TestDessinerQuadVide vérifie que la valeur zéro d'un quad ne dessine rien : c'est
// l'échec sûr sur lequel compte un hôte qui remplit la structure de zéros.
func TestDessinerQuadVide(t *testing.T) {
	a := atlasDe(t, 1, 1, rouge)
	tampon := NouveauTampon(2, 2)

	tampon.Dessiner(rendu.Rendu{{}}, a)

	if n := peints(tampon); n != 0 {
		t.Errorf("pixels peints : %d, attendu aucun", n)
	}
}

// TestDessinerNAlloueRien vérifie l'invariant du chemin de rendu : aucune
// allocation par image. Un tampon réutilisé, une liste réutilisée, et rien qui
// échappe — sans quoi le ramasse-miettes s'invite à soixante hertz.
func TestDessinerNAlloueRien(t *testing.T) {
	a := atlasDe(t, 2, 2, rouge, vert, bleu, demiRge)
	tampon := NouveauTampon(8, 8)

	q := quadDe(1, 1, 2, 2)
	miroir := quadDe(-1, 5, 2, 2)
	miroir.Drapeaux = rendu.MiroirX | rendu.MiroirY
	aplat := quadDe(4, 4, 3, 3)
	aplat.Drapeaux = rendu.Aplat
	r := rendu.Rendu{q, miroir, aplat}

	if n := testing.AllocsPerRun(10, func() { tampon.Dessiner(r, a) }); n != 0 {
		t.Errorf("allocations par image : %v, attendu aucune", n)
	}
}

// TestDessinerSourceDebordante vérifie qu'un rectangle de source plus grand que la
// planche s'arrête à la dernière ligne au lieu de lire la suivante.
func TestDessinerSourceDebordante(t *testing.T) {
	a := atlasDe(t, 2, 1, rouge, vert)
	tampon := NouveauTampon(2, 2)

	tampon.Dessiner(rendu.Rendu{quadDe(0, 0, 2, 2)}, a)

	if got := pixel(tampon, 0, 0); got != rouge {
		t.Errorf("première ligne : %v, attendu %v", got, rouge)
	}
	if n := peints(tampon); n != 2 {
		t.Errorf("pixels peints : %d, attendu 2", n)
	}
}
