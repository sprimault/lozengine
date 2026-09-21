// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// encoder rend le PNG d'une image, pour éprouver le chargement sans fichier sur
// disque.
func encoder(t *testing.T, img image.Image) *bytes.Buffer {
	t.Helper()

	var tampon bytes.Buffer
	if err := png.Encode(&tampon, img); err != nil {
		t.Fatalf("encodage du PNG : %v", err)
	}
	return &tampon
}

// TestChargerPlanchePremultiplie vérifie la conversion que le PNG ne fait pas :
// un rouge à demi transparent arrive prémultiplié, seul état dans lequel le
// mélange se ramène à une multiplication et une addition.
func TestChargerPlanchePremultiplie(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	p, err := ChargerPlanche(encoder(t, img))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}

	if veut := (rendu.Couleur{R: 128, A: 128}); p.Pixels[0] != veut {
		t.Errorf("rouge à demi transparent : %v, attendu %v", p.Pixels[0], veut)
	}
	if veut := (rendu.Couleur{R: 255, V: 255, B: 255, A: 255}); p.Pixels[1] != veut {
		t.Errorf("blanc opaque : %v, attendu %v", p.Pixels[1], veut)
	}
}

// TestChargerPlancheFormats vérifie que les formats qu'un PNG peut porter arrivent
// tous en RVBA : le chargement ne doit pas dépendre de la façon dont l'image a été
// écrite par un outil de fabrication.
func TestChargerPlancheFormats(t *testing.T) {
	gris := image.NewGray(image.Rect(0, 0, 1, 1))
	gris.SetGray(0, 0, color.Gray{Y: 200})

	palette := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.NRGBA{R: 10, G: 20, B: 30, A: 255}})
	palette.SetColorIndex(0, 0, 0)

	cas := []struct {
		nom  string
		img  image.Image
		veut rendu.Couleur
	}{
		{"niveaux de gris", gris, rendu.Couleur{R: 200, V: 200, B: 200, A: 255}},
		{"palette", palette, rendu.Couleur{R: 10, V: 20, B: 30, A: 255}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p, err := ChargerPlanche(encoder(t, c.img))
			if err != nil {
				t.Fatalf("chargement : %v", err)
			}
			if p.Pixels[0] != c.veut {
				t.Errorf("pixel : %v, attendu %v", p.Pixels[0], c.veut)
			}
		})
	}
}

// TestChargerPlancheOrigineDecalee éprouve le cas qu'une sous-image produit : des
// bornes dont l'origine n'est pas en (0, 0). Lire les pixels sans en tenir compte
// décale toute la planche d'une ligne et d'une colonne.
func TestChargerPlancheOrigineDecalee(t *testing.T) {
	entier := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	entier.SetNRGBA(1, 1, color.NRGBA{G: 255, A: 255})

	p, err := ChargerPlanche(encoder(t, entier.SubImage(image.Rect(1, 1, 3, 3))))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}

	if p.Largeur != 2 || p.Hauteur != 2 {
		t.Fatalf("dimensions : %d×%d, attendu 2×2", p.Largeur, p.Hauteur)
	}
	if veut := (rendu.Couleur{V: 255, A: 255}); p.Pixels[0] != veut {
		t.Errorf("premier pixel : %v, attendu %v", p.Pixels[0], veut)
	}
}

// TestChargerPlancheTropLarge vérifie le refus d'une planche qu'un quad ne saurait
// pas désigner, sa source ne tenant que sur seize bits.
func TestChargerPlancheTropLarge(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 65536, 1))

	if _, err := ChargerPlanche(encoder(t, img)); err == nil {
		t.Error("planche de 65 536 pixels de côté acceptée")
	}
}

// TestChargerPlancheInvalide vérifie que l'erreur de décodage est enveloppée avec
// son contexte, et non rendue nue.
func TestChargerPlancheInvalide(t *testing.T) {
	_, err := ChargerPlanche(strings.NewReader("ceci n'est pas un PNG"))
	if err == nil {
		t.Fatal("contenu invalide accepté")
	}
	if !strings.Contains(err.Error(), "PNG") {
		t.Errorf("erreur sans contexte : %v", err)
	}
}
