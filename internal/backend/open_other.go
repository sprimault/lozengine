// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !windows && !linux

package backend

import (
	"errors"
	"runtime"
)

// Open rend une erreur nommant ce qui manque, sur toute plateforme dont le backend
// n'est pas encore écrit.
//
// Un repli et non une absence de fonction : le moteur compile alors pour tous les
// couples que `CIBLES` énumère, et la suite de tests tourne partout. Ce qui ne marche pas le dit
// à l'exécution, en nommant le jalon qui le fera marcher, plutôt que d'échouer à la
// compilation sur un symbole introuvable.
//
// Le jalon 5 écrit le canvas du navigateur, le 6 Wayland et DRM/KMS. macOS, iOS et
// Android natifs restent hors périmètre, et le navigateur est ce qui les couvre — le
// message ne promet donc aucune des deux familles à la plateforme qui le lit, puisque
// certaines n'en auront jamais.
//
// **La contrainte de construction se resserre à chaque backend livré, et le message
// avec elle.** Le jalon 3 a retiré `linux` de ce repli, sans quoi deux Open se
// disputeraient la plateforme et la compilation tomberait ; et le texte qui nommait ce
// jalon comme à venir est parti du même geste, faute de quoi il aurait survécu ici en
// disant faux.
//
// **Les BSD parlent le même protocole X11 et restent pourtant ici**, ce qui est un
// choix et non un oubli : le client est écrit, mais aucune machine du parc ne permet de
// l'y voir tourner, et annoncer une plateforme sur la seule foi d'une compilation est
// précisément ce que les README refusent de faire. La contrainte s'élargira le jour où
// il y aura une machine pour la vérifier.
func Open(o Options) (Window, error) {
	return nil, errors.New("no window on " + runtime.GOOS + ": no backend is written for it yet; the browser comes at milestone 5, Wayland and DRM/KMS at milestone 6")
}
