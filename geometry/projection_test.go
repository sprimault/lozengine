// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'aller-retour monde vers écran, la tolérance qu'il admet — l'inverse d'une matrice
// en virgule fixe n'est pas exacte —, et les bornes que la projection sature au lieu
// de les laisser déborder.

package geometry

import (
	"math"
	"testing"
)

// TestDepthCountsInQuarters vérifie l'unité du champ que la clé de tri réserve à
// la profondeur, et qu'elle sort du même arrondi que le pixel.
func TestDepthCountsInQuarters(t *testing.T) {
	m := Dimetric(32, 16)

	cases := []struct {
		name string
		x, y float64
		want uint32
	}{
		{"l'origine", 0, 0, 0},
		{"un quart de cellule", 0.25, 0, 1},
		{"une cellule", 1, 0, 4},
		{"la somme des deux axes", 1, 1, 8},
		{"un demi-quart s'arrondit vers le bas", 0.12, 0, 0},
		{"un négatif se range devant tout", -5, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, p, _ := m.Project(c.x, c.y, 0); p != c.want {
				t.Errorf("profondeur %d, attendu %d", p, c.want)
			}
		})
	}
}

// TestHeightCountsInQuarters vérifie l'unité du champ que la clé de tri réserve à la
// hauteur, et les deux bouts où elle s'arrête.
//
// Ce que ce cas est seul à pouvoir faire échouer : une hauteur recalculée ailleurs,
// depuis le nombre d'origine. Elle y perdrait la conversion saturée, donc rendrait un
// résultat différent selon la machine, et sortirait d'un second arrondi qui peut
// tomber de l'autre côté d'un quart.
func TestHeightCountsInQuarters(t *testing.T) {
	m := Dimetric(32, 16)

	cases := []struct {
		name string
		z    float64
		want uint16
	}{
		{"le sol", 0, 0},
		{"un quart de cellule", 0.25, 1},
		{"une cellule", 1, 4},
		{"trois cellules", 3, 12},
		{"sous le sol, ramenée au sol", -2, 0},
		{"au-delà du champ, saturée", 1e6, math.MaxUint16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, _, h := m.Project(0, 0, c.z); h != c.want {
				t.Errorf("hauteur %d, attendu %d", h, c.want)
			}
		})
	}
}

// TestQuantizeSaturates vérifie ce sans quoi toute la virgule fixe serait vaine.
//
// Hors plage, la conversion d'un flottant vers un int32 est laissée à
// l'implémentation : amd64 rend le plus petit entier là où arm64 sature au plus
// grand, donc une même scène rendrait des pixels différents selon la machine. Le NaN
// est à part parce qu'aucune comparaison de borne ne l'attrape.
func TestQuantizeSaturates(t *testing.T) {
	cases := []struct {
		name string
		v    float64
		want int32
	}{
		{"au-delà de la plage", 1e12, math.MaxInt32},
		{"en deçà de la plage", -1e12, math.MinInt32},
		{"l'infini", math.Inf(1), math.MaxInt32},
		{"l'infini négatif", math.Inf(-1), math.MinInt32},
		{"le NaN, qu'aucune borne n'attrape", math.NaN(), 0},
		{"une valeur ordinaire", 3.5, 3.5 * unit},
		{"l'arrondi au plus proche", 1.0 / 3, 21845},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := quantize(c.v); got != c.want {
				t.Errorf("quantize(%g) = %d, attendu %d", c.v, got, c.want)
			}
		})
	}
}

// TestUnprojectExactOnPowersOfTwo vérifie que l'aller-retour ne perd rien
// quand le ratio est une puissance de deux : là, aucun coefficient n'est arrondi, et
// un écart signalerait une erreur de calcul et non un quantum.
func TestUnprojectExactOnPowersOfTwo(t *testing.T) {
	m := Dimetric(32, 16)

	for _, p := range [][2]float64{{0, 0}, {3, 5}, {1, 1}, {100, 250}, {-4, 7}} {
		ex, ey, _, _ := m.Project(p[0], p[1], 0)
		x, y := m.Unproject(ex, ey)

		if x != p[0] || y != p[1] {
			t.Errorf("(%g, %g) → (%d, %d) → (%g, %g)", p[0], p[1], ex, ey, x, y)
		}
	}
}

// TestRoundTripTolerance mesure l'écart de l'aller-retour sur les trois ratios et
// vérifie qu'il reste sous le quantum annoncé par la godoc d'Unproject.
//
// Ce que le cas protège vraiment : que l'écart reste dominé par l'arrondi au pixel et
// non par l'inexactitude des coefficients de l'inverse. Réduire inverseBits le ferait
// rougir, et c'est la seule chose qui le garde.
func TestRoundTripTolerance(t *testing.T) {
	const quantum = 0.05

	cases := []struct {
		name string
		m    Matrix
	}{
		{"dimétrique", Dimetric(32, 16)},
		{"isométrie vraie", Isometric(32)},
		{"ratio arbitraire", Dimetric(37, 13)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var worst float64

			for i := 0; i < 4000; i++ {
				ax, ay := float64(i)*0.37, float64(i)*0.61
				ex, ey, _, _ := c.m.Project(ax, ay, 0)
				rx, ry := c.m.Unproject(ex, ey)

				worst = max(worst, math.Abs(rx-ax), math.Abs(ry-ay))
			}

			t.Logf("écart maximal : %.6f cellule", worst)
			if worst > quantum {
				t.Errorf("écart de %.6f cellule, au-delà du quantum de %g", worst, quantum)
			}
		})
	}
}

// TestMaxCellsIsAnUpperBound vérifie que le majorant couvre vraiment ce que le relevé
// trouvera, pour toute position de la vue : un relevé qui n'alloue pas déborderait
// sinon, et c'est ce que la signature sert à empêcher.
func TestMaxCellsIsAnUpperBound(t *testing.T) {
	const width, height = 480, 270

	m := Dimetric(32, 16)
	upperBound := m.MaxCells(width, height)

	// Le relevé réel, pour quelques positions de vue : l'englobant en cellules des
	// quatre coins du tampon, comme le relevé des cellules visibles le calculera.
	for _, origin := range [][2]int32{{0, 0}, {7, 3}, {1000, -500}, {-13, 9999}} {
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

		for _, corner := range [4][2]int32{{0, 0}, {width, 0}, {0, height}, {width, height}} {
			x, y := m.Unproject(origin[0]+corner[0], origin[1]+corner[1])
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}

		cells := (int(maxX) - int(minX) + 1) * (int(maxY) - int(minY) + 1)
		if cells > upperBound {
			t.Errorf("vue en (%d, %d) : %d cellules relevées, majorant %d", origin[0], origin[1], cells, upperBound)
		}
	}
}

// TestUnprojectClampsItsInput vérifie que l'inverse sature au lieu de déborder : au
// -delà de la plage, le produit d'un pixel par un coefficient quitterait un int64 et
// rendrait un point arbitraire plutôt qu'un point lointain.
func TestUnprojectClampsItsInput(t *testing.T) {
	m := Dimetric(32, 16)

	far, _ := m.Unproject(math.MaxInt32, 0)
	bound, _ := m.Unproject(m.span, 0)

	if far != bound {
		t.Errorf("l'entrée extrême rend %g, la borne %g", far, bound)
	}
}
