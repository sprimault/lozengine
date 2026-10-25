// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le décodage du clavier éprouvé sans clavier : un message de touche est un numéro et
// un mot de trente-deux bits, donc il se fabrique.
//
// C'est là que vivent les défauts de cette partie du backend — un décalage de bits, un
// bit d'extension perdu, une variante système oubliée —, et aucun ne demande qu'on
// appuie sur quoi que ce soit.

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/input"
)

// lParamOf fabrique le mot d'un message de touche pour un code de balayage donné.
func lParamOf(scanCode uintptr, extended bool) uintptr {
	lParam := scanCode << scanCodeShift
	if extended {
		lParam |= extendedBit
	}
	return lParam
}

// TestKeyOfMatchesThePublishedKeys relie le décodage du backend aux places que le
// paquet des entrées publie, et c'est le seul cas à le pouvoir.
//
// Les deux côtés portent le bit d'extension séparément : le paquet des entrées ne
// l'exporte pas, puisqu'un jeu nomme les places au lieu de les composer. Si l'un des
// deux changeait, rien ne le signalerait à la compilation et un jeu qui lit les
// flèches cesserait de les voir.
func TestKeyOfMatchesThePublishedKeys(t *testing.T) {
	cases := []struct {
		name     string
		scanCode uintptr
		extended bool
		want     input.Key
	}{
		{"la place du W américain", 0x11, false, input.KeyW},
		{"la place du A américain", 0x1E, false, input.KeyA},
		{"espace", 0x39, false, input.KeySpace},
		{"échap", 0x01, false, input.KeyEscape},
		{"flèche gauche, étendue", 0x4B, true, input.KeyLeft},
		{"flèche haut, étendue", 0x48, true, input.KeyUp},
		{"contrôle droit, étendu", 0x1D, true, input.KeyRightControl},
		{"contrôle gauche, non étendu", 0x1D, false, input.KeyLeftControl},
		{"F12", 0x58, false, input.KeyF12},
		{"suppression, étendue", 0x53, true, input.KeyDelete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := keyOf(lParamOf(c.scanCode, c.extended)); got != c.want {
				t.Errorf("place : %#x, attendu %#x", got, c.want)
			}
		})
	}
}

// TestExtendedSeparatesTwoPlaces garde la raison d'être du neuvième bit : deux places
// portent le même code de base, et sans lui un jeu qui lit les flèches répondrait au
// pavé numérique.
func TestExtendedSeparatesTwoPlaces(t *testing.T) {
	const shared = 0x4B // la flèche gauche et le 4 du pavé numérique

	arrow := keyOf(lParamOf(shared, true))
	numpad := keyOf(lParamOf(shared, false))

	if arrow == numpad {
		t.Fatalf("les deux places rendent %#x", arrow)
	}
	if arrow != input.KeyLeft {
		t.Errorf("la flèche rend %#x, attendu %#x", arrow, input.KeyLeft)
	}
}

// TestKeyMessagesPressAndRelease vérifie les quatre messages, dont les deux variantes
// système.
//
// Celles-ci arrivent quand Alt est tenu. Ne traiter que les deux premières laisserait
// une touche enfoncée pour toujours dès qu'un joueur garde Alt appuyé — le même défaut
// que la perte de focus, par une autre porte.
func TestKeyMessagesPressAndRelease(t *testing.T) {
	cases := []struct {
		name string
		down uint32
		up   uint32
	}{
		{"ordinaires", wmKeyDown, wmKeyUp},
		{"système, Alt tenu", wmSysKeyDown, wmSysKeyUp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var state input.State
			lParam := lParamOf(0x11, false) // la place du W américain

			if !keyMessage(c.down, lParam, &state) {
				t.Fatal("le message d'appui n'a pas été reconnu")
			}
			if !state.Frame().Held(input.KeyW) {
				t.Error("la touche n'est pas tenue après l'appui")
			}

			if !keyMessage(c.up, lParam, &state) {
				t.Fatal("le message de relâchement n'a pas été reconnu")
			}
			if state.Frame().Held(input.KeyW) {
				t.Error("la touche est encore tenue après le relâchement")
			}
		})
	}
}

