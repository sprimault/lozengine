// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario du découpage : huit quads qui débordent des bords et des coins, deux
// entièrement dehors. C'est celui qui casse en silence — un décalage d'un pixel ne se
// voit pas à l'œil, et les deux quads dehors sont là pour qu'un index hors bornes se
// manifeste au lieu de peindre au hasard.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Edges éprouve le découpage : huit quads débordent des quatre bords et des quatre
// coins, deux autres sont entièrement dehors.
//
// C'est le scénario qui casse en silence. Un découpage faux d'un pixel ne se voit pas
// à l'œil, un quad dessiné à une ligne de décalage non plus, et les deux quads
// entièrement dehors sont là pour qu'un débordement d'index se manifeste plutôt que
// de peindre au hasard.
func Edges() Scenario {
	const (
		width, height = 48, 32
		side          = 8
	)

	return Scenario{
		Name:       "edges",
		Width:      width,
		Height:     height,
		Frames:     1,
		Background: render.Color{R: 20, G: 24, B: 32, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(markedSheet(side))
		},

		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			positions := [][2]int{
				{-4, -4}, {width - 4, -4}, {-4, height - 4}, {width - 4, height - 4},
				{20, -4}, {20, height - 4}, {-4, 12}, {width - 4, 12},
				{width + 8, 12}, {-side - 8, 12},
			}

			quads := reserve
			for _, p := range positions {
				quads = append(quads, render.Quad{
					X: int16(p[0]), Y: int16(p[1]), Width: side, Height: side,
					Tint: render.White(),
				})
			}
			return quads
		},
	}
}
