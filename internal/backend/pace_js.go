// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La cadence de la famille web : l'attente de l'image que la page s'apprête à composer,
// et ce qui la rend sûre quand la page cesse d'en demander.

package backend

import "syscall/js"

// Paced dit que cette fenêtre cadence la boucle, en attendant dans `Input` que la page
// réclame une image.
//
// **C'est la seule des trois familles dans ce cas**, et la godoc de [Window.Paced] dit ce
// que la boucle en fait. Ce qui l'impose ici est mesuré : une boucle qui dort pour tenir
// son pas écrit deux tiers de ses images hors du tour d'animation à quatre-vingt-dix pour
// cent de charge, et le navigateur compose alors un tampon que le moteur réécrit.
func (w *window) Paced() bool { return true }

// armFrame demande à la page de rappeler quand elle s'apprêtera à composer une image.
//
// **Une demande par attente, et non une chaîne qui se rearme côté page.** Les deux
// marchent — le relevé a éprouvé la seconde —, mais la chaîne demande du code dans la
// page, donc un contrat de plus avec qui l'héberge. Ici la fenêtre tient sa propre
// cadence, et une page qui ne sait rien du moteur suffit.
//
// Le coût est un trou : entre la présentation d'un tour et la demande du suivant, une
// image de la page peut passer. Il est borné par la durée d'un pas, et il ne coûte au
// pire qu'une image sur un redémarrage de boucle — là qu'une chaîne entretenue coûterait
// un rappel perdu par image, le relevé en comptant jusqu'à deux cent quarante-deux sur
// deux secondes.
func (w *window) armFrame() {
	if w.armed || w.closed {
		return
	}
	w.armed = true
	js.Global().Call("requestAnimationFrame", w.onFrame)
}

// awaitFrame attend l'image demandée et rend la main à la page pendant ce temps.
//
// **C'est cette attente qui fait tourner la page**, et c'est le point le plus contre-
// intuitif de la cible : une goroutine qui ne se gare jamais empêche la boucle
// d'événements de la page de tourner, donc l'image demandée n'arrive jamais. Recevoir sur
// un canal gare la goroutine, et le module rend la main.
//
// **Une page qui cesse de composer arrête le jeu, et c'est voulu.** Un onglet caché ou
// une fenêtre recouverte ne reçoit plus d'image d'animation — mesuré le 2 novembre 2026,
// deux rappels par seconde sur une fenêtre recouverte, alors que la page se déclare
// visible et avec le focus. Le pas suivant attend donc le retour, ce qui est le
// comportement d'un jeu dans une page ; et la boucle ne rattrape pas le temps perdu,
// `pace` repartant de maintenant au-delà de deux ticks de retard.
func (w *window) awaitFrame() {
	if w.closed {
		return
	}
	w.armFrame()
	<-w.frames
}

// newFrames monte le canal d'images et le rappel qui l'alimente.
//
// **Le canal porte une place et l'envoi ne bloque pas**, ce que la documentation de
// `js.FuncOf` exige : un rappel qui bloquerait retiendrait la boucle d'événements de la
// page, et la page cesserait de répondre. Un rappel qui arrive alors qu'une image attend
// déjà est donc perdu, ce qui est juste — la boucle n'en consomme qu'une par pas.
func (w *window) newFrames() {
	w.frames = make(chan struct{}, 1)
	w.onFrame = js.FuncOf(func(js.Value, []js.Value) any {
		w.armed = false
		select {
		case w.frames <- struct{}{}:
		default:
		}
		return nil
	})
}

// releaseFrames libère le rappel de cadence. Appeler deux fois est sans effet.
//
// Le canal n'est pas fermé : une fermeture ferait rendre l'attente immédiatement et en
// boucle, là qu'une fenêtre fermée ne doit plus rien attendre du tout — ce dont
// [window.awaitFrame] s'assure par son premier test.
func (w *window) releaseFrames() {
	if w.onFrame.Truthy() {
		w.onFrame.Release()
		w.onFrame = js.Func{}
	}
}
