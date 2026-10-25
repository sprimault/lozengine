// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package geometry

import "math"

// Project rend le pixel absolu où tombe un point du monde, et les deux grandeurs sur
// lesquelles la clé de tri l'ordonnera : sa profondeur et sa hauteur.
//
// Quatre valeurs pour un seul arrondi, et c'est la raison de leur réunion : le pixel
// se compte en seize-millièmes de cellule, la profondeur et la hauteur en quarts, et
// deux conversions indépendantes depuis le même nombre peuvent tomber de part et
// d'autre d'un quart. La divergence réordonnerait deux quads sans que rien ne le
// signale. L'appelant qui n'a que faire des deux dernières les ignore, ce qui ne
// coûte rien.
//
// Les pixels sont absolus, et c'est la scène qui retranche l'origine de la vue avant
// de remplir un Quad. En absolu sur seize bits, la plus grande carte représentable
// ferait deux mille quarante-sept cellules de côté à tuile 32 — seize fois moins que
// ce que la clé distingue, et un chiffre qui dépendrait de la taille de tuile.
//
// La hauteur z ne se fond pas dans les deux autres axes : elle se retranche de
// l'ordonnée après la projection du sol, parce que la clé de tri la range dans son
// propre champ. Les fondre perdrait l'information à l'entrée du pipeline et rendrait
// le tri irréparable.
func (m Matrix) Project(x, y, z float64) (ex, ey int32, depth uint32, height uint16) {
	fx, fy, fz := quantize(x), quantize(y), quantize(z)

	ex = pixel(int64(m.a)*int64(fx) + int64(m.b)*int64(fy))
	ey = pixel(int64(m.c)*int64(fx) + int64(m.d)*int64(fy) - int64(m.floor)*int64(fz))

	return ex, ey, depthOf(fx, fy), heightOf(fz)
}

// Unproject rend le point du monde qui tombe sur un pixel de l'écran.
//
// Elle prend des pixels entiers et rend des cellules flottantes, ce que ses deux
// usages demandent. Le premier lui donne les quatre coins de l'image pour savoir
// quelles cellules celle-ci peut montrer, et ne garde que la partie entière de ce
// qu'elle rend ; le second, la désignation au pointeur, lui donnera la position d'un
// curseur. Aucun type de coordonnée en virgule fixe n'a donc à traverser cette
// frontière.
//
// L'aller-retour admet un quantum, et il vient du pixel et non de la matrice :
// projeter arrondit au pixel entier, donc revenir au monde ne peut pas distinguer
// deux points qu'un même pixel recouvre. Mesuré sous un vingtième de cellule sur les
// trois ratios, et l'isométrie vraie — dont la hauteur ne tombe pas sur un entier —
// n'y est pas moins exacte que le dimétrique : c'est ce qui montre que le plancher
// est bien l'arrondi au pixel et non la quantification des coefficients.
func (m Matrix) Unproject(ex, ey int32) (x, y float64) {
	px, py := int64(m.clamp(ex)), int64(m.clamp(ey))

	const bias = 1 << (inverseBits - 1)
	fx := (m.ia*px + m.ib*py + bias) >> inverseBits
	fy := (m.ic*px + m.id*py + bias) >> inverseBits

	return float64(fx) / unit, float64(fy) / unit
}

// MaxCells borne le nombre de cellules qu'un rectangle d'écran peut couvrir,
// quelle que soit la position de la vue.
//
// Elle existe pour dimensionner un relevé qui n'alloue jamais : celui qui retire ce
// qui sort de l'image a besoin d'un majorant que la position de la caméra ne change
// pas, et celui-ci ne dépend que du rectangle et de la matrice, fixes tous les deux
// pour la durée d'une partie. C'est une signature que seul un moteur sans allocation
// par image réclame.
//
// Le losange qu'un rectangle découpe s'étend, sur chacun des deux axes du monde, de
// la largeur rapportée à l'empreinte d'une cellule plus la hauteur rapportée à la
// sienne. Un intervalle de cette longueur rencontre au plus sa partie entière plus
// deux bornes de cellule ; deux cellules de marge complètent le compte, une face
// posée depuis son sommet s'étendant vers le bas — celle dont le sommet vient de
// sortir de l'écran y a encore son corps.
func (m Matrix) MaxCells(width, height int32) int {
	footprintW := int64(abs(m.a)) + int64(abs(m.b))
	footprintH := int64(abs(m.c)) + int64(abs(m.d))
	if width <= 0 || height <= 0 || footprintW == 0 || footprintH == 0 {
		return 0
	}

	side := int64(width)*unit/footprintW + int64(height)*unit/footprintH + 4
	return int(side * side)
}

// depthOf compte la somme des deux axes du monde en quarts de cellule.
//
// Non signée, parce que le champ que la clé de tri lui réserve l'est et que c'est la
// scène qui s'en porte garante : elle possède l'étendue de sa grille, donc elle
// exprime une rotation en remappant les coordonnées plutôt qu'en changeant un signe.
// Un négatif qui arriverait malgré tout se range devant tout le reste au lieu de
// repasser derrière, ce qui est la façon visible de se tromper.
func depthOf(fx, fy int32) uint32 {
	quarters := (int64(fx) + int64(fy)) >> quartBits

	if quarters < 0 {
		return 0
	}
	if quarters > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(quarters)
}

// heightOf compte la hauteur au-dessus du sol en quarts de cellule.
//
// Seize bits, soit 16 384 cellules de hauteur : la clé de tri réserve à la hauteur son
// propre champ, à côté de la profondeur et dans la même unité. Elle sort du même
// arrondi que le pixel, et pour la même raison que la profondeur.
//
// Si elle se calculait ailleurs, il faudrait y refaire la conversion saturée qui rend
// le résultat identique d'une machine à l'autre — celle que quantize porte, et qui
// n'a aucune raison d'exister en deux endroits.
//
// Non signée : un point sous le sol se range au niveau du sol plutôt que de repasser
// par-dessus tout le reste, ce qui est la façon visible de se tromper.
func heightOf(fz int32) uint16 {
	quarters := int64(fz) >> quartBits

	if quarters < 0 {
		return 0
	}
	if quarters > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(quarters)
}

// clamp ramène un pixel dans la plage où son produit par un coefficient de
// l'inverse tient dans un int64.
//
// Saturer plutôt que déborder : un pixel hors plage rend alors le point le plus
// lointain que la matrice sait exprimer, au lieu d'un point arbitraire tombé d'un
// produit replié. La plage vient de la matrice, les coefficients de l'inverse
// dépendant de son ratio.
func (m Matrix) clamp(v int32) int32 {
	if v > m.span {
		return m.span
	}
	if v < -m.span {
		return -m.span
	}
	return v
}

// abs rend la valeur absolue d'un coefficient.
func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
