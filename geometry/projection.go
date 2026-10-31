// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le passage du monde à l'écran et son retour, plus la borne que la profondeur de la
// clé de tri impose à une carte.

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
// **L'aller-retour n'a pas le même statut dans les deux sens, et c'est ce qu'il faut
// lire avant de s'y appuyer.**
//
// D'un pixel vers le monde et retour, `Project(Unproject(p))` rend `p` **exactement**,
// et c'est **garanti** : un cadrage calculé en pixels atteint donc le pixel qu'il
// visait, sans marge à prévoir. Deux conditions, et les deux comptent. Le point du
// monde rendu doit tenir dans la plage du fixe, ±32 768 cellules ; et la matrice doit
// venir de [Dimetric], [Isometric] ou [Grid] — une matrice de [Raw] presque
// singulière perd l'exactitude **à l'intérieur** de la plage, mesuré jusqu'à 200 000
// pixels d'écart sur tous ses points, l'inverse ayant alors des coefficients énormes.
//
// Du monde vers l'écran et retour, l'écart est la préimage d'un demi-pixel sur chaque
// axe : il vaut donc `(1/l + 1/h)/2` cellule pour une tuile de `l` sur `h`, plus
// quelques millièmes que l'arrondi des coefficients de l'inverse ajoute. Ce régime est
// **décrit** et non garanti, mais sa loi suffit à ce qu'on en fait : l'écart
// **décroît** quand la tuile grandit — 0,047 cellule à 32×16, 0,023 à 64×32 — et
// croît quand elle rétrécit, 0,052 à 37×13 et 0,094 à 16×8. Aucun nombre commun ne le
// borne, et c'est pourquoi la loi s'écrit ici plutôt qu'un chiffre.
//
// **Ce que la loi interdit de faire, et qui est l'erreur à ne pas commettre** :
// convertir cet écart en pixels pour s'en prémunir. Il est la largeur de la préimage
// d'un pixel, donc il vit tout entier *à l'intérieur* d'un pixel — reprojeter un point
// qui en souffre ne peut pas changer de pixel, ce que la garantie du premier sens dit
// autrement.
//
// **Elle rend aussi une direction, et c'est son troisième usage** : la matrice étant
// sans translation, lui donner un vecteur d'écran rend le vecteur de monde
// correspondant. C'est ce qu'il faut à un jeu qui veut qu'une touche « haut » monte à
// l'écran, car les axes du monde sont les diagonales de l'image :
//
//	dx, dy := m.Unproject(0, -1)   // la direction « vers le haut de l'écran »
//	x, y = x+dx*vitesse, y+dy*vitesse
//
// Sans cela il faut deviner la combinaison, et se tromper est facile : monter demande
// de reculer sur les deux axes à la fois, aller à gauche d'avancer sur l'un en
// reculant sur l'autre. Un jeu qui préfère se déplacer selon les axes du monde — ce
// que beaucoup de jeux de stratégie veulent — n'a rien à convertir ; le choix lui
// appartient, le moteur ne le fait pas à sa place.
func (m Matrix) Unproject(ex, ey int32) (x, y float64) {
	px, py := int64(m.clamp(ex)), int64(m.clamp(ey))

	const bias = 1 << (inverseBits - 1)
	fx := (m.ia*px + m.ib*py + bias) >> inverseBits
	fy := (m.ic*px + m.id*py + bias) >> inverseBits

	return float64(fx) / unit, float64(fy) / unit
}

// Rise rend de combien de pixels une hauteur monte à l'écran.
//
// C'est la part que [Matrix.Project] retranche de l'ordonnée, isolée pour qui doit la
// défaire : désigner une case à un étage demande de redescendre le pixel visé jusqu'au
// sol avant d'appliquer l'inverse, celle-ci ne portant que le plan des deux premiers
// axes.
//
// Elle rend la montée et non le facteur, qui reste un détail de la matrice. Un
// appelant qui l'aurait voudrait le multiplier lui-même, et referait alors la
// quantification que le déterminisme veut faire à un seul endroit.
func (m Matrix) Rise(z float64) int32 {
	return pixel(int64(m.floor) * int64(quantize(z)))
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

	return saturate[uint32](quarters, 0, math.MaxUint32)
}

// heightOf compte la hauteur au-dessus du sol en quarts de cellule.
//
// Seize bits, soit 16 384 cellules de hauteur : la clé de tri réserve à la hauteur son
// propre champ, à côté de la profondeur et dans la même unité. Elle sort du même
// arrondi que le pixel, et pour la même raison que la profondeur.
//
// Si elle se calculait ailleurs, il faudrait y refaire la conversion saturée qui rend
// le résultat identique d'une machine à l'autre — celle que [saturate] porte, et qui
// n'a aucune raison d'exister en deux endroits.
//
// Non signée : un point sous le sol se range au niveau du sol plutôt que de repasser
// par-dessus tout le reste, ce qui est la façon visible de se tromper.
func heightOf(fz int32) uint16 {
	quarters := int64(fz) >> quartBits

	return saturate[uint16](quarters, 0, math.MaxUint16)
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
