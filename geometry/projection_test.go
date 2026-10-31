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

// TestRoundTripTolerance mesure l'écart du sens monde vers écran et retour, et vérifie
// qu'il suit la loi qu'annonce la godoc d'Unproject : la préimage d'un demi-pixel sur
// chaque axe, soit `(1/l + 1/h)/2` cellule pour une tuile de `l` sur `h`.
//
// Ce que le cas protège : que l'écart reste dominé par l'arrondi au pixel et non par
// l'inexactitude des coefficients de l'inverse. Réduire inverseBits le ferait rougir,
// et c'est la seule chose qui le garde.
//
// **Deux choses ont changé le 31 octobre 2026, et chacune corrige un défaut du cas.**
// Il comparait à un quantum commun de 0,05 cellule, que la godoc annonçait « sur les
// trois ratios » : c'est faux, l'écart dépendant de la tuile — 0,047 à 32×16 mais
// 0,052 à 37×13 et 0,094 à 16×8. Et il tirait ses points sur une **droite**, donc les
// deux axes y étaient corrélés et le pire cas hors d'atteinte : mesuré, cette forme
// rend 0,0484 à 37×13 là que les deux axes balayés séparément rendent 0,0519 — juste
// au-delà du seuil qu'il comparait. **C'est ce tirage qui a laissé vivre le chiffre
// faux**, et un cas qui ne peut pas atteindre le pire ne garde pas ce qu'il annonce.
func TestRoundTripTolerance(t *testing.T) {
	cases := []struct {
		name          string
		m             Matrix
		width, height float64
	}{
		{"dimétrique", Dimetric(32, 16), 32, 16},
		{"isométrie vraie", Isometric(32), 32, 32 / math.Sqrt(3)},
		{"ratio arbitraire", Dimetric(37, 13), 37, 13},
		{"petite tuile", Dimetric(16, 8), 16, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// La loi, plus un dixième pour ce que l'arrondi des coefficients ajoute.
			// Sans cette marge le cas rougirait sur du code juste ; au-delà il cesserait
			// de distinguer la loi d'un plancher deux fois plus haut.
			law := (1/c.width + 1/c.height) / 2
			limit := law * 1.1

			var worst float64
			for i := 0; i < 700; i++ {
				for j := 0; j < 700; j++ {
					ax, ay := float64(i)*0.013, float64(j)*0.017
					ex, ey, _, _ := c.m.Project(ax, ay, 0)
					rx, ry := c.m.Unproject(ex, ey)

					worst = max(worst, math.Abs(rx-ax), math.Abs(ry-ay))
				}
			}

			t.Logf("écart maximal %.6f cellule, loi %.6f", worst, law)
			if worst > limit {
				t.Errorf("écart de %.6f cellule, au-delà de la loi %.6f et de sa marge",
					worst, law)
			}
		})
	}
}

// TestRoundTripFromPixelsIsExact garde ce que la godoc d'Unproject annonce comme
// **garanti** : d'un pixel vers le monde et retour, le pixel revient à l'identique.
//
// C'est ce qui permet à un jeu de cadrer en pixels sans marge empirique, et c'est la
// seule des deux directions à être promise. Le cas rougirait si inverseBits baissait ou
// si l'arrondi de `pixel` changeait de sens.
//
// **Il balaie aussi la bordure de la plage**, qui est l'une des deux conditions de la
// garantie : au-delà de ±32 768 cellules de monde, le fixe sature et le pixel ne
// revient plus. L'autre condition — une matrice d'un constructeur — ne se garde pas
// ici : une matrice de Raw presque singulière perd l'exactitude dans la plage, ce que
// la godoc dit et que ce cas n'a pas à démentir.
func TestRoundTripFromPixelsIsExact(t *testing.T) {
	// La plage du fixe, dérivée du format plutôt qu'écrite : un int32 dont seize bits
	// sont de la fraction laisse ±32 768 à la partie entière. Le cas suit donc si le
	// format change, au lieu de garder une borne devenue fausse.
	const maxCells = 1 << (31 - fractionBits)

	cases := []struct {
		name string
		m    Matrix
	}{
		{"dimétrique", Dimetric(32, 16)},
		{"isométrie vraie", Isometric(32)},
		{"grille", Grid(16)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var seen, missed int

			for ex := int32(-40000); ex <= 40000; ex += 97 {
				for ey := int32(-40000); ey <= 40000; ey += 701 {
					x, y := c.m.Unproject(ex, ey)

					// Hors de la plage du fixe, la garantie ne porte pas.
					if x <= -maxCells || x >= maxCells || y <= -maxCells || y >= maxCells {
						continue
					}
					seen++

					if rx, ry, _, _ := c.m.Project(x, y, 0); rx != ex || ry != ey {
						missed++
						if missed == 1 {
							t.Errorf("(%d, %d) → (%g, %g) → (%d, %d)", ex, ey, x, y, rx, ry)
						}
					}
				}
			}

			t.Logf("%d pixels dans la plage, %d non exacts", seen, missed)
			if seen == 0 {
				t.Error("aucun pixel dans la plage : le balayage ne garde rien")
			}
		})
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

// TestUnprojectGivesAScreenDirection garde la promesse que la godoc fait à un jeu : la
// matrice étant sans translation, l'inverse d'un vecteur d'écran est le vecteur de
// monde correspondant.
//
// C'est ce qui permet à une touche « haut » de monter à l'écran sans deviner la
// combinaison des deux axes — et deviner est facile à rater, puisque les axes du monde
// sont les diagonales de l'image. Le cas reprojette la direction rendue et exige de
// retrouver le vecteur demandé.
func TestUnprojectGivesAScreenDirection(t *testing.T) {
	cases := []struct {
		name   string
		ex, ey int32
	}{
		{"vers le haut", 0, -32},
		{"vers le bas", 0, 32},
		{"vers la gauche", -32, 0},
		{"vers la droite", 32, 0},
	}

	for _, m := range []Matrix{Isometric(32), Dimetric(32, 16), Grid(16)} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				dx, dy := m.Unproject(c.ex, c.ey)

				ex, ey, _, _ := m.Project(dx, dy, 0)
				if ex != c.ex || ey != c.ey {
					t.Errorf("la direction (%g, %g) se reprojette en (%d, %d), attendu (%d, %d)",
						dx, dy, ex, ey, c.ex, c.ey)
				}
			})
		}
	}
}

