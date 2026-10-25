// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import "errors"

// Open monte une fenêtre et rend de quoi la piloter.
//
// Le corps arrive au lot suivant, qui l'écrit par appels directs à user32 et gdi32.
// Le contrat, lui, existe dès maintenant : c'est lui que la boucle consomme, et le
// fixer d'abord évite de le dériver de la première plateforme écrite.
func Open(o Options) (Window, error) {
	return nil, errors.New("not implemented yet: milestone 2")
}
