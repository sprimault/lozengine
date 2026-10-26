// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les exemples que la documentation montre et que la suite exécute. Ils sont dans le
// paquet de test externe, donc ils n'ont droit qu'à ce qu'un jeu a : un seul import.

package lozengine_test

import (
	"fmt"
	"log"

	"github.com/sprimault/lozengine"
)

// Example monte la seconde porte d'entrée, celle d'un hôte qui garde sa boucle : une
// scène, un atlas, une étape appelée à la demande, et l'image qu'elle rend.
//
// Aucune fenêtre n'est ouverte et aucune horloge n'est lue — c'est exactement ce que
// l'étape ne fait pas. L'appelant décide quand appeler, et ce qu'il fait de l'image.
func Example() {
	sheet := lozengine.NewSheet(2, 2)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = lozengine.White()
	}

	atlas := &lozengine.Atlas{}
	index, err := atlas.Add(sheet)
	if err != nil {
		log.Fatal(err)
	}

	world, err := lozengine.NewScene(8, 8, lozengine.Dimetric(32, 16), lozengine.SceneOptions{Entities: 1})
	if err != nil {
		log.Fatal(err)
	}
	world.Place(lozengine.Entity{
		X: 4.5, Y: 4.5,
		Aspect: lozengine.Aspect{Sheet: index, Width: 2, Height: 2, Tint: lozengine.White()},
	})

	engine, err := lozengine.NewEngine(world, lozengine.Options{Atlas: atlas, Width: 64, Height: 32})
	if err != nil {
		log.Fatal(err)
	}

	// Un jeu nul : la scène est dessinée telle qu'elle est, sans qu'aucun pas
	// l'avance.
	engine.Step(nil, lozengine.Input{})

	frame := engine.Frame()
	painted := 0
	for _, p := range frame.Pixels {
		if p != (lozengine.Color{A: 255}) {
			painted++
		}
	}
	fmt.Println(frame.Width, frame.Height, painted)
	// Output: 64 32 4
}

// ExampleState éprouve la méthode d'un jeu sans ouvrir de fenêtre, en fabriquant les
// entrées que la boucle aurait données.
//
// Les deux régimes ne disent pas la même chose : un appui est tenu tant que la touche
// l'est, et il n'est neuf qu'une image.
func ExampleState() {
	var keys lozengine.State

	keys.Press(lozengine.KeyRight)
	first := keys.Frame()
	second := keys.Frame()

	fmt.Println(first.Held(lozengine.KeyRight), first.JustPressed(lozengine.KeyRight))
	fmt.Println(second.Held(lozengine.KeyRight), second.JustPressed(lozengine.KeyRight))
	// Output:
	// true true
	// true false
}
