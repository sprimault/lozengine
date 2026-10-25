// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que chaque constructeur promet sur son ratio, et ce que la matrice nulle fait
// plutôt que de paniquer.

package geometry

import "testing"

// TestConstructorsSetTheirRatio vérifie que chacun place une cellule là où son
// ratio l'exige.
//
// C'est le seul cas qui distingue l'isométrie vraie du dimétrique : les deux ont la
// même forme de matrice, et seule la hauteur les sépare. Si Isometric prenait une
// hauteur en paramètre, aucun cas de ce tableau ne pourrait plus rougir.
func TestConstructorsSetTheirRatio(t *testing.T) {
	cases := []struct {
		name   string
		m      Matrix
		x, y   float64
		ex, ey int32
	}{
		{"dimétrique, l'axe y descend vers la droite", Dimetric(32, 16), 0, 1, 16, 8},
		{"dimétrique, l'axe x descend vers la gauche", Dimetric(32, 16), 1, 0, -16, 8},
		{"dimétrique, la cellule diagonale", Dimetric(32, 16), 1, 1, 0, 16},
		{"isométrie vraie, hauteur dérivée par racine de trois", Isometric(32), 1, 1, 0, 18},
		{"grille, x vers la droite", Grid(32), 1, 0, 32, 0},
		{"grille, y vers le bas", Grid(32), 0, 1, 0, 32},
		{"brute, les coefficients donnés", Raw(2, 3, 5, 7), 1, 1, 5, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ex, ey, _, _ := c.m.Project(c.x, c.y, 0)

			if ex != c.ex || ey != c.ey {
				t.Errorf("(%g, %g) → (%d, %d), attendu (%d, %d)", c.x, c.y, ex, ey, c.ex, c.ey)
			}
		})
	}
}

// TestFloorHeight vérifie de combien z monte à l'écran, et que le défaut est la hauteur
// qu'une cellule occupe — le cas d'un cube dont le dessus couvre le sol de sa case.
func TestFloorHeight(t *testing.T) {
	cases := []struct {
		name string
		m    Matrix
		want int32
	}{
		{"défaut du dimétrique, sa hauteur de tuile", Dimetric(32, 16), -16},
		{"défaut de la grille, son côté", Grid(32), -32},
		{"défaut de l'isométrie, sa hauteur dérivée", Isometric(32), -18},
		{"ajusté", Dimetric(32, 16).WithFloorHeight(8), -8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ey, _, _ := c.m.Project(0, 0, 1)

			if ey != c.want {
				t.Errorf("z = 1 monte à %d, attendu %d", ey, c.want)
			}
		})
	}
}

// TestWithFloorHeightLeavesTheOriginal vérifie qu'ajuster rend une matrice de plus au lieu
// d'en modifier une : elle circule par valeur, et un appelant qui garde la sienne ne
// doit pas la voir changer sous ses pieds.
func TestWithFloorHeightLeavesTheOriginal(t *testing.T) {
	m := Dimetric(32, 16)
	m.WithFloorHeight(8)

	if _, ey, _, _ := m.Project(0, 0, 1); ey != -16 {
		t.Errorf("l'originale monte de %d, attendu -16", ey)
	}
}

// TestZeroMatrix vérifie que la valeur zéro échoue de façon visible : tout point
// tombe sur l'origine et l'inverse ne rend rien, au lieu de projeter une scène
// subtilement fausse qu'on chercherait ailleurs.
func TestZeroMatrix(t *testing.T) {
	var m Matrix

	if ex, ey, p, h := m.Project(3, 5, 2); ex != 0 || ey != 0 || p == 0 || h == 0 {
		t.Errorf("projeté (%d, %d), profondeur %d, hauteur %d — attendu l'origine, et les deux dernières non nulles", ex, ey, p, h)
	}
	if x, y := m.Unproject(100, 100); x != 0 || y != 0 {
		t.Errorf("inversé (%g, %g), attendu l'origine", x, y)
	}
	if n := m.MaxCells(480, 270); n != 0 {
		t.Errorf("cellules majorées à %d, attendu 0", n)
	}
}
