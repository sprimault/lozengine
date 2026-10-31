// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la liste des poses en pixels doit tenir : une seule liste pour les deux
// repères, un vidage qui garde la capacité, et l'ordre de pose comme ordre de dessin.

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestClearScreenKeepsTheCapacity garde la promesse qui fait de la liste des pièces un
// magasin et non une allocation par image.
//
// `ClearScreen` n'avait aucun cas : la remplacer par `s.screen = nil` laissait toute la
// suite verte, alors que sa godoc promet de garder la capacité — et c'est cette promesse
// qui tient la seule réserve à l'absence d'allocation par image. Un jeu qui pose puis
// vide à chaque image réallouerait soixante fois par seconde sans que rien ne le dise.
//
// **Les deux repères passent par la même liste**, donc le cas les mêle : une pose de
// surface et une pose d'aire, vidées ensemble, et la capacité doit survivre aux deux.
func TestClearScreenKeepsTheCapacity(t *testing.T) {
	s := layoutScene(t, 480, 270)
	a := Aspect{Width: 8, Height: 8, Tint: render.White()}

	for i := int32(0); i < 8; i++ {
		if !s.PlaceScreen(i, i, a) {
			t.Fatalf("pose de surface %d refusée", i)
		}
		if !s.PlaceArea(i, i, a) {
			t.Fatalf("pose d'aire %d refusée", i)
		}
	}

	grown := cap(s.screen)
	if grown < 16 {
		t.Fatalf("capacité de %d après seize poses", grown)
	}

	s.ClearScreen()

	if n := len(s.screen); n != 0 {
		t.Errorf("%d pièces après le vidage, attendu aucune", n)
	}
	if got := cap(s.screen); got != grown {
		t.Errorf("capacité %d après le vidage, attendue %d", got, grown)
	}
}

// TestBothPlacementsShareOneList garde que l'ordre de pose est l'ordre de dessin entre
// les deux repères, et non deux ordres qui se recolleraient.
//
// C'est ce qu'une ligne de texte posée sur l'aire au-dessus d'un panneau de surface
// demande : à couche égale, ce qui est posé après recouvre ce qui précède, quel que soit
// le rectangle que chacun vise. Deux listes séparées rendraient cet ordre dépendant de
// celle qu'on parcourt d'abord.
func TestBothPlacementsShareOneList(t *testing.T) {
	s := layoutScene(t, 480, 270)
	a := Aspect{Width: 8, Height: 8, Tint: render.White()}

	s.PlaceScreen(0, 0, a)
	s.PlaceArea(1, 1, a)
	s.PlaceScreen(2, 2, a)

	if n := len(s.screen); n != 3 {
		t.Fatalf("%d pièces, attendu 3", n)
	}
	for i, want := range []bool{false, true, false} {
		if got := s.screen[i].InArea; got != want {
			t.Errorf("pièce %d : InArea = %v, attendu %v — l'ordre de pose n'est pas tenu",
				i, got, want)
		}
	}
}

// TestCornerFromFootUndoesThePlacement garde que la conversion est l'exact inverse de ce
// que la pose par le pied fait, division entière comprise.
//
// **La parité de la largeur est tout le sujet** : la pose interne divise par deux en
// entiers, donc une largeur impaire est le seul endroit où les deux calculs pourraient
// diverger d'un pixel. Le cas rougit si la méthode arrondit dans l'autre sens, ou si elle
// cesse de retrancher toute la hauteur.
//
// Ce qu'il ne garde pas, et c'est ailleurs : que le coin rendu pose vraiment l'image là
// où une entité atterrit. Cela demande un trieur, donc
// `TestScreenPieceByTheFootMatchesAnEntity` et `TestAreaPieceLandsWhereAnEntityLands` dans
// `internal/sorting`, qui passent tous deux par cette méthode.
func TestCornerFromFootUndoesThePlacement(t *testing.T) {
	cases := []struct {
		name          string
		width, height uint16
		ex, ey        int32
		wantX, wantY  int32
	}{
		{"largeur paire", 10, 7, 100, 50, 95, 43},
		{"largeur impaire", 11, 7, 100, 50, 95, 43},
		{"hauteur nulle", 10, 0, 100, 50, 95, 50},
		{"pixel négatif", 10, 7, -3, -2, -8, -9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := Aspect{Width: c.width, Height: c.height}
			if x, y := a.CornerFromFoot(c.ex, c.ey); x != c.wantX || y != c.wantY {
				t.Errorf("coin (%d, %d), attendu (%d, %d)", x, y, c.wantX, c.wantY)
			}
		})
	}
}

// TestCornerFromFootIgnoresTheAnchor garde ce que sa godoc promet : l'ancre n'entre pas
// dans le calcul, les deux voies de pose l'appliquant d'elles-mêmes.
//
// Sans ce cas, quelqu'un qui « corrigerait » la méthode en y ajoutant l'ancre la ferait
// compter deux fois — exactement le défaut qu'elle sert à fermer, retourné.
func TestCornerFromFootIgnoresTheAnchor(t *testing.T) {
	plain := Aspect{Width: 10, Height: 7}
	anchored := Aspect{Width: 10, Height: 7, AnchorX: 3, AnchorY: -2}

	px, py := plain.CornerFromFoot(100, 50)
	ax, ay := anchored.CornerFromFoot(100, 50)

	if px != ax || py != ay {
		t.Errorf("avec ancre (%d, %d), sans ancre (%d, %d) — l'ancre est entrée dans le calcul",
			ax, ay, px, py)
	}
}
