// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les quatre quarts de tour sur une grille rectangulaire — un carré cacherait une
// confusion entre les deux côtés — et le recadrage qui suit une rotation.

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
)

// testRectangle monte une scène dont les deux côtés diffèrent.
//
// La grille carrée des autres cas ne conviendrait pas ici : un quart de tour échange
// les deux côtés, et une confusion entre eux passerait inaperçue sur un carré.
func testRectangle(t *testing.T) *Scene {
	t.Helper()

	s, err := New(8, 5, geometry.Dimetric(32, 16), Options{Entities: 1})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	return s
}

// TestRemapRoundTrip vérifie que les quatre quarts de tour se défont, et qu'ils
// ne produisent jamais de coordonnée négative.
//
// Les deux propriétés portent le reste. Le relevé du décor remonte de l'image vers des
// coordonnées remappées et doit retrouver la cellule que le jeu a posée ; et le champ
// de profondeur de la clé de tri est non signé, donc une coordonnée négative s'y
// replierait en une valeur énorme au lieu de se ranger devant.
func TestRemapRoundTrip(t *testing.T) {
	s := testRectangle(t)
	points := [][2]float64{{0, 0}, {3.5, 2.25}, {8, 5}, {1, 4}}

	for _, o := range []Orientation{NoRotation, QuarterTurn, HalfTurn, ThreeQuarters} {
		s.Orient(o)

		for _, p := range points {
			rx, ry := s.Remap(p[0], p[1])
			if rx < 0 || ry < 0 {
				t.Errorf("quart %d : (%g, %g) devient (%g, %g), négatif", o, p[0], p[1], rx, ry)
			}

			if x, y := s.Unremap(rx, ry); x != p[0] || y != p[1] {
				t.Errorf("quart %d : (%g, %g) → (%g, %g) → (%g, %g)", o, p[0], p[1], rx, ry, x, y)
			}
		}
	}
}

// TestExtentFollowsRotation vérifie qu'un quart de tour échange les deux côtés, ce
// dont le relevé a besoin pour savoir jusqu'où balayer.
func TestExtentFollowsRotation(t *testing.T) {
	s := testRectangle(t)

	cases := []struct {
		o             Orientation
		width, height int
	}{
		{NoRotation, 8, 5},
		{QuarterTurn, 5, 8},
		{HalfTurn, 8, 5},
		{ThreeQuarters, 5, 8},
	}
	for _, c := range cases {
		s.Orient(c.o)

		if l, h := s.Extent(); l != c.width || h != c.height {
			t.Errorf("quart %d : étendue %d×%d, attendu %d×%d", c.o, l, h, c.width, c.height)
		}
	}
}

// TestFocusCentersTheTarget vérifie que le point visé tombe au milieu du tampon, et
// que la taille rendue est bien celle qu'on a donnée — le moteur n'en suppose aucune.
func TestFocusCentersTheTarget(t *testing.T) {
	s := testRectangle(t)
	s.Focus(2, 3, 100, 60)

	ex, ey, _, _ := s.Projection().Project(2, 3, 0)
	x, y, width, height := s.View()

	if width != 100 || height != 60 {
		t.Errorf("tampon %d×%d, attendu 100×60", width, height)
	}
	if ex-x != 50 || ey-y != 30 {
		t.Errorf("le point visé tombe en (%d, %d) du tampon, attendu (50, 30)", ex-x, ey-y)
	}
}

// TestFocusOnAPixelIsExact garde la recette que la godoc de [Scene.Focus] donne pour
// cadrer sur une origine de vue en pixels, et elle doit l'atteindre **exactement**.
//
// C'est ce qui dispense un jeu d'une marge empirique, et le constat qui a ouvert ce lot
// affirmait l'inverse — un quantum de 1,6 pixel à rattraper. Il n'y en a aucun : l'écart
// du sens monde vers écran vit à l'intérieur d'un pixel, donc reprojeter ne peut pas
// changer de pixel.
//
// **La largeur impaire est le sujet, et les quatre orientations avec.** La division
// entière par deux a lieu deux fois — dans la recette et dans le recadrage — et c'est
// le seul endroit où les deux pourraient diverger d'un pixel ; les trois cas existants
// de ce fichier n'emploient que des tailles paires. La rotation est là parce que le
// remappage et son inverse doivent se compenser, ce qu'aucun d'eux ne vérifie à travers
// un cadrage.
func TestFocusOnAPixelIsExact(t *testing.T) {
	for _, size := range [][2]int32{{100, 60}, {101, 61}} {
		for _, o := range []Orientation{NoRotation, QuarterTurn, HalfTurn, ThreeQuarters} {
			s := testRectangle(t)
			s.Orient(o)
			s.Focus(2, 3, size[0], size[1])

			for _, want := range [][2]int32{{0, 0}, {-137, 89}, {1000, -1000}} {
				_, _, w, h := s.View()
				rx, ry := s.Projection().Unproject(want[0]+w/2, want[1]+h/2)
				fx, fy := s.Unremap(rx, ry)
				s.Focus(fx, fy, w, h)

				if x, y, _, _ := s.View(); x != want[0] || y != want[1] {
					t.Errorf("vue %d×%d, orientation %d : origine (%d, %d) visée, (%d, %d) atteinte",
						size[0], size[1], o, want[0], want[1], x, y)
				}
			}
		}
	}
}

// TestRotationReframes vérifie qu'un quart de tour recalcule la vue.
//
// Sans cela, la scène pivoterait sous une origine restée en place et le point suivi
// partirait hors de l'image — un défaut qu'on ne voit qu'en tournant, c'est-à-dire
// rarement.
func TestRotationReframes(t *testing.T) {
	s := testRectangle(t)
	s.Focus(2, 3, 100, 60)

	before, _, _, _ := s.View()
	s.Orient(QuarterTurn)
	after, _, _, _ := s.View()

	if before == after {
		t.Error("l'origine de la vue n'a pas bougé après un quart de tour")
	}

	rx, ry := s.Remap(2, 3)
	ex, _, _, _ := s.Projection().Project(rx, ry, 0)
	if ex-after != 50 {
		t.Errorf("après rotation, le point visé tombe en %d, attendu 50", ex-after)
	}
}
