// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La traduction du pointeur X11 : les numéros de bouton du protocole vers ceux du
// moteur, les crans de molette qui n'en sont pas, et la position dans l'événement.

package backend

import "github.com/sprimault/lozengine/input"

// Les numéros que le protocole donne aux boutons. Les trois premiers sont ceux d'une
// souris ordinaire ; quatre et cinq sont la molette, que le cœur de X11 n'a pas d'autre
// moyen de rapporter ; huit et neuf sont les boutons latéraux.
//
// Six et sept existent aussi, pour une molette horizontale que le moteur ne nomme pas :
// ils sont ignorés plutôt que confondus avec autre chose.
const (
	x11ButtonLeft     = 1
	x11ButtonMiddle   = 2
	x11ButtonRight    = 3
	x11WheelUp        = 4
	x11WheelDown      = 5
	x11ButtonBack     = 8
	x11ButtonForward  = 9
	pointerEventX     = 24
	pointerEventYSkip = 26
)

// buttonOf traduit un numéro de bouton du protocole, et dit s'il en porte un que le
// moteur nomme.
func buttonOf(detail byte) (input.Button, bool) {
	switch detail {
	case x11ButtonLeft:
		return input.ButtonLeft, true
	case x11ButtonMiddle:
		return input.ButtonMiddle, true
	case x11ButtonRight:
		return input.ButtonRight, true
	case x11ButtonBack:
		return input.ButtonBack, true
	case x11ButtonForward:
		return input.ButtonForward, true
	}
	return 0, false
}

// wheelStep dit si un numéro de bouton est en réalité un cran de molette, et lequel.
//
// Le protocole n'a pas d'axe : une molette s'y rapporte comme un bouton pressé puis
// relâché aussitôt. Les traiter comme des boutons les ferait apparaître tenus pendant
// une image, et un jeu qui lit [input.Input.ButtonHeld] verrait un clic.
func wheelStep(detail byte) (steps int, ok bool) {
	switch detail {
	case x11WheelUp:
		return 1, true
	case x11WheelDown:
		return -1, true
	}
	return 0, false
}

// pointerAt relève la position que porte un événement de pointeur.
//
// Les coordonnées relatives à la fenêtre sont au même endroit dans les cinq événements
// qui nous intéressent — appui, relâchement, mouvement, entrée, sortie —, ce qui évite
// un décodage par type. Elles sont signées : un appui qui commence dans la fenêtre et
// se poursuit dehors continue d'être rapporté, avec des coordonnées négatives.
func (w *window) pointerAt(m []byte) {
	x := int16(order.Uint16(m[pointerEventX:]))
	y := int16(order.Uint16(m[pointerEventYSkip:]))
	w.state.Move(x, y)
}
