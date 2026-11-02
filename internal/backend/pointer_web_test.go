// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les trois conversions du pointeur d'une page, éprouvées sur l'hôte : aucune ne touche à
// la page, donc leurs cas tournent partout.

package backend

import (
	"math"
	"testing"

	"github.com/sprimault/lozengine/input"
)

// TestPointerAtConvertsBetweenTheTwoUnits garde la conversion d'une position d'événement
// vers les pixels d'image, dans les deux régimes de densité et aux bornes.
//
// **Le cas de la densité entière est celui qui doit rendre l'identité**, et c'est ce qui
// épargne toute conversion sur les téléphones et les écrans fins : le canvas y porte
// autant de pixels d'image que de pixels CSS.
func TestPointerAtConvertsBetweenTheTwoUnits(t *testing.T) {
	for _, c := range []struct {
		name         string
		box          canvasBox
		offsetX      float64
		offsetY      float64
		wantX, wantY int16
	}{
		{
			"densité entière, les deux unités se confondent",
			canvasBox{width: 480, height: 270, cssWidth: 480, cssHeight: 270},
			120, 60, 120, 60,
		},
		{
			"densité de 125 %, la position se multiplie",
			canvasBox{width: 600, height: 338, cssWidth: 480, cssHeight: 270.4},
			120, 60, 150, 75,
		},
		{
			"le coin haut-gauche reste le coin",
			canvasBox{width: 600, height: 338, cssWidth: 480, cssHeight: 270.4},
			0, 0, 0, 0,
		},
		{
			"un pointeur capturé hors du canvas passe sous zéro",
			canvasBox{width: 480, height: 270, cssWidth: 480, cssHeight: 270},
			-30, -12, -30, -12,
		},
		{
			"un canvas masqué ne désigne rien",
			canvasBox{},
			120, 60, 0, 0,
		},
		{
			"une position aberrante se borne au lieu de dépendre de l'architecture",
			canvasBox{width: 480, height: 270, cssWidth: 480, cssHeight: 270},
			1e9, -1e9, math.MaxInt16, math.MinInt16,
		},
		{
			"une position indéfinie vaut l'origine",
			canvasBox{width: 480, height: 270, cssWidth: 480, cssHeight: 270},
			math.NaN(), math.NaN(), 0, 0,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			x, y := pointerAt(c.offsetX, c.offsetY, c.box)
			if x != c.wantX || y != c.wantY {
				t.Errorf("(%g, %g) donne (%d, %d), attendu (%d, %d)",
					c.offsetX, c.offsetY, x, y, c.wantX, c.wantY)
			}
		})
	}
}

// TestPlayButtonsKeepsAClickShorterThanAStep garde ce que l'affectation d'un masque
// perdrait, et c'est la raison pour laquelle [input.State] n'offre pas cette affectation.
//
// Un clic qui commence et finit entre deux pas laisse un masque final nul : seules la
// pression et le relâchement joués laissent la trace que le pas voit.
//
// **Le pas le rapporte tenu et monté**, et non seulement monté : un pas dit ce qui s'est
// passé pendant lui et non l'état à sa fin, ce que la godoc de `Frame` pose pour les
// touches comme pour les boutons. Le pas suivant, lui, ne doit plus rien en porter.
func TestPlayButtonsKeepsAClickShorterThanAStep(t *testing.T) {
	var state input.State

	mask := playButtons(&state, 0, 1)
	mask = playButtons(&state, mask, 0)
	if mask != 0 {
		t.Errorf("le masque rendu vaut %d, attendu 0", mask)
	}

	in := state.Frame()
	if !in.ButtonJustPressed(input.ButtonLeft) {
		t.Error("le clic n'a pas laissé de trace, alors qu'il a eu lieu entre deux pas")
	}
	if !in.ButtonHeld(input.ButtonLeft) {
		t.Error("le clic n'est pas tenu au pas où il a eu lieu")
	}

	next := state.Frame()
	if next.ButtonHeld(input.ButtonLeft) || next.ButtonJustPressed(input.ButtonLeft) {
		t.Error("le clic survit au pas suivant, alors qu'il était relâché")
	}
}

