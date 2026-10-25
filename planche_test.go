// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/rendu"
)

// poserPNG écrit un PNG opaque de deux pixels de côté au chemin donné.
//
// Le fichier se fabrique au lieu d'être versionné : une image de deux pixels n'apprend
// rien à personne dans le dépôt, et la suite de non-régression a ses propres
// références pour ce qui se juge à l'œil.
func poserPNG(t *testing.T, chemin string) {
	t.Helper()

	tampon := rendu.NouveauTampon(2, 2)
	raster.Effacer(tampon, Blanc())

	fichier, err := os.Create(chemin)
	if err != nil {
		t.Fatalf("création de %s : %v", chemin, err)
	}
	defer fichier.Close()

	if err := raster.EcrirePNG(tampon, fichier); err != nil {
		t.Fatalf("écriture du PNG : %v", err)
	}
}

// TestOuvrirPlanche vérifie ce que ChargerPlanche ne couvre pas : ouvrir le fichier,
// le décoder, et le refermer.
func TestOuvrirPlanche(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "tuile.png")
	poserPNG(t, chemin)

	planche, err := OuvrirPlanche(chemin)
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	if planche.Largeur != 2 || planche.Hauteur != 2 {
		t.Errorf("planche de %d×%d, attendu 2×2", planche.Largeur, planche.Hauteur)
	}
	if n := len(planche.Pixels); n != 4 {
		t.Errorf("%d pixels, attendu 4", n)
	}
}

// TestOuvrirPlancheRefuse vérifie que les deux échecs remontent, et ils sont de nature
// différente : l'un vient du système de fichiers, l'autre du décodage.
//
// Le second est celui qui compte ici, parce qu'il traverse une fonction de plus : sans
// lui, rien ne dirait qu'une erreur de décodage sort bien d'OuvrirPlanche au lieu
// d'être avalée avec le fichier qu'on referme.
func TestOuvrirPlancheRefuse(t *testing.T) {
	repertoire := t.TempDir()
	pasUnPNG := filepath.Join(repertoire, "texte.png")
	if err := os.WriteFile(pasUnPNG, []byte("ceci n'est pas une image"), 0o644); err != nil {
		t.Fatalf("écriture de %s : %v", pasUnPNG, err)
	}

	cas := []struct {
		nom    string
		chemin string
	}{
		{"fichier absent", filepath.Join(repertoire, "absent.png")},
		{"contenu qui n'est pas un PNG", pasUnPNG},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := OuvrirPlanche(c.chemin); err == nil {
				t.Error("ouverture acceptée")
			}
		})
	}
}
