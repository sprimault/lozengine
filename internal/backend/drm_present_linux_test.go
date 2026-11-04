// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'une vraie carte détient quand on lui présente une image, et ce que coûte l'écriture
// dans un tampon que le noyau prête.

package backend

import (
	"testing"

	"github.com/sprimault/lozengine/internal/bench"
)

// TestTheCardHoldsExactlyWhatWasPresented est la validation de la présentation sur du vrai
// matériel.
//
// **Ce qu'il affirme est aussi direct qu'en Wayland, et pour la même raison** : le tampon
// est projeté chez nous, donc le relire est relire ce que le contrôleur balaie. Il n'y a
// pas de transfert à éprouver, seulement une conversion et un pas de ligne.
//
// **Le mode est posé pour de vrai, et il se remet en partant** : la fermeture repasse au
// contrôleur l'état que la découverte avait lu. C'est ce qui permet d'exécuter ce cas sur
// une machine partagée sans y laisser l'affichage dans l'état où on l'a mis.
//
// **Ce qu'il ne dit pas** : que l'image s'affiche. Sur la machine de contrôle le connecteur
// retenu est celui d'écriture différée, donc il n'y a aucun écran au bout — et aucun client
// ne peut observer ce qu'un contrôleur balaie. C'est le job qui lit un connecteur virtuel
// par écriture différée qui le montrera.
//
// **La position d'octet attendue se dérive du format que le pilote a accordé**, et non de
// la disposition que le backend en a construite : un cas qui éprouve une conversion ne peut
// pas s'en servir pour dire ce qu'il attend.
func TestTheCardHoldsExactlyWhatWasPresented(t *testing.T) {
	card := requireCard(t)

	if err := card.mount(); err != nil {
		t.Fatalf("montage des tampons : %v", err)
	}

	width, height := card.size()
	image := drmTestImage(width, height)

	card.Present(image)
	if card.err != nil {
		t.Fatalf("présentation : %v", card.err)
	}
	if !card.lit {
		t.Fatal("la présentation n'a pas posé le mode")
	}

	f := &card.frames[card.front]
	if f.pitch < uint32(width)*4 {
		t.Fatalf("le pas du pilote vaut %d, moins que les %d octets d'une ligne",
			f.pitch, uint32(width)*4)
	}
	t.Logf("tampon de %d×%d, pas de %d octets (%d pour une ligne jointive)",
		width, height, f.pitch, uint32(width)*4)

	red, blue := 0, 2
	if card.order.kind == pixelSwapped {
		red, blue = 2, 0
	}

	// La relecture porte sur les quatre coins et les deux diagonales plutôt que sur
	// toute la surface : à 2560×1600 une comparaison complète prend des secondes, et ce
	// qu'on cherche — un pas mal appliqué, une transposition, un décalage de ligne —
	// se voit dès qu'on lit deux lignes éloignées.
	for _, p := range [][2]int{
		{0, 0}, {int(width) - 1, 0}, {0, int(height) - 1},
		{int(width) - 1, int(height) - 1}, {int(width) / 2, int(height) / 2},
		{1, int(height) - 2}, {int(width) - 2, 1},
	} {
		x, y := p[0], p[1]
		want := image.Pixels[y*int(width)+x]
		at := y*int(f.pitch) + x*4

		if got := f.mem[at+red]; got != want.R {
			t.Errorf("en (%d, %d), le rouge du tampon vaut %d, on a écrit %d", x, y, got, want.R)
		}
		if got := f.mem[at+1]; got != want.G {
			t.Errorf("en (%d, %d), le vert du tampon vaut %d, on a écrit %d", x, y, got, want.G)
		}
		if got := f.mem[at+blue]; got != want.B {
			t.Errorf("en (%d, %d), le bleu du tampon vaut %d, on a écrit %d", x, y, got, want.B)
		}
	}
}

