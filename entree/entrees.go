// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package entree

// Entrees est ce que le clavier dit à une image : les touches tenues, et celles qui
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
type Entrees struct {
	tenues  [mots]uint64
	montees [mots]uint64
}

// Enfoncee dit si la touche est tenue à cette image, qu'elle vienne d'être pressée ou
// qu'elle le soit depuis un moment.
//
// C'est le régime du déplacement : tant qu'elle rend vrai, le personnage avance.
func (e Entrees) Enfoncee(t Touche) bool {
	mot, bit, ok := place(t)
	return ok && e.tenues[mot]&bit != 0
}

// Pressee dit si la touche vient d'être enfoncée à cette image, et une seule fois par
// appui.
//
// C'est le régime de ce qui se consomme — tirer, ramasser, valider. Elle reste fausse
// tant que le doigt ne se relève pas, donc un appui long ne vaut qu'une action.
func (e Entrees) Pressee(t Touche) bool {
	mot, bit, ok := place(t)
	return ok && e.montees[mot]&bit != 0
}
