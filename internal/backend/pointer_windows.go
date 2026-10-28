// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La traduction du pointeur Win32 : les messages de bouton et de molette, la position
// qu'ils portent, et le suivi qu'il faut demander pour savoir que le pointeur est
// sorti.

package backend

import (
	"unsafe"

	"github.com/sprimault/lozengine/input"
)

// Les messages de pointeur. Les deux derniers couvrent les boutons latéraux, qu'un
// seul couple de messages rapporte — c'est le mot haut de wParam qui dit lequel.
const (
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmRButtonDown  = 0x0204
	wmRButtonUp    = 0x0205
	wmMButtonDown  = 0x0207
	wmMButtonUp    = 0x0208
	wmMouseWheel   = 0x020A
	wmXButtonDown  = 0x020B
	wmXButtonUp    = 0x020C
	wmMouseLeave   = 0x02A3
	xButtonBack    = 1
	xButtonForward = 2

	// wheelDelta est ce qu'un cran de molette vaut. Le système compte en
	// cent-vingtièmes de cran pour que les molettes libres, qui tournent sans
	// crantage, puissent rapporter des fractions.
	wheelDelta = 120

	// tmeLeave demande à être prévenu quand le pointeur quitte la fenêtre.
	tmeLeave = 0x0002
)

// trackMouseEvent décrit le suivi qu'on demande au système.
//
// La structure dit sa propre taille, ce que l'interface Win32 emploie pour distinguer
// ses versions. L'oublier fait échouer l'appel sans rien dire d'utile.
type trackMouseEvent struct {
	size      uint32
	flags     uint32
	track     uintptr
	hoverTime uint32
}

// pointerMessage traduit un message de pointeur, et dit s'il en était un.
//
// **Le suivi se réarme à chaque mouvement, et ce n'est pas un oubli de conception.**
// Windows ne prévient de la sortie qu'une fois : la demande se consomme avec
// l'événement qu'elle produit. Sans le réarmement, le premier retour du pointeur dans
// la fenêtre ne serait plus jamais suivi d'une sortie, et un jeu croirait le curseur
// chez lui pour toujours.
func pointerMessage(msg uint32, hwnd, wParam, lParam uintptr, state *input.State) bool {
	switch msg {
	case wmMouseMove:
		state.Move(pointerX(lParam), pointerY(lParam))
		track := trackMouseEvent{
			size:  uint32(unsafe.Sizeof(trackMouseEvent{})),
			flags: tmeLeave,
			track: hwnd,
		}
		trackMouseEventProc.Call(uintptr(unsafe.Pointer(&track)))
		return true

	case wmMouseLeave:
		// La position reste celle du dernier mouvement : elle dit par où le pointeur
		// est sorti, et une remise à zéro désignerait le coin de l'image.
		state.Leave()
		return true

	case wmLButtonDown:
		state.PressButton(input.ButtonLeft)
		return true
	case wmLButtonUp:
		state.ReleaseButton(input.ButtonLeft)
		return true

	case wmRButtonDown:
		state.PressButton(input.ButtonRight)
		return true
	case wmRButtonUp:
		state.ReleaseButton(input.ButtonRight)
		return true

	case wmMButtonDown:
		state.PressButton(input.ButtonMiddle)
		return true
	case wmMButtonUp:
		state.ReleaseButton(input.ButtonMiddle)
		return true

	case wmXButtonDown:
		if b, ok := extraButton(wParam); ok {
			state.PressButton(b)
		}
		return true
	case wmXButtonUp:
		if b, ok := extraButton(wParam); ok {
			state.ReleaseButton(b)
		}
		return true

	case wmMouseWheel:
		state.Wheel(int(int16(wParam>>16)) / wheelDelta)
		return true
	}
	return false
}

// pointerX et pointerY lisent la position que porte lParam.
//
// Les deux mots sont signés : un bouton enfoncé dans la fenêtre continue d'être suivi
// quand le pointeur en sort, et les coordonnées passent alors sous zéro. Les lire non
// signés les ferait sauter à soixante-cinq mille.
func pointerX(lParam uintptr) int16 { return int16(uint16(lParam)) }
func pointerY(lParam uintptr) int16 { return int16(uint16(lParam >> 16)) }

// extraButton dit lequel des deux boutons latéraux un message porte.
func extraButton(wParam uintptr) (input.Button, bool) {
	switch uint16(wParam >> 16) {
	case xButtonBack:
		return input.ButtonBack, true
	case xButtonForward:
		return input.ButtonForward, true
	}
	return 0, false
}
