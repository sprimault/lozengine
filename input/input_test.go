// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la valeur zéro rend, et ce qu'une touche hors plage rend plutôt que de
// déborder de la table.

package input

import "testing"

// TestZeroValueHoldsNothing vérifie l'entrée d'un pas où rien n'est arrivé : c'est ce
// que reçoit un jeu avant le premier appui, et ce que la suite de non-régression passe
// à un scénario qui ne simule aucune touche.
func TestZeroValueHoldsNothing(t *testing.T) {
	var empty Input

	if empty.Held(keyA) || empty.JustPressed(keyA) {
		t.Error("la valeur zéro tient une touche")
	}
}

// TestOutOfRangeReturnsFalse vérifie qu'une touche qu'aucun clavier ne porte se lit
// sans paniquer.
//
// C'est une donnée du monde extérieur et non une faute de programmation : un clavier
// exotique ou un pilote bavard peut rapporter un code qu'on ne connaît pas, et le
// moteur n'a pas à s'arrêter pour cela. Le cas couvre la borne exacte et ce qu'un
// uint16 porte de plus grand.
func TestOutOfRangeReturnsFalse(t *testing.T) {
	var state State
	cases := []struct {
		name string
		key  Key
	}{
		{"la borne elle-même", keysMax},
		{"au-delà de la borne", keysMax + 1},
		{"le plus grand code possible", 65535},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state.Press(c.key)
			state.Release(c.key)

			frame := state.Frame()
			if frame.Held(c.key) || frame.JustPressed(c.key) {
				t.Error("une touche hors bornes se lit comme tenue")
			}
		})
	}
}

// TestLastKeyInRange vérifie que la borne n'est pas mangée d'une unité : la dernière
// place utilisable doit s'enfoncer et se lire.
//
// Un hors-par-un ici retirerait une touche que le moteur annonce distinguer, et aucun
// autre cas ne le verrait — les touches d'un clavier ordinaire sont loin de la borne.
func TestLastKeyInRange(t *testing.T) {
	last := Key(keysMax - 1)

	var state State
	state.Press(last)

	frame := state.Frame()
	if !frame.Held(last) || !frame.JustPressed(last) {
		t.Errorf("la touche %d ne se lit pas : tenue %v, montée %v",
			last, frame.Held(last), frame.JustPressed(last))
	}
}
