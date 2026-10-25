// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
)

// rectanglePourTest monte une scène dont les deux côtés diffèrent.
//
// La grille carrée des autres cas ne conviendrait pas ici : un quart de tour échange
// les deux côtés, et une confusion entre eux passerait inaperçue sur un carré.
func rectanglePourTest(t *testing.T) *Scene {
	t.Helper()

	s, err := Nouvelle(8, 5, geometry.Dimetric(32, 16), Options{Entites: 1})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	return s
}

// TestRemappageAllerRetour vérifie que les quatre quarts de tour se défont, et qu'ils
// ne produisent jamais de coordonnée négative.
//
// Les deux propriétés portent le reste. Le relevé du décor remonte de l'image vers des
// coordonnées remappées et doit retrouver la cellule que le jeu a posée ; et le champ
// de profondeur de la clé de tri est non signé, donc une coordonnée négative s'y
// replierait en une valeur énorme au lieu de se ranger devant.
func TestRemappageAllerRetour(t *testing.T) {
	s := rectanglePourTest(t)
	points := [][2]float64{{0, 0}, {3.5, 2.25}, {8, 5}, {1, 4}}

	for _, o := range []Orientation{SansRotation, UnQuart, DemiTour, TroisQuarts} {
		s.Orienter(o)

		for _, p := range points {
			rx, ry := s.Remapper(p[0], p[1])
			if rx < 0 || ry < 0 {
				t.Errorf("quart %d : (%g, %g) devient (%g, %g), négatif", o, p[0], p[1], rx, ry)
			}

			if x, y := s.Deremapper(rx, ry); x != p[0] || y != p[1] {
				t.Errorf("quart %d : (%g, %g) → (%g, %g) → (%g, %g)", o, p[0], p[1], rx, ry, x, y)
			}
		}
	}
}

// TestEtendueSuitLaRotation vérifie qu'un quart de tour échange les deux côtés, ce
// dont le relevé a besoin pour savoir jusqu'où balayer.
func TestEtendueSuitLaRotation(t *testing.T) {
	s := rectanglePourTest(t)

	cas := []struct {
		o                Orientation
		largeur, hauteur int
	}{
		{SansRotation, 8, 5},
		{UnQuart, 5, 8},
		{DemiTour, 8, 5},
		{TroisQuarts, 5, 8},
	}
	for _, c := range cas {
		s.Orienter(c.o)

		if l, h := s.Etendue(); l != c.largeur || h != c.hauteur {
			t.Errorf("quart %d : étendue %d×%d, attendu %d×%d", c.o, l, h, c.largeur, c.hauteur)
		}
	}
}

// TestCadrerCentreLePointVise vérifie que le point visé tombe au milieu du tampon, et
// que la taille rendue est bien celle qu'on a donnée — le moteur n'en suppose aucune.
func TestCadrerCentreLePointVise(t *testing.T) {
	s := rectanglePourTest(t)
	s.Cadrer(2, 3, 100, 60)

	ex, ey, _, _ := s.Projection().Project(2, 3, 0)
	x, y, largeur, hauteur := s.Vue()

	if largeur != 100 || hauteur != 60 {
		t.Errorf("tampon %d×%d, attendu 100×60", largeur, hauteur)
	}
	if ex-x != 50 || ey-y != 30 {
		t.Errorf("le point visé tombe en (%d, %d) du tampon, attendu (50, 30)", ex-x, ey-y)
	}
}

// TestRotationRecadre vérifie qu'un quart de tour recalcule la vue.
//
// Sans cela, la scène pivoterait sous une origine restée en place et le point suivi
// partirait hors de l'image — un défaut qu'on ne voit qu'en tournant, c'est-à-dire
// rarement, et jamais dans cohue qui n'a aucune orientation.
func TestRotationRecadre(t *testing.T) {
	s := rectanglePourTest(t)
	s.Cadrer(2, 3, 100, 60)

	avant, _, _, _ := s.Vue()
	s.Orienter(UnQuart)
	apres, _, _, _ := s.Vue()

	if avant == apres {
		t.Error("l'origine de la vue n'a pas bougé après un quart de tour")
	}

	rx, ry := s.Remapper(2, 3)
	ex, _, _, _ := s.Projection().Project(rx, ry, 0)
	if ex-apres != 50 {
		t.Errorf("après rotation, le point visé tombe en %d, attendu 50", ex-apres)
	}
}
