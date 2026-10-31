// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'assemblage du backend Windows sur une vraie fenêtre : création de classe,
// procédure réelle, pompe de messages et sortie propre.

package backend

import (
	"os"
	"testing"
)

// windowRequired est la variable par laquelle un job d'intégration continue interdit au
// cas de fenêtre d'être sauté, sur le modèle de `LOZENGINE_DISPLAY` côté X11.
const windowRequired = "LOZENGINE_WINDOW"

// TestOpenDrivesARealWindow ouvre une vraie fenêtre, la pompe, lui envoie la fermeture
// par la file du système, et vérifie qu'elle se déclare fermée.
//
// **C'est un renversement de la décision 11 du jalon 2**, qui posait qu'aucun test du
// backend n'ouvre de fenêtre. Ce qui a changé depuis : le backend Linux a résolu le même
// problème par un affichage sans écran et un refus-de-sauter, donc le précédent existe ;
// et le coût du statu quo s'est accumulé — Windows était la seule famille livrée dont
// aucun contrôle n'exerçait `Open`, la création de classe, la procédure réelle ni la
// pompe, là que Linux en a deux. `DECISIONS.md` porte l'arbitrage et son écarté.
//
// **Ce que le cas couvre, et ce qu'il laisse dehors.** Il couvre l'enchaînement que les
// cas sans écran ne peuvent pas atteindre : la classe s'enregistre, la fenêtre se crée,
// `byHandle` la retrouve, la pompe dispatche vers la vraie procédure, et `Close` rend ce
// qu'`Open` a pris. Il ne relit **aucun pixel** : la présentation est le seul régime dont
// le comportement n'est pas garanti dans la session d'un runner, et le laisser dehors
// suffit à couvrir tout le reste.
//
// **Le saut doit se voir, ou il ne sert à rien.** Sans station de fenêtre, le cas est
// sauté avec son motif ; avec `LOZENGINE_WINDOW` posée, il échoue au lieu d'être sauté —
// sans quoi un runner qui refuserait les fenêtres rendrait un vert que personne ne
// distinguerait d'un vrai.
func TestOpenDrivesARealWindow(t *testing.T) {
	w, err := Open(Options{Title: "lozengine", Width: 320, Height: 180})
	if err != nil {
		if os.Getenv(windowRequired) != "" {
			t.Fatalf("%s est posée et l'ouverture a échoué : %v", windowRequired, err)
		}
		t.Skipf("aucune station de fenêtre : %v", err)
	}
	defer w.Close()

	if w.Closed() {
		t.Fatal("la fenêtre se déclare fermée dès l'ouverture")
	}
	if width, height := w.Size(); width == 0 || height == 0 {
		t.Errorf("zone utile %d×%d, attendue non nulle", width, height)
	}

	// La poignée réelle, pour poster dans la file du système plutôt que d'appeler la
	// procédure à la main : c'est la pompe qu'on veut éprouver, et elle seule
	// distingue ce cas des autres du paquet.
	real, ok := w.(*window)
	if !ok {
		t.Fatalf("Open rend un %T, attendu *window", w)
	}
	if real.hwnd == 0 {
		t.Fatal("la fenêtre est ouverte sans poignée")
	}

	// `PostMessageW` et non `SendMessageW` : le second appellerait la procédure
	// directement, sans passer par la file — donc sans rien dire de la pompe.
	if ret, _, errno := postMessageW.Call(real.hwnd, wmClose, 0, 0); ret == 0 {
		t.Fatalf("PostMessageW a échoué : %v", errno)
	}

	if w.Closed() {
		t.Fatal("la fenêtre est fermée avant que la pompe ait tourné")
	}

	w.Input()

	if !w.Closed() {
		t.Error("la fermeture postée n'a pas traversé la pompe")
	}
	if err := w.Err(); err != nil {
		t.Errorf("fermeture par le joueur rapportée comme une erreur : %v", err)
	}
}
