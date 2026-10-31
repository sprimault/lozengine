// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La comparaison pixel à pixel avec les images qui font foi, et le seul endroit du
// dépôt qui les réécrit — sur un drapeau explicite, jamais tout seul. Un cas éprouve
// la comparaison elle-même : une suite qui passerait sur des images fausses ne
// garderait rien.

package viewer

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// regenerate réécrit les images de référence au lieu de les comparer.
//
// Jamais automatique : une image qui change sans qu'on l'ait demandé est un bug
// entériné. La cible `make references` le pose, et le commit qui en découle doit dire
// pourquoi le rendu a changé.
var regenerate = flag.Bool("regenerate", false, "réécrit les images de référence")

// referencesDir est le répertoire des images qui font foi, versionnées avec le code.
const referencesDir = "testdata/references"

// failuresDir reçoit le rendu et l'écart quand une comparaison échoue. Il est sous le
// `.tmp` du dépôt, qui n'est pas suivi : ces deux images servent à voir ce qu'on a
// cassé, pas à entrer dans l'historique.
var failuresDir = filepath.Join("..", "..", ".tmp", "references")

// TestReferences rend chaque scénario du catalogue et compare ses images à celles qui
// font foi.
//
// C'est le critère de fin du jalon 0 : le rendu se juge sans écran, au pixel, et toute
// modification du rastériseur laisse ces images inchangées ou s'accompagne d'une
// régénération justifiée.
func TestReferences(t *testing.T) {
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			scenario, ok := ByName(name)
			if !ok {
				t.Fatalf("scénario %q absent du catalogue", name)
			}

			replay, err := NewReplay(scenario)
			if err != nil {
				t.Fatalf("montage : %v", err)
			}

			for _, frame := range scenario.ReferenceFrames() {
				buffer := replay.Frame(frame, "")
				file := fmt.Sprintf("%s-%04d.png", name, frame)
				if *regenerate {
					regenerateOne(t, replay, file, frame, buffer)
					continue
				}
				compare(t, file, buffer)
			}
		})
	}
}

// regenerateOne réécrit une image de référence **seulement si ses pixels changent**.
//
// **Sans cela, une régénération réécrit tout le paquet**, et les octets changent même
// quand l'image ne change pas : l'encodage PNG ne se reproduit pas d'une chaîne Go à la
// suivante. Ajouter un scénario réécrivait donc les images de tous les autres, dont le
// contenu visuel était intact. Le dépôt a payé cette classe — un commit a versé sept
// images réécrites sans qu'un pixel bouge, et l'entrée est restée ouverte onze jours
// faute de savoir pourquoi.
//
// La parade était un geste manuel : restaurer après coup ce qui n'avait pas à changer,
// puis relancer la suite. Elle vit maintenant dans le code qui écrit, donc elle ne
// s'oublie plus — et la consigne correspondante a quitté la doctrine des tests.
//
// Deux régimes passent sans comparer : l'image absente, qu'on écrit, et celle dont les
// dimensions diffèrent, qu'on écrit aussi — comparer des suites de longueurs
// différentes n'a pas de sens, et `pixelGaps` parcourt la plus courte.
func regenerateOne(t *testing.T, replay *Replay, name string, frame int, buffer *render.Buffer) {
	t.Helper()

	write := func() {
		if err := replay.Write(referencesDir, frame, 1); err != nil {
			t.Fatalf("régénération : %v", err)
		}
	}

	file, err := os.Open(filepath.Join(referencesDir, name))
	if err != nil {
		write()
		return
	}
	reference, err := raster.LoadSheet(file)
	file.Close()
	if err != nil || reference.Width != buffer.Width || reference.Height != buffer.Height {
		write()
		return
	}
	if gaps, _ := pixelGaps(buffer.Pixels, reference.Pixels); gaps != 0 {
		write()
	}
}

// compare confronte un tampon à l'image de référence du même nom.
func compare(t *testing.T, name string, buffer *render.Buffer) {
	t.Helper()

	file, err := os.Open(filepath.Join(referencesDir, name))
	if err != nil {
		t.Fatalf("%s : %v — la créer par `make references`", name, err)
	}
	defer file.Close()

	reference, err := raster.LoadSheet(file)
	if err != nil {
		t.Fatalf("%s : %v", name, err)
	}
	if reference.Width != buffer.Width || reference.Height != buffer.Height {
		t.Fatalf("%s : rendu %d×%d, référence %d×%d",
			name, buffer.Width, buffer.Height, reference.Width, reference.Height)
	}

	gaps, first := pixelGaps(buffer.Pixels, reference.Pixels)
	if gaps == 0 {
		return
	}

	x, y := first%int(buffer.Width), first/int(buffer.Width)
	t.Errorf("%s : %d pixels diffèrent, le premier en (%d, %d) — %v au lieu de %v%s",
		name, gaps, x, y, buffer.Pixels[first], reference.Pixels[first],
		dump(t, name, buffer, reference))
}

