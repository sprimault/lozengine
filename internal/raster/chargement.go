// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"

	"github.com/sprimault/lozengine/internal/rendu"
)

// ChargerPlanche lit un PNG et rend une planche en alpha prémultiplié.
//
// Un io.Reader et non un chemin : la même fonction sert au fichier, à la ressource
// embarquée et à l'archive. Le confort d'un chemin appartient au paquet racine.
//
// Le passage par image.RGBA fait tout le travail ingrat en une fois — la
// prémultiplication, que le PNG ne fait pas, la palette, les niveaux de gris, et
// une image dont l'origine n'est pas en (0, 0), ce qu'une sous-image produit. Le
// coût est une copie au chargement, jamais à l'image.
func ChargerPlanche(r io.Reader) (Planche, error) {
	source, err := png.Decode(r)
	if err != nil {
		return Planche{}, fmt.Errorf("décodage du PNG : %w", err)
	}

	taille := source.Bounds()
	if taille.Dx() > math.MaxUint16 || taille.Dy() > math.MaxUint16 {
		return Planche{}, fmt.Errorf("planche de %d×%d : %d pixels de côté au plus, un quad n'en désignant la source que sur seize bits",
			taille.Dx(), taille.Dy(), math.MaxUint16)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, taille.Dx(), taille.Dy()))
	draw.Draw(rgba, rgba.Bounds(), source, taille.Min, draw.Src)

	planche := NouvellePlanche(uint16(taille.Dx()), uint16(taille.Dy()))
	for i := range planche.Pixels {
		o := i * 4
		planche.Pixels[i] = rendu.Couleur{
			R: rgba.Pix[o],
			V: rgba.Pix[o+1],
			B: rgba.Pix[o+2],
			A: rgba.Pix[o+3],
		}
	}
	return planche, nil
}
