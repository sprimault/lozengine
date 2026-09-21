// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// regenerer réécrit les images de référence au lieu de les comparer.
//
// Jamais automatique : une image qui change sans qu'on l'ait demandé est un bug
// entériné. La cible `make references` le pose, et le commit qui en découle doit dire
// pourquoi le rendu a changé.
var regenerer = flag.Bool("regenerer", false, "réécrit les images de référence")

// referencesDir est le répertoire des images qui font foi, versionnées avec le code.
const referencesDir = "testdata/references"

// echecsDir reçoit le rendu et l'écart quand une comparaison échoue. Il est sous le
// `.tmp` du dépôt, seul endroit où la chaîne Go a le droit d'écrire sur ce poste, et
// il n'est pas suivi.
var echecsDir = filepath.Join("..", "..", ".tmp", "references")

// TestReferences rend chaque scénario du catalogue et compare ses images à celles qui
// font foi.
//
// C'est le critère de fin du jalon 0 : le rendu se juge sans écran, au pixel, et toute
// modification du rastériseur laisse ces images inchangées ou s'accompagne d'une
// régénération justifiée.
func TestReferences(t *testing.T) {
	for _, nom := range Noms() {
		t.Run(nom, func(t *testing.T) {
			scenario, ok := Par(nom)
			if !ok {
				t.Fatalf("scénario %q absent du catalogue", nom)
			}

			rejeu, err := NouveauRejeu(scenario)
			if err != nil {
				t.Fatalf("montage : %v", err)
			}

			for _, numero := range scenario.ImagesDeReference() {
				tampon := rejeu.Image(numero, "")
				if *regenerer {
					if err := rejeu.Ecrire(referencesDir, numero, 1); err != nil {
						t.Fatalf("régénération : %v", err)
					}
					continue
				}
				comparer(t, fmt.Sprintf("%s-%04d.png", nom, numero), tampon)
			}
		})
	}
}

// comparer confronte un tampon à l'image de référence du même nom.
func comparer(t *testing.T, nom string, tampon *raster.Tampon) {
	t.Helper()

	fichier, err := os.Open(filepath.Join(referencesDir, nom))
	if err != nil {
		t.Fatalf("%s : %v — la créer par `make references`", nom, err)
	}
	defer fichier.Close()

	reference, err := raster.ChargerPlanche(fichier)
	if err != nil {
		t.Fatalf("%s : %v", nom, err)
	}
	if reference.Largeur != tampon.Largeur || reference.Hauteur != tampon.Hauteur {
		t.Fatalf("%s : rendu %d×%d, référence %d×%d",
			nom, tampon.Largeur, tampon.Hauteur, reference.Largeur, reference.Hauteur)
	}

	ecarts, premier := 0, -1
	for i := range tampon.Pixels {
		if tampon.Pixels[i] != reference.Pixels[i] {
			ecarts++
			if premier < 0 {
				premier = i
			}
		}
	}
	if ecarts == 0 {
		return
	}

	x, y := premier%int(tampon.Largeur), premier/int(tampon.Largeur)
	t.Errorf("%s : %d pixels diffèrent, le premier en (%d, %d) — %v au lieu de %v%s",
		nom, ecarts, x, y, tampon.Pixels[premier], reference.Pixels[premier],
		deposer(t, nom, tampon, reference))
}

// deposer écrit le rendu obtenu et la carte des écarts, et rend la phrase qui dit où
// ils sont.
//
// Une mesure dit qu'une image a changé, pas en quoi : sans ces deux fichiers, il
// faudrait réécrire la référence pour voir ce qu'on a cassé, c'est-à-dire effacer la
// preuve.
func deposer(t *testing.T, nom string, tampon *raster.Tampon, reference raster.Planche) string {
	t.Helper()

	ecart := raster.NouveauTampon(tampon.Largeur, tampon.Hauteur)
	for i := range ecart.Pixels {
		ecart.Pixels[i] = rendu.Couleur{R: 0, V: 0, B: 0, A: 255}
		if tampon.Pixels[i] != reference.Pixels[i] {
			ecart.Pixels[i] = rendu.Couleur{R: 255, V: 0, B: 0, A: 255}
		}
	}

	for suffixe, image := range map[string]*raster.Tampon{"rendu": tampon, "ecart": ecart} {
		chemin := filepath.Join(echecsDir, suffixe+"-"+nom)
		if err := os.MkdirAll(echecsDir, 0o755); err != nil {
			return ""
		}

		fichier, err := os.Create(chemin)
		if err != nil {
			return ""
		}
		if err := image.EcrirePNG(fichier); err != nil {
			fichier.Close()
			return ""
		}
		fichier.Close()
	}
	return " ; rendu et écart déposés dans " + echecsDir
}
