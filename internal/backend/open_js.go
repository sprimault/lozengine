// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'ouverture d'une fenêtre sur un canvas de la page, et les deux choses qu'elle refuse.

package backend

import (
	"errors"
	"syscall/js"
)

// canvasID est l'identifiant de l'élément que le moteur cherche dans la page.
//
// **Une convention plutôt qu'une option**, et c'est ce qui garde la surface publique du
// moteur hors de ce choix : un jeu qui veut placer le canvas lui-même écrit
// `<canvas id="lozengine">` où sa mise en page le demande, et le moteur le trouve. Une
// option aurait dû remonter jusqu'aux options du paquet racine, donc jusqu'à une
// signature traitée comme définitive, pour un réglage qu'une ligne de page exprime.
//
// Absent, le moteur crée le canvas et l'attache au corps de la page. C'est ce qui fait
// tourner un jeu dans une page vide, et c'est le régime du relevé sans écran.
const canvasID = "lozengine"

// Open ouvre une fenêtre sur un canvas de la page.
//
// # Ce qu'elle refuse, et pourquoi deux refus seulement
//
// **Une exécution sans page.** C'est le régime de `node`, sous lequel la suite de
// référence tourne : il n'y a ni document ni canvas, donc rien à ouvrir. Le refus est
// nommé plutôt que laissé à une exception de la page, qu'un module Go ne rattrape pas.
//
// **Un contexte qui garde son quatrième octet.** Le moteur ne convertit aucun pixel
// parce qu'un contexte `{alpha: false}` ignore cet octet par la spécification ; s'il le
// lisait, les trois octets de couleur étant prémultipliés, la page afficherait autre
// chose que Windows et X11. Le cas arrive pour de vrai : si la page a déjà demandé un
// contexte 2D sur le même canvas, `getContext` rend celui-là et ignore les options
// qu'on lui passe. D'où la relecture de ce que le contexte porte vraiment, plutôt que la
// confiance dans ce qu'on a demandé.
//
// # Ce qu'elle pose et ce qu'elle ne pose pas
//
// `image-rendering: pixelated` est posé, et il est fiable ici : la page n'étire le canvas
// que d'un facteur entier, [canvasFor] s'en assurant, et la clause de CSS Images Level 3
// qui impose une étape lisse ne vaut que pour un reste non entier. Sans lui, un canvas
// agrandi par la densité serait lissé, ce qui est exactement ce que l'échelle entière du
// moteur existe pour éviter.
//
// Le titre, l'icône et le curseur ne sont pas posés : ils viennent avec le lot qui les
// pose, comme les écouteurs d'entrées viennent avec celui qui les branche.
//
// La taille demandée sert de taille d'ouverture **au canvas que le moteur crée**, en
// pixels CSS — l'équivalent des points d'un système de fenêtrage. Un canvas que la page
// fournit garde la taille que la page lui donne : la mise en page ne lui appartient pas.
func Open(o Options) (Window, error) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return nil, errors.New("no window in this runtime: the browser backend needs a page, and this one has no document")
	}

	canvas := doc.Call("getElementById", canvasID)
	if !canvas.Truthy() {
		body := doc.Get("body")
		if !body.Truthy() {
			return nil, errors.New("no canvas and no body to attach one to: the page is loading, or it is not a document")
		}

		canvas = doc.Call("createElement", "canvas")
		canvas.Set("id", canvasID)
		style := canvas.Get("style")
		style.Set("width", cssPixels(float64(o.Width)))
		style.Set("height", cssPixels(float64(o.Height)))
		body.Call("appendChild", canvas)
	}
	style := canvas.Get("style")
	style.Set("imageRendering", "pixelated")

	// Sans cela, un doigt posé sur le canvas fait défiler ou zoomer la page au lieu de
	// jouer, et l'événement de pointeur est annulé en route. C'est l'équivalent tactile
	// de ce que [scrollCodes] empêche au clavier.
	style.Set("touchAction", "none")

	ctx := canvas.Call("getContext", "2d", map[string]any{"alpha": false})
	if !ctx.Truthy() {
		return nil, errors.New("no 2d context on canvas #" + canvasID + ": the page may hold another kind of context on it")
	}
	if attrs := ctx.Call("getContextAttributes"); attrs.Truthy() && attrs.Get("alpha").Bool() {
		return nil, errors.New("the 2d context of canvas #" + canvasID + " keeps an alpha channel: the engine's pixels are premultiplied, and the page would not show what the other backends show")
	}

	w := &window{canvas: canvas, ctx: ctx}
	w.newFrames()
	if memory := js.Global().Get(linearMemory); memory.Truthy() {
		w.frame.memory = memory
	}

	// La taille se lit avant de rendre la fenêtre : sans cela, le premier pas
	// présenterait dans un canvas dont la boucle croit la taille nulle.
	w.measure()
	w.listen()
	return w, nil
}
