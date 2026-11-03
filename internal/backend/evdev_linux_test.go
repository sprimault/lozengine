// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la table du noyau doit tenir une fois partagée : les codes qu'X11 refuse et
// qu'elle accepte, et les deux noms d'un même bouton latéral.

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/input"
)

// TestEvdevKeyOfTakesTheCodesX11Refuses garde la frontière entre la table et le décalage.
//
// **C'est le cas qui garde la décision du jalon, et il rougirait sur le défaut qu'elle
// nomme** : si le refus sous huit d'[x11KeyOf] descendait dans [evdevKeyOf] — ce qu'un
// partage fait naturellement, le corps étant plus court à déplacer que la table —, ces
// huit touches deviendraient muettes sous Wayland et sous `evdev`. Échap et les chiffres 1
// à 7 sont exactement les codes que le protocole X11 réserve pour autre chose.
//
// La seconde moitié garde l'autre sens : les mêmes codes, passés à [x11KeyOf], désignent
// autre chose ou rien, et c'est ce qui rend les deux fonctions non interchangeables.
func TestEvdevKeyOfTakesTheCodesX11Refuses(t *testing.T) {
	cases := []struct {
		name string
		code uint32
		want input.Key
	}{
		{"échappement, le premier code du noyau", 1, input.KeyEscape},
		{"le chiffre un", 2, input.Key1},
		{"le chiffre sept, dernier code sous le décalage", 8, input.Key7},
		{"le chiffre huit, premier code qu'X11 accepte", 9, input.Key8},
		{"la lettre A", 30, input.KeyA},
		{"F12, dernier code partagé", 88, input.KeyF12},
		{"la flèche du haut, une place étendue", 103, input.KeyUp},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := evdevKeyOf(c.code)
			if !ok {
				t.Fatalf("code evdev %d refusé", c.code)
			}
			if got != c.want {
				t.Errorf("code evdev %d a rendu %#x, attendu %#x", c.code, got, c.want)
			}
		})
	}

	// Le même code passé au traducteur X11 ne rend pas la même place : il y est un
	// code décalé, donc le code 9 y désigne Échap et non le chiffre huit.
	if got, ok := x11KeyOf(9); !ok || got != input.KeyEscape {
		t.Errorf("code X11 9 a rendu %#x (%v), attendu %#x", got, ok, input.KeyEscape)
	}
	if _, ok := x11KeyOf(7); ok {
		t.Error("le code X11 7 est sous le décalage et devait être refusé")
	}
}

// TestEvdevButtonOfJoinsBothNamesOfASideButton fige la correspondance des boutons.
//
// **Les quatre codes latéraux mènent à deux places, et c'est le fait que ce cas garde** :
// une souris émet `SIDE` et `EXTRA` ou bien `BACK` et `FORWARD` selon son micrologiciel,
// et n'en reconnaître qu'une paire rendrait muette la moitié des souris. Le cas rougirait
// si quelqu'un retirait l'une des deux paires en la croyant redondante.
func TestEvdevButtonOfJoinsBothNamesOfASideButton(t *testing.T) {
	cases := []struct {
		name string
		code uint32
		want input.Button
	}{
		{"le bouton principal", evdevButtonLeft, input.ButtonLeft},
		{"le bouton secondaire", evdevButtonRight, input.ButtonRight},
		{"la molette pressée", evdevButtonMiddle, input.ButtonMiddle},
		{"le latéral nommé côté", evdevButtonSide, input.ButtonBack},
		{"le latéral nommé précédent", evdevButtonBack, input.ButtonBack},
		{"le latéral nommé extra", evdevButtonExtra, input.ButtonForward},
		{"le latéral nommé suivant", evdevButtonForward, input.ButtonForward},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := evdevButtonOf(c.code)
			if !ok {
				t.Fatalf("code de bouton %#x refusé", c.code)
			}
			if got != c.want {
				t.Errorf("code %#x a rendu le bouton %d, attendu %d", c.code, got, c.want)
			}
		})
	}

	// `BTN_TASK` suit immédiatement les sept reconnus, et le moteur ne nomme aucune
	// place pour lui : il est refusé plutôt que rangé d'office dans la dernière.
	for _, code := range []uint32{0, 1, 0x10f, 0x117, 0x120} {
		if _, ok := evdevButtonOf(code); ok {
			t.Errorf("le code %#x n'est pas un bouton que le moteur nomme, et il a été accepté", code)
		}
	}
}
