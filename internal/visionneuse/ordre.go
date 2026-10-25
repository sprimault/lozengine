// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Ordre éprouve l'ordre de dessin et les miroirs : trois quads en escalier, deux
// quads exactement superposés, et la planche marquée dessinée dans ses quatre
// orientations.
//
// L'ordre du tableau est la seule vérité du résultat, et c'est une vérité qu'aucune
// autre image ne vérifie : partout ailleurs les quads se recouvrent peu. Les miroirs
// sont là pour la même raison — un axe inversé donne une image plausible, donc
// invisible sans référence.
func Ordre() Scenario {
	const (
		largeur, hauteur = 48, 32
		cote             = 8
	)

	return Scenario{
		Nom:     "ordre",
		Largeur: largeur,
		Hauteur: hauteur,
		Images:  1,
		Fond:    render.Color{R: 24, G: 20, B: 20, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasDUnePlanche(plancheMarquee(cote))
		},

		Quads: func(numero int, e Entrees, reserve render.DrawList) render.DrawList {
			quads := reserve

			// Escalier : chacun recouvre le précédent d'un quart.
			for i, teinte := range []render.Color{
				{R: 220, G: 60, B: 60, A: 255},
				{R: 60, G: 220, B: 60, A: 255},
				{R: 60, G: 60, B: 220, A: 255},
			} {
				quads = append(quads, render.Quad{
					X: int16(2 + i*6), Y: int16(2 + i*4), Width: cote, Height: cote,
					Tint: teinte,
				})
			}

			// Superposés : le second gagne, entièrement.
			quads = append(quads,
				render.Quad{X: 30, Y: 2, Width: cote, Height: cote,
					Tint: render.Color{R: 220, G: 220, B: 60, A: 255}},
				render.Quad{X: 30, Y: 2, Width: cote, Height: cote,
					Tint: render.Color{R: 60, G: 200, B: 200, A: 255}},
			)

			for i, drapeaux := range []render.Flags{
				0, render.MirrorX, render.MirrorY, render.MirrorX | render.MirrorY,
			} {
				quads = append(quads, render.Quad{
					X: int16(2 + i*11), Y: 20, Width: cote, Height: cote,
					Tint: render.White(), Flags: drapeaux,
				})
			}
			return quads
		},
	}
}
