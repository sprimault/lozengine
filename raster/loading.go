// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le chargement d'une planche depuis un PNG, et la prémultiplication qui l'accompagne :
// le format stocke l'alpha à part, le moteur le veut déjà appliqué pour que composer un
// pixel reste une multiplication.

package raster

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"

	"github.com/sprimault/lozengine/render"
)

// LoadSheet lit un PNG et rend une planche en alpha prémultiplié.
//
// Un io.Reader et non un chemin : la même fonction sert au fichier, à la ressource
// embarquée et à l'archive. Le confort d'un chemin appartient au paquet racine.
//
// Le passage par image.RGBA fait tout le travail ingrat en une fois — la
// prémultiplication, que le PNG ne fait pas, la palette, les niveaux de gris, et
// une image dont l'origine n'est pas en (0, 0), ce qu'une sous-image produit. Le
// coût est une copie au chargement, jamais à l'image.
func LoadSheet(r io.Reader) (Sheet, error) {
	source, err := png.Decode(r)
	if err != nil {
		return Sheet{}, fmt.Errorf("decoding PNG: %w", err)
	}

	size := source.Bounds()
	if size.Dx() > math.MaxUint16 || size.Dy() > math.MaxUint16 {
		return Sheet{}, fmt.Errorf("sheet of %d×%d: %d pixels per side at most, a quad addressing its source in sixteen bits only",
			size.Dx(), size.Dy(), math.MaxUint16)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, size.Dx(), size.Dy()))
	draw.Draw(rgba, rgba.Bounds(), source, size.Min, draw.Src)

	sheet := NewSheet(uint16(size.Dx()), uint16(size.Dy()))
	for i := range sheet.Pixels {
		o := i * 4
		sheet.Pixels[i] = render.Color{
			R: rgba.Pix[o],
			G: rgba.Pix[o+1],
			B: rgba.Pix[o+2],
			A: rgba.Pix[o+3],
		}
	}
	return sheet, nil
}
