// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"fmt"
	"strings"

	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// Mire est le scénario de démonstration : un sol en damier de losanges, un pion qui
// traverse l'image et en sort, et un voile translucide qui balaie le tout.
//
// Il est fait pour exercer les quatre chemins du blit d'un seul coup — copie pour le
// pion, modulation pour les tuiles, mélange pour leurs bords, aplat pour le voile — et
// le découpage, puisque le pion entre et sort par les bords.
func Mire() Scenario {
	const (
		largeur, hauteur = 128, 72
		tuileL, tuileH   = 32, 16
		pionCote         = 8
	)

	// Deux teintes sur une planche blanche : la couleur du sol vient de la modulation,
	// ce qui évite deux planches pour un damier.
	sol := [2]rendu.Couleur{
		{R: 90, V: 130, B: 90, A: 255},
		{R: 70, V: 100, B: 75, A: 255},
	}

	return Scenario{
		Nom:     "mire",
		Largeur: largeur,
		Hauteur: hauteur,
		Images:  24,
		Fond:    rendu.Couleur{R: 12, V: 12, B: 18, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			var atlas raster.Atlas
			for _, p := range []raster.Planche{plancheTuile(), planchePion()} {
				if _, err := atlas.Ajouter(p); err != nil {
					return nil, err
				}
			}
			return &atlas, nil
		},

		Quads: func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu {
			quads := reserve

			for ligne := 0; ligne*tuileH/2 < hauteur+tuileH; ligne++ {
				for colonne := 0; colonne*tuileL < largeur+tuileL; colonne++ {
					quads = append(quads, rendu.Quad{
						X:       int16(colonne*tuileL - tuileL + (ligne%2)*(tuileL/2)),
						Y:       int16(ligne*tuileH/2 - tuileH),
						Largeur: tuileL,
						Hauteur: tuileH,
						Teinte:  sol[(colonne+ligne)%2],
					})
				}
			}

			x, y := -pionCote+numero*6, 30
			if px, py, ok := positionPion(e); ok {
				x, y = px, py
			}
			quads = append(quads, rendu.Quad{
				X: int16(x), Y: int16(y), Largeur: pionCote, Hauteur: pionCote,
				Planche: 1, Teinte: rendu.Blanc(),
			})

			if !strings.Contains(string(e), "sans-voile") {
				quads = append(quads, rendu.Quad{
					X: int16(largeur - 4 - numero*5), Y: 0, Largeur: 24, Hauteur: hauteur,
					Teinte: rendu.Couleur{A: 96}, Drapeaux: rendu.Aplat,
				})
			}
			return quads
		},
	}
}

// positionPion lit la directive « pion X Y » d'une entrée.
//
// Une position absolue et non un déplacement : un scénario doit rester une fonction
// pure de son numéro d'image, donc rien ne s'accumule d'une image à la suivante.
func positionPion(e Entrees) (x, y int, ok bool) {
	if _, err := fmt.Sscanf(string(e), "pion %d %d", &x, &y); err != nil {
		return 0, 0, false
	}
	return x, y, true
}

// plancheTuile dessine un losange 32×16 blanc au bord adouci, que la teinte colore.
func plancheTuile() raster.Planche {
	p := raster.NouvellePlanche(32, 16)
	for y := range 16 {
		rang := y
		if y >= 8 {
			rang = 15 - y
		}
		demi := (rang + 1) * 2

		for x := 16 - demi; x < 16+demi; x++ {
			couleur := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
			if x == 16-demi || x == 16+demi-1 {
				couleur = rendu.Couleur{R: 128, V: 128, B: 128, A: 128}
			}
			p.Pixels[y*32+x] = couleur
		}
	}
	return p
}

// planchePion dessine un carré de 8 au contour translucide.
func planchePion() raster.Planche {
	p := raster.NouvellePlanche(8, 8)
	for y := range 8 {
		for x := range 8 {
			couleur := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
			if x == 0 || y == 0 || x == 7 || y == 7 {
				couleur = rendu.Couleur{R: 96, V: 96, B: 96, A: 96}
			}
			p.Pixels[y*8+x] = couleur
		}
	}
	return p
}
