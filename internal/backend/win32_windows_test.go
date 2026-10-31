// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le backend emprunte à Windows, éprouvé sans fenêtre : la recherche d'une
// procédure, qui décide si l'ouverture se replie, refuse, ou plante.

package backend

import "testing"

// TestProcAddrRefusesAnAbsentProcedure vérifie que la recherche a bien lieu avant
// qu'on demande l'adresse.
//
// **Le cas rougit en paniquant, et non en échouant** : qui retirerait la recherche du
// corps de [procAddr] laisserait `LazyProc.Addr` appeler `mustFind` sur un nom
// introuvable, et la panique arrête le paquet. C'est le régime que le défaut
// produisait sur tout Windows antérieur à la version 1703 de Windows 10 — aucune
// machine du parc n'en porte un, donc un nom qui n'existe nulle part en tient lieu.
//
// La cible `procedures` garde l'autre moitié, celle qu'un cas ne peut pas atteindre :
// qu'aucune procédure n'échappe à cette garde ailleurs dans le backend.
//
// Ce qu'il ne garde pas, faute de pouvoir injecter la procédure dans `Open` : que
// l'ouverture **refuse** quand la présentation manque. Seule la lecture du code le
// dit.
func TestProcAddrRefusesAnAbsentProcedure(t *testing.T) {
	if _, err := procAddr(user32.NewProc("SetProcessDpiAwarenessContextLozengine")); err == nil {
		t.Error("une procédure que le système ne porte pas devrait rendre une erreur")
	}
}
