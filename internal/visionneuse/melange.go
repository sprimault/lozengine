// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// Melange éprouve la composition : une échelle d'opacités sur un fond opaque, deux
// aplats translucides qui se recouvrent, et le bord translucide de la planche posé sur
// une couleur.
//
// Un arrondi qui dérive d'un cran ne se voit nulle part ailleurs. Ici il se lit sur
// l'échelle, où chaque marche vaut un mélange connu, et sur le recouvrement, où deux
// mélanges s'enchaînent — c'est-à-dire là où une erreur d'un pixel se double.
func Melange() Scenario {
	const (
		largeur, hauteur = 48, 32
		cote             = 8
	)

	return Scenario{
		Nom:     "melange",
		Largeur: largeur,
		Hauteur: hauteur,
		Images:  1,
		Fond:    rendu.Couleur{R: 200, V: 200, B: 200, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasDUnePlanche(plancheMarquee(cote))
		},

		Quads: func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu {
			quads := reserve

			// Une bande sombre sous l'échelle : un mélange ne se juge pas sur un seul
			// fond, et le blanc cache ce que le sombre révèle.
			quads = append(quads, rendu.Quad{
				X: 0, Y: 12, Largeur: largeur, Hauteur: 8,
				Teinte: rendu.Couleur{R: 30, V: 30, B: 40, A: 255}, Drapeaux: rendu.Aplat,
			})

			for i, alpha := range []uint8{255, 192, 128, 64, 16} {
				x := int16(2 + i*9)
				for _, y := range []int16{2, 12} {
					quads = append(quads, rendu.Quad{
						X: x, Y: y, Largeur: cote, Hauteur: cote,
						Teinte: rendu.Couleur{R: alpha, V: alpha, B: alpha, A: alpha},
					})
				}
			}

			// Deux aplats qui se recouvrent : le mélange s'applique une fois par quad,
			// donc la zone commune doit être plus opaque que chacune des deux.
			quads = append(quads,
				rendu.Quad{X: 4, Y: 22, Largeur: 20, Hauteur: 8,
					Teinte: rendu.Couleur{R: 0, V: 0, B: 96, A: 96}, Drapeaux: rendu.Aplat},
				rendu.Quad{X: 14, Y: 22, Largeur: 20, Hauteur: 8,
					Teinte: rendu.Couleur{R: 96, V: 0, B: 0, A: 96}, Drapeaux: rendu.Aplat},
			)
			return quads
		},
	}
}
