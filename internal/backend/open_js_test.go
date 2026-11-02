// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'ouverture refuse, éprouvé sous l'exécuteur de la suite : il porte un moteur
// JavaScript et aucune page, qui est exactement le régime que le refus vise.

package backend

import (
	"strconv"
	"strings"
	"testing"
)

// TestOpenRefusesARuntimeWithoutAPage garde le refus que la suite de référence est seule
// à pouvoir éprouver, et elle le fait à chaque lot.
//
// **C'est le régime de l'exécuteur de la suite** : un moteur JavaScript sans document ni
// canvas. Sans ce refus, l'ouverture appellerait une méthode sur la valeur indéfinie, ce
// qui lève une exception de la page — et un module Go ne la rattrape pas, donc le jeu
// s'arrêterait sans message plutôt qu'avec le sien.
//
// Il rougit si l'on retire le contrôle du document : le cas ne tombe pas sur une
// assertion mais sur la panique de l'appel.
func TestOpenRefusesARuntimeWithoutAPage(t *testing.T) {
	window, err := Open(Options{Title: "essai", Width: 480, Height: 270})

	if err == nil {
		t.Fatal("ouverture acceptée dans une exécution sans page")
	}
	if window != nil {
		t.Error("une fenêtre est rendue en même temps que l'erreur")
	}
	if !strings.Contains(err.Error(), "document") {
		t.Errorf("le message ne dit pas ce qui manque : %v", err)
	}
}

// TestCSSPixelsKeepsEveryDecimal garde ce que la taille CSS doit porter : la valeur que
// [canvasFor] a dérivée, sans arrondi.
//
// **Un arrondi ici déferait le rapport exact entre les deux unités**, qui est tout
// l'objet du calcul de densité : à densité fractionnaire la taille CSS a des décimales,
// et les perdre remettrait une mise à l'échelle fractionnaire dans la page. Le cas rougit
// dès qu'on demande une précision fixe au formatage.
func TestCSSPixelsKeepsEveryDecimal(t *testing.T) {
	for _, c := range []struct {
		name  string
		value float64
		want  string
	}{
		{"une taille entière", 480, "480px"},
		{"zéro, pour un canvas masqué", 0, "0px"},
		{"une densité de 125 %", 487 / 1.25, "389.6px"},
		{"une densité de 150 %", 585 / 1.5, "390px"},
		{"une densité de 262,5 %", 1023 / 2.625, strconv.FormatFloat(1023/2.625, 'f', -1, 64) + "px"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := cssPixels(c.value); got != c.want {
				t.Errorf("%g pixels CSS s'écrit %q, attendu %q", c.value, got, c.want)
			}
		})
	}
}
