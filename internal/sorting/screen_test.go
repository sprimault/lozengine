// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le placement en pixels de l'image doit tenir : ne pas bouger avec la caméra,
// sortir dans l'ordre de pose, et se ranger par sa couche comme tout le reste.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// TestScreenPieceDoesNotMoveWithTheCamera garde la raison d'être de cette porte.
//
// **C'est le seul cas à pouvoir la faire échouer**, et le défaut qu'il attrape est
// exactement celui qu'on aurait en posant l'interface en coordonnées du monde : le
// panneau suivrait la caméra d'un pixel par cran de déplacement, et une ligne de texte
// tremblerait pendant la marche du personnage.
//
// Il déplace la vue d'un nombre de cellules qui ne tombe pas juste en pixels, pour
// qu'un arrondi qui se glisserait dans la compensation se voie.
func TestScreenPieceDoesNotMoveWithTheCamera(t *testing.T) {
	s := testScene(t, 32, 1)

	a := figure()
	a.Layer = 1
	if !s.PlaceScreen(37, 11, a) {
		t.Fatal("pièce d'écran refusée")
	}

	at := func(focusX, focusY float64) (int16, int16) {
		s.Focus(focusX, focusY, 480, 270)

		image := New(s).Frame()
		if len(image) != 1 {
			t.Fatalf("%d quads, attendu 1", len(image))
		}
		return image[0].X, image[0].Y
	}

	x0, y0 := at(16, 16)
	x1, y1 := at(19.3, 22.7)

	if x0 != x1 || y0 != y1 {
		t.Errorf("la pièce a bougé de (%d, %d) à (%d, %d) quand la caméra a bougé",
			x0, y0, x1, y1)
	}
	if x0 != 37 || y0 != 11 {
		t.Errorf("pièce en (%d, %d), attendue au pixel posé (37, 11)", x0, y0)
	}
}

// TestScreenPiecesComeOutInOrderOfPlacement garde ce qu'une ligne de texte demande :
// ce qui est posé après recouvre ce qui précède.
//
// Les pièces partagent couche, planche et hauteur, donc seul le rang de pose les
// sépare. S'il ne passait pas dans la clé, elles tomberaient toutes dans le même
// groupe de départage — un tri par insertion, quadratique sur le nombre de glyphes.
func TestScreenPiecesComeOutInOrderOfPlacement(t *testing.T) {
	s := testScene(t, 32, 1)

	tints := []render.Color{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
	}
	for i, tint := range tints {
		a := figure()
		a.Layer, a.Tint = 1, tint
		if !s.PlaceScreen(int32(i*4), 0, a) {
			t.Fatalf("pièce %d refusée", i)
		}
	}

	image := New(s).Frame()
	if len(image) != len(tints) {
		t.Fatalf("%d quads, attendu %d", len(image), len(tints))
	}
	for i, want := range tints {
		if image[i].Tint != want {
			t.Errorf("rang %d : teinte %v, attendue %v", i, image[i].Tint, want)
		}
	}
}

// TestScreenPieceRanksByItsLayer vérifie qu'une pièce d'écran se range par sa couche
// comme n'importe quoi d'autre, et n'échappe pas à l'ordre commun.
//
// C'est ce qui fait que le placement en pixels n'est pas un second chemin de rendu :
// en couche basse, la pièce passe sous le monde ; en couche haute, au-dessus. Le jeu
// décide, et rien n'est câblé.
func TestScreenPieceRanksByItsLayer(t *testing.T) {
	for _, c := range []struct {
		name  string
		layer uint8
		first bool
	}{
		{"couche basse : sous le monde", 0, true},
		{"couche haute : au-dessus", 1, false},
	} {
		s := testScene(t, 32, 1)
		if _, ok := s.Place(scene.Entity{X: 16.5, Y: 16.5, Aspect: figure()}); !ok {
			t.Fatal("entité refusée")
		}

		piece := figure()
		piece.Layer, piece.Tint = c.layer, render.Color{R: 255, A: 255}
		if !s.PlaceScreen(0, 0, piece) {
			t.Fatal("pièce d'écran refusée")
		}

		image := New(s).Frame()
		if len(image) != 2 {
			t.Fatalf("%s : %d quads, attendu 2", c.name, len(image))
		}

		if got := image[0].Tint == piece.Tint; got != c.first {
			t.Errorf("%s : la pièce est peinte en premier = %v, attendu %v",
				c.name, got, c.first)
		}
	}
}
