// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La couleur à alpha prémultiplié, et la teinte neutre sur laquelle le rastériseur
// reconnaît qu'il n'a rien à moduler.

package render

// Color est une couleur RVBA sur quatre octets, à alpha prémultiplié.
//
// Le prémultiplié ramène le mélange d'un pixel à une multiplication et une
// addition par composante. Le PNG, lui, stocke ses composantes non
// prémultipliées : la conversion appartient au chargement de la planche, jamais
// au chemin de rendu.
type Color struct{ R, G, B, A uint8 }

// White rend la teinte neutre, celle qui laisse la source intacte.
//
// Une fonction plutôt qu'une variable de paquet : la seconde se laisserait
// changer sous les pieds de tous ceux qui la lisent.
func White() Color { return Color{255, 255, 255, 255} }

// Premultiply rend la couleur prémultipliée que décrivent des composantes séparées,
// celles qu'on lit dans un éditeur d'images ou dans un fichier de réglages.
//
// Un gris à moitié transparent s'écrit 128, 128, 128 avec un alpha de 128 partout
// ailleurs ; le moteur le veut en 64, 64, 64 pour 128, parce qu'il applique l'alpha
// une fois pour toutes au lieu de le refaire à chaque pixel dessiné. Cette fonction
// est ce passage, et c'est le seul endroit où un jeu en a besoin : le chargement d'un
// PNG le fait déjà.
//
// L'arrondi est au plus proche, et il importe : tronquer décale un dégradé d'un cran
// sur toute sa longueur, et un blanc opaque y perdrait sa dernière valeur. Il coïncide
// avec celui du rastériseur, ce qu'un cas de test vérifie sur les 65 536 couples
// possibles — deux arrondis qui divergeraient donneraient deux teintes différentes
// pour une même couleur décrite.
func Premultiply(r, g, b, a uint8) Color {
	return Color{
		R: applyAlpha(r, a),
		G: applyAlpha(g, a),
		B: applyAlpha(b, a),
		A: a,
	}
}

// applyAlpha rend l'arrondi au plus proche de c × a / 255, sans division.
//
// Les deux décalages reconstituent la division par 255 à partir de celle par 256, que
// le processeur fait gratuitement.
func applyAlpha(c, a uint8) uint8 {
	t := uint32(c)*uint32(a) + 128
	return uint8((t + t>>8) >> 8)
}

// Scale met une couleur prémultipliée à l'échelle d'une fraction de 255, et c'est la
// seule façon de l'atténuer : 255 la laisse intacte, 128 la met à moitié, 0 l'efface.
//
// **Les quatre composantes ensemble, alpha compris.** Amputer les trois couleurs sans
// toucher l'alpha donnerait quelque chose qui n'est plus une couleur prémultipliée du
// tout — la cohérence que [Color.Valid] exige serait rompue —, et à l'image cela
// assombrit au lieu d'estomper : un éclat qui s'éteint deviendrait une tache noire
// avant de disparaître.
//
// Une fraction de 255 plutôt qu'un nombre à virgule, et c'est la doctrine du moteur
// autant qu'une commodité : tout ce qui touche une couleur ici est entier, donc une
// même atténuation rend les mêmes pixels d'une machine à l'autre. C'est aussi la forme
// que la valeur prend déjà chez qui s'en sert — un alpha lu dans une teinte, qu'il
// fallait auparavant diviser par 255 pour le rendre à la virgule flottante.
func (c Color) Scale(part uint8) Color {
	return Color{
		R: applyAlpha(c.R, part),
		G: applyAlpha(c.G, part),
		B: applyAlpha(c.B, part),
		A: applyAlpha(c.A, part),
	}
}

// Valid dit si la couleur est un prémultiplié cohérent, c'est-à-dire si aucune de ses
// trois composantes ne dépasse son alpha.
//
// Une composante au-dessus de l'alpha ne décrit aucune couleur : elle donne un aplat
// bien plus dense que l'opacité annoncée, et rien d'autre ne le signale — ni la
// compilation, ni le rendu, qui dessine ce qu'on lui donne.
//
// **Elle existe pour qui remplit la structure sans passer par [Premultiply]**, et
// c'est d'abord un hôte dans un autre langage : les champs sont publics parce que la
// disposition doit rester celle du C, donc du code compilé ailleurs écrira ces quatre
// octets lui-même, sans appeler aucune fonction d'ici. Un jeu en Go qui construit ses
// couleurs par [Premultiply] n'en a pas besoin.
//
// À appeler au chargement, jamais par image : le chemin de rendu ne vérifie rien et
// n'a personne à qui se plaindre.
func (c Color) Valid() bool {
	return c.R <= c.A && c.G <= c.A && c.B <= c.A
}
