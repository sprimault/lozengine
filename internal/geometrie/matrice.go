// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package geometrie convertit entre le repère du monde et celui de l'écran.
//
// Le monde compte en cellules, l'écran en pixels, et la conversion est une matrice
// construite une fois par un de ses quatre constructeurs. Aucune constante de ratio
// ne vit ailleurs : pas de division par deux disséminée dans le code, pas de
// disposition en losanges écrite à la main.
//
// Tout y est en virgule fixe à partir de l'entrée. Une même scène rend ainsi la
// même image au pixel près du poste au Raspberry Pi, ce que le flottant ne
// garantirait pas — la spécification du langage autorise à fusionner « x·y + z » en
// une instruction unique, que certaines architectures pratiquent et d'autres non, et
// une matrice n'est faite que de cela.
package geometrie

import "math"

// Matrice est une projection du monde vers l'écran, avec son inverse gardée.
//
// Elle s'obtient d'un constructeur et s'emploie par [Matrice.Projeter] et
// [Matrice.Inverser] ; ses coefficients ne se lisent pas. C'est ce qui lui permet de
// rester interne tout en étant republiée par alias à la racine : un type qu'on
// construit par fonction et qu'on lit par méthode n'a pas besoin de montrer ses
// champs, contrairement à ceux qu'un hôte remplit lui-même.
//
// Sa valeur zéro ne projette rien d'utile — tout point y tombe sur l'origine et
// l'inverse est dégénérée. Un constructeur est obligatoire, et c'est l'échec visible
// plutôt que l'image subtilement fausse.
type Matrice struct {
	// ex = a·x + b·y, et ey = c·x + d·y − etage·z. Coefficients et coordonnées sont
	// en virgule fixe, donc le produit de deux d'entre eux porte deux fois les bits
	// de fraction et se ramène au pixel par un seul décalage.
	a, b, c, d, etage int32

	// L'inverse, calculée à la construction parce qu'on s'en sert à chaque image pour
	// savoir quelles cellules l'écran peut montrer, et que son calcul est une
	// division. En entiers comme l'aller : c'est elle qui décide quelles cellules sont
	// relevées, donc un arrondi variant d'une architecture à l'autre y changerait
	// l'image et non seulement un chiffre.
	ia, ib, ic, id int64

	// La plus grande entrée que l'inverse accepte avant que le produit par un de ses
	// coefficients quitte un int64. Elle ne peut pas être une constante : une
	// cellule minuscule donne de grands coefficients et donc une plage courte.
	plage int32
}

// Dimetrique rend la projection d'une cellule losange au ratio libre, celle d'un
// rendu isométrique de jeu : une tuile de 32 sur 16 avance de seize pixels par
// cellule en abscisse et de huit en ordonnée.
//
// C'est le constructeur du cas courant, et le seul qui laisse la hauteur libre.
func Dimetrique(largeur, hauteur int) Matrice {
	demiL, demiH := float64(largeur)/2, float64(hauteur)/2
	return matrice(-demiL, demiL, demiH, demiH)
}

// Isometrique rend la projection de l'isométrie vraie, dont elle dérive la hauteur
// par largeur sur racine de trois — dix-huit pixels et demi pour une largeur de
// trente-deux, là où le dimétrique courant en prend seize.
//
// La hauteur n'est pas un paramètre parce qu'un constructeur qui n'impose rien n'en
// est pas un : avec elle, celui-ci serait [Dimetrique] sous un autre nom, et aucun
// cas de test ne pourrait plus rougir d'une erreur de ratio. La racine carrée est à
// arrondi correct comme la division, et ne s'évalue qu'ici.
func Isometrique(largeur int) Matrice {
	hauteur := float64(largeur) / math.Sqrt(3)
	demiL, demiH := float64(largeur)/2, hauteur/2
	return matrice(-demiL, demiL, demiH, demiH)
}

