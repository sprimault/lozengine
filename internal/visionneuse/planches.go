// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// plancheMarquee dessine un carré blanc dont aucun quart de tour ni aucun miroir ne
// redonne le même dessin : un coin sombre en haut à gauche, une colonne translucide à
// droite.
//
// C'est ce qui rend une image de référence utile. Un carré uni cache une inversion
// d'axe, un miroir posé par erreur ou une source lue à l'envers ; celui-ci les montre.
func plancheMarquee(cote uint16) raster.Planche {
	p := raster.NouvellePlanche(cote, cote)

	for y := range int(cote) {
		for x := range int(cote) {
			couleur := rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
			switch {
			case x < 2 && y < 2:
				couleur = rendu.Couleur{R: 40, V: 40, B: 40, A: 255}
			case x == int(cote)-1:
				couleur = rendu.Couleur{R: 128, V: 32, B: 32, A: 128}
			}
			p.Pixels[y*int(cote)+x] = couleur
		}
	}
	return p
}

// atlasDUnePlanche range une seule planche et rend l'atlas.
func atlasDUnePlanche(p raster.Planche) (*raster.Atlas, error) {
	var atlas raster.Atlas
	if _, err := atlas.Ajouter(p); err != nil {
		return nil, err
	}
	return &atlas, nil
}
