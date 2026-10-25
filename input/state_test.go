// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package input

import "testing"

// Deux touches arbitraires, dans deux mots différents du jeu de bits : la seconde est
// au-delà du soixante-quatrième bit, donc un décalage calculé sur le mauvais mot se
// verrait.
const (
	keyA Key = 17
	keyB Key = 130
)

// TestRisingEdgeLastsOneFrame est le cas qui porte tout l'intérêt des deux régimes, et
// celui qui casserait en silence.
//
// Une touche tenue sans interruption doit rendre JustPressed vrai à la première image
// et faux ensuite, Held restant vrai tout du long. Sans ce cas, un front montant qui se
// répéterait passerait inaperçu de la suite — l'image serait juste, et le jeu tirerait
// soixante fois par seconde.
func TestRisingEdgeLastsOneFrame(t *testing.T) {
	var state State
	state.Press(keyA)

	first := state.Frame()
	if !first.Held(keyA) || !first.JustPressed(keyA) {
		t.Fatalf("première image : tenue %v, montée %v, attendu les deux",
			first.Held(keyA), first.JustPressed(keyA))
	}

	second := state.Frame()
	if !second.Held(keyA) {
		t.Error("seconde image : touche relâchée alors que le doigt est resté")
	}
	if second.JustPressed(keyA) {
		t.Error("seconde image : la montée s'est répétée")
	}
}

// TestAutoRepeatDoesNotRearm vérifie qu'un second Press sans relâchement ne redonne pas
// de front montant.
//
// C'est le cas réel et non théorique : le système renvoie l'événement d'appui tant que
// le doigt reste posé, à la cadence de répétition du clavier. Un front par répétition
// ferait tirer en rafale sans que le joueur l'ait demandé.
func TestAutoRepeatDoesNotRearm(t *testing.T) {
	var state State
	state.Press(keyA)
	state.Frame()

	state.Press(keyA)
	if state.Frame().JustPressed(keyA) {
		t.Error("la répétition du système a rendu un nouveau front montant")
	}
}

// TestReleaseRearmsTheEdge vérifie qu'un appui suivant compte, ce qui est l'autre
// moitié du régime : sans cela, une touche ne servirait qu'une fois par partie.
func TestReleaseRearmsTheEdge(t *testing.T) {
	var state State
	state.Press(keyA)
	state.Frame()

	state.Release(keyA)
	released := state.Frame()
	if released.Held(keyA) || released.JustPressed(keyA) {
		t.Fatal("touche encore tenue après relâchement")
	}

	state.Press(keyA)
	if !state.Frame().JustPressed(keyA) {
		t.Error("le second appui n'a pas rendu de front montant")
	}
}

// TestForgetReleasesEverything couvre la perte de focus, où le système cesse d'envoyer
// les relâchements : une touche tenue à cet instant le resterait indéfiniment, et le
// personnage marcherait tout seul au retour dans la fenêtre.
func TestForgetReleasesEverything(t *testing.T) {
	var state State
	state.Press(keyA)
	state.Press(keyB)
	state.Frame()

	state.Forget()
	after := state.Frame()
	if after.Held(keyA) || after.Held(keyB) {
		t.Error("une touche est restée tenue après l'oubli")
	}

	// Et le front se réarme : revenir dans la fenêtre et appuyer doit compter.
	state.Press(keyA)
	if !state.Frame().JustPressed(keyA) {
		t.Error("l'appui qui suit un oubli n'a pas rendu de front montant")
	}
}

// TestKeysAreIndependent vérifie que deux touches de mots différents ne se marchent pas
// dessus : un décalage appliqué au mauvais mot se verrait ici, et nulle part ailleurs.
func TestKeysAreIndependent(t *testing.T) {
	var state State
	state.Press(keyB)

	frame := state.Frame()
	if !frame.Held(keyB) {
		t.Error("la touche du second mot n'est pas tenue")
	}
	if frame.Held(keyA) {
		t.Error("la touche du premier mot est tenue alors que rien ne l'a enfoncée")
	}
}

// TestFrameDoesNotAllocate vérifie que la lecture des entrées reste hors du chemin
// d'allocation, comme l'effacement et le dessin qui la suivent dans un pas.
func TestFrameDoesNotAllocate(t *testing.T) {
	var state State
	state.Press(keyA)

	if n := testing.AllocsPerRun(10, func() { state.Frame() }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
