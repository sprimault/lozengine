// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !windows

package backend

import (
	"errors"
	"runtime"
)

// Ouvrir rend une erreur nommant ce qui manque, sur toute plateforme dont le backend
// n'est pas encore écrit.
//
// Un repli et non une absence de fonction : le moteur compile alors pour les huit
// couples annoncés, et la suite de tests tourne partout. Ce qui ne marche pas le dit
// à l'exécution, en nommant le jalon qui le fera marcher, plutôt que d'échouer à la
// compilation sur un symbole introuvable.
//
// Le jalon 3 écrit le client X11 pour Linux et les BSD, le 5 le canvas du navigateur,
// le 6 Wayland et DRM/KMS. macOS, iOS et Android natifs restent hors périmètre, et le
// navigateur est ce qui les couvre.
func Ouvrir(o Options) (Fenetre, error) {
	return nil, errors.New("aucune fenêtre sur " + runtime.GOOS + " : X11 vient au jalon 3, le navigateur au jalon 5")
}
