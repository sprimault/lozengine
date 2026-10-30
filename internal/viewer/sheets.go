// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les planches que les scénarios se dessinent, plutôt que de les charger : aucune
// image n'entre dans le dépôt pour éprouver le rendu, et un carré volontairement
// asymétrique montre l'inversion d'axe qu'un carré uni cacherait.

package viewer

import (
	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
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

// atlasOf range les planches dans l'ordre reçu et rend l'atlas.
//
// **L'ordre est celui que les aspects désignent**, chacun portant l'index de sa planche :
// en intervertir deux change ce qu'un scénario montre sans qu'aucun contrôle ne le dise
// avant la comparaison des images.
//
// Variadique parce que les sept scénarios qui montaient leur atlas à la main écrivaient
// la même boucle, et que celle d'une planche unique n'était qu'un cas de celle-là.
func atlasOf(sheets ...raster.Sheet) (*raster.Atlas, error) {
	var atlas raster.Atlas
	for _, p := range sheets {
		if _, err := atlas.Add(p); err != nil {
			return nil, err
		}
	}
	return &atlas, nil
}

// sceneOf monte une grille carrée au pas des scénarios, cadrée sur son centre.
//
// Les cinq scénarios qui montent une scène écrivaient ce préambule à l'identique, seules
// leurs options changeant. Elles restent donc le paramètre, et le reste — la projection
// dimétrique du dépôt, le centre, la panique sur l'échec — est commun.
//
// **La panique plutôt qu'une erreur rendue** : un scénario du catalogue est écrit dans ce
// paquet, donc un montage qui échoue est une faute d'ici et non une donnée d'entrée. La
// visionneuse n'a personne à qui se plaindre.
func sceneOf(side int, width, height int32, options scene.Options) *scene.Scene {
	s, err := scene.New(side, side, geometry.Dimetric(tileW, tileH), options)
	if err != nil {
		panic(err)
	}
	s.Focus(float64(side)/2, float64(side)/2, width, height)
	return s
}

// fillGround pose le même aspect sur toute la grille, sur la strate du sol.
//
// Quatre scénarios remplissaient leur sol par cette double boucle. C'est aussi la
// configuration qu'un jeu obtient sans rien régler, donc celle où un défaut d'ordre se
// produit : la garder en un seul endroit évite qu'un scénario la décale sans le vouloir.
func fillGround(s *scene.Scene, side int, a scene.Aspect) {
	for v := range side {
		for u := range side {
			s.PlaceCell(u, v, 0, a)
		}
	}
}