// TestKeyMessageIgnoresTheRest vérifie que le décodage ne prétend pas traiter ce qui
// n'est pas une touche : la procédure de fenêtre s'en sert pour décider si elle a
// fini, et un faux positif avalerait un message de fermeture.
func TestKeyMessageIgnoresTheRest(t *testing.T) {
	var state input.State

	for _, msg := range []uint32{wmClose, wmDestroy, wmSize, wmKillFocus, 0x0200} {
		if keyMessage(msg, 0, &state) {
			t.Errorf("le message %#04x a été pris pour une touche", msg)
		}
	}
}

// TestRepeatDoesNotRearmTheEdge vérifie ce que la répétition automatique ne doit pas
// faire : rendre une touche montée à chaque image.
//
// Le système renvoie des appuis tant que le doigt reste posé. Si chacun armait le
// front montant, une action à consommer — tirer, valider — partirait en rafale.
func TestRepeatDoesNotRearmTheEdge(t *testing.T) {
	var state input.State
	lParam := lParamOf(0x39, false) // espace

	keyMessage(wmKeyDown, lParam, &state)
	if !state.Frame().JustPressed(input.KeySpace) {
		t.Fatal("le premier appui n'a pas armé le front montant")
	}

	// La répétition arrive, et le pas suivant ne doit plus voir de montée.
	keyMessage(wmKeyDown, lParam, &state)
	frame := state.Frame()
	if frame.JustPressed(input.KeySpace) {
		t.Error("la répétition a réarmé le front montant")
	}
	if !frame.Held(input.KeySpace) {
		t.Error("la touche devrait rester tenue")
	}
}

// TestKeyMessagesThroughTheWindowProcedure vérifie le câblage, et non plus seulement
// le décodage : la procédure doit passer les messages de touche au décodage et
// répondre qu'elle a fini.
func TestKeyMessagesThroughTheWindowProcedure(t *testing.T) {
	var w window
	track(t, &w)

	windowProc(w.hwnd, wmKeyDown, 0, lParamOf(0x1F, false)) // la place du S américain
	if !w.state.Frame().Held(input.KeyS) {
		t.Error("la touche n'est pas arrivée jusqu'à l'état de la fenêtre")
	}
}

// TestResyncFindsTheHeldPlaces garde le défaut que la vérification manuelle a trouvé :
// un doigt posé avant la prise de focus n'envoie rien à la fenêtre, donc il faut
// relire le matériel.
//
// La lecture est simulée, ce qui rend le cas déterministe — sans cela il faudrait
// quelqu'un pour tenir une touche pendant la suite. Ce qu'il vérifie vraiment est la
// traduction : la place du W américain porte le code virtuel du Z sur un clavier
// français, et c'est cette place-là qui doit se retrouver tenue.
func TestResyncFindsTheHeldPlaces(t *testing.T) {
	// Le code virtuel de la place qu'on veut voir posée, demandé au système comme
	// la boucle le fait, pour que le cas vaille sur n'importe quelle disposition.
	vk, _, _ := mapVirtualKeyW.Call(0x11, mapvkVscToVkEx)
	if vk == 0 {
		t.Skip("cette disposition ne donne aucun code virtuel à la place du W")
	}

	var w window
	w.resyncWith(func(demande uintptr) bool { return demande == vk })

	frame := w.state.Frame()
	if !frame.Held(input.KeyW) {
		t.Error("la place tenue n'a pas été retrouvée")
	}
	if frame.Held(input.KeyA) {
		t.Error("une place qui n'était pas tenue a été posée")
	}
}

// TestResyncForgetsWhatIsNoLongerHeld vérifie que la relecture remplace l'état au lieu
// de s'y ajouter : une touche tenue avant un aller-retour de focus, puis relâchée
// pendant, ne doit pas revenir tenue.
func TestResyncForgetsWhatIsNoLongerHeld(t *testing.T) {
	var w window
	w.state.Press(input.KeySpace)

	w.resyncWith(func(uintptr) bool { return false })

	if w.state.Frame().Held(input.KeySpace) {
		t.Error("une touche relâchée pendant l'absence est revenue tenue")
	}
}
