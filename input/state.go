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

	// seen retient ce qui a été enfoncé depuis le pas précédent, même si le doigt
	// s'est relevé entre-temps. Sans lui, un appui plus court qu'un pas disparaît
	// sans trace : son appui et son relâchement arrivent dans la même poignée de
	// messages, les deux s'annulent, et le pas ne voit rien. Relevé le 25 octobre
	// 2026 à la vérification manuelle du backend Windows, où une touche sur deux
	// paraissait ne pas répondre.
	seen [words]uint64

	// Le pointeur, dans l'unité où le backend le reçoit — des pixels de fenêtre.
	// C'est la boucle qui les ramène en pixels de l'image, par [Input.WithPointer] :
	// elle seule connaît le facteur d'agrandissement et les bandes.
	pointerX, pointerY int16
	inside             bool

	// Les boutons, sur le même modèle que les touches : ce qui est tenu, ce qui a été
	// vu depuis le pas précédent, et ce qui l'était au pas précédent.
	buttons, buttonsSeen, buttonsPrevious uint8

	// Les crans de molette accumulés depuis le pas précédent, comptés et non tenus :
	// une molette n'a pas d'état, seulement des mouvements.
	wheel int16
}

// Move note où le pointeur se trouve, dans l'unité du backend, et qu'il est dans la
// fenêtre.
//
// Les coordonnées ne sont pas vérifiées : une plateforme qui rapporte un pointeur
// capturé hors de la fenêtre dit quelque chose de vrai, et c'est au jeu d'en décider.
func (s *State) Move(x, y int16) {
	s.pointerX, s.pointerY, s.inside = x, y, true
}

// Leave note que le pointeur a quitté la fenêtre, sa dernière position restant lisible.
//
// La garder plutôt que l'effacer : un jeu qui suit le curseur veut savoir par où il
// est sorti, et une position remise à zéro désignerait le coin de l'image.
func (s *State) Leave() { s.inside = false }

// PressButton note qu'un bouton du pointeur est tenu. Le répéter est sans effet, comme
// pour une touche.
func (s *State) PressButton(b Button) {
	if bit, ok := b.bit(); ok {
		s.buttons |= bit
		s.buttonsSeen |= bit
	}
}

// ReleaseButton note qu'un bouton ne l'est plus, sans effacer la trace de l'appui pour
// le pas en cours.
func (s *State) ReleaseButton(b Button) {
	if bit, ok := b.bit(); ok {
		s.buttons &^= bit
	}
}

// Wheel ajoute des crans de molette au pas qui vient, positifs vers le haut.
//
// Ils s'ajoutent plutôt que de se remplacer : plusieurs crans arrivent couramment dans
// la même poignée d'événements, et n'en garder qu'un perdrait la vitesse du geste.
func (s *State) Wheel(steps int) {
	s.wheel += int16(steps)
}

// Press note qu'une touche est tenue. Le répéter est sans effet, ce qui absorbe la
// répétition automatique du système : celle-ci renvoie l'événement d'appui tant que le
// doigt reste posé, et un front montant par répétition ferait tirer en rafale.
func (s *State) Press(k Key) {
	if word, bit, ok := locate(k); ok {
		s.held[word] |= bit
		s.seen[word] |= bit
	}
}

// Release note qu'une touche ne l'est plus. C'est le relâchement qui réarme le front
// montant, donc une action ne se répète qu'en relevant le doigt.
//
// Il n'efface pas la trace de l'appui pour le pas en cours — ce qui s'est passé
// pendant un pas appartient à ce pas, même si c'est fini quand il se calcule.
func (s *State) Release(k Key) {
	if word, bit, ok := locate(k); ok {
		s.held[word] &^= bit
	}
}

// Forget relâche tout, trace comprise.
//
// C'est ce qu'une fenêtre appelle en perdant le focus : le système cesse alors
// d'envoyer les relâchements, donc une touche tenue au moment du changement le
// resterait indéfiniment et le personnage marcherait tout seul au retour. La trace
// part avec, sinon l'appui perdu réapparaîtrait au pas suivant.
func (s *State) Forget() {
	s.held = [words]uint64{}
	s.seen = [words]uint64{}
	s.buttons, s.buttonsSeen = 0, 0
	s.inside = false
}

// Frame rend les entrées du pas qui commence, et arme le front montant pour le suivant.
//
// **Un pas rapporte ce qui s'est passé pendant lui, et non l'état à sa fin.** Une
// touche enfoncée puis relâchée entre deux pas y est donc tenue et montée, ce qui est
// la seule lecture utile : à soixante images par seconde, un appui ordinaire dure
// moins d'un pas une fois sur deux, et le perdre ferait « une touche sur deux ne
// répond pas ».
//
// Une touche est montée quand elle est tenue à ce pas et ne l'était pas au précédent.
// Appeler deux fois pour un même pas rendrait donc des montées vides la seconde fois :
// c'est une fois par pas, depuis la boucle, et jamais depuis le jeu.
//
// **Ce que ce modèle ne peut pas rendre, et il faut le dire** : deux appuis de la même
// touche dans le même pas ne font qu'une montée. Un état par pas n'a pas de quoi en
// compter deux, et c'est le prix de l'arbitrage qui a préféré un état à une file
// d'événements — lequel évite qu'un pas coûte selon ce qui est arrivé depuis le
// précédent. Le cas demande de taper deux fois en moins de seize millisecondes, ce
// qu'une main ne fait pas ; il devient atteignable quand la boucle traîne, et c'est
// alors la boucle qu'il faut regarder. Mesuré le 25 octobre 2026 : une boucle qui
// dort à l'aveugle au lieu de mesurer son pas rate des images, et deux tapes
// ordinaires finissent dans la même.
func (s *State) Frame() Input {
	var frame Input
	for i := range s.held {
		frame.held[i] = s.held[i] | s.seen[i]
		frame.edges[i] = frame.held[i] &^ s.previous[i]
	}
	s.previous = frame.held
	s.seen = [words]uint64{}

	frame.pointerX, frame.pointerY, frame.inside = s.pointerX, s.pointerY, s.inside

	frame.buttons = s.buttons | s.buttonsSeen
	frame.buttonEdges = frame.buttons &^ s.buttonsPrevious
	s.buttonsPrevious, s.buttonsSeen = frame.buttons, 0

	frame.wheel, s.wheel = s.wheel, 0

	return frame
}
