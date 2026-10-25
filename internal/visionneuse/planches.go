// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// plancheMarquee dessine un carré blanc dont aucun quart de tour ni aucun miroir ne
// redonne le même dessin : un coin sombre en haut à gauche, une colonne translucide à
// droite.
//
// C'est ce qui rend une image de référence utile. Un carré uni cache une inversion
// d'axe, un miroir posé par erreur ou une source lue à l'envers ; celui-ci les montre.
func plancheMarquee(cote uint16) raster.Sheet {
	p := raster.NewSheet(cote, cote)

	for y := range int(cote) {
		for x := range int(cote) {
			couleur := render.Color{R: 255, G: 255, B: 255, A: 255}
			switch {
			case x < 2 && y < 2:
				couleur = render.Color{R: 40, G: 40, B: 40, A: 255}
			case x == int(cote)-1:
				couleur = render.Color{R: 128, G: 32, B: 32, A: 128}
			}
			p.Pixels[y*int(cote)+x] = couleur
		}
	}
	return p
}

// plancheBordee dessine un rectangle plein cerné d'un bord sombre d'un pixel.
//
// Le bord est ce qui rend l'ordre de dessin lisible : deux aplats de même teinte qui
// se recouvrent ne montrent pas lequel est devant, deux rectangles bordés le montrent
// à l'endroit exact où l'un mange le contour de l'autre.
func plancheBordee(largeur, hauteur uint16, c render.Color) raster.Sheet {
	bord := render.Color{R: c.R / 3, G: c.G / 3, B: c.B / 3, A: 255}
	p := raster.NewSheet(largeur, hauteur)

	for y := range int(hauteur) {
		for x := range int(largeur) {
			teinte := c
			if x == 0 || y == 0 || x == int(largeur)-1 || y == int(hauteur)-1 {
				teinte = bord
			}
			p.Pixels[y*int(largeur)+x] = teinte
		}
	}
	return p
}

// atlasDUnePlanche range une seule planche et rend l'atlas.
func atlasDUnePlanche(p raster.Sheet) (*raster.Atlas, error) {
	var atlas raster.Atlas
	if _, err := atlas.Add(p); err != nil {
		return nil, err
	}
	return &atlas, nil
}
