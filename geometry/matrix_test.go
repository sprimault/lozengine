// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que chaque constructeur promet sur son ratio, ce que la matrice nulle fait plutôt
// que de paniquer, et ce que devient une matrice presque singulière dont l'inverse ne
// tient plus dans les entiers du moteur.

package geometry

import "testing"

// TestConstructorsSetTheirRatio vérifie que chacun place une cellule là où son
// ratio l'exige.
//
// C'est le seul cas qui distingue l'isométrie vraie du dimétrique : les deux ont la
// même forme de matrice, et seule la hauteur les sépare. Si Isometric prenait une
// hauteur en paramètre, aucun cas de ce tableau ne pourrait plus rougir.
//
// **Chaque constructeur y porte un sous-cas par axe, et pas seulement sa diagonale.**
// L'isométrie vraie n'avait que sa cellule diagonale, dont les deux coordonnées valent
// un : son abscisse est nulle quel que soit le signe des coefficients. Mesuré le
// 29 octobre 2026 — en retournant le signe de ses deux premiers coefficients, la suite
// complète des douze paquets restait verte, là où la même mutation sur le dimétrique
// faisait rougir geometry, internal/sorting et internal/viewer. Un cas symétrique ne
// garde pas un sens.
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
		{"isométrie vraie, l'axe y descend vers la droite", Isometric(32), 0, 1, 16, 9},
		{"isométrie vraie, l'axe x descend vers la gauche", Isometric(32), 1, 0, -16, 9},
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
}

// TestNarrowSpanIsNeverNegative garde que la plage de l'inverse ne devient pas négative
// sur une matrice presque singulière, dont les coefficients d'inverse sont énormes.
//
// **Une plage négative est pire qu'une plage nulle** : [Matrix.clamp] rend alors `-span`
// pour un pixel positif et `+span` pour un négatif, donc le relevé des cellules visibles
// part des quatre coins opposés à ceux qu'on lui a donnés. Rien ne déborde, rien ne
// panique, et l'écran montre l'autre bout de la carte.
//
// **Zéro est ici la valeur juste et non un pis-aller** : le calcul ne pouvait déborder
// qu'au-delà du tiers d'un int64, et passé ce seuil le produit d'un seul pixel par un
// coefficient quitte déjà la plage. Une telle matrice n'accepte donc aucun pixel non
// nul, et attendre mieux de ce régime serait attendre ce qu'aucune arithmétique ne
// donne.
//
// Le cas ne peut pas se lire sur une image : de grands coefficients d'inverse veulent de
// petits coefficients d'aller, donc une telle matrice projette tout près de l'origine.
// C'est [Matrix.Unproject] qui s'y trompe, pas [Matrix.Project].
func TestNarrowSpanIsNeverNegative(t *testing.T) {
	m := Raw(65.0/unit, 64.0/unit, 1.0/unit, 1.0/unit)

	if m.span < 0 {
		t.Errorf("plage de %d, attendue positive ou nulle", m.span)
	}
}

// TestUnrepresentableInverseDegenerates garde qu'un inverse qui ne tient pas dans un
// int64 rend la matrice dégénérée, et non des coefficients que la machine choisit.
//
// Hors plage, la conversion d'un flottant vers un entier est laissée à l'implémentation
// par la spécification du langage : amd64 rend le plus petit entier là où arm64 sature
// au plus grand. C'est ce que [quantize] porte pour l'aller, et le retour ne
// l'appliquait pas.
//
// **Le cas porte sur les coefficients et pas seulement sur ce qu'une image rend**, et il
// faut le dire : la plage retombait déjà à zéro par un second débordement — la valeur
// absolue du plus petit int64 étant elle-même négative —, donc [Matrix.Unproject]
// rendait l'origine comme on l'attend. C'était une coïncidence entre deux défauts et non
// une garantie, et c'est elle que ce cas retire.
//
// Le déterminant vaut une unité pour un cofacteur de mille et un : le quotient dépasse
// les cent vingt-huit au-delà desquelles l'échelle de l'inverse ne tient plus.
func TestUnrepresentableInverseDegenerates(t *testing.T) {
	m := Raw(1000.0/unit, 999.0/unit, 1001.0/unit, 1000.0/unit)

	if m.ia != 0 || m.ib != 0 || m.ic != 0 || m.id != 0 {
		t.Errorf("inverse (%d, %d, %d, %d), attendue nulle", m.ia, m.ib, m.ic, m.id)
	}
	if m.span != 0 {
		t.Errorf("plage de %d, attendue nulle", m.span)
	}
	if x, y := m.Unproject(100, 100); x != 0 || y != 0 {
		t.Errorf("inversé (%g, %g), attendu l'origine", x, y)
	}
}

// TestSingularMatrixDegenerates garde le repli d'une matrice dont le déterminant est
// nul, et il s'atteint depuis la surface publique par deux portes.
//
// Deux coefficients égaux sur les deux lignes décrivent une projection qui écrase le
// monde sur une droite : aucune inverse n'existe, et le moteur rend alors une matrice
// qui ramène tout point sur l'origine plutôt qu'un résultat tiré d'une division par
// zéro. `Grid(0)` y arrive aussi, un côté nul annulant les quatre coefficients.
//
// Aucun cas ne l'exerçait : les deux `Raw` de ce fichier rendent un déterminant de 1 ou
// de −1, et retirer la branche laissait la suite verte.
func TestSingularMatrixDegenerates(t *testing.T) {
	for _, c := range []struct {
		name string
		m    Matrix
	}{
		{"lignes proportionnelles", Raw(1, 1, 1, 1)},
		{"côté nul", Grid(0)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.m.ia != 0 || c.m.ib != 0 || c.m.ic != 0 || c.m.id != 0 {
				t.Errorf("inverse (%d, %d, %d, %d), attendue nulle",
					c.m.ia, c.m.ib, c.m.ic, c.m.id)
			}
			if x, y := c.m.Unproject(100, 100); x != 0 || y != 0 {
				t.Errorf("inversé (%g, %g), attendu l'origine", x, y)
			}
		})
	}
}
