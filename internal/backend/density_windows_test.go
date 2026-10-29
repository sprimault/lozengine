// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La traduction d'une densité d'écran en facteur entier, éprouvée sur les échelles que
// Windows propose plutôt que sur celle de la machine qui exécute le test.

package backend

import "testing"

// TestFactorForDPICoversEveryWindowsScale garde le facteur sur toutes les échelles que
// le système offre, et c'est le seul cas à le pouvoir.
//
// **Aucune machine du parc ne dépasse 125 %**, donc le chemin corrigé y rend le même
// facteur qu'avant ce lot et ne s'y distingue de rien. Ce cas est ce qui tient les
// autres échelles, et notamment les deux qui décident : 150 %, où l'arrondi va vers le
// haut plutôt que vers le bas, et 100 %, où le facteur doit valoir un pour qu'un jeu
// existant ne bouge pas d'un pixel.
//
// La borne basse est là pour un système qui rendrait zéro : le facteur ne descend pas
// sous un, sans quoi la fenêtre s'ouvrirait vide.
func TestFactorForDPICoversEveryWindowsScale(t *testing.T) {
	for _, c := range []struct {
		dpi   int
		scale string
		want  int
	}{
		{0, "système muet", 1},
		{96, "100 %", 1},
		{120, "125 %", 1},
		{144, "150 %", 2},
		{168, "175 %", 2},
		{192, "200 %", 2},
		{240, "250 %", 3},
		{288, "300 %", 3},
		{384, "400 %", 4},
	} {
		t.Run(c.scale, func(t *testing.T) {
			if got := factorForDPI(c.dpi); got != c.want {
				t.Errorf("%d dpi donne le facteur %d, attendu %d", c.dpi, got, c.want)
			}
		})
	}
}

// TestDensityFactorIsUsableOnThisMachine vérifie que l'appel système répond et que son
// résultat est exploitable, ce qu'un cas sur l'arithmétique seule ne dit pas.
//
// Il ne compare à aucune valeur attendue : la densité dépend de l'écran, et le poste
// qui a écrit ce lot en porte trois à deux densités différentes. Ce qu'il garde est que
// la procédure se trouve, que l'appel ne panique pas, et que le facteur reste dans ce
// qu'une fenêtre peut porter.
func TestDensityFactorIsUsableOnThisMachine(t *testing.T) {
	got := densityFactor()
	if got < 1 || got > 8 {
		t.Errorf("facteur de densité %d, attendu entre 1 et 8", got)
	}
	t.Logf("densité de cet écran : facteur %d", got)
}