// pixelGaps compte les pixels qui diffèrent et rend l'index du premier, ou -1
// quand les deux suites sont identiques.
//
// Sorti de compare pour être éprouvable, et pour une raison qui vaut plus que le
// rangement : c'est le contrôle dont les sept images de référence dépendent, et rien
// ne disait qu'il savait échouer. Une comparaison qui rendrait toujours zéro laisse
// la suite verte en ne gardant plus rien.
func pixelGaps(got, want []render.Color) (count, first int) {
	first = -1
	for i := range got {
		if got[i] != want[i] {
			count++
			if first < 0 {
				first = i
			}
		}
	}
	return count, first
}

// TestPixelGapsCanFail donne à la comparaison des suites volontairement
// fausses, et vérifie qu'elle les voit.
//
// Le dernier cas relie cette fonction à l'aller-retour PNG : un seul cran d'écart
// compte comme un écart, donc une référence à alpha partiel ferait rougir la suite
// sans qu'aucun défaut existe. C'est ce qui oblige les scénarios à rendre des images
// opaques.
func TestPixelGapsCanFail(t *testing.T) {
	blue := render.Color{B: 255, A: 255}
	red := render.Color{R: 255, A: 255}

	cases := []struct {
		name         string
		got, want    []render.Color
		count, first int
	}{
		{"identiques", []render.Color{blue, blue}, []render.Color{blue, blue}, 0, -1},
		{"le second faux", []render.Color{blue, red}, []render.Color{blue, blue}, 1, 1},
		{"le premier faux", []render.Color{red, blue}, []render.Color{blue, blue}, 1, 0},
		{"les deux faux", []render.Color{red, red}, []render.Color{blue, blue}, 2, 0},
		{"un seul cran", []render.Color{{R: 1, A: 2}}, []render.Color{{A: 2}}, 1, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			count, first := pixelGaps(c.got, c.want)
			if count != c.count || first != c.first {
				t.Errorf("%d écarts dès l'index %d, attendu %d dès %d",
					count, first, c.count, c.first)
			}
		})
	}
}

// dump écrit le rendu obtenu et la carte des écarts, et rend la phrase qui dit où
// ils sont — ou une chaîne vide si le dépôt lui-même échoue, l'échec de la
// comparaison se rapportant alors seul.
//
// Une mesure dit qu'une image a changé, pas en quoi : sans ces deux fichiers, il
// faudrait réécrire la référence pour voir ce qu'on a cassé, c'est-à-dire effacer la
// preuve. Un dépôt qui échoue n'aggrave donc pas l'échec, il le rend seulement moins
// lisible — et c'est tout ce qu'on peut faire de mieux ici, puisque le test a déjà
// quelque chose à rapporter.
func dump(t *testing.T, name string, buffer *render.Buffer, reference raster.Sheet) string {
	t.Helper()

	gap := render.NewBuffer(buffer.Width, buffer.Height)
	for i := range gap.Pixels {
		gap.Pixels[i] = render.Color{R: 0, G: 0, B: 0, A: 255}
		if buffer.Pixels[i] != reference.Pixels[i] {
			gap.Pixels[i] = render.Color{R: 255, G: 0, B: 0, A: 255}
		}
	}

	for suffix, image := range map[string]*render.Buffer{"rendu": buffer, "ecart": gap} {
		path := filepath.Join(failuresDir, suffix+"-"+name)
		if err := os.MkdirAll(failuresDir, 0o755); err != nil {
			return ""
		}

		file, err := os.Create(path)
		if err != nil {
			return ""
		}
		if err := raster.WritePNG(image, file); err != nil {
			file.Close()
			return ""
		}
		file.Close()
	}
	return " ; rendu et écart déposés dans " + failuresDir
}

// TestReferencesMatchTheCatalog compare les fichiers présents à ce que le catalogue
// déclare, dans les deux sens.
//
// **Ce qu'il est seul à pouvoir faire échouer est l'orphelin** : une image qu'aucun
// scénario ne réclame plus, laissée par un scénario retiré ou renommé.
// [TestReferences] n'ouvre que les fichiers qu'il attend, donc il ne la voit jamais, et
// `make references` ne la réécrit pas davantage — elle resterait au dépôt en donnant à
// croire qu'elle garde quelque chose.
//
// Le manquant est gardé des deux côtés, ici et par l'échec de lecture de
// [TestReferences]. Le redire coûte trois lignes et dit la bonne cause : un fichier
// absent est une régénération oubliée, pas un rendu qui a changé.
func TestReferencesMatchTheCatalog(t *testing.T) {
	var declared []string
	expected := map[string]bool{}

	for _, name := range Names() {
		scenario, ok := ByName(name)
		if !ok {
			t.Fatalf("scénario %q absent du catalogue", name)
		}
		for _, frame := range scenario.ReferenceFrames() {
			file := fmt.Sprintf("%s-%04d.png", name, frame)
			declared = append(declared, file)
			expected[file] = true
		}
	}

	entries, err := os.ReadDir(referencesDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", referencesDir, err)
	}

	present := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		present[entry.Name()] = true
		if !expected[entry.Name()] {
			t.Errorf("%s : aucun scénario du catalogue ne la déclare", entry.Name())
		}
	}

	for _, file := range declared {
		if !present[file] {
			t.Errorf("%s : déclarée par le catalogue et absente du dépôt", file)
		}
	}
}