// TestPixelCannotSaturate démontre que les deux bornes de [pixel] ne peuvent pas être
// atteintes, et c'est une propriété du décalage et non de ce que la projection produit.
//
// `2*fractionBits` vaut exactement la largeur d'un `int32` : le plus grand `int64`
// décalé de trente-deux bits rend donc `MaxInt32` tout juste, jamais au-delà, et le plus
// petit rend `MinInt32`. Les deux comparaisons de saturation sont par conséquent du code
// que rien n'exécute.
//
// **Le cas garde la coïncidence, pas la saturation.** Il rougit le jour où
// `fractionBits` change, et c'est alors qu'il faudra reprendre la godoc de [pixel] et se
// demander si les bornes redeviennent utiles. Le laisser croire qu'il éprouve une
// saturation serait pire que de ne rien écrire.
func TestPixelCannotSaturate(t *testing.T) {
	if 2*fractionBits != 32 {
		t.Fatalf("le décalage vaut %d bits : la propriété ci-dessous est à reprendre", 2*fractionBits)
	}

	// Le biais s'ajoute avant le décalage, donc l'entrée la plus grande qui ne déborde
	// pas est celle qui laisse place à ce biais.
	const bias = int64(1) << (2*fractionBits - 1)

	for _, c := range []struct {
		name  string
		fixed int64
		want  int32
	}{
		{"le plus grand entier que le biais laisse passer", math.MaxInt64 - bias, math.MaxInt32},
		{"le plus petit entier représentable", math.MinInt64, math.MinInt32},
		{"un pixel ordinaire", 3 << (2 * fractionBits), 3},
	} {
		if got := pixel(c.fixed); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.name, got, c.want)
		}
	}
}

// TestDepthStaysFarBelowItsCeiling démontre que le rabattement de [depthOf] à
// `MaxUint32` ne peut pas être atteint, et garde la borne négative, qui l'est.
//
// Les deux axes sont des `int32`, donc leur somme tient sur trente-trois bits ; le
// décalage de `quartBits` ramène le quotient sous dix-neuf bits, soit quatre ordres de
// grandeur sous le plafond. La borne basse, elle, s'atteint depuis la surface publique :
// une coordonnée négative range la chose devant tout le reste, ce qui est la façon
// visible de se tromper que la godoc annonce.
func TestDepthStaysFarBelowItsCeiling(t *testing.T) {
	if got := depthOf(math.MaxInt32, math.MaxInt32); got >= math.MaxUint32>>8 {
		t.Errorf("profondeur maximale %d : le plafond n'est plus hors de portée", got)
	}

	if got := depthOf(-1<<20, -1<<20); got != 0 {
		t.Errorf("profondeur négative : %d, attendu 0", got)
	}
}

// TestUnprojectClampsBothSigns garde les deux bornes de [Matrix.clamp], là qu'un seul
// signe était éprouvé.
//
// Retirer la branche négative laissait toute la suite verte : les autres cas donnent des
// pixels qui restent dans la plage, et le seul cas de saturation partait du positif. Un
// pixel très négatif doit rendre le point le plus lointain que la matrice sache exprimer,
// et le symétrique de ce que le positif rend.
//
// **Les signes attendus viennent du repère et non de l'intuition** : `x` croît vers le
// bas-gauche de l'écran et `y` vers le bas-droite, donc un pixel loin à droite décroît en
// `x` et croît en `y`. L'écrire à l'envers est l'erreur que ce cas a d'abord faite.
func TestUnprojectClampsBothSigns(t *testing.T) {
	m := Isometric(32)

	xHigh, yHigh := m.Unproject(math.MaxInt32, 0)
	xLow, yLow := m.Unproject(math.MinInt32, 0)

	if xHigh >= 0 || yHigh <= 0 {
		t.Errorf("pixel très positif : (%g, %g), attendu x négatif et y positif", xHigh, yHigh)
	}
	if xLow <= 0 || yLow >= 0 {
		t.Errorf("pixel très négatif : (%g, %g), attendu x positif et y négatif", xLow, yLow)
	}
	if xHigh != -xLow || yHigh != -yLow {
		t.Errorf("les deux bornes ne sont pas symétriques : (%g, %g) contre (%g, %g)",
			xHigh, yHigh, xLow, yLow)
	}
}
