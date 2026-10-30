// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Où l'aire de monde se pose dans la surface de dessin quand un plafond la rend plus
// petite : l'énuméré, ce que chacune de ses valeurs met au bord, et le partage du
// reste d'une marge impaire.

package scene

// Alignment dit où l'aire de monde se pose dans la surface, sur un axe.
//
// Ses valeurs sont [AlignCenter], [AlignStart] et [AlignEnd].
//
// **Les axes sont ceux de la surface, pas ceux du monde** : [AlignStart] sur l'axe
// vertical désigne le haut de l'image, et [Scene.Orient] ne le fait pas tourner. La
// scène remappe les coordonnées du monde à chaque quart de tour, donc la confusion est
// à portée de main, et « début d'axe » est précisément le mot qui l'invite.
//
// C'est un réglage et non un défaut du moteur, parce que la place de l'aire est une
// propriété de la mise en page du jeu : un bandeau en haut veut l'aire en bas, un
// panneau latéral la veut d'un côté, un encadrement décoratif la veut centrée. Aucune
// de ces places n'en sert plus d'une, donc aucune n'est celle du moteur.
//
// Ce qu'il n'exprime pas, et c'est assumé : deux marges **inégales** sur le même axe,
// comme les deux barres d'un jeu de rôle classique. Une fraction les exprimerait, mais
// son zéro vaudrait l'un des bords là qu'une aire plafonnée est centrée aujourd'hui,
// et deux façons de dire la même quantité coûtent plus qu'un manque énoncé.
type Alignment uint8

// Les trois places qu'une aire de monde peut prendre dans la surface, sur un axe.
const (
	// AlignCenter partage la marge également de part et d'autre.
	//
	// Le reste d'une marge impaire va **après** l'aire, à droite ou en bas : c'est la
	// convention que l'agrandissement suit déjà pour ses bandes, et une image de
	// référence devant être reproductible au pixel, l'arbitraire se documente au lieu
	// de se découvrir. Une marge de soixante-dix-sept pixels donne donc trente-huit
	// avant et trente-neuf après.
	AlignCenter Alignment = iota

	// AlignStart pose l'aire au bord haut ou gauche, et toute la marge vient après
	// elle.
	AlignStart

	// AlignEnd pose l'aire au bord bas ou droit, et toute la marge vient avant elle.
	AlignEnd
)

// offset rend la place de l'aire dans la surface sur un axe, la marge étant ce que la
// surface a de plus que l'aire.
//
// Une marge négative ne peut pas venir du dimensionnement, qui borne l'aire par la
// surface ; si elle venait d'ailleurs, zéro est la seule place qui ne sorte pas de la
// surface.
func (a Alignment) offset(margin int32) int32 {
	if margin <= 0 {
		return 0
	}

	switch a {
	case AlignStart:
		return 0
	case AlignEnd:
		return margin
	default:
		return margin / 2
	}
}
