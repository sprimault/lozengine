// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
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
	tampon := NouveauTampon(2, 2)
	tampon.Pixels[0] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	tampon.Pixels[1] = rendu.Couleur{}
	tampon.Pixels[2] = rendu.Couleur{R: 128, A: 128}
	tampon.Pixels[3] = rendu.Couleur{B: 64, A: 255}

	var fichier bytes.Buffer
	if err := tampon.EcrirePNG(&fichier); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	p, err := ChargerPlanche(&fichier)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Largeur != 2 || p.Hauteur != 2 {
		t.Fatalf("dimensions : %d×%d, attendu 2×2", p.Largeur, p.Hauteur)
	}
	for i, veut := range tampon.Pixels {
		if p.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, p.Pixels[i], veut)
		}
	}
}

// TestEcrirePNGOpaque vérifie le cas que l'encodeur traite à part — une image sans
// transparence, qu'il écrit sans canal alpha —, puisque c'est celui de presque
// toutes les images de référence.
func TestEcrirePNGOpaque(t *testing.T) {
	tampon := NouveauTampon(3, 1)
	tampon.Effacer(rendu.Couleur{R: 10, V: 20, B: 30, A: 255})

	var fichier bytes.Buffer
	if err := tampon.EcrirePNG(&fichier); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	p, err := ChargerPlanche(&fichier)
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
	err := NouveauTampon(1, 1).EcrirePNG(ecrivainEnPanne{})
	if err == nil {
		t.Fatal("écrivain en panne accepté")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("PNG")) {
		t.Errorf("erreur sans contexte : %v", err)
	}
}
