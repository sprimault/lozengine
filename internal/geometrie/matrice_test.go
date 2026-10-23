// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package geometrie

import "testing"

// TestConstructeursPosentLeurRatio vérifie que chacun place une cellule là où son
// ratio l'exige.
//
// C'est le seul cas qui distingue l'isométrie vraie du dimétrique : les deux ont la
// même forme de matrice, et seule la hauteur les sépare. Si Isometrique prenait une
// hauteur en paramètre, aucun cas de ce tableau ne pourrait plus rougir.
func TestConstructeursPosentLeurRatio(t *testing.T) {
	cas := []struct {
		nom    string
		m      Matrice
		x, y   float64
		ex, ey int32
	}{
		{"dimétrique, l'axe y descend vers la droite", Dimetrique(32, 16), 0, 1, 16, 8},
		{"dimétrique, l'axe x descend vers la gauche", Dimetrique(32, 16), 1, 0, -16, 8},
		{"dimétrique, la cellule diagonale", Dimetrique(32, 16), 1, 1, 0, 16},
		{"isométrie vraie, hauteur dérivée par racine de trois", Isometrique(32), 1, 1, 0, 18},
		{"grille, x vers la droite", Grille(32), 1, 0, 32, 0},
		{"grille, y vers le bas", Grille(32), 0, 1, 0, 32},
		{"brute, les coefficients donnés", Brute(2, 3, 5, 7), 1, 1, 5, 12},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ex, ey, _ := c.m.Projeter(c.x, c.y, 0)

			if ex != c.ex || ey != c.ey {
				t.Errorf("(%g, %g) → (%d, %d), attendu (%d, %d)", c.x, c.y, ex, ey, c.ex, c.ey)
			}
		})
	}
}

// TestEtage vérifie de combien z monte à l'écran, et que le défaut est la hauteur
// qu'une cellule occupe — le cas d'un cube dont le dessus couvre le sol de sa case.
func TestEtage(t *testing.T) {
	cas := []struct {
		nom     string
		m       Matrice
		attendu int32
	}{
		{"défaut du dimétrique, sa hauteur de tuile", Dimetrique(32, 16), -16},
		{"défaut de la grille, son côté", Grille(32), -32},
		{"défaut de l'isométrie, sa hauteur dérivée", Isometrique(32), -18},
		{"ajusté", Dimetrique(32, 16).AvecEtage(8), -8},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, ey, _ := c.m.Projeter(0, 0, 1)

			if ey != c.attendu {
				t.Errorf("z = 1 monte à %d, attendu %d", ey, c.attendu)
			}
		})
	}
}

// TestAvecEtageLaisseLOriginale vérifie qu'ajuster rend une matrice de plus au lieu
// d'en modifier une : elle circule par valeur, et un appelant qui garde la sienne ne
// doit pas la voir changer sous ses pieds.
func TestAvecEtageLaisseLOriginale(t *testing.T) {
	m := Dimetrique(32, 16)
	m.AvecEtage(8)

	if _, ey, _ := m.Projeter(0, 0, 1); ey != -16 {
		t.Errorf("l'originale monte de %d, attendu -16", ey)
	}
}

// TestMatriceZero vérifie que la valeur zéro échoue de façon visible : tout point
// tombe sur l'origine et l'inverse ne rend rien, au lieu de projeter une scène
// subtilement fausse qu'on chercherait ailleurs.
func TestMatriceZero(t *testing.T) {
	var m Matrice

	if ex, ey, p := m.Projeter(3, 5, 2); ex != 0 || ey != 0 || p == 0 {
		t.Errorf("projeté (%d, %d) profondeur %d, attendu l'origine et une profondeur non nulle", ex, ey, p)
	}
	if x, y := m.Inverser(100, 100); x != 0 || y != 0 {
		t.Errorf("inversé (%g, %g), attendu l'origine", x, y)
	}
	if n := m.CellulesMax(480, 270); n != 0 {
		t.Errorf("cellules majorées à %d, attendu 0", n)
	}
}
