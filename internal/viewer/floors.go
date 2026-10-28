// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario des deux niveaux : un sol, une plateforme posée dessus par son étage, et
// deux figures qui disent où chaque niveau commence. C'est lui qui garde que l'étage
// est une hauteur et pas seulement un rang dans l'ordre de dessin.

package viewer

import (
	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario. Trois teintes franchement distinctes : ce qu'on lit sur
// l'image est de combien chaque chose est montée, donc il faut pouvoir suivre un bord
// sans compter les pixels.
const (
	floorsGroundSheet = iota
	floorsPlatformSheet
	floorsFigureSheet
)

// La grille et la vue. Six cellules de côté, et quatre hauteurs de tuile de marge en
// haut : une plateforme à l'étage un monte d'une hauteur de cellule, et la figure qui
// s'y tient d'autant.
const (
	floorsSide = 6

	floorsWidth  = floorsSide*tileW + tileW
	floorsHeight = floorsSide*tileH + 4*tileH
)

// L'emprise de la plateforme, en cellules. Décalée vers le fond pour que le sol reste
// visible devant elle, ce qui est la seule façon de voir qu'elle est plus haute.
const (
	floorsPlatformFrom = 1
	floorsPlatformTo   = 3
)

// Floors éprouve qu'un décor à deux niveaux se dessine à deux hauteurs.
//
// **C'est le seul scénario à pouvoir faire échouer cette entrée.** Aucun autre ne pose
// de décor à un étage : [Core] n'élève qu'une entité, et les autres posent leurs quads
// à la main sans passer par le tri. Le régime qu'il garde est celui d'une cellule de
// décor dont l'étage n'est pas zéro, et d'une entité qui se tient dessus.
//
// Ce que le défaut donnait, et que l'image montre : le décor était projeté à hauteur
// nulle quel que soit son étage, parce que la hauteur d'une cellule n'avait aucun champ
// où vivre — [scene.Scene.PlaceCell] ne prend que deux coordonnées et une strate. La
// plateforme se rangeait donc devant tout le sol, ce que son étage demande, mais se
// dessinait au ras de ce sol : elle passait devant sans jamais monter. La correction
// fait de l'étage une hauteur autant qu'un rang, pour une cellule comme pour une
// entité.
//
// Les deux figures couvrent les deux régimes du même champ. Celle du bas est au sol et
// ne doit pas bouger ; celle du haut se tient sur la plateforme et monte par son seul
// étage, sans z — porter la hauteur dans les deux l'élèverait deux fois.
func Floors() Scenario {
	return Scenario{
		Name:       "floors",
		Width:      floorsWidth,
		Height:     floorsHeight,
		Frames:     1,
		Background: render.Color{R: 20, G: 22, B: 28, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			var atlas raster.Atlas
			for _, p := range []raster.Sheet{
				borderedSheet(tileW, tileH, render.Color{R: 72, G: 96, B: 72, A: 255}),
				borderedSheet(tileW, tileH, render.Color{R: 168, G: 152, B: 112, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, render.Color{R: 208, G: 96, B: 96, A: 255}),
			} {
				if _, err := atlas.Add(p); err != nil {
					return nil, err
				}
			}
			return &atlas, nil
		},

		Quads: func(_ int, _ InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, sorting.New(floorsScene()).Frame()...)
		},
	}
}

// floorsScene monte le sol, la plateforme et les deux figures.
//
// Montée à neuf à chaque image, comme celles de [Core] et [Ground] : une image doit
// pouvoir se rendre seule, deux fois, et donner les mêmes pixels.
func floorsScene() *scene.Scene {
	s, err := scene.New(floorsSide, floorsSide, geometry.Dimetric(tileW, tileH),
		scene.Options{Entities: 8})
	if err != nil {
		panic(err)
	}
	s.Focus(floorsSide/2, floorsSide/2, floorsWidth, floorsHeight)

	// Le sol entier, à l'étage zéro, sur la strate zéro.
	ground := scene.Aspect{
		Sheet: floorsGroundSheet, Width: tileW, Height: tileH, Tint: render.White(),
	}
	for v := range floorsSide {
		for u := range floorsSide {
			s.PlaceCell(u, v, 0, ground)
		}
	}

	// La plateforme, à l'étage un et sur la strate un. La strate la met au-dessus du
	// sol de sa propre cellule ; l'étage la met au-dessus du niveau entier, et c'est
	// lui qui la fait monter à l'écran. Les deux notions se voient ici dans le même
	// quad, ce qui est la raison de les poser ensemble plutôt que séparément.
	platform := scene.Aspect{
		Sheet: floorsPlatformSheet, Width: tileW, Height: tileH, Tint: render.White(),
		Floor: 1,
	}
	for v := floorsPlatformFrom; v <= floorsPlatformTo; v++ {
		for u := floorsPlatformFrom; u <= floorsPlatformTo; u++ {
			s.PlaceCell(u, v, 1, platform)
		}
	}

	figure := scene.Aspect{
		Sheet: floorsFigureSheet, Width: tileW / 3, Height: tileH * 3 / 2,
		Tint: render.White(),
	}

	// Au sol, et hors de ce que la plateforme recouvre : son rôle est de témoigner
	// qu'un étage nul ne monte pas, pas de départager les deux niveaux. Posée devant
	// la plateforme, elle serait recouverte par elle — l'étage domine la profondeur
	// dans la clé —, et c'est l'occlusion qui traite ce cas, pas ce scénario.
	s.Place(scene.Entity{X: 4.5, Y: 4.5, Aspect: figure})

	// Sur la plateforme, au même étage qu'elle. Sans z : l'étage porte la hauteur.
	upper := figure
	upper.Floor = 1
	s.Place(scene.Entity{X: 2.5, Y: 2.5, Aspect: upper})

	return s
}
