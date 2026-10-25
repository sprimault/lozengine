// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'un jeu lit du clavier à l'image courante, et rien de plus : il n'y a aucune
// façon d'y écrire depuis l'extérieur du moteur.

package input

// Input est ce que le clavier dit à une image : les touches tenues, et celles qui
// viennent d'être enfoncées.
//
// Un état et non une file d'événements. Une file ferait dépendre le travail d'un pas du
// nombre d'événements arrivés depuis le précédent, ce qui défait le pas fixe ; et elle
// obligerait un jeu à tenir lui-même quelles touches sont tenues, c'est-à-dire à
// réécrire ceci.
//
// Une valeur, donc un jeu la lit sans pouvoir la modifier, et sans qu'elle change sous
// lui pendant qu'il la lit. Sa valeur zéro ne tient aucune touche, ce qui est l'entrée
// d'un pas où rien n'est arrivé.
type Input struct {
	held  [words]uint64
	edges [words]uint64
}

// Held dit si la touche est tenue à cette image, qu'elle vienne d'être pressée ou
// qu'elle le soit depuis un moment.
//
// C'est le régime du déplacement : tant qu'elle rend vrai, le personnage avance.
func (i Input) Held(k Key) bool {
	word, bit, ok := locate(k)
	return ok && i.held[word]&bit != 0
}

// JustPressed dit si la touche vient d'être enfoncée à cette image, et une seule fois
// par appui.
//
// C'est le régime de ce qui se consomme — tirer, ramasser, valider. Elle reste fausse
// tant que le doigt ne se relève pas, donc un appui long ne vaut qu'une action.
func (i Input) JustPressed(k Key) bool {
	word, bit, ok := locate(k)
	return ok && i.edges[word]&bit != 0
}