// TestPlayButtonsPlaysEveryChangedBit garde que les cinq places suivent le masque de la
// page, et que seules les places changées sont jouées.
//
// Le masque de la page coïncidant bit pour bit avec celui du moteur, ce cas garde surtout
// l'absence de table : une correspondance écrite à la main se tromperait ici.
func TestPlayButtonsPlaysEveryChangedBit(t *testing.T) {
	var state input.State

	// Les cinq boutons d'un coup, puis les deux latéraux relâchés.
	mask := playButtons(&state, 0, 0b11111)
	in := state.Frame()
	for b := input.ButtonLeft; b <= input.ButtonForward; b++ {
		if !in.ButtonHeld(b) {
			t.Errorf("le bouton %d n'est pas tenu alors que le masque le portait", b)
		}
	}

	playButtons(&state, mask, 0b00111)
	in = state.Frame()
	if in.ButtonHeld(input.ButtonBack) || in.ButtonHeld(input.ButtonForward) {
		t.Error("un bouton latéral reste tenu après son relâchement")
	}
	if !in.ButtonHeld(input.ButtonMiddle) {
		t.Error("le bouton du milieu a été relâché alors que le masque le portait encore")
	}
}

// TestWheelKeepsFractionsOfANotch garde ce qu'une division entière perdrait, et c'est le
// régime d'un pavé tactile : la page y rapporte des fractions de cran, là qu'un système
// crante avant de rapporter.
//
// Le signe s'inverse au passage — une page compte vers le bas, le moteur vers le haut —, et
// ce cas le garde aussi : c'est une erreur qui ne se voit qu'à l'usage, la molette
// marchant alors à l'envers.
func TestWheelKeepsFractionsOfANotch(t *testing.T) {
	t.Run("quatre fractions font un cran", func(t *testing.T) {
		var w wheel
		for i := range 3 {
			if steps := w.add(25, deltaPixels); steps != 0 {
				t.Fatalf("un cran rendu dès la fraction %d, attendu aucun", i+1)
			}
		}
		if steps := w.add(25, deltaPixels); steps != -1 {
			t.Errorf("la quatrième fraction rend %d crans, attendu -1", steps)
		}
	})

	for _, c := range []struct {
		name  string
		delta float64
		mode  int
		want  int
	}{
		{"cent pixels font un cran vers le bas", 100, deltaPixels, -1},
		{"cent pixels vers le haut", -100, deltaPixels, 1},
		{"trois lignes font un cran", 3, deltaLines, -1},
		{"une page fait un cran", 1, 2, -1},
		{"deux crans d'un coup", 200, deltaPixels, -2},
		{"un défilement indéfini ne compte pas", math.NaN(), deltaPixels, 0},
		{"un défilement infini ne compte pas", math.Inf(1), deltaPixels, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			var w wheel
			if steps := w.add(c.delta, c.mode); steps != c.want {
				t.Errorf("%g en mode %d rend %d crans, attendu %d",
					c.delta, c.mode, steps, c.want)
			}
		})
	}
}

// TestScrollKeysAreOnlyTheOnesThatScroll garde les deux côtés de la liste : ce qui défile
// est empêché, et ce qui sert au joueur ou au navigateur ne l'est pas.
//
// **Le second côté est celui qui compte**, et il rougirait si quelqu'un étendait la liste
// à tout ce que le moteur nomme : la touche d'actualisation et celle de l'onglet
// cesseraient de répondre dans une page de jeu, ce qu'aucun joueur ne pardonne.
func TestScrollKeysAreOnlyTheOnesThatScroll(t *testing.T) {
	for _, code := range []string{"Space", "ArrowUp", "ArrowDown", "ArrowLeft",
		"ArrowRight", "PageUp", "PageDown", "Home", "End"} {
		if !scrollsThePage(code) {
			t.Errorf("%q fait défiler la page et n'est pas empêchée", code)
		}
	}

	for _, code := range []string{"F5", "KeyW", "KeyR", "Tab", "Escape", "Enter",
		"ShiftLeft", "NumpadAdd", "Digit0"} {
		if scrollsThePage(code) {
			t.Errorf("%q ne fait pas défiler la page et se trouve pourtant empêchée", code)
		}
	}
}
