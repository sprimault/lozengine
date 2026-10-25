// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

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

// TestRotationReframes vérifie qu'un quart de tour recalcule la vue.
//
// Sans cela, la scène pivoterait sous une origine restée en place et le point suivi
// partirait hors de l'image — un défaut qu'on ne voit qu'en tournant, c'est-à-dire
// rarement, et jamais dans cohue qui n'a aucune orientation.
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
