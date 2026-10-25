// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// ecrivainEnPanne refuse tout ce qu'on lui donne, pour éprouver le chemin d'erreur.
type ecrivainEnPanne struct{}

// Write échoue toujours.
func (ecrivainEnPanne) Write([]byte) (int, error) { return 0, errors.New("disque plein") }

// TestEcrirePNGAllerRetour vérifie qu'une image écrite puis rechargée rend les mêmes
// pixels. C'est l'aller-retour dont la suite de non-régression dépend : elle compare
// un tampon à une référence relue du disque, et un décalage d'un cran sur l'un des
// deux bouts suffirait à la rendre inutilisable.
func TestEcrirePNGAllerRetour(t *testing.T) {
	tampon := rendu.NouveauTampon(2, 2)
	tampon.Pixels[0] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	tampon.Pixels[1] = rendu.Couleur{}
	tampon.Pixels[2] = rendu.Couleur{R: 128, A: 128}
	tampon.Pixels[3] = rendu.Couleur{B: 64, A: 255}

	var fichier bytes.Buffer
	if err := WritePNG(tampon, &fichier); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	p, err := LoadSheet(&fichier)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Width != 2 || p.Height != 2 {
		t.Fatalf("dimensions : %d×%d, attendu 2×2", p.Width, p.Height)
	}
	for i, veut := range tampon.Pixels {
		if p.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, p.Pixels[i], veut)
		}
	}
}

// TestEcrirePNGAlphaPartielPerdUnCran établit ce que l'aller-retour ne rend pas, et
// c'est le pendant du test ci-dessus : les quatre pixels qu'il compare sont exacts
// par construction — deux opaques, un transparent, un dont la composante égale son
// alpha, donc qui vaut 255 une fois divisé.
//
// Partout ailleurs, le PNG stockant ses composantes non prémultipliées, l'écriture
// divise par l'alpha et le rechargement remultiplie, sans que les deux arrondis se
// compensent. Mesuré sur les 32 896 couples prémultipliés valides : 11 352 perdent
// un cran, soit un tiers, et jamais davantage.
//
// D'où la contrainte que la suite de non-régression porte : un scénario rend un
// tampon opaque. Une référence à alpha partiel ferait échouer une comparaison
// stricte sans qu'aucun défaut existe, et personne ne chercherait la cause ici.
func TestEcrirePNGAlphaPartielPerdUnCran(t *testing.T) {
	cas := []struct {
		nom           string
		pose, attendu rendu.Couleur
	}{
		{"un demi-cran sous deux", rendu.Couleur{R: 1, V: 1, B: 1, A: 2}, rendu.Couleur{A: 2}},
		{"deux sur quatre", rendu.Couleur{R: 2, V: 2, B: 2, A: 4}, rendu.Couleur{R: 1, V: 1, B: 1, A: 4}},
		{"un sur six", rendu.Couleur{R: 1, V: 1, B: 1, A: 6}, rendu.Couleur{A: 6}},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			tampon := rendu.NouveauTampon(1, 1)
			tampon.Pixels[0] = c.pose

			var fichier bytes.Buffer
			if err := WritePNG(tampon, &fichier); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			p, err := LoadSheet(&fichier)
			if err != nil {
				t.Fatalf("rechargement : %v", err)
			}
			if p.Pixels[0] != c.attendu {
				t.Errorf("%v revient %v, attendu %v", c.pose, p.Pixels[0], c.attendu)
			}
		})
	}
}

// TestEcrirePNGOpaque vérifie le cas que l'encodeur traite à part — une image sans
// transparence, qu'il écrit sans canal alpha —, puisque c'est celui de presque
// toutes les images de référence.
func TestEcrirePNGOpaque(t *testing.T) {
	tampon := rendu.NouveauTampon(3, 1)
	Clear(tampon, rendu.Couleur{R: 10, V: 20, B: 30, A: 255})

	var fichier bytes.Buffer
	if err := WritePNG(tampon, &fichier); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	p, err := LoadSheet(&fichier)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	for i, got := range p.Pixels {
		if veut := (rendu.Couleur{R: 10, V: 20, B: 30, A: 255}); got != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, got, veut)
		}
	}
}

// TestEcrirePNGEnPanne vérifie que l'erreur de l'écrivain remonte avec son contexte,
// et non nue : c'est au chargement et à l'écriture que les erreurs ont le droit
// d'exister, donc elles doivent se lire.
func TestEcrirePNGEnPanne(t *testing.T) {
	err := WritePNG(rendu.NouveauTampon(1, 1), ecrivainEnPanne{})
	if err == nil {
		t.Fatal("écrivain en panne accepté")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("PNG")) {
		t.Errorf("erreur sans contexte : %v", err)
	}
}
