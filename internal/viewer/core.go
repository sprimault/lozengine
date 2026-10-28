// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le seul scénario qui passe par le noyau entier — grille, décor, entités, caméra,
// puis tri — là où les autres posent leurs quads à la main. C'est lui qui éprouve
// l'ordre que le moteur décide, et non celui qu'un tableau lui dicte.

package viewer

import (
	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario, et leurs tailles. Trois formes distinctes pour que l'ordre
// de dessin se lise : un losange de sol, une borne haute qui dépasse sur la cellule
// derrière elle, et une figure étroite qui se glisse entre les deux.
const (
	lightGroundSheet = iota
	darkGroundSheet
	pillarSheet
	figureSheet
)

// La grille du scénario et la vue qui la montre entière.
//
// Les dimensions se dérivent de la tuile plutôt que d'être posées : huit cellules de
// côté à tuile 32×16 couvrent 256 pixels de large et 128 de haut, auxquels s'ajoutent
// la hauteur des bornes et la marge d'un étage.
const (
	gridSide = 8
	tileW    = 32
	tileH    = 16

	coreWidth  = gridSide*tileW + tileW
	coreHeight = gridSide*tileH + 4*tileH
)

// Core rend le scénario qui éprouve le chemin complet du moteur : une scène décrite,
// ordonnée par le tri, puis dessinée.
//
// C'est le seul scénario qui ne pose pas ses quads à la main. Les autres vérifient le
// rastériseur en lui donnant exactement ce qu'il doit peindre ; celui-ci vérifie que
// **le moteur décide juste** — l'ordre de dessin, le retrait de ce qui sort de l'image,
// le placement d'un point du monde sur un pixel. Un quad mal placé par le tri s'y voit,
// là où aucun des autres ne le verrait.
//
// Ses quatre images sont les quatre quarts de tour de la même scène. C'est la seule
// chose qui garde les orientations : aucun consommateur réel n'en emploie, donc rien
// d'autre ne les exercerait.
//
// **Le sol s'y lit en briques et non en losanges, et c'est voulu.** Ses planches sont
// des rectangles pleins, là où un vrai sol isométrique serait un losange transparent
// aux coins. Deux losanges jointifs ne se recouvrent pas, donc leur ordre ne décide de
// rien et une image ne montrerait pas qu'il a changé ; deux rectangles se chevauchent,
// et le bord que l'un mange à l'autre dit lequel est passé devant. Ce scénario garde
// l'ordre, pas l'apparence d'un décor.
func Core() Scenario {
	return Scenario{
		Name:       "core",
		Width:      coreWidth,
		Height:     coreHeight,
		Frames:     4,
		Background: render.Color{R: 18, G: 18, B: 24, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			var atlas raster.Atlas
			sheets := []raster.Sheet{
				borderedSheet(tileW, tileH, render.Color{R: 96, G: 112, B: 88, A: 255}),
				borderedSheet(tileW, tileH, render.Color{R: 64, G: 80, B: 64, A: 255}),
				borderedSheet(tileW/2, tileH*2, render.Color{R: 176, G: 144, B: 96, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, render.Color{R: 208, G: 96, B: 96, A: 255}),
			}
			for _, p := range sheets {
				if _, err := atlas.Add(p); err != nil {
					return nil, err
				}
			}
			return &atlas, nil
		},

		Quads: func(frame int, _ InputLine, reserve render.DrawList) render.DrawList {
			s := coreScene()
			s.Orient(scene.Orientation(frame))

			return append(reserve, sorting.New(s).Frame()...)
		},
	}
}

// coreScene monte la scène à neuf, ce que la pureté exigée d'un scénario demande :
// une image doit pouvoir se rendre seule, deux fois, et donner les mêmes pixels.
func coreScene() *scene.Scene {
	s, err := scene.New(gridSide, gridSide, geometry.Dimetric(tileW, tileH), scene.Options{Entities: 8})
	if err != nil {
		panic(err)
	}
	s.Focus(gridSide/2, gridSide/2, coreWidth, coreHeight)

	// Le sol, en damier pour que chaque cellule se distingue de ses voisines : un
	// décalage d'une cellule à la projection se verrait, là où un sol uni le cacherait.
	for v := range gridSide {
		for u := range gridSide {
			sheet := uint16(lightGroundSheet)
			if (u+v)%2 == 1 {
				sheet = darkGroundSheet
			}
			s.PlaceCell(u, v, 0, scene.Aspect{
				Sheet: sheet, Width: tileW, Height: tileH, Tint: render.White(),
			})
		}
	}

	// Trois bornes en strate 1, posées sur le sol sans l'effacer. Leur disposition est
	// asymétrique pour qu'un quart de tour se voie : en ligne, une rotation de cent
	// quatre-vingts degrés rendrait la même image.
	pillar := scene.Aspect{
		Sheet: pillarSheet, Width: tileW / 2, Height: tileH * 2, Tint: render.White(),
	}
	for _, c := range [][2]int{{1, 1}, {2, 1}, {1, 5}} {
		s.PlaceCell(c[0], c[1], 1, pillar)
	}

	figure := scene.Aspect{
		Sheet: figureSheet, Width: tileW / 3, Height: tileH * 3 / 2, Tint: render.White(),
	}

	// Une figure derrière les bornes, une devant : c'est le couple qui montre que
	// l'ordre suit la profondeur et non l'ordre de pose.
	s.Place(scene.Entity{X: 1.5, Y: 0.5, Aspect: figure})
	s.Place(scene.Entity{X: 1.5, Y: 2.5, Aspect: figure})

	// À cheval sur quatre cellules, ce qu'une position entière ne montrerait pas.
	s.Place(scene.Entity{X: 3.5, Y: 3.5, Aspect: figure})

	// Un étage au-dessus, qui passe devant tout l'étage du sol quelle que soit sa
	// profondeur, et que son étage monte à l'écran. Pas de z : il porterait la même
	// hauteur une seconde fois, et l'entité monterait de deux crans.
	tall := figure
	tall.Floor = 1
	s.Place(scene.Entity{X: 5.5, Y: 5.5, Aspect: tall})

	return s
}
