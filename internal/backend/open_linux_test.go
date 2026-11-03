// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la cascade des familles Linux doit tenir : quelle famille s'essaie, ce qui
// épingle une session sur X11, et ce que l'échec des deux rapporte.

package backend

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestWLNamedAsksTheEnvironmentAndNotTheFilesystem garde la question que la cascade pose.
//
// **Elle ne se confond pas avec « la socket existe-t-elle »**, et c'est tout l'objet du
// cas : la découverte de chemin se replie sur `wayland-0`, donc une session X11 pure où un
// compositeur voisin tourne serait servie par lui. Le cas rougirait si quelqu'un dérivait
// la réponse de la présence d'un fichier.
func TestWLNamedAsksTheEnvironmentAndNotTheFilesystem(t *testing.T) {
	cases := []struct {
		name    string
		display string
		socket  string
		want    bool
	}{
		{"une session X11 pure, ni l'une ni l'autre", "", "", false},
		{"un bureau Wayland ordinaire", "wayland-1", "", true},
		{"un client que le compositeur a lancé", "", "7", true},
		{"les deux, ce qu'un lanceur peut laisser", "wayland-1", "7", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("WAYLAND_DISPLAY", c.display)
			t.Setenv("WAYLAND_SOCKET", c.socket)

			// Le répertoire d'exécution est posé dans les quatre régimes : la
			// réponse n'en dépend pas, et le laisser varier masquerait une
			// écriture qui le consulterait.
			t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

			if got := wlNamed(); got != c.want {
				t.Errorf("WAYLAND_DISPLAY=%q WAYLAND_SOCKET=%q a rendu %v, attendu %v",
					c.display, c.socket, got, c.want)
			}
		})
	}
}

// TestOpenPinsASessionOnX11WhenNoCompositorIsNamed garde l'épinglage par absence.
//
// **C'est le cas qui porte le lot, et il se lit dans son message d'erreur.** Sans
// compositeur nommé, la cascade ne doit pas même essayer Wayland : l'erreur rendue est
// alors celle d'X11 seule, et elle nomme `DISPLAY`. Le cas rougit sur l'écriture qui
// paraît la plus naturelle — essayer les deux familles d'office —, puisque l'erreur
// porterait aussi la cause Wayland.
//
// C'est le régime du job qui éprouve Xwayland : il pose `DISPLAY` et le répertoire
// d'exécution, jamais `WAYLAND_DISPLAY`, et c'est ainsi qu'il s'épingle sur X11 alors
// qu'un compositeur tourne à côté.
func TestOpenPinsASessionOnX11WhenNoCompositorIsNamed(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("WAYLAND_SOCKET", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("DISPLAY", "")

	w, err := Open(wlTestOptions())
	if err == nil {
		w.Close()
		t.Fatal("une fenêtre s'est ouverte sans serveur d'affichage")
	}

	if !strings.Contains(err.Error(), "DISPLAY") {
		t.Errorf("l'erreur ne nomme pas DISPLAY : %v", err)
	}
	if strings.Contains(err.Error(), "compositor") {
		t.Errorf("la cascade a essayé Wayland alors que rien ne le nommait : %v", err)
	}
}

// TestOpenReportsBothCausesWhenNeitherFamilyAnswers garde ce qu'un échec complet dit.
//
// Une cause seule ne dirait pas si c'est le compositeur qui manquait ou le serveur X, et
// c'est exactement la question qu'on se pose devant un jeu qui ne s'ouvre pas. Le cas
// nomme un compositeur qui n'existe pas, ce qui est le seul moyen de faire échouer la
// première famille après l'avoir réellement essayée.
func TestOpenReportsBothCausesWhenNeitherFamilyAnswers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WAYLAND_DISPLAY", filepath.Join(dir, "compositeur-absent"))
	t.Setenv("WAYLAND_SOCKET", "")
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("DISPLAY", "")

	w, err := Open(wlTestOptions())
	if err == nil {
		w.Close()
		t.Fatal("une fenêtre s'est ouverte sans compositeur ni serveur d'affichage")
	}

	for _, want := range []string{"compositor", "DISPLAY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("l'erreur ne porte pas la cause %q :\n%v", want, err)
		}
	}
}
