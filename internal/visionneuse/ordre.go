// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
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
		Fond:    rendu.Couleur{R: 24, V: 20, B: 20, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasDUnePlanche(plancheMarquee(cote))
		},

		Quads: func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu {
			quads := reserve

			// Escalier : chacun recouvre le précédent d'un quart.
			for i, teinte := range []rendu.Couleur{
				{R: 220, V: 60, B: 60, A: 255},
				{R: 60, V: 220, B: 60, A: 255},
				{R: 60, V: 60, B: 220, A: 255},
			} {
				quads = append(quads, rendu.Quad{
					X: int16(2 + i*6), Y: int16(2 + i*4), Largeur: cote, Hauteur: cote,
					Teinte: teinte,
				})
			}

			// Superposés : le second gagne, entièrement.
			quads = append(quads,
				rendu.Quad{X: 30, Y: 2, Largeur: cote, Hauteur: cote,
					Teinte: rendu.Couleur{R: 220, V: 220, B: 60, A: 255}},
				rendu.Quad{X: 30, Y: 2, Largeur: cote, Hauteur: cote,
					Teinte: rendu.Couleur{R: 60, V: 200, B: 200, A: 255}},
			)

			for i, drapeaux := range []rendu.Drapeaux{
				0, rendu.MiroirX, rendu.MiroirY, rendu.MiroirX | rendu.MiroirY,
			} {
				quads = append(quads, rendu.Quad{
					X: int16(2 + i*11), Y: 20, Largeur: cote, Hauteur: cote,
					Teinte: rendu.Blanc(), Drapeaux: drapeaux,
				})
			}
			return quads
		},
	}
}
