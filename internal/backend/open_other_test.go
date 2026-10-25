// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

//go:build !windows

package backend

import (
	"runtime"
	"strings"
	"testing"
)

// TestOuvrirReplieSansFenetre vérifie le seul comportement que ce lot livre : sur une
// plateforme dont le backend n'est pas écrit, l'ouverture échoue proprement au lieu de
// rendre une fenêtre inutilisable.
//
// Le cas vérifie aussi que le message nomme la plateforme et le jalon. Une erreur qui
// dirait seulement « non pris en charge » obligerait à chercher dans la feuille de
// route ce que le moteur attend encore, et c'est précisément ce qu'on veut éviter à
// quelqu'un qui essaie le moteur sous Linux aujourd'hui.
func TestOpenFallsBackWithoutWindow(t *testing.T) {
	window, err := Open(Options{Title: "essai", Width: 480, Height: 270})

	if err == nil {
		t.Fatal("ouverture acceptée sur une plateforme sans backend")
	}
	if window != nil {
		t.Error("une fenêtre est rendue en même temps que l'erreur")
	}
	if !strings.Contains(err.Error(), runtime.GOOS) {
		t.Errorf("le message ne nomme pas la plateforme : %v", err)
	}
	if !strings.Contains(err.Error(), "milestone") {
		t.Errorf("le message ne nomme pas le jalon qui l'écrira : %v", err)
	}
}
