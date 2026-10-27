// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// visualOf décrit un serveur par ce qui décide de la disposition d'un pixel : l'ordre
// de ses octets d'image et les masques de son visuel.
func visualOf(imageOrder uint8, red, green, blue uint32) setup {
	return setup{
		depth:        24,
		bitsPerPixel: 32,
		imageOrder:   imageOrder,
		redMask:      red,
		greenMask:    green,
		blueMask:     blue,
	}
}

// TestPixelOrderFollowsTheServerAndNotTheMachine est le cas qui porte la correction du
// brief du jalon.
//
// Celui-ci annonçait que la conversion était « un échange des octets 0 et 2 » et qu'un
// serveur gros-boutien aux mêmes masques recevrait le tampon tel quel. Le premier point
// est juste pour le serveur qu'on rencontre ; le second est faux, et le calcul le
// montre : avec les masques courants, un serveur gros-boutien attend l'ordre `_RVB`,
// soit un décalage. Le cas « tel quel » existe bel et bien, mais c'est celui d'un
// serveur petit-boutien à visuel BVR.
//
// Un échange codé en dur passerait donc sur la machine de contrôle et sortirait des
// couleurs permutées ailleurs, sans qu'aucune erreur de protocole ne le dise.
func TestPixelOrderFollowsTheServerAndNotTheMachine(t *testing.T) {
	cases := []struct {
		name  string
		setup setup
		kind  pixelKind
		at    [3]int
	}{
		{
			"petit-boutien à visuel RVB, le serveur courant",
			visualOf(lsbFirst, 0xFF0000, 0x00FF00, 0x0000FF),
			pixelSwapped,
			[3]int{2, 1, 0},
		},
		{
			"petit-boutien à visuel BVR, où le tampon tombe déjà juste",
			visualOf(lsbFirst, 0x0000FF, 0x00FF00, 0xFF0000),
			pixelDirect,
			[3]int{0, 1, 2},
		},
		{
			"gros-boutien à visuel RVB, que le brief donnait pour direct",
			visualOf(msbFirst, 0xFF0000, 0x00FF00, 0x0000FF),
			pixelMapped,
			[3]int{1, 2, 3},
		},
		{
			"gros-boutien à visuel BVR",
			visualOf(msbFirst, 0x0000FF, 0x00FF00, 0xFF0000),
			pixelMapped,
			[3]int{3, 2, 1},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, err := pixelOrderOf(c.setup)
			if err != nil {
				t.Fatalf("visuel refusé : %v", err)
			}
			if o.at != c.at {
				t.Errorf("index d'octets %v, attendu %v", o.at, c.at)
			}
			if o.kind != c.kind {
				t.Errorf("disposition %d, attendu %d", o.kind, c.kind)
			}
		})
	}
}

// TestPixelOrderWritesWhereTheMasksPointTo vérifie que la conversion écrit bien les
// composantes aux index que les masques désignent.
//
// Le cas vaut pour les trois boucles : deux sont spécialisées pour la vitesse, la
// troisième est générale, et rien ne garantirait sans cela qu'elles s'accordent. Une
// spécialisation fausse serait invisible — elle ne touche que l'ordre des octets d'une
// image que personne ne relit octet par octet.
func TestPixelOrderWritesWhereTheMasksPointTo(t *testing.T) {
	pixels := []render.Color{{R: 10, G: 20, B: 30, A: 255}}

	cases := []struct {
		name  string
		setup setup
		want  [3]byte // ce que portent les trois premiers octets
	}{
		{"échangé", visualOf(lsbFirst, 0xFF0000, 0x00FF00, 0x0000FF), [3]byte{30, 20, 10}},
		{"direct", visualOf(lsbFirst, 0x0000FF, 0x00FF00, 0xFF0000), [3]byte{10, 20, 30}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, err := pixelOrderOf(c.setup)
			if err != nil {
				t.Fatalf("visuel refusé : %v", err)
			}

			got := o.appendPixels(nil, pixels)
			if len(got) != 4 {
				t.Fatalf("%d octets écrits pour un pixel, attendu 4", len(got))
			}
			if got[0] != c.want[0] || got[1] != c.want[1] || got[2] != c.want[2] {
				t.Errorf("octets %v, attendus %v", got[:3], c.want)
			}
		})
	}

	// Le cas général, où aucune des deux spécialisations ne s'applique : les trois
	// composantes partent aux index 1, 2 et 3.
	o, err := pixelOrderOf(visualOf(msbFirst, 0xFF0000, 0x00FF00, 0x0000FF))
	if err != nil {
		t.Fatalf("visuel refusé : %v", err)
	}
	got := o.appendPixels(nil, pixels)
	if got[1] != 10 || got[2] != 20 || got[3] != 30 {
		t.Errorf("octets %v, attendus [_ 10 20 30]", got)
	}
}

