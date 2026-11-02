// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La correspondance entre ce que la page nomme une touche et ce que le moteur appelle une
// place. Le fichier n'importe pas `syscall/js`, pour que ses cas tournent sur l'hôte.

package backend

import "github.com/sprimault/lozengine/input"

// placeOfCode traduit le `code` d'un événement de clavier de la page vers la place que le
// moteur nomme, et dit si cette place existe.
//
// **Le nom ne suit pas la convention des familles, et il ne peut pas.** Chacune a son
// `keyOf` — un `byte` sous X11, un `uintptr` sous Windows —, le nom se réutilisant parce
// que leurs fichiers sont mutuellement exclusifs. Ce fichier n'a pas de contrainte de
// construction, pour que ses cas tournent sur l'hôte, donc il coexiste avec les deux et
// doit nommer autrement. Tout fichier `_web.go` ultérieur retombera sur cette règle.
//
// # Pourquoi `code`, et jamais `key` ni `keyCode`
//
// Une [input.Key] désigne **l'endroit** où la touche se trouve, pas le caractère qu'elle
// produit : c'est ce qui fait que `KeyW` reste la touche au-dessus de `KeyS` quelle que
// soit la disposition du joueur. De ses trois attributs, `code` est le seul que la
// spécification définit ainsi. `key` suit la disposition active — mesuré sur un clavier
// AZERTY réel, `code=KeyA` donne `key="q"` et `code=KeyM` donne `key=","` —, et `keyCode`
// est déprécié.
//
// # Les dix renommages que personne n'annonce
//
// Vingt-six places de lettres portent le même nom des deux côtés, ce qui donne l'illusion
// d'une table mécanique. Les chiffres de la rangée du haut la défont : le moteur les
// appelle `Key0` à `Key9`, la page `Digit0` à `Digit9`. C'est la seule famille où les deux
// vocabulaires divergent sans que rien ne le signale, et une table écrite au jugé les
// manquerait toutes les dix.
//
// # Les trois ambiguïtés que le relevé a fermées
//
// Aucune ne demande de cas particulier, et c'est ce qui compte, car chacune aurait pu en
// demander un. **Le pavé numérique** rend le même `code` que le verrou soit allumé ou
// éteint — seul `key` bascule, de « 1 » à « End » —, donc les dix places du pavé sont
// distinctes des six de navigation sans effort. **AltGr** livre `ControlLeft` puis
// `AltRight`, exactement ce que le backend Windows rapporte déjà ; en traiter le cas ici
// ferait deux comportements pour un même clavier. **Les modificateurs** sont deux `code`
// distincts par paire, donc deux entrées.
//
// Reste une place dont la justesse n'est pas vérifiée par une frappe réelle : celle du
// verrou numérique. La spécification lui donne `"NumLock"` et la table le suit — c'est ce
// qui fait foi ici, une implémentation ne faisant pas le contrat. Mais les tables de
// balayage de Chrome et de Windows sont inversées sur cette touche, donc si un appui réel
// rendait `"Pause"`, la place serait perdue en silence. `"Pause"` n'est pas une des places
// du moteur, donc elle ne peut pas se traduire par erreur vers une autre touche : le
// mode de panne est une touche muette, pas une touche fausse.
func placeOfCode(code string) (input.Key, bool) {
	k, ok := codePlaces[code]
	return k, ok
}

