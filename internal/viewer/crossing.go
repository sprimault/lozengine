// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le scénario du critère de fin : un décor à deux niveaux qu'un personnage traverse
// de part en part, en passant sous la plateforme qui le couvrirait entièrement.

package viewer

import (
	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Les planches du scénario.
const (
	crossingGroundSheet = iota
	crossingPlatformSheet
	crossingFigureSheet
)

// La grille, la vue, et la traversée.
const (
	crossingSide   = 8
	crossingFrames = 8

	crossingWidth  = crossingSide*tileW + tileW
	crossingHeight = crossingSide*tileH + 5*tileH

	// L'emprise de la plateforme, au milieu de la carte : le personnage entre
	// dessous, la traverse entièrement et en ressort.
	crossingPlatformFrom = 2
	crossingPlatformTo   = 4

	// De combien le personnage se trouve caché avant de se redessiner. Son sprite
	// couvre deux cent quarante pixels ; cent est franchi dès qu'un tiers de lui
	// disparaît, et jamais par le seul chevauchement d'un pied sur une bordure.
	crossingRevealAfter = 100
)

// crossingFigureTint est la teinte du personnage, et aucune autre planche ne la porte.
// C'est elle que [TestTheCharacterIsNeverLost] compte.
var crossingFigureTint = render.Color{R: 224, G: 96, B: 96, A: 255}

// Crossing éprouve le critère de fin du jalon : un décor à deux niveaux se traverse
// sans perdre le personnage.
//
// **C'est le seul scénario à poser le problème entier**, les autres n'en gardant
// qu'une pièce : [Floors] garde que l'étage est une hauteur, [Reveal] garde qu'une
// chose cachée se redessine, et celui-ci garde que les deux tiennent ensemble pendant
// un déplacement. La différence n'est pas théorique — le personnage y passe par des
// régimes qu'une image fixe n'atteint pas : dehors, au bord, entièrement dessous, et
// ressorti.
//
// Ce que l'image montre : le personnage reste visible à chaque pas, devant la
// plateforme quand elle le couvrirait. Sans la révélation il disparaîtrait au tiers de
// sa course et reviendrait aux deux tiers, ce qui est précisément « perdre son
// personnage » — et ce qui arrive avec un ordre de dessin pourtant juste, l'étage
// dominant la profondeur dans la clé.
//
// **Trois images seulement sont versées** : la 0, la 4 et la 7. Et aucune des trois ne
// montre la révélation à l'œuvre, ce qu'il faut dire plutôt que de laisser croire
// l'inverse — la godoc annonçait « au milieu du passage ». Mesuré : elle s'exerce aux
// images 0 à 3, qui rendent deux quads de personnage, et plus aux 4 à 7, qui n'en
// rendent qu'un. La 0 est donc la seule des trois versées à la montrer, et elle est
// aussi le premier pas.
//
// Le comptage de pixels, lui, parcourt les huit — c'est lui qui rend le critère de fin
// mécanique, une image de référence verrouillant des pixels et non une impression, et
// c'est lui qui voit les quatre images où la chose se révèle.
func Crossing() Scenario {
	return Scenario{
		Name:       "crossing",
		Width:      crossingWidth,
		Height:     crossingHeight,
		Frames:     crossingFrames,
		References: []int{0, 4, 7},
		Background: render.Color{R: 18, G: 20, B: 28, A: 255},

		Atlas: func() (*raster.Atlas, error) {
			return atlasOf(
				borderedSheet(tileW, tileH, render.Color{R: 76, G: 92, B: 76, A: 255}),
				borderedSheet(tileW, tileH*2, render.Color{R: 164, G: 150, B: 120, A: 255}),
				borderedSheet(tileW/3, tileH*3/2, crossingFigureTint),
			)
		},

		Quads: func(frame int, _ InputLine, reserve render.DrawList) render.DrawList {
			return append(reserve, sorting.New(crossingScene(frame)).Frame()...)
		},
	}
}

// crossingScene monte la carte et place le personnage là où ce pas le met.
//
// La position se calcule depuis le numéro d'image et non d'un pas accumulé : une image
// doit pouvoir se rendre seule, deux fois, et donner les mêmes pixels.
func crossingScene(frame int) *scene.Scene {
	s := sceneOf(crossingSide, crossingWidth, crossingHeight, scene.Options{
		Entities:  4,
		Revealing: scene.RevealOptions{After: crossingRevealAfter},
	})

	fillGround(s, crossingSide, scene.Aspect{
		Sheet: crossingGroundSheet, Width: tileW, Height: tileH, Tint: render.White(),
	})

	// La plateforme, à l'étage un : elle monte à l'écran et passe devant tout l'étage
	// du sol, ce qui est l'ordre juste et ce qui cacherait le personnage.
	platform := scene.Aspect{
		Sheet: crossingPlatformSheet, Width: tileW, Height: tileH * 2,
		Tint: render.White(), Floor: 1,
	}
	for v := crossingPlatformFrom; v <= crossingPlatformTo; v++ {
		for u := crossingPlatformFrom; u <= crossingPlatformTo; u++ {
			s.PlaceCell(u, v, 1, platform)
		}
	}

	// La traversée, en diagonale d'un coin à l'autre : c'est le trajet qui passe le
	// plus longtemps sous la plateforme.
	//
	// Le produit est entier et la division vient après, pour qu'il ne reste ici aucun
	// « x·y + z » flottant : c'est le motif qu'arm64 et riscv64 fusionnent en une
	// instruction unique et qu'amd64 ne fusionne pas, et ce scénario produit des images
	// de référence comparées au pixel. Un produit entier ne se fusionne pas, donc la
	// forme tient par construction et non par vigilance — là où séparer les deux
	// opérations sur deux lignes n'aurait rien garanti, la spécification autorisant la
	// fusion « possibly across statements ».
	at := 0.5 + float64(frame*(crossingSide-1))/float64(crossingFrames-1)

	figure := scene.Aspect{
		Sheet: crossingFigureSheet, Width: tileW / 3, Height: tileH * 3 / 2,
		Tint: render.White(), Reveal: 1,
	}
	s.Place(scene.Entity{X: at, Y: at, Aspect: figure})

	return s
}
