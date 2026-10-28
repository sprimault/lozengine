// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la cadence doit tenir : la même image pour le même pas, une animation jouée
// une fois qui s'arrête sur sa fin, et le fait qui justifie de ne pas choisir à la
// place du jeu ce que le pas compte.

package appearance

import (
	"math"
	"testing"

	"github.com/sprimault/lozengine/geometry"
)

// TestLoopRepeatsAndHolds vérifie la boucle, la durée d'une image et le décalage.
func TestLoopRepeatsAndHolds(t *testing.T) {
	for _, c := range []struct {
		name   string
		clip   Clip
		step   int
		offset int
		want   int
	}{
		{"première image au premier pas", Clip{Frames: 4}, 0, 0, 0},
		{"une image par pas", Clip{Frames: 4}, 2, 0, 2},
		{"la boucle revient au début", Clip{Frames: 4}, 4, 0, 0},
		{"deux pas par image", Clip{Frames: 4, Hold: 2}, 3, 0, 1},
		{"le décalage avance la lecture", Clip{Frames: 4}, 0, 3, 3},
		{"le décalage boucle aussi", Clip{Frames: 4}, 0, 5, 1},
		{"une seule image ne bouge pas", Clip{Frames: 1}, 7, 3, 0},
		{"la valeur zéro ne bouge pas", Clip{}, 7, 3, 0},
	} {
		if got := c.clip.Loop(c.step, c.offset); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.name, got, c.want)
		}
	}
}

// TestLoopIsAFunctionOfItsStep garde ce dont la suite de non-régression dépend : deux
// lectures du même pas donnent la même image.
//
// Le cas ne peut pas rougir tant que la fonction reste pure, et c'est dit ici plutôt
// que laissé à croire qu'il protège autre chose. Ce qu'il garde est qu'elle le reste —
// un état ajouté dans [Clip], un compteur de paquet, une horloge le feraient tomber.
func TestLoopIsAFunctionOfItsStep(t *testing.T) {
	c := Clip{Frames: 6, Hold: 3}

	for step := range 40 {
		if first, second := c.Loop(step, 2), c.Loop(step, 2); first != second {
			t.Fatalf("pas %d : %d puis %d", step, first, second)
		}
	}
}

// TestOnceStopsOnItsLastFrame vérifie qu'une animation jouée une fois progresse, puis
// tient sur sa dernière image en se disant finie.
func TestOnceStopsOnItsLastFrame(t *testing.T) {
	c := Clip{Frames: 3, Hold: 2}

	for _, w := range []struct {
		elapsed int
		frame   int
		done    bool
	}{
		{0, 0, false},
		{1, 0, false},
		{2, 1, false},
		{4, 2, false},
		{5, 2, false},
		{6, 2, true},
		{100, 2, true},
	} {
		frame, done := c.Once(w.elapsed)
		if frame != w.frame || done != w.done {
			t.Errorf("écoulé %d : image %d finie %v, attendu %d et %v",
				w.elapsed, frame, done, w.frame, w.done)
		}
	}
}

// TestOnceRefusesNothing garde les deux entrées qu'un jeu atteint sans le vouloir : une
// cadence vide, et un temps écoulé négatif qu'une soustraction de pas produit.
func TestOnceRefusesNothing(t *testing.T) {
	if frame, done := (Clip{}).Once(5); frame != 0 || !done {
		t.Errorf("cadence vide : image %d finie %v, attendu 0 et true", frame, done)
	}
	if frame, done := (Clip{Frames: 3}).Once(-2); frame != 0 || done {
		t.Errorf("écoulé négatif : image %d finie %v, attendu 0 et false", frame, done)
	}
}

// TestAScreenStepIsNotTheSameLengthInEveryDirection mesure le fait qui justifie que
// [Clip.Loop] ne choisisse pas ce que son pas compte.
//
// Un pas du monde ne couvre pas la même distance à l'écran selon sa direction, et
// l'écart est large : cadencer une marche sur le numéro de pas fait donc glisser les
// pieds, et cadencer sur une distance ne le fait pas. Le moteur ne peut pas trancher à
// la place du jeu, le rapport dépendant d'une projection que la cadence ne voit pas —
// ce cas le chiffre pour que la godoc n'affirme rien qu'on n'ait mesuré.
func TestAScreenStepIsNotTheSameLengthInEveryDirection(t *testing.T) {
	m := geometry.Dimetric(32, 16)

	shortest, longest := math.Inf(1), 0.0
	for _, step := range screenSteps {
		x0, y0, _, _ := m.Project(8, 8, 0)
		x1, y1, _, _ := m.Project(8+step[0], 8+step[1], 0)

		dx, dy := float64(x1-x0), float64(y1-y0)
		length := math.Hypot(dx, dy)

		shortest, longest = math.Min(shortest, length), math.Max(longest, length)
	}

	// Deux : le pas le plus long vaut le double du plus court, et non quelques pour
	// cent. Si la projection changeait au point que ce ne soit plus vrai, la godoc de
	// [Clip.Loop] serait à relire plutôt que ce cas à corriger.
	if ratio := longest / shortest; math.Abs(ratio-2) > 0.01 {
		t.Errorf("rapport de %.3f entre le pas le plus long et le plus court, attendu 2",
			ratio)
	}
}
