// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package viewer

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// testScenario rend un scénario minuscule et rapide, pour ce qui n'a pas besoin de la
// mire : trois images, un seul quad qui se déplace d'un pixel par image.
func testScenario() Scenario {
	return Scenario{
		Name: "essai", Width: 4, Height: 2, Frames: 3,
		Atlas: func() (*raster.Atlas, error) {
			p := raster.NewSheet(1, 1)
			p.Pixels[0] = render.Color{R: 255, G: 255, B: 255, A: 255}

			var atlas raster.Atlas
			_, err := atlas.Add(p)
			return &atlas, err
		},
		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, render.Quad{
				X: int16(frame), Width: 1, Height: 1, Tint: render.White(),
			})
		},
	}
}

// samePixels dit si deux tampons portent exactement les mêmes pixels.
func samePixels(a, b *render.Buffer) bool {
	if a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			return false
		}
	}
	return true
}

// copyPixels garde les pixels d'un tampon, que l'image suivante réécrira.
func copyPixels(t *render.Buffer) *render.Buffer {
	clone := render.NewBuffer(t.Width, t.Height)
	copy(clone.Pixels, t.Pixels)
	return clone
}

// TestReplayIsDeterministic vérifie la propriété dont tout le reste dépend : deux rejeux du
// même scénario rendent exactement les mêmes pixels. Sans elle, une comparaison à une
// image de référence ne veut rien dire.
func TestReplayIsDeterministic(t *testing.T) {
	first, err := NewReplay(Target())
	if err != nil {
		t.Fatalf("premier rejeu : %v", err)
	}
	second, err := NewReplay(Target())
	if err != nil {
		t.Fatalf("second rejeu : %v", err)
	}

	// Les images que la suite de référence compare, et non les quatre premières : un
	// scénario peut très bien être déterministe au début et dériver ensuite, et c'est
	// sur celles-là que la non-régression s'appuie.
	for _, frame := range []int{0, 8, 16, 23} {
		a := copyPixels(first.Frame(frame, ""))
		b := second.Frame(frame, "")
		if !samePixels(a, b) {
			t.Fatalf("image %d : les deux rejeux diffèrent", frame)
		}
	}
}

// TestFrameIndependentOfThePrevious vérifie qu'une image ne dépend que de son
// numéro et de son entrée : rendue à nouveau après une autre, elle est identique. C'est
// ce qui interdit à un scénario d'accumuler un état.
func TestFrameIndependentOfThePrevious(t *testing.T) {
	replay, err := NewReplay(Target())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	avant := copyPixels(replay.Frame(5, ""))
	replay.Frame(2, "")
	apres := replay.Frame(5, "")

	if !samePixels(avant, apres) {
		t.Error("l'image 5 diffère selon ce qui a été rendu avant")
	}
}

// TestFrameFollowsTheInput vérifie que l'entrée simulée change bien le rendu, sans quoi
// le fichier d'entrées serait décoratif.
func TestFrameFollowsTheInput(t *testing.T) {
	replay, err := NewReplay(Target())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	sans := copyPixels(replay.Frame(0, ""))
	avec := replay.Frame(0, "pion 60 30")

	if samePixels(sans, avec) {
		t.Error("l'entrée « pion 60 30 » n'a rien changé")
	}
}

// TestRunWritesNumberedFrames vérifie les noms et le nombre de fichiers, puis
// qu'un PNG se relit à la résolution du scénario.
func TestRunWritesNumberedFrames(t *testing.T) {
	replay, err := NewReplay(testScenario())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	dir := filepath.Join(t.TempDir(), "images")
	if err := replay.Run(dir, nil, 1); err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		t.Fatalf("listage : %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("images écrites : %v, attendu 3", files)
	}

	first, err := os.Open(filepath.Join(dir, "essai-0000.png"))
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	defer first.Close()

	p, err := raster.LoadSheet(first)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Width != 4 || p.Height != 2 {
		t.Errorf("dimensions : %d×%d, attendu 4×2", p.Width, p.Height)
	}
}

// TestWriteUpscales vérifie que le facteur agrandit le fichier écrit, et lui seul :
// c'est un habillage pour regarder, pas un résultat de rendu.
func TestWriteUpscales(t *testing.T) {
	replay, err := NewReplay(testScenario())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	dir := t.TempDir()
	interne := replay.Frame(0, "")
	if err := replay.Write(dir, 0, 3); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	if interne.Width != 4 || interne.Height != 2 {
		t.Errorf("tampon interne : %d×%d, attendu 4×2", interne.Width, interne.Height)
	}

	file, err := os.Open(filepath.Join(dir, "essai-0000.png"))
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	defer file.Close()

	p, err := raster.LoadSheet(file)
	if err != nil {
		t.Fatalf("rechargement : %v", err)
	}
	if p.Width != 12 || p.Height != 6 {
		t.Errorf("dimensions : %d×%d, attendu 12×6", p.Width, p.Height)
	}
}

// TestWriteRefusesAnOverflowingFactor vérifie que l'agrandissement refuse au lieu de
// replier. La taille de sortie passait par un uint16 avant tout contrôle : sur ce
// scénario de quatre pixels de large, le facteur 16 385 en demandait 65 540 et écrivait
// une image de quatre, sans erreur et sans que rien ne distingue le fichier obtenu de
// celui qu'on voulait.
func TestWriteRefusesAnOverflowingFactor(t *testing.T) {
	replay, err := NewReplay(testScenario())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	replay.Frame(0, "")

	if err := replay.Write(t.TempDir(), 0, 16385); err == nil {
		t.Error("facteur accepté")
	}
}

// TestNewReplayRefuses vérifie que les erreurs de montage se voient au montage, et
// avec leur contexte : passé ce point, le rendu ne rend plus d'erreur.
func TestNewReplayRefuses(t *testing.T) {
	cases := []struct {
		name     string
		scenario Scenario
	}{
		{"sans atlas", Scenario{Name: "vide", Quads: testScenario().Quads}},
		{"sans quads", Scenario{Name: "vide", Atlas: testScenario().Atlas}},
		{"atlas en échec", Scenario{
			Name:  "cassé",
			Quads: testScenario().Quads,
			Atlas: func() (*raster.Atlas, error) { return nil, errors.New("planche introuvable") },
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewReplay(c.scenario); err == nil {
				t.Error("scénario accepté")
			}
		})
	}
}

// TestCatalog vérifie que chaque scénario du catalogue est montable et porte le nom
// sous lequel il est rangé — c'est ce nom qui donne celui de ses images de référence.
func TestCatalog(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("catalogue vide")
	}
	if !slices.IsSorted(names) {
		t.Errorf("noms non triés : %v", names)
	}

	for _, name := range names {
		scenario, ok := ByName(name)
		if !ok {
			t.Errorf("%q absent alors qu'il est listé", name)
			continue
		}
		if scenario.Name != name {
			t.Errorf("%q rangé sous le nom %q", scenario.Name, name)
		}
		if scenario.Frames == 0 || scenario.Atlas == nil || scenario.Quads == nil {
			t.Errorf("%q : scénario incomplet", name)
		}
	}

	if _, ok := ByName("inconnu"); ok {
		t.Error("scénario inconnu accepté")
	}
}
