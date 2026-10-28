// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la rotation des directions doit tenir : l'accord avec la rotation que la
// scène applique aux points, qui est la seule chose capable de dire si le décalage va
// dans le bon sens.

package appearance

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/scene"
)

// screenSteps donne les huit directions, du sud et dans le sens horaire à l'écran, par
// le pas du monde qui les produit.
//
// La projection envoie un pas (dx, dy) sur (dy−dx, dx+dy) à un facteur près, ce qui
// range ces huit pas dans cet ordre. La table est écrite plutôt que calculée pour que
// le cas qui s'en sert ne dépende pas de la fonction qu'il éprouve.
var screenSteps = [8][2]float64{
	{1, 1},   // sud
	{1, 0},   // sud-ouest
	{1, -1},  // ouest
	{0, -1},  // nord-ouest
	{-1, -1}, // nord
	{-1, 0},  // nord-est
	{-1, 1},  // est
	{0, 1},   // sud-est
}

// remappedStep rend le pas que l'orientation donne à un pas du monde.
//
// Il se mesure sur la scène, par la différence de deux points remappés, plutôt que de
// se recalculer ici : c'est ce qui relie le cas à la rotation réelle, et non à l'idée
// qu'on s'en fait.
func remappedStep(t *testing.T, o scene.Orientation, dx, dy float64) (float64, float64) {
	t.Helper()

	s, err := scene.New(16, 16, geometry.Dimetric(32, 16), scene.Options{})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Orient(o)

	x0, y0 := s.Remap(8, 8)
	x1, y1 := s.Remap(8+dx, 8+dy)
	return x1 - x0, y1 - y0
}

// stepIndex rend le rang d'un pas du monde dans [screenSteps].
func stepIndex(t *testing.T, dx, dy float64) int {
	t.Helper()

	for i, s := range screenSteps {
		if s[0] == dx && s[1] == dy {
			return i
		}
	}
	t.Fatalf("pas (%g, %g) absent de la table", dx, dy)
	return 0
}

// TestTurnFollowsTheCameraRotation vérifie que le décalage des directions va dans le
// même sens que la rotation que la scène applique aux points.
//
// **C'est le seul cas capable d'attraper un sens inversé**, et c'est le défaut que la
// fonction peut avoir sans que rien d'autre ne bronche : une apparence resterait
// cohérente avec elle-même à chaque orientation, et montrerait seulement un personnage
// tourné à l'envers — ce qui se lit comme un défaut de planches.
//
// La vérification ne recalcule pas la rotation : elle la mesure sur la scène, puis
// demande à [Turn] la même chose et compare. Les deux chemins sont indépendants, donc
// un sens inversé les sépare.
func TestTurnFollowsTheCameraRotation(t *testing.T) {
	for _, o := range []scene.Orientation{
		scene.NoRotation, scene.QuarterTurn, scene.HalfTurn, scene.ThreeQuarters,
	} {
		for facing, step := range screenSteps {
			dx, dy := remappedStep(t, o, step[0], step[1])

			want := stepIndex(t, dx, dy)
			if got := Turn(Facing(facing), o, len(screenSteps)); int(got) != want {
				t.Errorf("orientation %d, direction %d : %d, attendu %d",
					o, facing, got, want)
			}
		}
	}
}

// TestTurnLeavesADirectionAloneWithoutRotation garde le cas d'un jeu dont la caméra ne
// tourne pas, qui doit pouvoir appeler la fonction sans y penser.
func TestTurnLeavesADirectionAloneWithoutRotation(t *testing.T) {
	for facing := range screenSteps {
		if got := Turn(Facing(facing), scene.NoRotation, 8); int(got) != facing {
			t.Errorf("direction %d : %d, attendu %d", facing, got, facing)
		}
	}
}

// TestTurnRefusesNothingOnAnEmptyProvider garde le repli d'un fournisseur qui ne
// déclare aucune direction.
//
// Zéro n'est pas une valeur légitime, mais la valeur zéro d'un entier l'atteint sans
// qu'on l'ait voulu — et une division par elle arrêterait le programme là où le jeu
// attend une image. La direction revient alors telle quelle.
func TestTurnRefusesNothingOnAnEmptyProvider(t *testing.T) {
	if got := Turn(3, scene.QuarterTurn, 0); got != 3 {
		t.Errorf("%d, attendu 3", got)
	}
}
