// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/input"
)

// TestKeyOfSubtractsEightOnTheSharedCodes vérifie la soustraction de huit sur les
// places que les deux conventions partagent.
//
// Les valeurs attendues viennent des en-têtes UAPI du noyau, relevées le 27 octobre
// 2026, et non d'une correspondance devinée. Chacune est un fait d'interface : le code
// du noyau, plus huit que le protocole réserve, donne le code de touche que le serveur
// envoie.
//
// L'échantillon couvre les six blocs qui pourraient divergir indépendamment — lettres,
// chiffres, modificateurs de gauche, touches de fonction aux deux bouts, pavé
// numérique, et les deux touches dont la place est la plus haute de la zone partagée.
func TestKeyOfSubtractsEightOnTheSharedCodes(t *testing.T) {
	cases := []struct {
		name    string
		keycode byte
		want    input.Key
	}{
		{"échappement, le premier code du noyau", 1 + keycodeShift, input.KeyEscape},
		{"le chiffre un", 2 + keycodeShift, input.Key1},
		{"le chiffre zéro, en fin de rangée", 11 + keycodeShift, input.Key0},
		{"retour arrière", 14 + keycodeShift, input.KeyBackspace},
		{"la lettre Q, première de sa rangée", 16 + keycodeShift, input.KeyQ},
		{"entrée", 28 + keycodeShift, input.KeyEnter},
		{"contrôle de gauche", 29 + keycodeShift, input.KeyLeftControl},
		{"la lettre A", 30 + keycodeShift, input.KeyA},
		{"majuscule de gauche", 42 + keycodeShift, input.KeyLeftShift},
		{"majuscule de droite, qui n'est pas une touche étendue", 54 + keycodeShift, input.KeyRightShift},
		{"la lettre Z", 44 + keycodeShift, input.KeyZ},
		{"l'astérisque du pavé numérique", 55 + keycodeShift, input.KeyNumpadMultiply},
		{"alternative de gauche", 56 + keycodeShift, input.KeyLeftAlt},
		{"espace", 57 + keycodeShift, input.KeySpace},
		{"F1, première touche de fonction", 59 + keycodeShift, input.KeyF1},
		{"F10", 68 + keycodeShift, input.KeyF10},
		{"verrou numérique", 69 + keycodeShift, input.KeyNumLock},
		{"le zéro du pavé numérique", 82 + keycodeShift, input.KeyNumpad0},
		{"le point du pavé numérique", 83 + keycodeShift, input.KeyNumpadDecimal},
		{"F11, au-delà de la rangée de fonction", 87 + keycodeShift, input.KeyF11},
		{"F12, dernier code partagé", 88 + keycodeShift, input.KeyF12},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := keyOf(c.keycode)
			if !ok {
				t.Fatalf("code %d refusé", c.keycode)
			}
			if got != c.want {
				t.Errorf("place %#x, attendu %#x", got, c.want)
			}
		})
	}
}

// TestKeyOfTranslatesTheExtendedCodes vérifie la table des touches que le jeu 1
// atteignait par un préfixe.
//
// C'est là que les deux conventions divergent : le noyau leur a donné des codes à la
// suite, le moteur leur garde un bit de plus. Une entrée fausse ne se verrait pas à la
// compilation et donnerait une flèche qui déplace vers une autre direction.
func TestKeyOfTranslatesTheExtendedCodes(t *testing.T) {
	cases := []struct {
		name    string
		keycode byte
		want    input.Key
	}{
		{"entrée du pavé numérique", 96 + keycodeShift, input.KeyNumpadEnter},
		{"contrôle de droite", 97 + keycodeShift, input.KeyRightControl},
		{"division du pavé numérique", 98 + keycodeShift, input.KeyNumpadDivide},
		{"alternative de droite", 100 + keycodeShift, input.KeyRightAlt},
		{"origine", 102 + keycodeShift, input.KeyHome},
		{"flèche haute", 103 + keycodeShift, input.KeyUp},
		{"page précédente", 104 + keycodeShift, input.KeyPageUp},
		{"flèche gauche", 105 + keycodeShift, input.KeyLeft},
		{"flèche droite", 106 + keycodeShift, input.KeyRight},
		{"fin", 107 + keycodeShift, input.KeyEnd},
		{"flèche basse", 108 + keycodeShift, input.KeyDown},
		{"page suivante", 109 + keycodeShift, input.KeyPageDown},
		{"insertion", 110 + keycodeShift, input.KeyInsert},
		{"suppression", 111 + keycodeShift, input.KeyDelete},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := keyOf(c.keycode)
			if !ok {
				t.Fatalf("code %d refusé", c.keycode)
			}
			if got != c.want {
				t.Errorf("place %#x, attendu %#x", got, c.want)
			}
		})
	}
}

