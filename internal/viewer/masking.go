// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario de ce qu'un masquant déclare être : deux murs posés de la même façon, dont
// un seul se dit décor, et les deux teintes de révélation qui en découlent.

package viewer

import (
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario, les mêmes que celles de la révélation : un sol, un mur
// opaque, une figure.
const (
	maskingGroundSheet = iota
	maskingWallSheet
	maskingFigureSheet
)

// La grille et la vue. Six cellules de côté, et de la marge en haut pour des murs de
// trois hauteurs de tuile.
const (
	maskingSide = 6

	maskingWidth  = maskingSide*tileW + tileW
	maskingHeight = maskingSide*tileH + 5*tileH

	// Le seuil de révélation. Chaque mur cache sa figure bien au-delà, de sorte qu'un
	// déplacement d'une cellule ne fasse pas basculer le cas.
	maskingAfter = 100
)

// Masking éprouve que la nature déclarée d'un masquant décide de la teinte de révélation,
// là où sa voie de pose ne le peut pas.
//
// **Les deux murs sont posés en entités, à la même distance, avec la même image.** Seul
// leur champ `Masks` diffère : celui de gauche se déclare décor, celui de droite laisse sa
// voie de pose décider. La bascule de teinte employant le recouvrement que le décor
// **seul** atteindrait, les deux figures se redessinent sous deux teintes différentes.
//
// Ce que l'image montre, et ce qu'elle montrerait sans le lot : deux figures révélées, la
// gauche sous la teinte réservée au décor et la droite sous la teinte ordinaire. Sans la
// déclaration, les deux seraient identiques — un mur posé en entité ne comptait jamais
// comme décor, et aucune image du dépôt n'aurait dit que la distinction était fonction de
// la voie de pose plutôt que de la nature.
//
// Ce que ce scénario ne garde pas : le compte du recouvrement, ni la propagation jusqu'à
// la mesure au pixel. Les cas du noyau de tri gardent le premier, et un cas du paquet
// racine le second — il lui faut un atlas, que le noyau de tri ne monte pas.
func Masking() Scenario {
	return Scenario{
		Name:       "masking",
		Width:      maskingWidth,
		Height:     maskingHeight,
		Frames:     1,
		Background: render.Color{R: 16, G: 18, B: 26, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(
				borderedSheet(tileW, tileH, render.Color{R: 72, G: 88, B: 72, A: 255}),
				borderedSheet(tileW, tileH*3, render.Color{R: 150, G: 140, B: 128, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, render.Color{R: 224, G: 96, B: 96, A: 255}),
			)
		},

		Quads: func(_ int, _ InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, sorting.New(maskingScene()).Frame()...)
		},
	}
}

// maskingScene monte le sol, les deux figures et les deux murs qui les cachent.
func maskingScene() *scene.Scene {
	s := sceneOf(maskingSide, maskingWidth, maskingHeight, scene.Options{
		Entities: 8,
		Revealing: scene.RevealOptions{
			After: maskingAfter,

			// Deux teintes franchement distinctes : l'image doit dire laquelle s'est
			// appliquée, et deux nuances voisines laisseraient la comparaison pixel à
			// pixel trancher là où l'œil ne verrait rien.
			Tint:        render.Color{R: 224, G: 224, B: 96, A: 255},
			TintByDecor: render.Color{R: 96, G: 128, B: 224, A: 255},
		},
	})

	fillGround(s, maskingSide, scene.Aspect{
		Sheet: maskingGroundSheet, Width: tileW, Height: tileH, Tint: render.White(),
	})

	wall := scene.Aspect{
		Sheet: maskingWallSheet, Width: tileW, Height: tileH * 3, Tint: render.White(),
	}
	figure := scene.Aspect{
		Sheet: maskingFigureSheet, Width: tileW / 3, Height: tileH * 3 / 2,
		Tint: render.White(), Reveal: 1,
	}

	// Chaque mur est plus proche que sa figure d'une cellule, donc il se peint après et
	// la recouvre. Les deux couples sont séparés d'une cellule vide pour qu'aucun mur ne
	// morde sur la figure de l'autre.
	declared := wall
	declared.Masks = scene.MaskAsDecor

	s.Place(scene.Entity{X: 1.5, Y: 1.5, Aspect: figure})
	s.Place(scene.Entity{X: 1.5, Y: 2.5, Aspect: declared})

	s.Place(scene.Entity{X: 4.5, Y: 1.5, Aspect: figure})
	s.Place(scene.Entity{X: 4.5, Y: 2.5, Aspect: wall})

	return s
}
