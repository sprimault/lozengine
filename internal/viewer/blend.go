// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario de la composition : une échelle d'opacités, deux aplats translucides qui
// se recouvrent, et un bord translucide posé sur une couleur. Il existe parce qu'un
// arrondi qui dérive d'un cran ne se voit nulle part ailleurs.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Blend éprouve la composition : une échelle d'opacités sur un fond opaque, deux
// aplats translucides qui se recouvrent, et le bord translucide de la planche posé sur
// une couleur.
//
// Un arrondi qui dérive d'un cran ne se voit nulle part ailleurs. Ici il se lit sur
// l'échelle, où chaque marche vaut un mélange connu, et sur le recouvrement, où deux
// mélanges s'enchaînent — c'est-à-dire là où une erreur d'un pixel se double.
func Blend() Scenario {
	const (
		width, height = 48, 32
		side          = 8
	)

	return Scenario{
		Name:       "blend",
		Width:      width,
		Height:     height,
		Frames:     1,
		Background: render.Color{R: 200, G: 200, B: 200, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(markedSheet(side))
		},

		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			quads := reserve

			// Une bande sombre sous l'échelle : un mélange ne se juge pas sur un seul
			// fond, et le blanc cache ce que le sombre révèle.
			quads = append(quads, render.Quad{
				X: 0, Y: 12, Width: width, Height: 8,
				Tint: render.Color{R: 30, G: 30, B: 40, A: 255}, Flags: render.Flat,
			})

			// Les quatre composantes égales font un blanc prémultiplié à cette
			// opacité, et c'est ce que l'échelle garde : l'alpha de la teinte n'agit
			// qu'une fois, donc une marche vaut l'opacité qu'elle porte.
			for i, alpha := range []uint8{255, 192, 128, 64, 16} {
				x := int16(2 + i*9)
				for _, y := range []int16{2, 12} {
					quads = append(quads, render.Quad{
						X: x, Y: y, Width: side, Height: side,
						Tint: render.Color{R: alpha, G: alpha, B: alpha, A: alpha},
					})
				}
			}

			// Deux aplats qui se recouvrent : le mélange s'applique une fois par quad,
			// donc la zone commune doit être plus opaque que chacune des deux.
			quads = append(quads,
				render.Quad{X: 4, Y: 22, Width: 20, Height: 8,
					Tint: render.Color{R: 0, G: 0, B: 96, A: 96}, Flags: render.Flat},
				render.Quad{X: 14, Y: 22, Width: 20, Height: 8,
					Tint: render.Color{R: 96, G: 0, B: 0, A: 96}, Flags: render.Flat},
			)
			return quads
		},
	}
}
