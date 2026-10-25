// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les planches que les scénarios se dessinent, plutôt que de les charger : aucune
// image n'entre dans le dépôt pour éprouver le rendu, et un carré volontairement
// asymétrique montre l'inversion d'axe qu'un carré uni cacherait.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// markedSheet dessine un carré blanc dont aucun quart de tour ni aucun miroir ne
// redonne le même dessin : un coin sombre en haut à gauche, une colonne translucide à
// droite.
//
// C'est ce qui rend une image de référence utile. Un carré uni cache une inversion
// d'axe, un miroir posé par erreur ou une source lue à l'envers ; celui-ci les montre.
func markedSheet(side uint16) raster.Sheet {
	p := raster.NewSheet(side, side)

	for y := range int(side) {
		for x := range int(side) {
			color := render.Color{R: 255, G: 255, B: 255, A: 255}
			switch {
			case x < 2 && y < 2:
				color = render.Color{R: 40, G: 40, B: 40, A: 255}
			case x == int(side)-1:
				color = render.Color{R: 128, G: 32, B: 32, A: 128}
			}
			p.Pixels[y*int(side)+x] = color
		}
	}
	return p
}

// borderedSheet dessine un rectangle plein cerné d'un bord sombre d'un pixel.
//
// Le bord est ce qui rend l'ordre de dessin lisible : deux aplats de même teinte qui
// se recouvrent ne montrent pas lequel est devant, deux rectangles bordés le montrent
// à l'endroit exact où l'un mange le contour de l'autre.
func borderedSheet(width, height uint16, c render.Color) raster.Sheet {
	border := render.Color{R: c.R / 3, G: c.G / 3, B: c.B / 3, A: 255}
	p := raster.NewSheet(width, height)

	for y := range int(height) {
		for x := range int(width) {
			tint := c
			if x == 0 || y == 0 || x == int(width)-1 || y == int(height)-1 {
				tint = border
			}
			p.Pixels[y*int(width)+x] = tint
		}
	}
	return p
}

// atlasOfOneSheet range une seule planche et rend l'atlas.
func atlasOfOneSheet(p raster.Sheet) (*raster.Atlas, error) {
	var atlas raster.Atlas
	if _, err := atlas.Add(p); err != nil {
		return nil, err
	}
	return &atlas, nil
}
