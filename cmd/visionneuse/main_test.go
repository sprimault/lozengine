// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLireEntreesSansFichier vérifie qu'un chemin vide ne rend ni entrées ni erreur :
// rejouer sans fichier d'entrées est le cas courant, pas une omission.
func TestLireEntreesSansFichier(t *testing.T) {
	entrees, err := lireEntrees("")
	if err != nil {
		t.Fatalf("chemin vide : %v", err)
	}
	if entrees != nil {
		t.Errorf("entrées : %v, attendu aucune", entrees)
	}
}

// TestLireEntreesFichierAbsent vérifie que l'erreur nomme ce qu'on tentait de faire.
//
// Un `open : no such file` nu laisse chercher lequel des deux chemins de la ligne de
// commande était faux, celui des images ou celui des entrées.
func TestLireEntreesFichierAbsent(t *testing.T) {
	_, err := lireEntrees(filepath.Join(t.TempDir(), "absent.txt"))
	if err == nil {
		t.Fatal("un fichier absent devrait échouer")
	}
	if !strings.Contains(err.Error(), "ouverture des entrées") {
		t.Errorf("erreur : %v, attendu le contexte de l'ouverture", err)
	}
}

// TestLireEntreesFichier vérifie qu'une ligne du fichier devient une entrée d'image.
func TestLireEntreesFichier(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "entrees.txt")
	if err := os.WriteFile(chemin, []byte("sans-voile\npion 4 8\n"), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	entrees, err := lireEntrees(chemin)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(entrees) != 2 {
		t.Fatalf("entrées : %d, attendu 2", len(entrees))
	}
	if string(entrees[0]) != "sans-voile" {
		t.Errorf("première entrée : %q, attendu sans-voile", entrees[0])
	}
}

// TestRejouerScenarioInconnu vérifie que l'erreur énumère les scénarios connus.
//
// C'est la seule aide qu'un appelant reçoit : la commande n'a pas d'autre moyen de
// dire ce qu'elle accepte, et une faute de frappe est le cas le plus fréquent.
func TestRejouerScenarioInconnu(t *testing.T) {
	err := rejouer("introuvable", t.TempDir(), "", 1, 1)
	if err == nil {
		t.Fatal("un scénario inconnu devrait échouer")
	}
	if !strings.Contains(err.Error(), "connus :") {
		t.Errorf("erreur : %v, attendu la liste des scénarios", err)
	}
}

// TestRejouerEcritLesImages vérifie le chemin complet de la commande : un scénario
// monté, rejoué, et ses images écrites là où on les demande.
//
// Les images vont dans un répertoire temporaire que le test efface : ce sont des PNG
// et non des exécutables, donc rien n'oblige à les écrire dans le dépôt.
func TestRejouerEcritLesImages(t *testing.T) {
	repertoire := t.TempDir()

	if err := rejouer("mire", repertoire, "", 1, 2); err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	images, err := filepath.Glob(filepath.Join(repertoire, "*.png"))
	if err != nil {
		t.Fatalf("relevé : %v", err)
	}
	if len(images) != 2 {
		t.Errorf("images écrites : %d, attendu 2", len(images))
	}
}
