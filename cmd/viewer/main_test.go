// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadInputLinesWithoutFile vérifie qu'un chemin vide ne rend ni entrées ni erreur :
// rejouer sans fichier d'entrées est le cas courant, pas une omission.
func TestReadInputLinesWithoutFile(t *testing.T) {
	lines, err := readInputLines("")
	if err != nil {
		t.Fatalf("chemin vide : %v", err)
	}
	if lines != nil {
		t.Errorf("entrées : %v, attendu aucune", lines)
	}
}

// TestReadInputLinesMissingFile vérifie que l'erreur nomme ce qu'on tentait de faire.
//
// Un `open : no such file` nu laisse chercher lequel des deux chemins de la ligne de
// commande était faux, celui des images ou celui des entrées.
func TestReadInputLinesMissingFile(t *testing.T) {
	_, err := readInputLines(filepath.Join(t.TempDir(), "absent.txt"))
	if err == nil {
		t.Fatal("un fichier absent devrait échouer")
	}
	if !strings.Contains(err.Error(), "ouverture des entrées") {
		t.Errorf("erreur : %v, attendu le contexte de l'ouverture", err)
	}
}

// TestReadInputLinesFile vérifie qu'une ligne du fichier devient une entrée d'image.
func TestReadInputLinesFile(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(chemin, []byte("sans-voile\npion 4 8\n"), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	lines, err := readInputLines(chemin)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("entrées : %d, attendu 2", len(lines))
	}
	if string(lines[0]) != "sans-voile" {
		t.Errorf("première entrée : %q, attendu sans-voile", lines[0])
	}
}

// TestRunUnknownScenario vérifie que l'erreur énumère les scénarios connus.
//
// C'est la seule aide qu'un appelant reçoit : la commande n'a pas d'autre moyen de
// dire ce qu'elle accepte, et une faute de frappe est le cas le plus fréquent.
func TestRunUnknownScenario(t *testing.T) {
	err := run("introuvable", t.TempDir(), "", 1, 1)
	if err == nil {
		t.Fatal("un scénario inconnu devrait échouer")
	}
	if !strings.Contains(err.Error(), "connus :") {
		t.Errorf("erreur : %v, attendu la liste des scénarios", err)
	}
}

// TestRunWritesTheImages vérifie le chemin complet de la commande : un scénario
// monté, rejoué, et ses images écrites là où on les demande.
//
// Les images vont dans un répertoire temporaire que le test efface : ce sont des PNG
// et non des exécutables, donc rien n'oblige à les écrire dans le dépôt.
func TestRunWritesTheImages(t *testing.T) {
	dir := t.TempDir()

	if err := run("target", dir, "", 1, 2); err != nil {
		t.Fatalf("rejeu : %v", err)
	}

	images, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		t.Fatalf("relevé : %v", err)
	}
	if len(images) != 2 {
		t.Errorf("images écrites : %d, attendu 2", len(images))
	}
}
