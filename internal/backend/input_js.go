// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les écouteurs par lesquels une page rapporte ses entrées, et ce que la fenêtre en fait.

package backend

import (
	"syscall/js"

	"github.com/sprimault/lozengine/input"
)

// listener retient de quoi retirer un écouteur et libérer la fonction qui le porte.
//
// **Une fonction posée dans la page ne se ramasse pas toute seule** : `js.FuncOf` retient
// la valeur côté Go jusqu'à sa libération, et la page garde sa référence jusqu'au retrait
// de l'écouteur. Les deux gestes vont ensemble, d'où ce triplet.
type listener struct {
	target js.Value
	name   string
	fn     js.Func
}

// listen pose les écouteurs de la page et retient de quoi les retirer.
//
// **Le clavier s'écoute sur la page entière et le pointeur sur le canvas**, et ce n'est
// pas une asymétrie gratuite : un canvas ne reçoit les touches que s'il porte un index de
// tabulation et que le joueur l'a cliqué, là qu'une position de pointeur hors du canvas
// n'a pas de sens. C'est la répartition qu'une page de jeu attend, et celle qui demande le
// moins à la page qui l'héberge.
//
// Aucun écouteur ne touche à l'image : ils ne font qu'écrire dans l'état que le pas
// suivant lira. L'invariant qui interdit un rappel dans le chemin de rendu n'est donc pas
// en cause — ce sont des événements, et ils arrivent entre deux pas.
func (w *window) listen() {
	page := js.Global()

	w.on(page, "keydown", false, func(e js.Value) {
		code := e.Get("code").String()
		if k, ok := placeOfCode(code); ok {
			w.state.Press(k)
		}
		if scrollsThePage(code) && !modified(e) {
			e.Call("preventDefault")
		}
	})
	w.on(page, "keyup", true, func(e js.Value) {
		if k, ok := placeOfCode(e.Get("code").String()); ok {
			w.state.Release(k)
		}
	})

	// La page perd le focus et cesse d'envoyer les relâchements : sans cet oubli, une
	// touche tenue au moment du changement d'onglet le resterait pour toujours et le
	// personnage marcherait tout seul au retour.
	w.on(page, "blur", true, func(js.Value) {
		w.state.Forget()
		w.buttons = 0
	})

	w.on(w.canvas, "pointerdown", false, func(e js.Value) {
		if !w.claim(e) {
			return
		}
		// La capture fait suivre les mouvements et le relâchement même hors du canvas.
		// Sans elle, un bouton relâché ailleurs resterait tenu, ce qui est le même
		// défaut que la touche oubliée au départ du focus.
		w.canvas.Call("setPointerCapture", e.Get("pointerId"))
		w.track(e)
	})
	w.on(w.canvas, "pointermove", true, func(e js.Value) {
		if w.owns(e) {
			w.track(e)
		}
	})
	w.on(w.canvas, "pointerup", true, func(e js.Value) {
		if w.owns(e) {
			w.track(e)
			w.releaseContact()
		}
	})
	w.on(w.canvas, "pointercancel", true, func(e js.Value) {
		if w.owns(e) {
			w.buttons = playButtons(&w.state, w.buttons, 0)
			w.state.Leave()
			w.releaseContact()
		}
	})
	w.on(w.canvas, "pointerleave", true, func(e js.Value) {
		if w.owns(e) {
			w.state.Leave()
		}
	})

	w.on(w.canvas, "wheel", false, func(e js.Value) {
		if steps := w.wheel.add(e.Get("deltaY").Float(), e.Get("deltaMode").Int()); steps != 0 {
			w.state.Wheel(steps)
		}
		// Sans cela la page défile sous le jeu, pour la raison que donne
		// [scrollsThePage].
		e.Call("preventDefault")
	})
}

// on pose un écouteur et le retient.
//
// `passive` dit à la page qu'on ne l'empêchera pas d'agir, ce qui lui permet de défiler
// sans attendre : il est donc vrai partout sauf là où l'écouteur appelle vraiment
// `preventDefault`. Le déclarer et ne pas le tenir fait ignorer l'appel, en avertissant
// dans la console.
func (w *window) on(target js.Value, name string, passive bool, handle func(e js.Value)) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			handle(args[0])
		}
		return nil
	})
	target.Call("addEventListener", name, fn, map[string]any{"passive": passive})
	w.listeners = append(w.listeners, listener{target: target, name: name, fn: fn})
}

// silence retire les écouteurs et libère leurs fonctions. Appeler deux fois est sans
// effet.
func (w *window) silence() {
	for _, l := range w.listeners {
		l.target.Call("removeEventListener", l.name, l.fn)
		l.fn.Release()
	}
	w.listeners = nil
}

// modified dit si l'événement porte un modificateur que la page ou le système emploie.
//
// `shiftKey` n'en est pas : il ne change aucun raccourci de navigation, et un jeu l'emploie
// couramment avec une flèche.
func modified(e js.Value) bool {
	return e.Get("ctrlKey").Bool() || e.Get("metaKey").Bool() || e.Get("altKey").Bool()
}

// claim fait du contact de cet événement celui que la fenêtre suit, et dit s'il l'est.
//
// **Un seul contact à la fois, et les suivants sont ignorés** : `input.State` n'offre
// qu'un pointeur, et lui en ajouter serait une signature publique qu'aucun jeu n'a
// demandée. La forme retenue l'accueillerait sans rien casser, le contact étant suivi par
// son identifiant.
func (w *window) claim(e js.Value) bool {
	if w.contact {
		return false
	}
	w.contactID, w.contact = e.Get("pointerId").Int(), true
	return true
}

// owns dit si l'événement vient du contact suivi, ou d'un pointeur libre.
//
// **Un pointeur libre est suivi aussi, et c'est ce qui distingue une souris d'un doigt** :
// une souris survole sans jamais appuyer, donc exiger un contact revendiqué ferait perdre
// tout mouvement hors clic. Un doigt, lui, n'existe qu'entre son appui et son relâchement.
func (w *window) owns(e js.Value) bool {
	return !w.contact || e.Get("pointerId").Int() == w.contactID
}

// releaseContact rend le suivi au prochain contact qui se présentera.
func (w *window) releaseContact() {
	w.contact, w.contactID = false, 0
}

// track porte dans l'état la position et les boutons que l'événement rapporte.
//
// L'ordre compte peu ici, mais la position vient d'abord par principe : c'est elle qui
// situe le clic que les boutons vont poser.
func (w *window) track(e js.Value) {
	x, y := pointerAt(e.Get("offsetX").Float(), e.Get("offsetY").Float(), w.box)
	w.state.Move(x, y)
	w.buttons = playButtons(&w.state, w.buttons, uint8(e.Get("buttons").Int()))
}

// Input attend l'image que la page s'apprête à composer, puis rend les entrées du pas
// qui commence.
//
// **Rien à vider ici, contrairement aux deux autres familles** : les écouteurs ont déjà
// écrit dans l'état quand la page les a appelés, et une page n'a pas de file de messages
// à pomper. Ce que cette méthode fait est donc l'inverse — elle **attend**, et c'est ce
// qui cadence la boucle, [Window.Paced] le déclarant.
//
// L'ordre compte : l'attente vient d'abord, de sorte que tout ce que les écouteurs ont
// rapporté pendant qu'elle durait entre dans ce pas-ci. Mesurer ensuite lit une mise en
// page que le navigateur vient de calculer pour composer, ce que la godoc de
// [window.measure] explique.
func (w *window) Input() input.Input {
	w.awaitFrame()
	w.measure()
	return w.state.Frame()
}
