// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la relecture en octets doit tenir : qu'elle alias, qu'elle couvre tout le
// tampon, et qu'un tampon vide ne la fasse pas paniquer. Et ce qu'un code de format
// demande, pour les deux familles qui partagent ce registre.

package backend

import (
	"strings"
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestFourCCMatchesTheKernelHeaders fige la composition d'un code de format contre une
// valeur relevée dans la source.
//
// `XB24` vaut 0x34324258 d'après les en-têtes UAPI, et ce cas est le seul endroit où
// cette valeur est écrite autrement que par la macro qui la compose. Sans lui, la macro
// se contrôlerait elle-même : une inversion de l'ordre des quatre octets passerait, et
// toutes les comparaisons de formats resteraient cohérentes entre elles tout en ne
// désignant plus rien de ce que la plateforme annonce.
func TestFourCCMatchesTheKernelHeaders(t *testing.T) {
	if got, want := fourcc('X', 'B', '2', '4'), uint32(0x34324258); got != want {
		t.Errorf("le code XB24 vaut %#08x, attendu %#08x", got, want)
	}
}

// TestPixelOrderOfFourCCPlacesTheChannels garde le classement des quatre formats écrits.
//
// **Le piège que ce cas existe pour attraper est l'inversion du direct et de l'échangé.**
// Les en-têtes décrivent le mot de trente-deux bits du bit de poids fort au bit de poids
// faible — `x:B:G:R` pour `XB24` —, là que la conversion a besoin de l'ordre des octets
// en mémoire, qui se lit à l'envers. Confondre les deux lectures donne une image aux
// couleurs permutées, qu'aucune erreur de protocole ne signale et qu'un job sans écran ne
// verrait pas. Les deux dispositions fautives sont valides et le code tournerait.
func TestPixelOrderOfFourCCPlacesTheChannels(t *testing.T) {
	cases := []struct {
		label string
		code  uint32
		at    [3]int
		kind  pixelKind
	}{
		{"x:B:G:R, l'ordre du moteur", formatXBGR8888, [3]int{0, 1, 2}, pixelDirect},
		{"x:R:G:B, le rouge et le bleu échangés", formatXRGB8888, [3]int{2, 1, 0}, pixelSwapped},
		{"R:G:B:x, décalé d'un octet", formatRGBX8888, [3]int{3, 2, 1}, pixelMapped},
		{"B:G:R:x, échangé et décalé", formatBGRX8888, [3]int{1, 2, 3}, pixelMapped},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			o, err := pixelOrderOfFourCC(c.code)
			if err != nil {
				t.Fatalf("le format %s est refusé : %v", fourccName(c.code), err)
			}
			if o.at != c.at {
				t.Errorf("les composantes vont en %v, attendu %v", o.at, c.at)
			}
			if o.kind != c.kind {
				t.Errorf("la disposition est classée %d, attendu %d", o.kind, c.kind)
			}
			if o.size != 4 {
				t.Errorf("un pixel y occupe %d octets, attendu 4", o.size)
			}
		})
	}
}

// TestPixelOrderOfFourCCRefusesAnAlphaFormat garde le refus des formats à alpha.
//
// **Le mode de panne qu'il attrape est muet et total** : les deux dispositions rapides
// composent le mot entier, donc elles écriraient zéro dans l'octet d'alpha et la surface
// serait entièrement transparente. Le protocole ne dirait rien, et une capture sans écran
// montrerait le fond du compositeur.
//
// Le message est contrôlé aussi, parce qu'il est la seule voie par laquelle un
// intégrateur apprend quel format sa plateforme a annoncé : un code en hexadécimal ne se
// reconnaît pas.
func TestPixelOrderOfFourCCRefusesAnAlphaFormat(t *testing.T) {
	for _, c := range []struct {
		label string
		code  uint32
	}{
		{"A:R:G:B", fourcc('A', 'R', '2', '4')},
		{"A:B:G:R", fourcc('A', 'B', '2', '4')},
		{"seize bits par pixel", fourcc('R', 'G', '1', '6')},
	} {
		t.Run(c.label, func(t *testing.T) {
			_, err := pixelOrderOfFourCC(c.code)
			if err == nil {
				t.Fatalf("le format %s est accepté, attendu un refus", fourccName(c.code))
			}
			if name := fourccName(c.code); !strings.Contains(err.Error(), name) {
				t.Errorf("le refus dit %q, attendu qu'il nomme %s en clair", err, name)
			}
		})
	}
}

// TestFourCCNameKeepsAMessageReadable garde la substitution d'un octet non imprimable.
//
// Un code que la plateforme aurait annoncé de travers ne doit pas écrire d'octet de
// contrôle dans un message d'erreur, qui finit dans un rapport de bug.
func TestFourCCNameKeepsAMessageReadable(t *testing.T) {
	if got, want := fourccName(fourcc('X', 0x00, 'B', 0x7f)), "X?B?"; got != want {
		t.Errorf("le nom rendu est %q, attendu %q", got, want)
	}
}

// TestBytesOfAliasesThePixels écrit par la tranche d'octets et relit le pixel.
//
// **C'est le seul cas de ce fichier qu'aucune autre couche n'attrape**, et le mode de panne
// qu'il garde est muet : une relecture qui copierait au lieu d'aliaser rendrait une
// présentation correcte à la première image et figée ensuite, sans erreur ni message. Rien
// dans la suite ne verrait la différence — les octets sont les bons, ils ne sont simplement
// plus ceux du tampon que le moteur repeint.
//
// L'ordre des composantes, lui, est gardé ailleurs et par le même chemin : `TestColorLayout`
// du paquet `render` fige les quatre décalages, donc ce cas n'a pas à les revérifier.
func TestBytesOfAliasesThePixels(t *testing.T) {
	pixels := []render.Color{{R: 1, G: 2, B: 3, A: 4}, {R: 5, G: 6, B: 7, A: 8}}

	raw := bytesOf(pixels)
	raw[4] = 50

	if got, want := pixels[1].R, uint8(50); got != want {
		t.Errorf("le rouge du second pixel vaut %d, attendu %d — la tranche copie au lieu d'aliaser", got, want)
	}
}

// TestBytesOfCoversTheWholeBuffer vérifie la longueur, quatre octets par pixel.
//
// Il rougit sur un facteur faux — trois, ou une longueur en pixels — qui laisserait le
// dernier quart du tampon hors de ce qu'on présente, donc une bande non peinte en bas de
// l'image. Le défaut serait visible, mais seulement une fois un canvas monté : ici il se
// voit sur l'hôte.
func TestBytesOfCoversTheWholeBuffer(t *testing.T) {
	for _, pixels := range []int{1, 2, 480 * 270} {
		got := len(bytesOf(make([]render.Color, pixels)))
		if want := pixels * 4; got != want {
			t.Errorf("%d pixels donnent %d octets, attendu %d", pixels, got, want)
		}
	}
}

// TestBytesOfAcceptsAnEmptyBuffer vérifie qu'un tampon vide ne panique pas.
//
// Sans la garde, `&pixels[0]` indexe un élément qui n'existe pas. Le cas n'est pas
// défensif pour rien : une fenêtre réduite rend une taille nulle, ce que le backend
// Windows documente déjà comme un régime à traverser sans casser.
func TestBytesOfAcceptsAnEmptyBuffer(t *testing.T) {
	if got := bytesOf(nil); len(got) != 0 {
		t.Errorf("un tampon vide rend %d octets, attendu 0", len(got))
	}
}
