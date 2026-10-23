// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package geometrie

import "math"

// Projeter rend le pixel absolu où tombe un point du monde, et la profondeur sur
// laquelle la clé de tri l'ordonnera.
//
// Trois valeurs pour un seul arrondi, et c'est la raison de leur réunion : le pixel
// se compte en seize-millièmes de cellule, la profondeur en quarts, et deux
// conversions indépendantes depuis le même flottant peuvent tomber de part et d'autre
// d'un quart. La divergence réordonnerait deux quads sans que rien ne le signale.
// L'appelant qui n'a que faire de la profondeur l'ignore, ce qui ne coûte rien.
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
func (m Matrice) Projeter(x, y, z float64) (ex, ey int32, profondeur uint32) {
	fx, fy, fz := quantifier(x), quantifier(y), quantifier(z)

	ex = pixel(int64(m.a)*int64(fx) + int64(m.b)*int64(fy))
	ey = pixel(int64(m.c)*int64(fx) + int64(m.d)*int64(fy) - int64(m.etage)*int64(fz))

	return ex, ey, profondeurDe(fx, fy)
}

// Inverser rend le point du monde qui tombe sur un pixel de l'écran.
//
// Elle prend des pixels entiers et rend des cellules flottantes, ce que ses deux
// usages demandent : le culling lui donne les coins du tampon et ne garde que la
// partie entière de ce qu'elle rend, la désignation au pointeur lui donnera la
// position d'un curseur. Aucun type de coordonnée en virgule fixe n'a donc à
// traverser cette frontière.
//
// L'aller-retour admet un quantum, et il vient du pixel et non de la matrice :
// projeter arrondit au pixel entier, donc revenir au monde ne peut pas distinguer
// deux points qu'un même pixel recouvre. Mesuré sous un vingtième de cellule sur les
// trois ratios, et l'isométrie vraie — dont la hauteur ne tombe pas sur un entier —
// n'y est pas moins exacte que le dimétrique : c'est ce qui montre que le plancher
// est bien l'arrondi au pixel et non la quantification des coefficients.
func (m Matrice) Inverser(ex, ey int32) (x, y float64) {
	px, py := int64(m.borner(ex)), int64(m.borner(ey))

	const biais = 1 << (inverseBits - 1)
	fx := (m.ia*px + m.ib*py + biais) >> inverseBits
	fy := (m.ic*px + m.id*py + biais) >> inverseBits

	return float64(fx) / unite, float64(fy) / unite
}

// CellulesMax borne le nombre de cellules qu'un rectangle d'écran peut couvrir,
// quelle que soit la position de la vue.
//
// Elle existe pour dimensionner un relevé qui n'alloue jamais : le culling a besoin
// d'un majorant que la position de la caméra ne change pas, et celui-ci ne dépend
// que du rectangle et de la matrice, fixes tous les deux pour la durée d'une partie.
// C'est une signature que seul un moteur sans allocation par image réclame.
//
// Le losange qu'un rectangle découpe s'étend, sur chacun des deux axes du monde, de
// la largeur rapportée à l'empreinte d'une cellule plus la hauteur rapportée à la
// sienne. Un intervalle de cette longueur rencontre au plus sa partie entière plus
// deux bornes de cellule ; deux cellules de marge complètent le compte, une face
// posée depuis son sommet s'étendant vers le bas — celle dont le sommet vient de
// sortir de l'écran y a encore son corps.
func (m Matrice) CellulesMax(largeur, hauteur int32) int {
	empreinteL := int64(abs(m.a)) + int64(abs(m.b))
	empreinteH := int64(abs(m.c)) + int64(abs(m.d))
	if largeur <= 0 || hauteur <= 0 || empreinteL == 0 || empreinteH == 0 {
		return 0
	}

	cote := int64(largeur)*unite/empreinteL + int64(hauteur)*unite/empreinteH + 4
	return int(cote * cote)
}

// profondeurDe compte la somme des deux axes du monde en quarts de cellule.
//
// Non signée, parce que le champ que la clé de tri lui réserve l'est et que c'est la
// scène qui s'en porte garante : elle possède l'étendue de sa grille, donc elle
// exprime une rotation en remappant les coordonnées plutôt qu'en changeant un signe.
// Un négatif qui arriverait malgré tout se range devant tout le reste au lieu de
// repasser derrière, ce qui est la façon visible de se tromper.
func profondeurDe(fx, fy int32) uint32 {
	quarts := (int64(fx) + int64(fy)) >> quartBits

	if quarts < 0 {
		return 0
	}
	if quarts > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(quarts)
}

// borner ramène un pixel dans la plage où son produit par un coefficient de
// l'inverse tient dans un int64.
//
// Saturer plutôt que déborder : un pixel hors plage rend alors le point le plus
// lointain que la matrice sait exprimer, au lieu d'un point arbitraire tombé d'un
// produit replié. La plage vient de la matrice, les coefficients de l'inverse
// dépendant de son ratio.
func (m Matrice) borner(v int32) int32 {
	if v > m.plage {
		return m.plage
	}
	if v < -m.plage {
		return -m.plage
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
