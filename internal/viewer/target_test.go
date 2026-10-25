// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package viewer

import "testing"

// TestTargetWithoutVeil éprouve la seule directive d'entrée que la mire interprète, et ce
// test est le seul endroit où elle l'est : aucune image de référence ne l'emploie,
// donc la retirer du scénario ne faisait rougir personne.
func TestTargetWithoutVeil(t *testing.T) {
	replay, err := NewReplay(Target())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	with := copyPixels(replay.Frame(0, ""))
	without := replay.Frame(0, "sans-voile")

	if samePixels(with, without) {
		t.Fatal("la directive ne change rien à l'image")
	}

	// Le voile est un aplat sombre à alpha partiel : le retirer ne peut qu'éclaircir,
	// jamais assombrir un seul pixel. C'est ce qui distingue ce cas d'une image
	// simplement différente.
	for i := range with.Pixels {
		if without.Pixels[i].R < with.Pixels[i].R {
			t.Fatalf("pixel %d plus sombre sans le voile : %v au lieu de %v",
				i, without.Pixels[i], with.Pixels[i])
		}
	}
}
