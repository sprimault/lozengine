// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La traduction du pointeur d'une page : la position dans l'unité du moteur, le masque
// de boutons joué bit à bit, et les crans d'une molette qui n'en compte pas.

package backend

import (
	"math"

	"github.com/sprimault/lozengine/input"
)

// pointerAt porte la position d'un événement en pixels d'image du canvas, qui est
// l'unité que le contrat de [Window] appelle des pixels de fenêtre.
//
// **L'événement donne sa position en pixels CSS, relative à la cible**, ce qui épargne
// de connaître où le canvas se trouve dans la page : une page qui défile entre deux
// images déplacerait son coin sans rien changer à cette position. Le facteur entre les
// deux unités est connu sans rien demander à la page — c'est le rapport des deux tailles
// que [canvasFor] a décidées —, et il vaut un dans le régime de densité entière.
//
// **La borne n'est pas de la défensive, c'est la spécification de Go** : la conversion
// d'un flottant hors de la plage du type de destination a un résultat indéfini, et un
// pointeur capturé peut sortir loin du canvas. Sans elle, une position aberrante
// dépendrait de l'architecture, ce que le déterminisme du moteur interdit.
//
// Une boîte sans taille CSS rend l'origine : c'est un canvas que la page a masqué, et
// aucune position n'y a de sens.
func pointerAt(offsetX, offsetY float64, box canvasBox) (x, y int16) {
	if box.cssWidth == 0 || box.cssHeight == 0 {
		return 0, 0
	}
	return clampToInt16(offsetX * float64(box.width) / box.cssWidth),
		clampToInt16(offsetY * float64(box.height) / box.cssHeight)
}

// clampToInt16 ramène un flottant dans la plage du type, en traitant l'indéfini comme
// l'origine.
func clampToInt16(v float64) int16 {
	switch {
	case math.IsNaN(v):
		return 0
	case v > math.MaxInt16:
		return math.MaxInt16
	case v < math.MinInt16:
		return math.MinInt16
	}
	return int16(v)
}

// playButtons joue sur l'état chaque bouton dont le masque a changé, et rend le masque
// qu'il faudra comparer à l'événement suivant.
//
// **Le masque se joue, il ne s'affecte pas**, et c'est [input.State] qui l'impose : il
// n'offre pas de voie pour poser un masque d'un coup, parce qu'une affectation perdrait
// le clic qui commence et finit entre deux pas. Presser puis relâcher laisse au contraire
// une trace que le pas voit.
//
// Le masque de la page coïncide bit pour bit avec celui du moteur — principal, secondaire,
// auxiliaire, quatrième, cinquième —, donc il n'y a aucune table à tenir. C'est
// `PointerEvent.buttons` et jamais `button`, qui vaut moins un sur un mouvement et dont
// l'ordre n'est pas celui du moteur.
func playButtons(state *input.State, before, now uint8) uint8 {
	changed := before ^ now
	for b := input.ButtonLeft; b <= input.ButtonForward; b++ {
		bit := uint8(1) << uint8(b)
		if changed&bit == 0 {
			continue
		}
		if now&bit != 0 {
			state.PressButton(b)
		} else {
			state.ReleaseButton(b)
		}
	}
	return now
}

// Les unités dans lesquelles une page exprime un défilement, et ce qu'un cran y vaut.
//
// **Ce sont des conventions de fait et non des valeurs normatives** : la spécification des
// événements de roue ne fixe aucune correspondance entre un delta et un cran, et laisse
// même le mode au choix de l'implémentation. Les deux premières valeurs sont celles que
// les navigateurs emploient — cent pixels ou trois lignes par cran —, la troisième étant
// la page elle-même.
const (
	deltaPixels = 0
	deltaLines  = 1

	pixelsPerNotch = 100.0
	linesPerNotch  = 3.0
)

// wheel accumule ce qu'une page rapporte de défilement et n'en rend que des crans
// entiers.
//
// **Un pavé tactile défile par fractions de cran**, et la division entière des autres
// familles les perdrait toutes : le système y crante avant de rapporter, une page non. Le
// reste se garde donc d'un événement à l'autre, et c'est le seul état que la traduction
// du pointeur retient.
type wheel struct {
	// notches est ce qui n'a pas encore fait un cran entier, dans l'unité du cran.
	notches float64
}

// add convertit un défilement en crans et rend ceux qui sont entiers, positifs vers le
// haut.
//
// Le signe s'inverse : une page compte vers le bas, [input.State.Wheel] vers le haut.
//
// Un mode que la page n'exprime pas en pixels ni en lignes est pris pour des pages, un
// cran valant alors la page — c'est ce que le troisième mode désigne, et il n'a pas
// d'autre lecture utile.
func (w *wheel) add(delta float64, mode int) int {
	if math.IsNaN(delta) || math.IsInf(delta, 0) {
		return 0
	}

	switch mode {
	case deltaPixels:
		w.notches -= delta / pixelsPerNotch
	case deltaLines:
		w.notches -= delta / linesPerNotch
	default:
		w.notches -= delta
	}

	steps := math.Trunc(w.notches)
	w.notches -= steps
	return int(clampToInt16(steps))
}
