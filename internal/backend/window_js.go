// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La fenêtre de la famille web : un canvas, son contexte, et la taille que la page lui
// laisse. Contraint par le suffixe de son nom, comme les deux autres familles.

package backend

import (
	"strconv"
	"syscall/js"

	"github.com/sprimault/lozengine/input"
)

// cssPixels écrit une longueur en pixels CSS, telle qu'une propriété de style l'attend.
//
// La précision n'est pas tronquée : une taille CSS dérivée d'une densité fractionnaire a
// des décimales, et les arrondir déferait le rapport exact entre les deux unités que
// [canvasFor] vient d'établir.
func cssPixels(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "px"
}

// window est ce qu'une page offre de fenêtre : un élément et un contexte de dessin.
//
// **Il n'y a ni poignée système ni socket**, et c'est ce qui distingue cette famille des
// deux autres : tout passe par des appels à la page, dont chacun coûte une traversée de
// la frontière entre le module et le moteur JavaScript. C'est pourquoi la taille se lit
// une fois par pas et se garde, au lieu d'être demandée par chaque méthode.
type window struct {
	// canvas et ctx sont l'élément et son contexte de dessin. Le contexte est demandé
	// une fois : `getContext` rend le même objet à chaque appel, mais l'appel lui-même
	// se paie.
	canvas js.Value
	ctx    js.Value

	// state accumule ce que la page rapporte des entrées. Les écouteurs d'événements
	// viennent avec le lot qui branche le clavier et le pointeur ; ce qui est déjà
	// câblé est la remise à zéro de ce qui vient d'être enfoncé, que `Frame` fait.
	state input.State

	// box est la taille décidée au dernier pas, sous les deux unités. Elle est la
	// réponse de `Size`, qui ne rappelle donc pas la page.
	box canvasBox

	// frame porte ce que la présentation garde d'une image à l'autre.
	frame presentation

	// listeners porte les écouteurs posés dans la page, pour les retirer et libérer
	// leurs fonctions à la fermeture.
	listeners []listener

	// buttons est le masque de boutons du dernier événement de pointeur. Il se compare
	// au suivant pour n'en jouer que les changements, parce que [input.State] veut
	// qu'un bouton se presse et se relâche et non qu'il s'affecte.
	buttons uint8

	// wheel accumule les fractions de cran qu'une page rapporte et qu'une division
	// entière perdrait.
	wheel wheel

	// contactID est le contact que la fenêtre suit, et contact dit qu'il y en a un. Un
	// seul à la fois : la godoc de [window.claim] dit pourquoi.
	contactID int
	contact   bool

	// closed retient que la fenêtre ne présentera plus rien. Il ne se relève jamais.
	closed bool
}

// measure lit la boîte que la page laisse au canvas et le redimensionne quand elle a
// changé.
//
// **Elle est appelée depuis `Input`, et nulle part ailleurs** : le contrat de [Window]
// permet à une famille sans événement de taille d'interroger, à condition que cela ne
// sorte pas de la machine, et il veut que les autres méthodes ne rapportent que ce que
// cette première a appris.
//
// La lecture de la boîte a lieu avant toute écriture dans la page, donc sur une mise en
// page que le navigateur vient de calculer pour peindre. L'ordre inverse — écrire puis
// mesurer — est ce qui force un recalcul, et il coûterait une image.
//
// **Le canvas se redimensionne par ses deux unités ensemble**, et c'est [canvasFor] qui
// les décide : les attributs portent les pixels d'image, le style les pixels CSS. Poser
// les uns sans les autres laisserait la page étirer le canvas d'une fraction.
//
// Écrire dans les attributs efface le canvas, donc on ne l'écrit que lorsque la taille
// change — et la présentation qui suit immédiatement le remplit, ce qui ne laisse rien
// voir du vide.
//
// **Un canvas détaché de la page ferme la fenêtre**, ce qui est l'équivalent de la croix
// pour cette famille : une page qui change de vue démonte son canvas, et le moteur n'a
// plus rien où peindre. Sans cela, la boucle continuerait de tourner et de présenter dans
// le vide, sans que rien n'arrête le jeu.
func (w *window) measure() {
	if w.closed {
		return
	}
	if !w.canvas.Get("isConnected").Bool() {
		w.closed = true
		return
	}

	rect := w.canvas.Call("getBoundingClientRect")
	box := canvasFor(
		rect.Get("width").Float(),
		rect.Get("height").Float(),
		js.Global().Get("devicePixelRatio").Float(),
	)
	if box == w.box {
		return
	}
	w.box = box

	w.canvas.Set("width", box.width)
	w.canvas.Set("height", box.height)
	style := w.canvas.Get("style")
	style.Set("width", cssPixels(box.cssWidth))
	style.Set("height", cssPixels(box.cssHeight))
}

// Size rend la taille du canvas en pixels d'image, telle que le pas l'a apprise.
//
// Jamais d'appel à la page : c'est [window.measure] qui l'a lue, et le contrat veut que
// cette méthode rende ce que le pas courant sait.
//
// Zéro est une réponse licite, et la boucle la traite : c'est ce que rend une page qui a
// masqué le canvas, dont la boîte est alors vide.
func (w *window) Size() (width, height uint16) {
	return w.box.width, w.box.height
}

// Closed dit que la fenêtre ne présentera plus rien.
//
// Pour cette famille, c'est un canvas que la page a démonté. La fermeture de l'onglet ne
// passe pas par ici : le module meurt avec la page, et il n'y a personne pour lire la
// réponse.
func (w *window) Closed() bool {
	return w.closed
}

// Err dit ce qui a mis fin à la fenêtre, et nil quand c'est la page.
//
// Toujours nil pour cette famille : il n'y a ni protocole à rompre ni serveur à perdre,
// et un canvas démonté est le régime normal. Ce qui échoue avant que la fenêtre existe
// est rendu par [Open].
func (w *window) Err() error {
	return nil
}

// Close rend à la page ce qu'[Open] a pris, et n'y laisse rien tourner.
//
// Le canvas n'est pas retiré du document : le moteur ne l'a pas forcément créé, et
// démonter l'élément d'une page qu'on n'a pas mise en page serait une décision qui ne
// lui appartient pas. Ce qui part est ce que le module a posé **dans** la page — ses
// écouteurs — et ce qu'il tient pour elle : la mémoire épinglée pour la présentation.
//
// **Les écouteurs se retirent et leurs fonctions se libèrent**, les deux et pas l'une :
// une fonction libérée mais toujours écoutée fait jeter la page au premier événement, et
// un écouteur retiré sans libération retient la valeur côté Go jusqu'à la fin du module.
//
// Appeler deux fois est sans effet.
func (w *window) Close() {
	w.closed = true
	w.silence()
	w.frame.release()
}
