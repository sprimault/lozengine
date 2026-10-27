// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import "testing"

// TestDisplaySocketsReadsTheUsualForms vérifie que les formes qu'une session écrit
// réellement se lisent, numéro de session et numéro d'écran séparés.
//
// Les deux nombres ne servent pas au même : le premier nomme la socket et l'entrée
// d'autorisation, le second choisit parmi les écrans qu'un seul serveur annonce. Les
// confondre ferait chercher une socket qui n'existe pas.
func TestDisplaySocketsReadsTheUsualForms(t *testing.T) {
	cases := []struct {
		name    string
		display string
		number  int
		screen  int
	}{
		{"la forme la plus courte", ":0", 0, 0},
		{"un serveur sans écran, comme en intégration continue", ":99", 99, 0},
		{"le second écran de la même session", ":0.1", 0, 1},
		{"la socket locale nommée explicitement", "unix:12", 12, 0},
		{"un numéro à deux chiffres et un écran", "unix:10.2", 10, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paths, number, screen, err := displaySockets(c.display)
			if err != nil {
				t.Fatalf("DISPLAY=%q refusé : %v", c.display, err)
			}
			if number != c.number {
				t.Errorf("numéro de session %d, attendu %d", number, c.number)
			}
			if screen != c.screen {
				t.Errorf("numéro d'écran %d, attendu %d", screen, c.screen)
			}
			if len(paths) == 0 {
				t.Fatal("aucun chemin de socket rendu")
			}
		})
	}
}

// TestDisplaySocketsOffersBothSocketForms vérifie que les deux formes de socket sont
// proposées, et dans l'ordre.
//
// C'est ce qui permet à un serveur sans écran d'être joint quand il n'offre que la
// socket abstraite, propre à Linux et sans entrée dans l'arborescence : un seul chemin
// ferait échouer l'ouverture là où le serveur écoute pourtant.
func TestDisplaySocketsOffersBothSocketForms(t *testing.T) {
	paths, _, _, err := displaySockets(":99")
	if err != nil {
		t.Fatalf("DISPLAY=:99 refusé : %v", err)
	}

	want := []string{"/tmp/.X11-unix/X99", "@/tmp/.X11-unix/X99"}
	if len(paths) != len(want) {
		t.Fatalf("%d chemin(s) rendu(s), %d attendus : %q", len(paths), len(want), paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("chemin %d vaut %q, attendu %q", i, paths[i], want[i])
		}
	}
}

// TestDisplaySocketsRefusesWhatItCannotServe vérifie que ce qui ne peut pas aboutir
// est refusé ici, avec un message, plutôt que plus loin sur un échec de connexion.
//
// Le cas qui compte est celui de la machine nommée : le refus dit que seules les
// sessions locales sont servies, là où une tentative de connexion rendrait
// « connexion refusée » et laisserait chercher un serveur qui tourne pourtant.
func TestDisplaySocketsRefusesWhatItCannotServe(t *testing.T) {
	cases := []struct {
		name    string
		display string
	}{
		{"la variable absente", ""},
		{"une machine nommée", "ailleurs:0"},
		{"une machine nommée par une adresse", "192.168.1.2:0"},
		{"une adresse IPv6, dont les deux-points piègent le découpage", "::1:0"},
		{"aucun numéro de session", "unix"},
		{"un numéro de session illisible", ":eh"},
		{"un numéro d'écran illisible", ":0.eh"},
		{"un numéro de session vide", ":"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paths, _, _, err := displaySockets(c.display)
			if err == nil {
				t.Fatalf("DISPLAY=%q accepté, chemins %q", c.display, paths)
			}
		})
	}
}
