// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario de ce qu'une entité foule : une figure posée ailleurs qu'au centre de sa
// cellule, et le sol de cette cellule, sur la même couche et la même strate.

package viewer

import (
	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario. Le sol est clair et la figure sombre, pour que le bord que
// l'un mange à l'autre se lise sans compter les pixels.
const (
	groundTileSheet = iota
	groundFigureSheet
)

// La grille et la vue. Quatre cellules de côté suffisent : le sujet est une cellule et
// ce qui s'y tient, pas une étendue.
const (
	groundSide = 4

	groundWidth  = groundSide*tileW + tileW
	groundHeight = groundSide*tileH + 3*tileH
)

// Ground éprouve qu'une entité passe devant la cellule qu'elle foule, où qu'elle se
// place dedans.
//
// **C'est le seul scénario à pouvoir faire échouer cette entrée**, et il existe parce
// qu'aucun autre ne la couvrait : les quatre figures de [Core] sont toutes à des
// centres de cellule exacts, et les autres scénarios posent leurs quads à la main sans
// passer par le tri. Le régime qu'il garde est celui d'une entité **hors centre**, sur
// la même couche et la même strate que son sol — c'est-à-dire ce qu'un jeu écrit quand
// il ne sait pas qu'il devrait séparer les deux.
//
// Ce que le défaut donnait, et que l'image montrait : la profondeur du décor venait du
// centre de sa cellule, celle d'une entité de sa position brute. Une figure dans la
// moitié nord d'une case avait donc une profondeur **plus petite** que la case, était
// peinte avant, et le sol lui passait sur les pieds. La correction donne au décor la
// profondeur de son coin le plus lointain, de sorte qu'il soit derrière par
// construction.
//
// Les cinq positions couvrent la cellule dans les deux sens : le nord, où le défaut
// se voyait, le centre, où c'était l'ordre des planches qui tranchait, et le sud, qui
// était déjà juste et qui doit le rester.
func Ground() Scenario {
	return Scenario{
		Name:       "ground",
		Width:      groundWidth,
		Height:     groundHeight,
		Frames:     1,
		Background: render.Color{R: 24, G: 24, B: 32, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			var atlas raster.Atlas
			for _, p := range []raster.Sheet{
				borderedSheet(tileW, tileH, render.Color{R: 190, G: 185, B: 170, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, render.Color{R: 60, G: 80, B: 150, A: 255}),
			} {
				if _, err := atlas.Add(p); err != nil {
					return nil, err
				}
			}
			return &atlas, nil
		},

		Quads: func(_ int, _ InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, sorting.New(groundScene()).Frame()...)
		},
	}
}

// groundScene monte la grille, son sol, et cinq figures réparties du nord au sud de
// leur cellule.
//
// Montée à neuf à chaque image, comme celle de [Core] : une image doit pouvoir se
// rendre seule, deux fois, et donner les mêmes pixels.
func groundScene() *scene.Scene {
	s, err := scene.New(groundSide, groundSide, geometry.Dimetric(tileW, tileH),
		scene.Options{Entities: 8})
	if err != nil {
		panic(err)
	}
	s.Focus(groundSide/2, groundSide/2, groundWidth, groundHeight)

	// Le sol entier, sur la strate 0 : c'est la configuration qu'un jeu obtient sans
	// rien régler, et donc celle où le défaut se produisait.
	tile := scene.Aspect{
		Sheet: groundTileSheet, Width: tileW, Height: tileH, Tint: render.White(),
	}
	for v := range groundSide {
		for u := range groundSide {
			s.PlaceCell(u, v, 0, tile)
		}
	}

	// Cinq figures sur la même rangée, de la plus au nord à la plus au sud de leur
	// cellule. La couche et la strate sont celles du sol — rien ne les sépare, et c'est
	// le point.
	figure := scene.Aspect{
		Sheet: groundFigureSheet, Width: tileW / 3, Height: tileH * 3 / 2,
		Tint: render.White(),
	}
	for i, offset := range []float64{0.05, 0.25, 0.5, 0.75, 0.95} {
		s.Place(scene.Entity{
			X:      float64(i) + 0.5,
			Y:      float64(i) + offset,
			Aspect: figure,
		})
	}

	return s
}
