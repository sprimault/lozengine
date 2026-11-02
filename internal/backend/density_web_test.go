// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les deux régimes de densité d'une page, éprouvés sur les densités que les écrans
// servent plutôt que sur celle de la machine qui exécute le test.

package backend

import (
	"math"
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestCanvasForCoversEveryScreenDensity garde les deux régimes sur les densités que les
// écrans servent, et sur ce qu'une page rend quand elle ne sait pas décrire sa boîte.
//
// Les cas ne portent que des produits exacts en binaire — des boîtes entières, et des
// densités qui sont des fractions de puissance de deux —, pour que les attendus soient
// des nombres et non des expressions recopiées du code qu'ils contrôlent. Les densités
// qui ne tombent pas juste sont tenues par les cas de propriété plus bas.
func TestCanvasForCoversEveryScreenDensity(t *testing.T) {
	for _, c := range []struct {
		name                string
		boxWidth, boxHeight float64
		ratio               float64
		width, height       uint16
		cssWidth, cssHeight float64
	}{
		{"densité un", 960, 540, 1, 960, 540, 960, 540},
		{"densité deux", 960, 540, 2, 960, 540, 960, 540},
		{"densité trois sur un téléphone", 390, 844, 3, 390, 844, 390, 844},
		{"densité quatre", 960, 540, 4, 960, 540, 960, 540},
		{"boîte fractionnaire à densité entière", 389.75, 843.5, 3, 389, 843, 389, 843},
		{"densité 125 %", 1280, 720, 1.25, 1600, 900, 1280, 720},
		{"densité 150 %", 1280, 720, 1.5, 1920, 1080, 1280, 720},
		{"densité 175 %", 1280, 720, 1.75, 2240, 1260, 1280, 720},
		{"densité sous un, un zoom arrière", 960, 540, 0.5, 480, 270, 960, 540},
		{"densité presque entière", 390, 844, 3 - 1e-12, 390, 844, 390, 844},
		{"densité quasi nulle", 960, 540, 1e-12, 0, 0, 0, 0},
		{"densité nulle", 960, 540, 0, 960, 540, 960, 540},
		{"densité négative", 960, 540, -2, 960, 540, 960, 540},
		{"densité indéfinie", 960, 540, math.NaN(), 960, 540, 960, 540},
		{"densité infinie", 960, 540, math.Inf(1), 960, 540, 960, 540},
		{"boîte nulle", 0, 0, 2, 0, 0, 0, 0},
		{"boîte négative", -5, -5, 2, 0, 0, 0, 0},
		{"boîte indéfinie", math.NaN(), math.NaN(), 2, 0, 0, 0, 0},
		{"un seul côté décrit", 960, 0, 2, 960, 0, 960, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := canvasFor(c.boxWidth, c.boxHeight, c.ratio)
			want := canvasBox{
				width: c.width, height: c.height,
				cssWidth: c.cssWidth, cssHeight: c.cssHeight,
			}
			if got != want {
				t.Errorf("boîte %g×%g à la densité %g donne %+v, attendu %+v",
					c.boxWidth, c.boxHeight, c.ratio, got, want)
			}
		})
	}
}