// codePlaces porte les quatre-vingt-six correspondances, et elle est close : le moteur ne
// déclare pas d'autre place, et un `code` que la page envoie sans qu'il y figure est une
// touche que le jeu ne lit pas.
var codePlaces = map[string]input.Key{
	"KeyA": input.KeyA, "KeyB": input.KeyB, "KeyC": input.KeyC, "KeyD": input.KeyD,
	"KeyE": input.KeyE, "KeyF": input.KeyF, "KeyG": input.KeyG, "KeyH": input.KeyH,
	"KeyI": input.KeyI, "KeyJ": input.KeyJ, "KeyK": input.KeyK, "KeyL": input.KeyL,
	"KeyM": input.KeyM, "KeyN": input.KeyN, "KeyO": input.KeyO, "KeyP": input.KeyP,
	"KeyQ": input.KeyQ, "KeyR": input.KeyR, "KeyS": input.KeyS, "KeyT": input.KeyT,
	"KeyU": input.KeyU, "KeyV": input.KeyV, "KeyW": input.KeyW, "KeyX": input.KeyX,
	"KeyY": input.KeyY, "KeyZ": input.KeyZ,

	// Les dix renommages : la page dit `Digit`, le moteur dit `Key`.
	"Digit0": input.Key0, "Digit1": input.Key1, "Digit2": input.Key2,
	"Digit3": input.Key3, "Digit4": input.Key4, "Digit5": input.Key5,
	"Digit6": input.Key6, "Digit7": input.Key7, "Digit8": input.Key8,
	"Digit9": input.Key9,

	"ArrowUp": input.KeyUp, "ArrowLeft": input.KeyLeft,
	"ArrowRight": input.KeyRight, "ArrowDown": input.KeyDown,

	"Escape": input.KeyEscape, "Backspace": input.KeyBackspace,
	"Tab": input.KeyTab, "Enter": input.KeyEnter, "Space": input.KeySpace,

	"ShiftLeft": input.KeyLeftShift, "ShiftRight": input.KeyRightShift,
	"ControlLeft": input.KeyLeftControl, "ControlRight": input.KeyRightControl,
	"AltLeft": input.KeyLeftAlt, "AltRight": input.KeyRightAlt,

	"F1": input.KeyF1, "F2": input.KeyF2, "F3": input.KeyF3, "F4": input.KeyF4,
	"F5": input.KeyF5, "F6": input.KeyF6, "F7": input.KeyF7, "F8": input.KeyF8,
	"F9": input.KeyF9, "F10": input.KeyF10, "F11": input.KeyF11, "F12": input.KeyF12,

	"Home": input.KeyHome, "End": input.KeyEnd,
	"PageUp": input.KeyPageUp, "PageDown": input.KeyPageDown,
	"Insert": input.KeyInsert, "Delete": input.KeyDelete,

	"Numpad0": input.KeyNumpad0, "Numpad1": input.KeyNumpad1,
	"Numpad2": input.KeyNumpad2, "Numpad3": input.KeyNumpad3,
	"Numpad4": input.KeyNumpad4, "Numpad5": input.KeyNumpad5,
	"Numpad6": input.KeyNumpad6, "Numpad7": input.KeyNumpad7,
	"Numpad8": input.KeyNumpad8, "Numpad9": input.KeyNumpad9,
	"NumpadDecimal": input.KeyNumpadDecimal, "NumpadMultiply": input.KeyNumpadMultiply,
	"NumpadSubtract": input.KeyNumpadSubtract, "NumpadAdd": input.KeyNumpadAdd,
	"NumpadDivide": input.KeyNumpadDivide, "NumpadEnter": input.KeyNumpadEnter,

	"NumLock": input.KeyNumLock,
}

// scrollsThePage dit si une place est de celles qu'une page emploie pour défiler, et que
// le moteur doit donc l'empêcher d'employer tant que la fenêtre est ouverte.
//
// **Un jeu qui déplace son personnage aux flèches ferait sinon défiler la page sous lui**,
// et c'est le défaut le plus visible qu'une page de jeu puisse avoir.
//
// **La liste est celle du défilement, pas celle des touches que le jeu lit.** Empêcher
// tout ce que le moteur nomme retirerait au joueur l'actualisation, la fermeture de
// l'onglet et la navigation au clavier, pour un bénéfice nul : ces places-là ne déplacent
// pas la page. Le backend n'empêche d'ailleurs rien quand un modificateur accompagne la
// touche, ce qui laisse passer `Ctrl+R`, `Cmd+W` et le retour en arrière par
// `Alt+Flèche` — seule la conséquence dans la page est retirée, jamais la lecture par le
// jeu.
//
// Sans contrainte de plateforme, comme la table ci-dessus et pour la même raison : son cas
// tourne alors partout, et pas seulement sur la cible qui la consomme.
func scrollsThePage(code string) bool {
	switch code {
	case "Space", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
		"PageUp", "PageDown", "Home", "End":
		return true
	}
	return false
}
