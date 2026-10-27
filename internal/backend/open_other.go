// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !windows

package backend

import (
	"errors"
	"runtime"
)

// Open rend une erreur nommant ce qui manque, sur toute plateforme dont le backend
// n'est pas encore écrit.
//
// Un repli et non une absence de fonction : le moteur compile alors pour les huit
// couples annoncés, et la suite de tests tourne partout. Ce qui ne marche pas le dit
// à l'exécution, en nommant le jalon qui le fera marcher, plutôt que d'échouer à la
// compilation sur un symbole introuvable.
//
// Le jalon 3 écrit le client X11 pour Linux et les BSD, le 5 le canvas du navigateur,
// le 6 Wayland et DRM/KMS. macOS, iOS et Android natifs restent hors périmètre, et le
// navigateur est ce qui les couvre — le message ne promet donc aucune des deux familles
// à la plateforme qui le lit, puisque certaines n'en auront jamais.
//
// **La contrainte de construction se resserre à chaque backend livré, et le message
// avec elle.** Le jalon qui écrit X11 doit retirer `linux` de ce repli, sans quoi deux
// Open se disputeraient la plateforme et la compilation tomberait ; et le texte qui
// nomme un jalon déjà livré est à reprendre au même moment, faute de quoi il survit
// ailleurs en disant faux.
func Open(o Options) (Window, error) {
	return nil, errors.New("no window on " + runtime.GOOS + ": no backend is written for it yet; X11 comes at milestone 3, the browser at milestone 5")
}