// TestEvdevSeparatesTheNumpadFromTheNavigationBlock est le cas qui garde contre la
// transposition du dispositif de Windows.
//
// Là-bas, le pavé numérique et le bloc de navigation produisent les mêmes codes et il
// faut consulter le verrou numérique pour les séparer — tout un mécanisme que le
// backend Windows a dû construire. **Ici le noyau les sépare de lui-même**, et
// transposer ce mécanisme reviendrait à résoudre un problème qui n'existe pas, avec le
// risque d'en créer un.
//
// Le cas le montre sur les cinq paires qui partagent un code de base : les deux places
// du moteur diffèrent, et chacune vient d'un code du noyau distinct.
func TestEvdevSeparatesTheNumpadFromTheNavigationBlock(t *testing.T) {
	pairs := []struct {
		name            string
		numpad, navigat byte
		wantNumpad      input.Key
		wantNavigation  input.Key
	}{
		{"zéro et insertion", 82, 110, input.KeyNumpad0, input.KeyInsert},
		{"un et fin", 79, 107, input.KeyNumpad1, input.KeyEnd},
		{"sept et origine", 71, 102, input.KeyNumpad7, input.KeyHome},
		{"huit et flèche haute", 72, 103, input.KeyNumpad8, input.KeyUp},
		{"neuf et page précédente", 73, 104, input.KeyNumpad9, input.KeyPageUp},
	}

	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			numpad, ok := keyOf(p.numpad + keycodeShift)
			if !ok {
				t.Fatalf("code %d du pavé numérique refusé", p.numpad)
			}
			navigation, ok := keyOf(p.navigat + keycodeShift)
			if !ok {
				t.Fatalf("code %d du bloc de navigation refusé", p.navigat)
			}

			if numpad != p.wantNumpad {
				t.Errorf("pavé numérique : place %#x, attendu %#x", numpad, p.wantNumpad)
			}
			if navigation != p.wantNavigation {
				t.Errorf("navigation : place %#x, attendu %#x", navigation, p.wantNavigation)
			}
			if numpad == navigation {
				t.Error("les deux blocs rendent la même place, donc le serveur ne les sépare plus")
			}
		})
	}
}

// TestKeyOfRefusesWhatHasNoHonestPlace vérifie les refus.
//
// Celui de la touche de pause est une décision et non un oubli : le jeu 1 l'envoyait en
// trois octets et Windows ne la rapporte pas uniformément, donc une correspondance ici
// la rendrait lisible sous X11 et muette sous Windows pour la même valeur. Une place
// qui ne veut pas dire la même chose partout vaut moins qu'une place absente, puisque
// c'est l'universalité de la convention qui fait sa valeur.
func TestKeyOfRefusesWhatHasNoHonestPlace(t *testing.T) {
	cases := []struct {
		name    string
		keycode byte
	}{
		{"la touche de pause, dont la correspondance serait fausse ailleurs", 119 + keycodeShift},
		{"un code que le protocole réserve", 4},
		{"le code nul", 0},
		{"un code sans touche derrière", 200},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, ok := keyOf(c.keycode); ok {
				t.Errorf("code %d accepté comme place %#x", c.keycode, got)
			}
		})
	}
}

// TestRelearnKeysReadsTheWholeKeyboardAtOnce vérifie la reprise de l'état depuis le
// vecteur de bits du serveur.
//
// C'est ce qui remplace la relecture touche par touche de Windows, et l'écart est le
// fait marquant du clavier X11 : une requête contre cinq cents appels. Un décalage
// d'un bit dans la lecture allumerait la touche voisine, ce qui est le genre de défaut
// qu'on attribue au clavier du joueur.
func TestRelearnKeysReadsTheWholeKeyboardAtOnce(t *testing.T) {
	w, _ := windowPair(t)

	// Le vecteur commence au code huit : l'octet n porte les codes 8+8n à 15+8n, du
	// bit de poids faible au bit de poids fort. La lettre A est le code 30 du noyau,
	// donc 38 pour le serveur, donc l'octet 3 et le bit 6.
	bitmap := make([]byte, 31)
	bitmap[3] |= 1 << 6

	w.relearnKeys(bitmap)
	in := w.state.Frame()

	if !in.Held(input.KeyA) {
		t.Error("la lettre A n'est pas tenue alors que son bit est posé")
	}
	if in.Held(input.KeyB) {
		t.Error("la lettre B est tenue alors qu'aucun bit ne la nomme")
	}
}

// TestRelearnKeysForgetsWhatTheVectorDoesNotName vérifie que la reprise part de
// l'état réel et non de ce qu'on croyait.
//
// L'oubli préalable est le cœur de cet événement : ce que le serveur ne nomme pas
// n'est pas enfoncé. Sans lui, une touche tenue pendant que la fenêtre n'avait pas le
// focus le resterait pour toujours, et le personnage marcherait tout seul au retour.
func TestRelearnKeysForgetsWhatTheVectorDoesNotName(t *testing.T) {
	w, _ := windowPair(t)
	w.state.Press(input.KeyLeft)

	w.relearnKeys(make([]byte, 31))

	if w.state.Frame().Held(input.KeyLeft) {
		t.Error("une touche absente du vecteur est restée tenue")
	}
}
