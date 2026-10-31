// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le repli des plateformes sans backend, éprouvé sur l'hôte plutôt que sur un GOOS
// qu'aucune cible n'exécute.

package backend

import (
	"strings"
	"testing"
)

// TestUnsupportedNamesThePlatformAndTheMilestone garde le seul comportement qu'une
// plateforme sans backend livre : une erreur qui dit où l'on est et ce qu'il faut
// attendre.
//
// **Il tourne sur l'hôte, et c'est la correction qu'il porte.** Le cas équivalent vivait
// sous `!windows && !linux`, donc aucune cible du dépôt ne l'exécutait — ni `make test`,
// qui tourne sur l'hôte, ni les trois runners, ni `make cross`, qui ignore les fichiers
// de test. Les quatre assertions ne tournaient nulle part, et un repli muet aurait passé
// tous les contrôles.
//
// Deux plateformes nommées, dont une qui n'aura jamais de fenêtre native : le message ne
// doit promettre ni le navigateur ni Wayland à celle qui le lit, seulement dire quels
// jalons les écrivent.
func TestUnsupportedNamesThePlatformAndTheMilestone(t *testing.T) {
	for _, goos := range []string{"darwin", "freebsd"} {
		t.Run(goos, func(t *testing.T) {
			err := unsupported(goos)
			if err == nil {
				t.Fatal("aucune erreur pour une plateforme sans backend")
			}
			if !strings.Contains(err.Error(), goos) {
				t.Errorf("le message ne nomme pas la plateforme : %v", err)
			}
			if !strings.Contains(err.Error(), "milestone") {
				t.Errorf("le message ne nomme pas le jalon qui l'écrira : %v", err)
			}
		})
	}
}
