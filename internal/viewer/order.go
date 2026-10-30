// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario de l'ordre de dessin et des miroirs : un escalier, deux quads
// exactement superposés, et la planche marquée dans ses quatre orientations. Partout
// ailleurs les quads se recouvrent trop peu pour qu'un ordre faux se voie.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Order éprouve l'ordre de dessin et les miroirs : trois quads en escalier, deux
// quads exactement superposés, et la planche marquée dessinée dans ses quatre
// orientations.
//
// L'ordre du tableau est la seule vérité du résultat, et c'est une vérité qu'aucune
// autre image ne vérifie : partout ailleurs les quads se recouvrent peu. Les miroirs
// sont là pour la même raison — un axe inversé donne une image plausible, donc
// invisible sans référence.
func Order() Scenario {
	const (
		width, height = 48, 32
		side          = 8
	)

	return Scenario{
		Name:       "order",
		Width:      width,
		Height:     height,
		Frames:     1,
		Background: render.Color{R: 24, G: 20, B: 20, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(markedSheet(side))
		},

		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			quads := reserve

			// Escalier : chacun recouvre le précédent d'un quart.
			for i, tint := range []render.Color{
				{R: 220, G: 60, B: 60, A: 255},
				{R: 60, G: 220, B: 60, A: 255},
				{R: 60, G: 60, B: 220, A: 255},
			} {
				quads = append(quads, render.Quad{
					X: int16(2 + i*6), Y: int16(2 + i*4), Width: side, Height: side,
					Tint: tint,
				})
			}

			// Superposés : le second gagne, entièrement.
			quads = append(quads,
				render.Quad{X: 30, Y: 2, Width: side, Height: side,
					Tint: render.Color{R: 220, G: 220, B: 60, A: 255}},
				render.Quad{X: 30, Y: 2, Width: side, Height: side,
					Tint: render.Color{R: 60, G: 200, B: 200, A: 255}},
			)

			for i, flags := range []render.Flags{
				0, render.MirrorX, render.MirrorY, render.MirrorX | render.MirrorY,
			} {
				quads = append(quads, render.Quad{
					X: int16(2 + i*11), Y: 20, Width: side, Height: side,
					Tint: render.White(), Flags: flags,
				})
			}
			return quads
		},
	}
}