// Grille rend la projection de la vue de dessus, où une cellule est un carré et non
// un losange.
//
// Elle existe parce que [Dimetrique] ne peut pas l'exprimer, quel que soit son
// ratio : une vue de dessus ne fait pas tourner les axes. La projection étant une
// matrice, c'est un constructeur de plus et non un second moteur.
func Grille(cote int) Matrice {
	c := float64(cote)
	return matrice(c, 0, 0, c)
}

// Brute rend la projection de coefficients donnés, pour ce qu'aucun des trois autres
// constructeurs n'exprime.
//
// Les quatre coefficients sont ceux d'une application linéaire : ex = a·x + b·y et
// ey = c·x + d·y. Pas de translation, parce que la scène retranche l'origine de la
// vue et qu'aucun appelant n'en a l'usage.
func Brute(a, b, c, d float64) Matrice {
	return matrice(a, b, c, d)
}

// AvecEtage change le facteur par lequel z monte à l'écran et rend la matrice
// modifiée, l'originale restant intacte.
//
// Ce facteur vit dans la matrice et non chez l'appelant : sinon chacun appliquerait
// lui-même la soustraction et le calcul se disséminerait. Il vaut par défaut la
// hauteur qu'une cellule occupe à l'écran, le cas d'un cube dont la face du dessus
// couvre exactement le sol de sa case.
//
// L'inverse n'est pas recalculée : elle ne porte que le plan des deux premiers axes,
// où z n'intervient pas.
func (m Matrice) AvecEtage(hauteur int) Matrice {
	m.etage = quantifier(float64(hauteur))
	return m
}

// matrice quantifie des coefficients et calcule l'inverse qui va avec.
//
// Le facteur d'étage s'y déduit au lieu d'être passé : la hauteur qu'une cellule
// occupe à l'écran est celle de l'empreinte de son côté, soit la somme des valeurs
// absolues des deux coefficients d'ordonnée. La formule rend seize pour une tuile de
// 32 sur 16, le côté pour une grille, et la hauteur dérivée pour l'isométrie —
// chacun étant bien le défaut que son constructeur annonce.
func matrice(a, b, c, d float64) Matrice {
	m := Matrice{
		a: quantifier(a), b: quantifier(b),
		c: quantifier(c), d: quantifier(d),
		etage: quantifier(math.Abs(c) + math.Abs(d)),
	}

	// Le déterminant se calcule sur les coefficients déjà entiers, et non sur les
	// flottants reçus : « a·d − b·c » est exactement le motif qu'une architecture
	// peut fusionner et une autre pas. En entiers, le produit est exact et la
	// question ne se pose plus.
	det := int64(m.a)*int64(m.d) - int64(m.b)*int64(m.c)
	if det == 0 {
		return m
	}

	m.ia, m.ib = coefficientInverse(m.d, det), -coefficientInverse(m.b, det)
	m.ic, m.id = -coefficientInverse(m.c, det), coefficientInverse(m.a, det)

	// Trois parts et non deux : l'inverse additionne deux produits et le biais
	// d'arrondi, et il reste ainsi de quoi les porter tous les trois.
	if pire := maxAbs(m.ia, m.ib, m.ic, m.id); pire > 0 {
		m.plage = int32(min(math.MaxInt64/(3*pire), math.MaxInt32))
	}
	return m
}

// maxAbs rend la plus grande valeur absolue d'un jeu de coefficients.
func maxAbs(valeurs ...int64) int64 {
	var pire int64
	for _, v := range valeurs {
		if v < 0 {
			v = -v
		}
		pire = max(pire, v)
	}
	return pire
}

// coefficientInverse met un cofacteur à l'échelle de l'inverse et le divise par le
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
func coefficientInverse(cofacteur int32, det int64) int64 {
	const echelle = 1 << (2*fractionBits + inverseBits)

	return int64(math.Round(float64(cofacteur) * echelle / float64(det)))
}
