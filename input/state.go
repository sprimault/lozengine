// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'état que le backend tient d'une image à l'autre, et dont il dérive le front
// montant en comparant avec l'image précédente. C'est le côté écriture du clavier,
// celui qu'un jeu ne voit pas.

package input

// State accumule ce qu'un backend relève du clavier, et en tire les entrées d'une
// image.
//
// Il vit du côté qui parle au système : une fenêtre reçoit des événements quand ils
// arrivent, à un rythme qui n'est pas celui des images, et quelqu'un doit les garder
// jusqu'au pas suivant. C'est aussi lui qui tient l'image précédente, sans quoi le
// front montant ne se calculerait pas.
//
// Sa valeur zéro est prête : aucune touche tenue, aucune montée.
//
// Un jeu n'en a pas besoin, mais rien ne le lui interdit — au pire il se mentirait à
// lui-même. C'est délibéré : la suite de non-régression doit pouvoir fabriquer des
// entrées sans fenêtre, et un scénario qui simule un appui passe par ici.
type State struct {
	held     [words]uint64
	previous [words]uint64
}

// Press note qu'une touche est tenue. Le répéter est sans effet, ce qui absorbe la
// répétition automatique du système : celle-ci renvoie l'événement d'appui tant que le
// doigt reste posé, et un front montant par répétition ferait tirer en rafale.
func (s *State) Press(k Key) {
	if word, bit, ok := locate(k); ok {
		s.held[word] |= bit
	}
}

// Release note qu'une touche ne l'est plus. C'est le relâchement qui réarme le front
// montant, donc une action ne se répète qu'en relevant le doigt.
func (s *State) Release(k Key) {
	if word, bit, ok := locate(k); ok {
		s.held[word] &^= bit
	}
}

// Forget relâche tout.
//
// C'est ce qu'une fenêtre appelle en perdant le focus : le système cesse alors
// d'envoyer les relâchements, donc une touche tenue au moment du changement le
// resterait indéfiniment et le personnage marcherait tout seul au retour.
func (s *State) Forget() {
	s.held = [words]uint64{}
}

// Frame rend les entrées du pas qui commence, et arme le front montant pour le suivant.
//
// Une touche est montée quand elle est tenue maintenant et ne l'était pas au pas
// précédent. Appeler deux fois pour un même pas rendrait donc des montées vides la
// seconde fois : c'est une fois par pas, depuis la boucle, et jamais depuis le jeu.
func (s *State) Frame() Input {
	var frame Input
	for i := range s.held {
		frame.held[i] = s.held[i]
		frame.edges[i] = s.held[i] &^ s.previous[i]
	}
	s.previous = s.held
	return frame
}
