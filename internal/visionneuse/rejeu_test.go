// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// scenarioEssai rend un scénario minuscule et rapide, pour ce qui n'a pas besoin de la
// mire : trois images, un seul quad qui se déplace d'un pixel par image.
func scenarioEssai() Scenario {
	return Scenario{
		Nom: "essai", Largeur: 4, Hauteur: 2, Images: 3,
		Atlas: func() (*raster.Atlas, error) {
			p := raster.NouvellePlanche(1, 1)
			p.Pixels[0] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

			var atlas raster.Atlas
			_, err := atlas.Ajouter(p)
			return &atlas, err
		},
		Quads: func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu {
			return append(reserve, rendu.Quad{
				X: int16(numero), Largeur: 1, Hauteur: 1, Teinte: rendu.Blanc(),
			})
		},
	}
}

// memesPixels dit si deux tampons portent exactement les mêmes pixels.
func memesPixels(a, b *raster.Tampon) bool {
	if a.Largeur != b.Largeur || a.Hauteur != b.Hauteur {
		return false
	}
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			return false
		}
	}
	return true
}

// copiePixels garde les pixels d'un tampon, que l'image suivante réécrira.
func copiePixels(t *raster.Tampon) *raster.Tampon {
	copie := raster.NouveauTampon(t.Largeur, t.Hauteur)
	copy(copie.Pixels, t.Pixels)
	return copie
}

// TestRejeuDeterministe vérifie la propriété dont tout le reste dépend : deux rejeux du
// même scénario rendent exactement les mêmes pixels. Sans elle, une comparaison à une
// image de référence ne veut rien dire.
func TestRejeuDeterministe(t *testing.T) {
	premier, err := NouveauRejeu(Mire())
	if err != nil {
		t.Fatalf("premier rejeu : %v", err)
	}
	second, err := NouveauRejeu(Mire())
	if err != nil {
		t.Fatalf("second rejeu : %v", err)
	}

	for numero := range 4 {
		a := copiePixels(premier.Image(numero, ""))
		b := second.Image(numero, "")
		if !memesPixels(a, b) {
			t.Fatalf("image %d : les deux rejeux diffèrent", numero)
		}
	}
}

// TestImageIndependanteDuNumeroPrecedent vérifie qu'une image ne dépend que de son
// numéro et de son entrée : rendue à nouveau après une autre, elle est identique. C'est
// ce qui interdit à un scénario d'accumuler un état.
func TestImageIndependanteDuNumeroPrecedent(t *testing.T) {
	rejeu, err := NouveauRejeu(Mire())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	avant := copiePixels(rejeu.Image(5, ""))
	rejeu.Image(2, "")
	apres := rejeu.Image(5, "")

	if !memesPixels(avant, apres) {
		t.Error("l'image 5 diffère selon ce qui a été rendu avant")
	}
}

// TestImageSuitLesEntrees vérifie que l'entrée simulée change bien le rendu, sans quoi
// le fichier d'entrées serait décoratif.
func TestImageSuitLesEntrees(t *testing.T) {
	rejeu, err := NouveauRejeu(Mire())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	sans := copiePixels(rejeu.Image(0, ""))
	avec := rejeu.Image(0, "pion 60 30")

	if memesPixels(sans, avec) {
		t.Error("l'entrée « pion 60 30 » n'a rien changé")
	}
}

// TestRejouerEcritDesImagesNumerotees vérifie les noms et le nombre de fichiers, puis
// qu'un PNG se relit à la résolution du scénario.
func TestRejouerEcritDesImagesNumerotees(t *testing.T) {
	rejeu, err := NouveauRejeu(scenarioEssai())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	repertoire := filepath.Join(t.TempDir(), "images")
	if err := rejeu.Rejouer(repertoire, nil, 1); err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	fichiers, err := filepath.Glob(filepath.Join(repertoire, "*.png"))
	if err != nil {
		t.Fatalf("listage : %v", err)
	}
	if len(fichiers) != 3 {
		t.Fatalf("images écrites : %v, attendu 3", fichiers)
	}

	premier, err := os.Open(filepath.Join(repertoire, "essai-0000.png"))
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	defer premier.Close()

	p, err := raster.ChargerPlanche(premier)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Largeur != 4 || p.Hauteur != 2 {
		t.Errorf("dimensions : %d×%d, attendu 4×2", p.Largeur, p.Hauteur)
	}
}

// TestEcrireAgrandit vérifie que le facteur agrandit le fichier écrit, et lui seul :
// c'est un habillage pour regarder, pas un résultat de rendu.
func TestEcrireAgrandit(t *testing.T) {
	rejeu, err := NouveauRejeu(scenarioEssai())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	repertoire := t.TempDir()
	interne := rejeu.Image(0, "")
	if err := rejeu.Ecrire(repertoire, 0, 3); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	if interne.Largeur != 4 || interne.Hauteur != 2 {
		t.Errorf("tampon interne : %d×%d, attendu 4×2", interne.Largeur, interne.Hauteur)
	}

	fichier, err := os.Open(filepath.Join(repertoire, "essai-0000.png"))
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	defer fichier.Close()

	p, err := raster.ChargerPlanche(fichier)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Largeur != 12 || p.Hauteur != 6 {
		t.Errorf("dimensions : %d×%d, attendu 12×6", p.Largeur, p.Hauteur)
	}
}

// TestNouveauRejeuRefuse vérifie que les erreurs de montage se voient au montage, et
// avec leur contexte : passé ce point, le rendu ne rend plus d'erreur.
func TestNouveauRejeuRefuse(t *testing.T) {
	cas := []struct {
		nom      string
		scenario Scenario
	}{
		{"sans atlas", Scenario{Nom: "vide", Quads: scenarioEssai().Quads}},
		{"sans quads", Scenario{Nom: "vide", Atlas: scenarioEssai().Atlas}},
		{"atlas en échec", Scenario{
			Nom:   "cassé",
			Quads: scenarioEssai().Quads,
			Atlas: func() (*raster.Atlas, error) { return nil, errors.New("planche introuvable") },
		}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := NouveauRejeu(c.scenario); err == nil {
				t.Error("scénario accepté")
			}
		})
	}
}

// TestCatalogue vérifie que chaque scénario du catalogue est montable et porte le nom
// sous lequel il est rangé — c'est ce nom qui donne celui de ses images de référence.
func TestCatalogue(t *testing.T) {
	noms := Noms()
	if len(noms) == 0 {
		t.Fatal("catalogue vide")
	}
	if !slices.IsSorted(noms) {
		t.Errorf("noms non triés : %v", noms)
	}

	for _, nom := range noms {
		scenario, ok := Par(nom)
		if !ok {
			t.Errorf("%q absent alors qu'il est listé", nom)
			continue
		}
		if scenario.Nom != nom {
			t.Errorf("%q rangé sous le nom %q", scenario.Nom, nom)
		}
		if scenario.Images == 0 || scenario.Atlas == nil || scenario.Quads == nil {
			t.Errorf("%q : scénario incomplet", nom)
		}
	}

	if _, ok := Par("inconnu"); ok {
		t.Error("scénario inconnu accepté")
	}
}
