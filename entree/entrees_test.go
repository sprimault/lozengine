// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package entree

import "testing"

// TestValeurZeroNeTientRien vérifie l'entrée d'un pas où rien n'est arrivé : c'est ce
// que reçoit un jeu avant le premier appui, et ce que la suite de non-régression passe
// à un scénario qui ne simule aucune touche.
func TestValeurZeroNeTientRien(t *testing.T) {
	var vide Entrees

	if vide.Enfoncee(toucheA) || vide.Pressee(toucheA) {
		t.Error("la valeur zéro tient une touche")
	}
}

// TestHorsBornesRendFaux vérifie qu'une touche qu'aucun clavier ne porte se lit sans
// paniquer.
//
// C'est une donnée du monde extérieur et non une faute de programmation : un clavier
// exotique ou un pilote bavard peut rapporter un code qu'on ne connaît pas, et le
// moteur n'a pas à s'arrêter pour cela. Le cas couvre la borne exacte et ce qu'un
// uint16 porte de plus grand.
func TestHorsBornesRendFaux(t *testing.T) {
	var etat Etat
	cas := []struct {
		nom    string
		touche Touche
	}{
		{"la borne elle-même", touchesMax},
		{"au-delà de la borne", touchesMax + 1},
		{"le plus grand code possible", 65535},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			etat.Enfoncer(c.touche)
			etat.Relacher(c.touche)

			image := etat.Image()
			if image.Enfoncee(c.touche) || image.Pressee(c.touche) {
				t.Error("une touche hors bornes se lit comme tenue")
			}
		})
	}
}

// TestDerniereToucheDeLaBorne vérifie que la borne n'est pas mangée d'une unité : la
// dernière place utilisable doit s'enfoncer et se lire.
//
// Un hors-par-un ici retirerait une touche que le moteur annonce distinguer, et aucun
// autre cas ne le verrait — les touches d'un clavier ordinaire sont loin de la borne.
func TestDerniereToucheDeLaBorne(t *testing.T) {
	derniere := Touche(touchesMax - 1)

	var etat Etat
	etat.Enfoncer(derniere)

	image := etat.Image()
	if !image.Enfoncee(derniere) || !image.Pressee(derniere) {
		t.Errorf("la touche %d ne se lit pas : tenue %v, montée %v",
			derniere, image.Enfoncee(derniere), image.Pressee(derniere))
	}
}
