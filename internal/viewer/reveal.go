// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario de ce qu'on ne doit pas perdre de vue : un personnage derrière un mur
// qui le couvre entièrement, et qui se redessine par-dessus.

package viewer

import (
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario. Le mur est opaque et large : s'il laissait voir au travers,
// l'image ne dirait pas si le personnage a été révélé ou simplement aperçu.
const (
	revealGroundSheet = iota
	revealWallSheet
	revealFigureSheet
)

// La grille et la vue. Six cellules de côté, et de la marge en haut pour un mur de
// trois hauteurs de tuile.
const (
	revealSide = 6

	revealWidth  = revealSide*tileW + tileW
	revealHeight = revealSide*tileH + 5*tileH

	// Le recouvrement à partir duquel le personnage se redessine. Le mur lui en cache
	// près de deux cent quarante : le seuil est donc franchi largement, et la moitié
	// de cette valeur ne tiendrait pas si le mur bougeait d'une cellule.
	revealAfter = 100
)

// Reveal éprouve qu'un personnage caché par le décor se redessine au-dessus de lui.
//
// **C'est le seul scénario à pouvoir faire échouer la révélation**, et il existe parce
// qu'aucun autre ne met une chose marquée derrière un décor qui la couvre. Le régime
// qu'il garde est celui qui arrive quand on est encerclé : le mur est plus proche que
// le personnage, donc il se peint après et le recouvre entièrement — c'est l'ordre
// juste, et c'est précisément ce qui fait perdre son personnage.
//
// Ce que l'image montre : le personnage apparaît **entier et devant le mur**, dans la
// couche qu'il a demandée. Sans la révélation, il ne resterait de lui aucun pixel, et
// c'est ce que la comparaison attrape — ce ne sont pas quelques bords qui changent,
// c'est une figure qui existe ou non.
//
// Ce que le scénario ne garde pas : le seuil. Celui-ci est franchi d'un facteur deux,
// de sorte qu'un déplacement d'une cellule ne le fasse pas basculer — c'est la mesure
// du recouvrement que les cas de `sorting` éprouvent, pas cette image.
func Reveal() Scenario {
	return Scenario{
		Name:       "reveal",
		Width:      revealWidth,
		Height:     revealHeight,
		Frames:     1,
		Background: render.Color{R: 16, G: 18, B: 26, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(
				borderedSheet(tileW, tileH, render.Color{R: 72, G: 88, B: 72, A: 255}),
				borderedSheet(tileW, tileH*3, render.Color{R: 150, G: 140, B: 128, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, render.Color{R: 224, G: 96, B: 96, A: 255}),
			)
		},

		Quads: func(_ int, _ InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, sorting.New(revealScene()).Frame()...)
		},
	}
}

// revealScene monte le sol, le mur et le personnage qu'il cache.
func revealScene() *scene.Scene {
	s := sceneOf(revealSide, revealWidth, revealHeight, scene.Options{
		Entities:  4,
		Revealing: scene.RevealOptions{After: revealAfter},
	})

	fillGround(s, revealSide, scene.Aspect{
		Sheet: revealGroundSheet, Width: tileW, Height: tileH, Tint: render.White(),
	})

	// Le mur, sur la rangée qui précède le personnage du point de vue de celui qui
	// regarde : plus proche, donc peint après, donc devant lui.
	wall := scene.Aspect{
		Sheet: revealWallSheet, Width: tileW, Height: tileH * 3, Tint: render.White(),
	}
	for u := 2; u <= 3; u++ {
		s.PlaceCell(u, 3, 1, wall)
	}

	// Le personnage, derrière le mur et marqué pour la couche un. Sans cette marque,
	// l'image n'en montrerait rien.
	figure := scene.Aspect{
		Sheet: revealFigureSheet, Width: tileW / 3, Height: tileH * 3 / 2,
		Tint: render.White(), Reveal: 1,
	}
	s.Place(scene.Entity{X: 2.5, Y: 2.5, Aspect: figure})

	return s
}
