// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La matrice et ses quatre constructeurs. C'est le seul endroit du moteur où une
// constante de ratio a le droit d'apparaître : ailleurs, une division par deux
// écrite à la main serait une seconde vérité sur la forme d'une cellule.

package geometry

import "math"

// Matrix est une projection du monde vers l'écran, avec son inverse gardée.
//
// Elle s'obtient d'un constructeur et s'emploie par [Matrix.Project] et
// [Matrix.Unproject] ; ses coefficients ne se lisent pas. C'est ce qui lui permet de
// rester interne tout en étant republiée par alias à la racine : un type qu'on
// construit par fonction et qu'on lit par méthode n'a pas besoin de montrer ses
// champs, contrairement à ceux qu'un hôte remplit lui-même.
//
// Sa valeur zéro ne projette rien d'utile — tout point y tombe sur l'origine et
// l'inverse est dégénérée. Un constructeur est obligatoire, et c'est l'échec visible
// plutôt que l'image subtilement fausse.
type Matrix struct {
	// ex = a·x + b·y, et ey = c·x + d·y − floor·z. Coefficients et coordonnées sont
	// en virgule fixe, donc le produit de deux d'entre eux porte deux fois les bits
	// de fraction et se ramène au pixel par un seul décalage.
	a, b, c, d, floor int32

	// L'inverse, calculée à la construction parce qu'on s'en sert à chaque image pour
	// savoir quelles cellules l'écran peut montrer, et que son calcul est une
	// division. En entiers comme l'aller : c'est elle qui décide quelles cellules sont
	// relevées, donc un arrondi variant d'une architecture à l'autre y changerait
	// l'image et non seulement un chiffre.
	ia, ib, ic, id int64

	// La plus grande entrée que l'inverse accepte avant que le produit par un de ses
	// coefficients quitte un int64. Elle ne peut pas être une constante : une
	// cellule minuscule donne de grands coefficients et donc une span courte.
	span int32
}

// Dimetric rend la projection d'une cellule losange au ratio libre, celle d'un
// rendu isométrique de jeu : une tuile de 32 sur 16 avance de seize pixels par
// cellule en abscisse et de huit en ordonnée.
//
// C'est le constructeur du cas courant, et le seul qui laisse la hauteur libre.
func Dimetric(width, height int) Matrix {
	halfW, halfH := float64(width)/2, float64(height)/2
	return matrix(-halfW, halfW, halfH, halfH)
}

// Isometric rend la projection de l'isométrie vraie, dont elle dérive la hauteur
// par largeur sur racine de trois — dix-huit pixels et demi pour une largeur de
// trente-deux, là où le dimétrique courant en prend seize.
//
// La hauteur n'est pas un paramètre parce qu'un constructeur qui n'impose rien n'en
// est pas un : avec elle, celui-ci serait [Dimetric] sous un autre nom, et aucun
// cas de test ne pourrait plus rougir d'une erreur de ratio. La racine carrée est à
// arrondi correct comme la division, et ne s'évalue qu'ici.
func Isometric(width int) Matrix {
	height := float64(width) / math.Sqrt(3)
	halfW, halfH := float64(width)/2, height/2
	return matrix(-halfW, halfW, halfH, halfH)
}

// Grid rend la projection de la vue de dessus, où une cellule est un carré et non
// un losange.
//
// Elle existe parce que [Dimetric] ne peut pas l'exprimer, quel que soit son
// ratio : une vue de dessus ne fait pas tourner les axes. La projection étant une
// matrice, c'est un constructeur de plus et non un second moteur.
func Grid(side int) Matrix {
	c := float64(side)
	return matrix(c, 0, 0, c)
}

// Raw rend la projection de coefficients donnés, pour ce qu'aucun des trois autres
// constructeurs n'exprime.
//
// Les quatre coefficients sont ceux d'une application linéaire : ex = a·x + b·y et
// ey = c·x + d·y. Pas de translation, parce que la scène retranche l'origine de la
// vue et qu'aucun appelant n'en a l'usage.
func Raw(a, b, c, d float64) Matrix {
	return matrix(a, b, c, d)
}

// WithFloorHeight change le facteur par lequel z monte à l'écran et rend la matrice
// modifiée, l'originale restant intacte.
//
// Ce facteur vit dans la matrice et non chez l'appelant : sinon chacun appliquerait
// lui-même la soustraction et le calcul se disséminerait. Il vaut par défaut la
// hauteur qu'une cellule occupe à l'écran, le cas d'un cube dont la face du dessus
// couvre exactement le sol de sa case.
//
// L'inverse n'est pas recalculée : elle ne porte que le plan des deux premiers axes,
// où z n'intervient pas.
func (m Matrix) WithFloorHeight(height int) Matrix {
	m.floor = quantize(float64(height))
	return m
}

// matrix quantifie des coefficients et calcule l'inverse qui va avec.
//
// Le facteur d'étage s'y déduit au lieu d'être passé : la hauteur qu'une cellule
// occupe à l'écran est celle de l'empreinte de son côté, soit la somme des valeurs
// absolues des deux coefficients d'ordonnée. La formule rend seize pour une tuile de
// 32 sur 16, le côté pour une grille, et la hauteur dérivée pour l'isométrie —
// chacun étant bien le défaut que son constructeur annonce.
func matrix(a, b, c, d float64) Matrix {
	m := Matrix{
		a: quantize(a), b: quantize(b),
		c: quantize(c), d: quantize(d),
		floor: quantize(math.Abs(c) + math.Abs(d)),
	}

	// Le déterminant se calcule sur les coefficients déjà entiers, et non sur les
	// flottants reçus : « a·d − b·c » est exactement le motif qu'une architecture
	// peut fusionner et une autre pas. En entiers, le produit est exact et la
	// question ne se pose plus.
	det := int64(m.a)*int64(m.d) - int64(m.b)*int64(m.c)
	if det == 0 {
		return m
	}

	m.ia, m.ib = inverseCoefficient(m.d, det), -inverseCoefficient(m.b, det)
	m.ic, m.id = -inverseCoefficient(m.c, det), inverseCoefficient(m.a, det)

	// Trois parts et non deux : l'inverse additionne deux produits et le biais
	// d'arrondi, et il reste ainsi de quoi les porter tous les trois.
	if pire := maxAbs(m.ia, m.ib, m.ic, m.id); pire > 0 {
		m.span = int32(min(math.MaxInt64/(3*pire), math.MaxInt32))
	}
	return m
}

// maxAbs rend la plus grande valeur absolue d'un jeu de coefficients.
func maxAbs(values ...int64) int64 {
	var worst int64
	for _, v := range values {
		if v < 0 {
			v = -v
		}
		worst = max(worst, v)
	}
	return worst
}

// inverseCoefficient met un cofacteur à l'échelle de l'inverse et le divise par le
// déterminant.
//
// L'échelle porte trois jeux de bits de fraction, et pas deux : ceux du cofacteur,
// ceux que le déterminant a reçus en multipliant deux coefficients, et ceux de
// l'inverse elle-même. Le produit d'un coefficient par un pixel rend alors
// directement du fixe, une fois décalé de inverseBits.
//
// Le numérateur reste exact en flottant — un cofacteur ne porte qu'une vingtaine de
// bits significatifs contre les cinquante-trois d'un float64 — et la division qui
// suit est à arrondi correct, donc identique sur toutes les architectures.
func inverseCoefficient(cofactor int32, det int64) int64 {
	const scale = 1 << (2*fractionBits + inverseBits)

	return int64(math.Round(float64(cofactor) * scale / float64(det)))
}
