// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le pointeur : ses boutons, sa position, et la molette. Ce qu'un jeu en lit à
// l'image courante, et la conversion que la boucle y applique avant lui.

package input

// Button désigne un bouton du pointeur.
//
// Un type à part et non une place de [Key], bien que les deux se pressent et que
// `evdev` les range ensemble : les cinq cent douze places du clavier sont exactement
// l'espace des codes de balayage, et y loger des boutons demanderait de le doubler
// pour cinq valeurs. Le jeu de bits d'un pointeur tient dans un octet.
//
// Ses valeurs sont [ButtonLeft], [ButtonRight], [ButtonMiddle], [ButtonBack] et
// [ButtonForward].
type Button uint8

// Les boutons que le moteur nomme, dans l'ordre où les plateformes les rapportent.
//
// Les deux derniers sont les boutons latéraux, que les souris de bureau portent
// couramment et que les pavés tactiles n'ont pas. Un jeu qui s'en sert prévoit de
// s'en passer.
const (
	ButtonLeft Button = iota
	ButtonRight
	ButtonMiddle
	ButtonBack
	ButtonForward
)

// buttonsMax est le nombre de boutons qu'un état distingue, soit ce qu'un octet porte.
const buttonsMax = 8

// bit rend le bit où un bouton se lit, et si celui-ci est dans les bornes.
//
// Hors bornes rend faux plutôt que de paniquer, pour la raison qui vaut pour une
// touche inconnue : c'est une donnée du monde extérieur, pas une faute de
// programmation.
func (b Button) bit() (bit uint8, ok bool) {
	if b >= buttonsMax {
		return 0, false
	}
	return 1 << uint8(b), true
}

// Pointer rend la position du pointeur à cette image, en pixels de l'image, et s'il
// est dans la fenêtre.
//
// **Les pixels sont ceux de la résolution interne, pas ceux de la fenêtre.** La boucle
// défait l'agrandissement entier et les bandes avant de rendre l'entrée au jeu, de
// sorte que la position se compare directement à ce qui est dessiné. Un jeu n'a donc
// jamais à connaître le facteur, ce qui serait lui demander de refaire un calcul que
// le moteur vient de faire.
//
// Sortie de la fenêtre, la dernière position reste lisible : un jeu qui suit le
// curseur veut savoir par où il est parti. C'est le second résultat qui dit de ne plus
// s'y fier, et non des coordonnées remises à zéro, qui désigneraient un coin.
func (i Input) Pointer() (x, y int16, inside bool) {
	return i.pointerX, i.pointerY, i.inside
}

// ButtonHeld dit si le bouton est tenu à cette image.
//
// C'est le régime de ce qui dure — tirer une sélection, faire glisser —, le même que
// [Input.Held] pour une touche.
func (i Input) ButtonHeld(b Button) bool {
	bit, ok := b.bit()
	return ok && i.buttons&bit != 0
}

// ButtonJustPressed dit si le bouton vient d'être enfoncé, et une seule fois par appui.
//
// C'est le régime de ce qui se consomme — désigner une case, valider —, le même que
// [Input.JustPressed] pour une touche.
func (i Input) ButtonJustPressed(b Button) bool {
	bit, ok := b.bit()
	return ok && i.buttonEdges&bit != 0
}

// Wheel rend les crans de molette du pas, positifs vers le haut, et zéro quand elle
// n'a pas tourné.
//
// Un compte et non un état : une molette n'a pas de position à tenir, seulement des
// mouvements, et ce qui s'est passé pendant un pas appartient à ce pas.
func (i Input) Wheel() int { return int(i.wheel) }

// WithPointer rend la même entrée, la position du pointeur remplacée.
//
// C'est par là que la boucle ramène en pixels de l'image ce qu'un backend a rapporté
// en pixels de fenêtre. Elle est publique parce qu'un hôte qui tient sa propre boucle
// présente lui-même et connaît donc son propre facteur : sans elle, il n'aurait aucune
// façon de fournir une position juste à [Input.Pointer].
func (i Input) WithPointer(x, y int16, inside bool) Input {
	i.pointerX, i.pointerY, i.inside = x, y, inside
	return i
}
