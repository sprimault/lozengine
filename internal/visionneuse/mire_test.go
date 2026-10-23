// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import "testing"

// TestMireSansVoile éprouve la seule directive d'entrée que la mire interprète, et ce
// test est le seul endroit où elle l'est : aucune image de référence ne l'emploie,
// donc la retirer du scénario ne faisait rougir personne.
func TestMireSansVoile(t *testing.T) {
	rejeu, err := NouveauRejeu(Mire())
	if err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	avec := copiePixels(rejeu.Image(0, ""))
	sans := rejeu.Image(0, "sans-voile")

	if memesPixels(avec, sans) {
		t.Fatal("la directive ne change rien à l'image")
	}

	// Le voile est un aplat sombre à alpha partiel : le retirer ne peut qu'éclaircir,
	// jamais assombrir un seul pixel. C'est ce qui distingue ce cas d'une image
	// simplement différente.
	for i := range avec.Pixels {
		if sans.Pixels[i].R < avec.Pixels[i].R {
			t.Fatalf("pixel %d plus sombre sans le voile : %v au lieu de %v",
				i, sans.Pixels[i], avec.Pixels[i])
		}
	}
}