// BenchmarkDRMWrite mesure l'écriture d'une image dans un tampon que le noyau prête.
//
// **Il existe pour répondre à une réserve que la godoc d'[pixelOrder.appendPixels] porte
// depuis le jalon 3** : ses chiffres ne valaient que pour un tampon du tas de Go, « une
// mémoire partagée projetée ou un tampon dumb du noyau sont une autre destination ». La
// première moitié a été mesurée par la famille Wayland, qui n'a rien changé ; celle-ci est
// la seconde, et c'est la dernière destination qu'aucun banc ne mesurait.
//
// **Le témoin est dans le banc et non dans une seconde série** : chaque géométrie est
// mesurée vers le tampon du pilote **et** vers une tranche du tas de même pas, à la même
// place dans la séquence. Leur écart est la réponse, et il ne dépend d'aucune discipline de
// remesure.
//
// **Les deux largeurs ne sont pas un échantillon, ce sont deux régimes** : 1920 tombe juste
// et l'écriture est une copie unique, 1366 ne tombe pas juste et elle se fait ligne par
// ligne. C'est le rapport entre les deux qui dit ce que le pas du pilote coûte, et il ne se
// devine pas — à 1366 il y a huit cents copies par image au lieu d'une.
//
// Il demande une carte, et se saute sans elle.
//
// # Ce que le lot a mesuré
//
// Trois séries sur la machine de contrôle le 4 novembre 2026, en part brute du budget d'une
// image à soixante par seconde :
//
//	                        tampon du pilote   tranche du tas
//	xbgr8888  1920, aligné      0,84 %            0,85 %
//	xbgr8888  1366, allongé     0,44 %            0,34 %
//	xrgb8888  1920, aligné     11,40 %           11,42 %
//	xrgb8888  1366, allongé     5,790 %           5,797 %
//
// **La destination ne compte que sur le chemin de copie, et seulement quand le pas est
// allongé.** Sous conversion, elle ne change rien à aucune des deux largeurs — deux
// dixièmes de millième —, parce que le coût y est dans les opérations : 4,3 Go/s de part et
// d'autre. Sous copie à pas aligné, rien non plus, l'écriture étant une seule grande copie.
// Sous copie à pas allongé, en revanche, le tampon du pilote coûte **trente pour cent de
// plus** : 57,5 Go/s contre 74,5, pour sept cent soixante-huit copies de 5 464 octets au
// lieu d'une de 8,3 Mo.
//
// **C'est ce qui lève la réserve du jalon 3 sans la remplacer par une approximation** :
// elle portait sur [pixelOrder.appendPixels], c'est-à-dire sur la conversion, et pour elle
// la réponse est que la destination est sans effet. Ce qui apparaît est un fait voisin et
// distinct, sur un chemin que cette réserve ne visait pas.
//
// **Et cette famille coûte ce que coûte Wayland**, à trois pour mille près : 0,84 % contre
// 0,85 % sous l'ordre du moteur, 11,4 % contre 11,4 % sous le repli. Les deux écrivent dans
// une projection, et c'est le format accordé qui décide, pas la famille.
func BenchmarkDRMWrite(b *testing.B) {
	card := requireCard(b)

	for _, format := range []struct {
		name string
		code uint32
	}{
		{"xbgr8888, l'ordre du moteur", formatXBGR8888},
		{"xrgb8888, le rouge et le bleu échangés", formatXRGB8888},
	} {
		o, err := pixelOrderOfFourCC(format.code)
		if err != nil {
			b.Fatalf("format %s refusé : %v", format.name, err)
		}

		for _, size := range []struct {
			name          string
			width, height uint16
		}{
			{"1920×1080, un pas qui tombe juste", 1920, 1080},
			{"1366×768, un pas que le pilote allonge", 1366, 768},
		} {
			frame, _, err := drmNewFrame(card.fd, size.width, size.height)
			if err != nil {
				b.Fatalf("tampon %s : %v", size.name, err)
			}
			image := drmTestImage(size.width, size.height)

			b.Run(format.name+"/"+size.name+"/tampon du pilote", func(b *testing.B) {
				frame.write(o, image)

				b.SetBytes(int64(4 * len(image.Pixels)))
				b.ResetTimer()

				for range b.N {
					frame.write(o, image)
				}
				bench.Part(b)
			})

			// Le témoin : le même pas, la même image, une tranche du tas. C'est lui
			// qui dit si la destination compte.
			heap := &drmFrame{
				pitch: frame.pitch,
				mem:   make([]byte, int(frame.pitch)*int(size.height)),
			}
			b.Run(format.name+"/"+size.name+"/tranche du tas", func(b *testing.B) {
				heap.write(o, image)

				b.SetBytes(int64(4 * len(image.Pixels)))
				b.ResetTimer()

				for range b.N {
					heap.write(o, image)
				}
				bench.Part(b)
			})

			frame.drop(card.fd)
		}
	}
}
