// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'écriture dans un tampon dumb doit tenir, et le piège qu'aucune largeur ronde ne
// montre.

package backend

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/sprimault/lozengine/render"
)

// drmTestFrame monte un tampon de la géométrie et du pas donnés, sans pilote.
//
// La projection est une tranche ordinaire : ce qu'on éprouve est l'arithmétique de
// l'écriture, et une vraie projection de tampon dumb n'y changerait rien — la mesurer est
// le travail du banc.
func drmTestFrame(width, height uint16, pitch uint32) *drmFrame {
	return &drmFrame{pitch: pitch, mem: make([]byte, int(pitch)*int(height))}
}

// drmTestImage rend une image dont chaque pixel dit où il est.
//
// **Le motif dépend des deux coordonnées**, parce qu'un tampon uni ne distinguerait ni une
// transposition, ni un décalage de ligne, ni un pas mal appliqué — les trois défauts que
// l'écriture peut produire et dont aucun ne lève d'erreur.
func drmTestImage(width, height uint16) *render.Buffer {
	t := render.NewBuffer(width, height)
	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			t.Pixels[y*int(width)+x] = render.Color{
				R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: 255,
			}
		}
	}
	return t
}

// TestDRMWriteHonoursTheDriversStride garde le piège central de cette famille.
//
// **Le pas de ligne vient du pilote et n'est pas celui du moteur**, mesuré le 4 novembre
// 2026 sur la machine de contrôle : une largeur de 1366 donne un pas de 5632 octets au lieu
// de 5464, et 1000 donne 4096 au lieu de 4000. Recopier l'image d'un bloc décalerait chaque
// ligne d'un peu plus que la précédente, et l'image sortirait en biais.
//
// **Et c'est un défaut qu'aucune largeur ronde ne montre** : 1920 et 2560 tombent juste, et
// ce sont les deux premières tailles qu'on essaie. Le cas emploie donc 1366 et 1000, les
// deux largeurs mesurées comme non alignées, plus une largeur alignée en témoin — sans elle,
// une écriture qui ignorerait le pas rougirait partout et on ne saurait pas distinguer un
// défaut d'arithmétique d'un défaut de chemin.
//
// **Les deux dispositions y sont**, parce que ce sont deux boucles distinctes : la copie en
// bloc quand le pilote accorde l'ordre du moteur, la permutation sinon.
func TestDRMWriteHonoursTheDriversStride(t *testing.T) {
	for _, format := range []struct {
		name string
		code uint32
	}{
		{"xbgr8888, l'ordre du moteur", formatXBGR8888},
		{"xrgb8888, le rouge et le bleu échangés", formatXRGB8888},
	} {
		o, err := pixelOrderOfFourCC(format.code)
		if err != nil {
			t.Fatalf("format %s refusé : %v", format.name, err)
		}

		for _, geometry := range []struct {
			name          string
			width, height uint16
			pitch         uint32
		}{
			{"1366 de large, pas de 5632", 1366, 8, 5632},
			{"1000 de large, pas de 4096", 1000, 8, 4096},
			{"un pas qui tombe juste", 64, 8, 64 * 4},
		} {
			t.Run(format.name+"/"+geometry.name, func(t *testing.T) {
				f := drmTestFrame(geometry.width, geometry.height, geometry.pitch)
				image := drmTestImage(geometry.width, geometry.height)

				f.write(o, image)

				red, blue := 0, 2
				if format.code == formatXRGB8888 {
					red, blue = 2, 0
				}

				for y := 0; y < int(geometry.height); y++ {
					for x := 0; x < int(geometry.width); x++ {
						want := image.Pixels[y*int(geometry.width)+x]
						at := y*int(geometry.pitch) + x*4

						if got := f.mem[at+red]; got != want.R {
							t.Fatalf("en (%d, %d), le rouge du tampon vaut %d, on a écrit %d",
								x, y, got, want.R)
						}
						if got := f.mem[at+1]; got != want.G {
							t.Fatalf("en (%d, %d), le vert du tampon vaut %d, on a écrit %d",
								x, y, got, want.G)
						}
						if got := f.mem[at+blue]; got != want.B {
							t.Fatalf("en (%d, %d), le bleu du tampon vaut %d, on a écrit %d",
								x, y, got, want.B)
						}
					}
				}
			})
		}
	}
}

// TestDRMWriteLeavesTheStridePaddingAlone garde que l'écriture ne dépasse pas la ligne.
//
// **Ce que le pas laisse en trop n'appartient à personne**, et un pilote peut y ranger ce
// qu'il veut : une écriture qui le remplirait marcherait aujourd'hui et pourrait abîmer
// demain ce qu'un pilote y range. Le cas remplit la bourre d'un octet repérable et vérifie
// qu'elle y reste.
//
// Il attrape en plus le défaut inverse de celui du cas précédent : une écriture qui
// prendrait le pas pour la largeur de l'image, et peindrait donc des pixels que l'image ne
// porte pas.
func TestDRMWriteLeavesTheStridePaddingAlone(t *testing.T) {
	const (
		width  = 100
		height = 4
		pitch  = 512
	)

	o, err := pixelOrderOfFourCC(formatXBGR8888)
	if err != nil {
		t.Fatalf("format refusé : %v", err)
	}

	f := drmTestFrame(width, height, pitch)
	for i := range f.mem {
		f.mem[i] = 0xAB
	}

	f.write(o, drmTestImage(width, height))

	for y := 0; y < height; y++ {
		for at := y*pitch + width*4; at < (y+1)*pitch; at++ {
			if f.mem[at] != 0xAB {
				t.Fatalf("la bourre de la ligne %d a été touchée à l'octet %d", y, at-y*pitch)
			}
		}
	}
}

