// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le pointeur doit tenir : les deux régimes d'un bouton comme ceux d'une
// touche, une molette qui se compte au lieu de se tenir, et une position qui survit à
// la sortie de la fenêtre.

package input

import "testing"

// TestButtonHasTheTwoRegimes vérifie qu'un bouton se lit tenu et monté, et que la
// montée ne vaut qu'une fois par appui.
func TestButtonHasTheTwoRegimes(t *testing.T) {
	var s State

	s.PressButton(ButtonLeft)
	first := s.Frame()
	if !first.ButtonHeld(ButtonLeft) || !first.ButtonJustPressed(ButtonLeft) {
		t.Fatal("premier pas : le bouton devrait être tenu et monté")
	}

	second := s.Frame()
	if !second.ButtonHeld(ButtonLeft) {
		t.Error("second pas : le bouton devrait rester tenu")
	}
	if second.ButtonJustPressed(ButtonLeft) {
		t.Error("second pas : la montée devrait être retombée")
	}

	s.ReleaseButton(ButtonLeft)
	if third := s.Frame(); third.ButtonHeld(ButtonLeft) {
		t.Error("après relâchement : le bouton ne devrait plus être tenu")
	}
}

// TestButtonPressedAndReleasedWithinAStepIsSeen garde ce que le clavier garde déjà :
// un appui plus court qu'un pas ne disparaît pas.
//
// C'est le défaut qui avait fait « une touche sur deux ne répond pas » à la
// vérification manuelle du backend Windows, et un clic est plus bref qu'un appui de
// touche, donc il l'atteint plus souvent.
func TestButtonPressedAndReleasedWithinAStepIsSeen(t *testing.T) {
	var s State

	s.PressButton(ButtonRight)
	s.ReleaseButton(ButtonRight)

	if frame := s.Frame(); !frame.ButtonJustPressed(ButtonRight) {
		t.Error("un clic entre deux pas devrait être vu")
	}
}

// TestWheelAccumulatesThenResets vérifie que les crans s'ajoutent dans le pas et
// repartent de zéro au suivant.
func TestWheelAccumulatesThenResets(t *testing.T) {
	var s State

	s.Wheel(2)
	s.Wheel(-5)
	if got := s.Frame().Wheel(); got != -3 {
		t.Errorf("%d crans, attendu -3", got)
	}
	if got := s.Frame().Wheel(); got != 0 {
		t.Errorf("%d crans au pas suivant, attendu 0", got)
	}
}

// TestPointerSurvivesLeaving vérifie qu'une sortie de fenêtre garde la dernière
// position et ne la remet pas à zéro.
//
// Un jeu qui suit le curseur veut savoir par où il est parti ; des coordonnées
// effacées désigneraient le coin de l'image, ce qui est une position comme une autre
// et donc indiscernable d'un vrai pointeur là-bas.
func TestPointerSurvivesLeaving(t *testing.T) {
	var s State

	s.Move(120, 45)
	s.Leave()

	x, y, inside := s.Frame().Pointer()
	if inside {
		t.Error("le pointeur devrait être dehors")
	}
	if x != 120 || y != 45 {
		t.Errorf("position (%d, %d), attendue (120, 45)", x, y)
	}
}

// TestForgetReleasesButtonsToo garde la parade de la perte de focus côté pointeur.
//
// Sans elle, un bouton tenu au moment du changement de fenêtre le resterait
// indéfiniment : le système cesse d'envoyer les relâchements, et le jeu croirait à un
// glissement sans fin.
func TestForgetReleasesButtonsToo(t *testing.T) {
	var s State

	s.PressButton(ButtonMiddle)
	s.Frame()
	s.Forget()

	frame := s.Frame()
	if frame.ButtonHeld(ButtonMiddle) {
		t.Error("le bouton devrait être relâché")
	}
	if _, _, inside := frame.Pointer(); inside {
		t.Error("le pointeur devrait être dehors")
	}
}

// TestUnknownButtonIsRefusedQuietly garde le même régime que pour une touche inconnue :
// hors bornes rend faux au lieu de paniquer.
func TestUnknownButtonIsRefusedQuietly(t *testing.T) {
	var s State

	s.PressButton(Button(buttonsMax))

	frame := s.Frame()
	if frame.ButtonHeld(Button(buttonsMax)) || frame.ButtonJustPressed(Button(buttonsMax)) {
		t.Error("un bouton hors bornes devrait rester faux")
	}
}

// TestWithPointerReplacesOnlyThePosition vérifie que la conversion de la boucle ne
// touche ni les touches ni les boutons.
func TestWithPointerReplacesOnlyThePosition(t *testing.T) {
	var s State

	s.Press(KeyA)
	s.PressButton(ButtonLeft)
	s.Move(400, 300)

	moved := s.Frame().WithPointer(12, 7, true)

	x, y, inside := moved.Pointer()
	if x != 12 || y != 7 || !inside {
		t.Errorf("position (%d, %d) dedans=%v, attendue (12, 7) dedans", x, y, inside)
	}
	if !moved.Held(KeyA) || !moved.ButtonHeld(ButtonLeft) {
		t.Error("la conversion a perdu une touche ou un bouton")
	}
}
