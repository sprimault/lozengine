// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !windows && !linux && !js

package backend

import (
	"runtime"
	"strings"
	"testing"
)

// TestOpenFallsBackWithoutWindow vérifie que l'ouverture échoue proprement, sans rendre
// de fenêtre inutilisable.
//
// **Ce cas ne tourne nulle part, et c'est dit plutôt que supposé** : sa contrainte de
// construction est celle de son fichier, `!windows && !linux && !js`, et aucune cible du
// dépôt n'exécute un tel `GOOS`. Il ne garde donc que le câblage — qu'`Open` appelle bien
// `unsupported` et ne rende rien d'autre —, le jour où quelqu'un compilera pour une de
// ces plateformes.
//
// **La phrase a été fausse, et c'est ce qui dit quand la relire** : la suite de référence
// est passée sous `js/wasm` avant que ce fichier ne retire `js`, donc ce cas y a tourné
// pendant deux lots. Une contrainte négative se périme à chaque cible ajoutée, et pas
// seulement quand on la modifie.
//
// Ce qui est réellement gardé, sur l'hôte, est le message lui-même :
// [TestUnsupportedNamesThePlatformAndTheMilestone], dans un fichier sans contrainte.
func TestOpenFallsBackWithoutWindow(t *testing.T) {
	window, err := Open(Options{Title: "essai", Width: 480, Height: 270})

	if err == nil {
		t.Fatal("ouverture acceptée sur une plateforme sans backend")
	}
	if window != nil {
		t.Error("une fenêtre est rendue en même temps que l'erreur")
	}
	if !strings.Contains(err.Error(), runtime.GOOS) {
		t.Errorf("le message ne nomme pas la plateforme : %v", err)
	}
}
