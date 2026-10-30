// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le plus petit jeu qui tienne : une tuile de sol, et la méthode unique qu'un jeu
// écrit. Il mesure un critère permanent du moteur — un hello world en moins de trente
// lignes —, donc le garder court n'est pas de la coquetterie.
package main

import (
	"fmt"

	"github.com/sprimault/lozengine"
)

// game est tout ce qu'un jeu écrit : un type, et une méthode que le moteur appelle une
// fois par image.
type game struct{}

// Step avance le jeu d'un pas. Rien ici, et c'est le propos : la scène est décrite une
// fois pour toutes, le moteur décidant ensuite de l'ordre de dessin et de l'échelle.
func (game) Step(in lozengine.Input) {}

// main monte la scène, la peuple une fois, et laisse la boucle au moteur. C'est le
// hello world du dépôt, et sa longueur est un critère permanent de la feuille de route.
func main() {
	scene, err := lozengine.NewScene(8, 8, lozengine.Isometric(32), lozengine.SceneOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	scene.PlaceCell(4, 4, 0, lozengine.Aspect{Width: 32, Height: 16, Tint: lozengine.White()})

	sheet := lozengine.NewSheet(32, 16)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = lozengine.White()
	}

	var atlas lozengine.Atlas
	if _, err := atlas.Add(sheet); err != nil {
		fmt.Println(err)
		return
	}

	if err := lozengine.Run(scene, game{}, lozengine.Options{Title: "lozengine", Atlas: &atlas}); err != nil {
		fmt.Println(err)
	}
}