// TestDRMTurnsItsFramesWithoutWritingTheVisibleOne garde la rotation des tampons.
//
// Trois régimes, et le troisième est celui qui compte : **le tampon affiché n'est pas
// libre.** Le contrôleur le balaie en continu, donc y écrire produit du déchirement
// qu'aucune erreur ne signale — c'est le mode de panne le plus désagréable de cette
// famille, puisqu'il ne se voit que sur un écran réel.
//
// Le cas n'ouvre aucune carte : la rotation est une propriété de la structure, et
// l'éprouver par un pilote mêlerait ce qu'elle décide à ce que le matériel transporte.
func TestDRMTurnsItsFramesWithoutWritingTheVisibleOne(t *testing.T) {
	var c drmCard

	if got := c.free(); got != 0 {
		t.Fatalf("avant tout affichage, le premier tampon choisi est %d, attendu 0", got)
	}

	c.lit, c.front = true, 0
	if got := c.free(); got != 1 {
		t.Fatalf("le tampon zéro étant à l'écran, le choix rend %d, attendu 1", got)
	}

	c.frames[1].pending = true
	if got := c.free(); got != -1 {
		t.Errorf("l'un affiché et l'autre en vol, le choix rend %d, attendu -1", got)
	}

	c.frames[1].pending = false
	if got := c.free(); got != 1 {
		t.Errorf("la bascule honorée, le choix rend %d, attendu 1", got)
	}
}

// TestDRMDrainFreesTheFlippedFrame garde la lecture des événements de la carte.
//
// **Il passe par un tube et non par une carte**, ce qui est ce qui le rend exécutable
// partout : ce qu'on éprouve est le décodage d'un flux d'événements, et le noyau n'y est
// pour rien. Le tube porte exactement ce que le pilote écrirait.
//
// Trois régimes, et chacun est un défaut qu'un décodage naïf produirait : **deux événements
// dans une seule lecture**, que le noyau groupe et qu'une boucle qui n'en lirait qu'un
// laisserait en arrière ; **un type inconnu**, qui doit se sauter par la longueur que son
// en-tête déclare et non par la taille d'une structure connue, sans quoi la suite du flux
// se désynchronise ; et un **index hors borne**, qu'un pilote ne produira pas mais qu'une
// lecture décalée produirait, et qui ferait paniquer la présentation.
func TestDRMDrainFreesTheFlippedFrame(t *testing.T) {
	event := func(kind uint32, size int, data uint64) []byte {
		b := make([]byte, size)
		binary.NativeEndian.PutUint32(b[0:], kind)
		binary.NativeEndian.PutUint32(b[4:], uint32(size))
		if size >= drmEventVblankSize {
			binary.NativeEndian.PutUint64(b[drmEventVblankDataAt:], data)
		}
		return b
	}

	for _, c := range []struct {
		name    string
		stream  []byte
		pending [drmFrames]bool
		want    [drmFrames]bool
	}{
		{
			"une bascule libère son tampon",
			event(drmEventFlipComplete, drmEventVblankSize, 1),
			[drmFrames]bool{true, true},
			[drmFrames]bool{true, false},
		},
		{
			"deux événements dans une lecture",
			append(
				event(drmEventFlipComplete, drmEventVblankSize, 0),
				event(drmEventFlipComplete, drmEventVblankSize, 1)...,
			),
			[drmFrames]bool{true, true},
			[drmFrames]bool{false, false},
		},
		{
			"un type inconnu se saute par sa longueur",
			append(
				event(0x99, drmEventVblankSize+8, 0),
				event(drmEventFlipComplete, drmEventVblankSize, 1)...,
			),
			[drmFrames]bool{true, true},
			[drmFrames]bool{true, false},
		},
		{
			"un retour de balayage ne libère rien",
			event(drmEventVblankType, drmEventVblankSize, 0),
			[drmFrames]bool{true, true},
			[drmFrames]bool{true, true},
		},
		{
			"un index hors borne est ignoré",
			event(drmEventFlipComplete, drmEventVblankSize, 42),
			[drmFrames]bool{true, true},
			[drmFrames]bool{true, true},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatalf("montage du tube : %v", err)
			}
			t.Cleanup(func() { read.Close() })

			if _, err := write.Write(c.stream); err != nil {
				t.Fatalf("écriture du flux d'essai : %v", err)
			}
			write.Close()

			// Le descripteur se relève comme la carte le fait, et jamais par
			// `os.File.Fd` : sur un tube, qui est toujours confié au scrutateur de
			// Go, `Fd` le remettrait en mode bloquant et la boucle de lecture
			// attendrait là où le pilote n'attend pas.
			fd, err := drmRawFd(read)
			if err != nil {
				t.Fatalf("relevé du descripteur du tube : %v", err)
			}

			card := &drmCard{file: read, fd: fd}
			for i := range card.frames {
				card.frames[i].pending = c.pending[i]
			}
			card.drain()

			for i := range card.frames {
				if got := card.frames[i].pending; got != c.want[i] {
					t.Errorf("le tampon %d est en vol : %v, attendu %v", i, got, c.want[i])
				}
			}
		})
	}
}
