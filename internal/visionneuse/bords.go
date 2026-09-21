// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// Bords éprouve le découpage : huit quads débordent des quatre bords et des quatre
// coins, deux autres sont entièrement dehors.
//
// C'est le scénario qui casse en silence. Un découpage faux d'un pixel ne se voit pas
// à l'œil, un quad dessiné à une ligne de décalage non plus, et les deux quads
// entièrement dehors sont là pour qu'un débordement d'index se manifeste plutôt que
// de peindre au hasard.
func Bords() Scenario {
	const (
		largeur, hauteur = 48, 32
		cote             = 8
	)

	return Scenario{
		Nom:     "bords",
		Largeur: largeur,
		Hauteur: hauteur,
		Images:  1,
		Fond:    rendu.Couleur{R: 20, V: 24, B: 32, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasDUnePlanche(plancheMarquee(cote))
		},

		Quads: func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu {
			positions := [][2]int{
				{-4, -4}, {largeur - 4, -4}, {-4, hauteur - 4}, {largeur - 4, hauteur - 4},
				{20, -4}, {20, hauteur - 4}, {-4, 12}, {largeur - 4, 12},
				{largeur + 8, 12}, {-cote - 8, 12},
			}

			quads := reserve
			for _, p := range positions {
				quads = append(quads, rendu.Quad{
					X: int16(p[0]), Y: int16(p[1]), Largeur: cote, Hauteur: cote,
					Teinte: rendu.Blanc(),
				})
			}
			return quads
		},
	}
}