// TestPixelOrderLeavesThePaddingByteAlone vérifie que le quatrième octet n'est pas
// écrit.
//
// En profondeur 24 sur trente-deux bits, les huit bits hors masques sont indéfinis et
// le serveur les ignore : les mettre à zéro coûterait une écriture de plus par pixel
// pour rien. Le cas fixe ce choix, parce qu'il a une conséquence pour le lot qui
// relira l'image — comparer des mots bruts serait intermittent, il faudra masquer des
// deux côtés.
func TestPixelOrderLeavesThePaddingByteAlone(t *testing.T) {
	o, err := pixelOrderOf(visualOf(lsbFirst, 0xFF0000, 0x00FF00, 0x0000FF))
	if err != nil {
		t.Fatalf("visuel refusé : %v", err)
	}

	// Un tampon dont le quatrième octet porte une marque reconnaissable, et une
	// capacité suffisante pour que la conversion le réemploie au lieu d'en allouer
	// un neuf.
	dst := make([]byte, 4, 4)
	dst[3] = 0xAA
	dst = dst[:0]

	dst = o.appendPixels(dst, []render.Color{{R: 1, G: 2, B: 3, A: 4}})

	if dst[3] != 0xAA {
		t.Errorf("le quatrième octet vaut %#x, et rien ne devait l'écrire", dst[3])
	}
}

// TestPixelOrderRefusesWhatAPermutationCannotServe vérifie que les visuels hors de
// portée sont refusés, avec un message qui nomme ce qui a été trouvé.
//
// Le refus est préférable à une conversion approximative : une image aux couleurs
// fausses ressemble à un défaut du rastériseur, et on la cherche à l'autre bout du
// moteur. Il tombe à l'ouverture, pas à la première image.
func TestPixelOrderRefusesWhatAPermutationCannotServe(t *testing.T) {
	cases := []struct {
		name  string
		setup setup
	}{
		{
			"seize bits par pixel, où une composante ne tient pas dans un octet",
			setup{bitsPerPixel: 16, redMask: 0xF800, greenMask: 0x07E0, blueMask: 0x001F},
		},
		{
			"vingt-quatre bits par pixel, que les boucles ne servent pas",
			setup{bitsPerPixel: 24, redMask: 0xFF0000, greenMask: 0x00FF00, blueMask: 0x0000FF},
		},
		{
			"un masque qui ne commence pas sur un octet",
			setup{bitsPerPixel: 32, redMask: 0x1FE000, greenMask: 0x00FF00, blueMask: 0x0000FF},
		},
		{
			"un masque vide",
			setup{bitsPerPixel: 32, redMask: 0, greenMask: 0x00FF00, blueMask: 0x0000FF},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := pixelOrderOf(c.setup); err == nil {
				t.Error("visuel accepté")
			}
		})
	}
}

// TestAppendPixelsDoesNotAllocateOnAWarmBuffer vérifie que la conversion n'allègue
// rien quand le tampon d'émission a déjà sa place.
//
// C'est la condition que le chemin de rendu impose : elle tourne soixante fois par
// seconde sur toute la surface de la fenêtre, et une allocation par image mettrait le
// ramasse-miettes dans ce chemin. La première image paie l'agrandissement, les
// suivantes réemploient.
func TestAppendPixelsDoesNotAllocateOnAWarmBuffer(t *testing.T) {
	o, err := pixelOrderOf(visualOf(lsbFirst, 0xFF0000, 0x00FF00, 0x0000FF))
	if err != nil {
		t.Fatalf("visuel refusé : %v", err)
	}

	pixels := make([]render.Color, 64*64)
	dst := o.appendPixels(nil, pixels)

	allocs := testing.AllocsPerRun(50, func() {
		dst = o.appendPixels(dst[:0], pixels)
	})
	if allocs != 0 {
		t.Errorf("%.1f allocation(s) par conversion, attendu aucune", allocs)
	}
}
