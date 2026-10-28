// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La cadence d'une animation, et les deux façons de la lire : en boucle, ou une fois.
// Rien n'y est gardé d'une image à la suivante.

package appearance

// Clip décrit la cadence d'une animation : son nombre d'images, et la durée de chacune
// comptée en pas de la boucle.
//
// **Il n'y a pas d'état d'animation par entité, et c'est tout l'intérêt.** L'image
// courante se dérive du numéro de pas, donc deux rendus du même pas donnent la même
// image — c'est ce qu'une suite de non-régression qui compare au pixel exige, et cela
// retire en même temps la mémoire qu'il aurait fallu tenir pour des milliers d'entités
// et la remise en cohérence après un chargement.
//
// Sa valeur zéro rend toujours l'image zéro, ce qui convient à ce qui ne s'anime pas :
// une chose immobile n'a pas à déclarer une cadence d'une image.
type Clip struct {
	// Frames est le nombre d'images que l'animation parcourt. Zéro et un se
	// comportent pareil, l'animation tenant alors sur sa seule image.
	Frames int

	// Hold est la durée d'une image, en pas. Zéro ou moins vaut un pas, de sorte
	// qu'une cadence oubliée avance au lieu de diviser par zéro.
	Hold int
}

// Loop rend l'image d'une animation qui tourne sans fin.
//
// Le décalage évite que toutes les choses d'un même type battent du pied ensemble. Il
// se tire de ce qui distingue l'entité — son rang, son identité —, jamais d'un tirage
// au sort : celui-ci donnerait une image différente à chaque rendu du même pas, et la
// comparaison au pixel tomberait.
//
// **Ce que porte le pas décide de la cadence, et le moteur ne le choisit pas.** Un
// numéro de pas fait avancer l'animation au rythme du temps ; une distance cumulée la
// fait avancer au rythme du déplacement, ce qu'il faut pour que les pieds ne glissent
// pas. La différence n'est pas une subtilité : un pas du monde ne couvre pas la même
// distance à l'écran selon sa direction, et le rapport dépend de la projection, donc
// il n'a pas sa place dans une fonction qui ne la voit pas.
func (c Clip) Loop(step, offset int) int {
	n := c.Frames
	if n <= 1 {
		return 0
	}

	return ((step+offset)/c.hold()%n + n) % n
}

// Once rend l'image d'une animation jouée une fois, et dit si elle est finie.
//
// Une animation finie tient sur sa dernière image plutôt que de revenir à la première :
// c'est ce qu'un coup porté ou une porte ouverte demandent, et c'est au jeu de décider
// quand passer à autre chose. Revenir au début ferait clignoter ce qui devait rester.
func (c Clip) Once(elapsed int) (frame int, done bool) {
	n := c.Frames
	if n <= 1 {
		return 0, true
	}
	if elapsed < 0 {
		return 0, false
	}

	if i := elapsed / c.hold(); i < n {
		return i, false
	}
	return n - 1, true
}

// hold rend la durée d'une image, ramenée à au moins un pas.
func (c Clip) hold() int {
	if c.Hold < 1 {
		return 1
	}
	return c.Hold
}