// TestCanvasForKeepsTheUnitsInAnIntegerRatio garde l'invariant d'échelle entière du
// moteur là où la page l'applique : entre les pixels d'image du canvas et les pixels
// d'appareil qu'il occupe.
//
// **C'est le cas qui tient la dérivation de la taille CSS**, et il est le seul à le
// pouvoir. Poser la taille CSS depuis la boîte plutôt que depuis la taille en pixels
// d'image le fait rougir : à densité 1,25 sur une boîte de 390, le rapport devient
// 1,001 au lieu de 1, soit une mise à l'échelle fractionnaire que rien d'autre ne
// signalerait — l'image paraîtrait seulement un peu sale.
//
// La tolérance n'est pas un relâchement : un rapport flottant ne retombe pas sur l'entier
// au bit près, et c'est précisément pourquoi on vérifie la distance à l'entier plutôt que
// l'égalité.
func TestCanvasForKeepsTheUnitsInAnIntegerRatio(t *testing.T) {
	boxes := [][2]float64{{390, 844}, {389.75, 843.5}, {1280, 720}, {960.5, 540.25}, {480, 270}}
	ratios := []float64{1, 1.25, 1.5, 1.75, 2, 2.625, 3, 3.5, 0.5}

	for _, box := range boxes {
		for _, ratio := range ratios {
			got := canvasFor(box[0], box[1], ratio)
			for _, side := range []struct {
				axis   string
				pixels uint16
				css    float64
			}{
				{"largeur", got.width, got.cssWidth},
				{"hauteur", got.height, got.cssHeight},
			} {
				if side.pixels == 0 {
					continue
				}

				units := side.css * ratio / float64(side.pixels)
				if nearest := math.Round(units); nearest < 1 || math.Abs(units-nearest) > 1e-9 {
					t.Errorf("boîte %g×%g à la densité %g : %s, %g pixel(s) d'appareil par pixel d'image",
						box[0], box[1], ratio, side.axis, units)
				}
			}
		}
	}
}

// TestCanvasForMatchesDensityOneWhereItCan vérifie que le régime entier rend exactement
// ce que la même boîte rend à densité un : c'est la définition de la taille apparente
// préservée, et c'est ce que le lot achète.
//
// **Il rougirait si le canvas prenait la taille en pixels d'appareil**, qui est l'autre
// forme possible : sur un téléphone de 390 pixels CSS à densité trois, elle rend 1170
// pixels d'image au lieu de 390, et un personnage de vingt-quatre pixels y mesure les
// deux tiers de ce qu'il mesure sur un écran ordinaire.
//
// Les densités fractionnaires n'y figurent pas, et ce n'est pas un oubli : préserver
// demanderait un facteur égal à la densité multipliée par un entier, qui n'est pas entier.
func TestCanvasForMatchesDensityOneWhereItCan(t *testing.T) {
	for _, box := range [][2]float64{{390, 844}, {389.75, 843.5}, {1280, 720}, {480, 270}} {
		reference := canvasFor(box[0], box[1], 1)
		for _, ratio := range []float64{2, 3, 4, 5} {
			got := canvasFor(box[0], box[1], ratio)
			if got.width != reference.width || got.height != reference.height {
				t.Errorf("boîte %g×%g : %d×%d pixels d'image à la densité %g, %d×%d à la densité un",
					box[0], box[1], got.width, got.height, ratio, reference.width, reference.height)
			}
		}
	}
}

// TestCanvasForClampsToTheBufferSide garde la borne du tampon, et surtout qu'elle morde
// avant la division : une taille CSS dérivée de la boîte plutôt que du côté borné
// décrirait un rectangle que l'élément ne porte pas, et la page étirerait le canvas pour
// l'y faire tenir.
func TestCanvasForClampsToTheBufferSide(t *testing.T) {
	for _, c := range []struct {
		name                string
		boxWidth, boxHeight float64
		ratio               float64
		cssWidth            float64
	}{
		{"boîte au-delà de la borne", 40000, 40000, 1, render.MaxBufferSide},
		{"densité qui pousse au-delà", 30000, 30000, 1.5, render.MaxBufferSide / 1.5},
		{"boîte infinie", math.Inf(1), math.Inf(1), 2, render.MaxBufferSide},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := canvasFor(c.boxWidth, c.boxHeight, c.ratio)
			if got.width != render.MaxBufferSide || got.height != render.MaxBufferSide {
				t.Errorf("boîte %g×%g à la densité %g donne %d×%d pixels d'image, attendu %d de côté",
					c.boxWidth, c.boxHeight, c.ratio, got.width, got.height, render.MaxBufferSide)
			}
			if got.cssWidth != c.cssWidth {
				t.Errorf("boîte %g×%g à la densité %g donne %g pixels CSS, attendu %g",
					c.boxWidth, c.boxHeight, c.ratio, got.cssWidth, c.cssWidth)
			}
		})
	}
}
