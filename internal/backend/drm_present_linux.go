// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Comment une image arrive à l'écran sans serveur graphique : un mode posé une fois, puis
// des bascules au retour de balayage.

package backend

import (
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// Les bornes de lecture d'un événement de la carte, dérivées des structures du relevé.
//
// **Elles se dérivent et ne se recopient pas**, pour la raison qui vaut pour les requêtes :
// une structure élargie changerait ces trois nombres, et les écrire en clair les ferait
// mentir en silence. `Sizeof` et `Offsetof` ne convertissent rien, donc rien de tout cela
// n'entre dans la liste close des conversions.
const (
	drmEventHeaderSize   = int(unsafe.Sizeof(drmEventHeader{}))
	drmEventVblankSize   = int(unsafe.Sizeof(drmEventVblank{}))
	drmEventVblankDataAt = int(unsafe.Offsetof(drmEventVblank{}.userData))
)

// mount fabrique les tampons de la carte et la rend prête à présenter.
//
// **Il se fait après la découverte et pas pendant**, parce que les deux ont des raisons
// d'échouer différentes : une carte qu'on découvre sans pouvoir y fabriquer de tampon est
// une carte dont la cascade d'ouverture doit se détourner, là qu'une découverte qui ne
// trouve pas d'écran n'est même pas une erreur.
//
// La disposition accordée est celle du premier tampon, et les suivants la reprennent : le
// pilote n'a aucune raison d'accorder deux formats différents pour la même géométrie, et
// lui en laisser la possibilité obligerait à porter une disposition par tampon pour un
// régime qui n'existe pas.
func (c *drmCard) mount() error {
	width, height := c.size()

	for i := range c.frames {
		frame, order, err := drmNewFrame(c.fd, width, height)
		if err != nil {
			c.unmount()
			return err
		}
		c.frames[i] = frame
		if i == 0 {
			c.order = order
		}
	}

	return nil
}

// unmount rend les tampons au pilote.
func (c *drmCard) unmount() {
	for i := range c.frames {
		c.frames[i].drop(c.fd)
	}
}

// Present pose l'image dans un tampon libre et demande au contrôleur de l'afficher.
//
// **Le premier tour ne bascule pas, il pose le mode**, et c'est ce qui allume l'écran :
// `SETCRTC` attache le tampon au contrôleur et lui donne le mode retenu. Les tours
// suivants n'ont plus qu'à échanger les tampons, ce qu'une bascule fait au retour de
// balayage.
//
// Trois raisons de ne rien faire, et aucune n'est un échec, ce que le contrat prévoit : la
// carte est fermée, l'image est vide, ou une bascule est déjà en vol — le contrôleur garde
// alors la dernière image, et le tour suivant réessaiera. Ce qui est terminal passe par
// [drmCard.fail] : un mode qu'on ne peut plus poser n'est pas un tour perdu.
//
// **`EBUSY` n'est pas une erreur ici et c'est l'erreur de conception à ne pas faire** : le
// pilote le rend quand une bascule n'a pas encore été honorée, c'est-à-dire quand on
// présente plus vite que l'écran n'affiche. Le traiter comme une panne fermerait la fenêtre
// d'un jeu qui tourne simplement plus vite que son moniteur.
func (c *drmCard) Present(t *render.Buffer) {
	if c.closed || t == nil || len(t.Pixels) == 0 {
		return
	}

	c.drain()

	i := c.free()
	if i < 0 {
		return
	}
	f := &c.frames[i]
	f.write(c.order, t)

	fd := c.fd
	if !c.lit {
		if err := c.light(fd, f); err != nil {
			c.fail(err)
			return
		}
		c.lit, c.front = true, i
		return
	}

	flip := drmModeCrtcPageFlip{
		crtcID:   c.crtc,
		fbID:     f.fb,
		flags:    drmPageFlipEvent,
		userData: uint64(i),
	}
	if err := drmIoctl(fd, drmModePageFlip, &flip); err != nil {
		if errors.Is(err, syscall.EBUSY) {
			return
		}
		c.fail(fmt.Errorf("flipping to buffer %d: %w", i, err))
		return
	}
	f.pending, c.front = true, i
}

// light pose le mode et allume l'écran sur le tampon donné.
//
// **La maîtrise de l'affichage ne se prend pas, et c'est une décision.** Le noyau l'accorde
// implicitement au premier client qui ouvre le nœud, ce qui est le régime d'une machine
// sans serveur graphique — mesuré le 4 novembre 2026, `SETCRTC` y réussit sans
// `SET_MASTER`. La demander n'achèterait donc rien là où on l'a déjà, et là où un
// compositeur tourne elle reviendrait à lui prendre l'écran. Le refus que le pilote oppose
// alors est exactement ce que la cascade d'ouverture doit lire : cette famille n'est pas la
// bonne sur cette machine.
func (c *drmCard) light(fd int, f *drmFrame) error {
	// Le connecteur est nommé par un tableau d'un élément, que la requête lit par son
	// adresse : il doit donc survivre à l'appel comme n'importe quel tableau
	// d'énumération, d'où le passage par [drmIoctlInto].
	connectors := []uint32{c.connector}

	set := drmModeCrtc{
		setConnectorsPtr: drmArrayAddr(connectors),
		countConnectors:  uint32(len(connectors)),
		crtcID:           c.crtc,
		fbID:             f.fb,
		modeValid:        1,
		mode:             c.mode,
	}
	if err := drmIoctlInto(fd, drmModeSetCrtc, &set, connectors); err != nil {
		return fmt.Errorf("lighting connector %d on controller %d: %w", c.connector, c.crtc, err)
	}
	return nil
}

// free rend l'index d'un tampon libre, ou -1 quand tout est en vol ou à l'écran.
//
// Celui qui est affiché n'est pas libre : le contrôleur le balaie en continu, et y écrire
// produirait du déchirement qu'aucune erreur ne signale.
func (c *drmCard) free() int {
	for i := range c.frames {
		if !c.frames[i].pending && (!c.lit || i != c.front) {
			return i
		}
	}
	return -1
}

// drain lit ce que le contrôleur a annoncé depuis le tour précédent.
//
// **Une bascule honorée est la seule chose qui libère un tampon**, donc une présentation
// qui ne lirait pas trouverait tout en vol au second tour et ne montrerait plus rien.
//
// La lecture est non bloquante et son absence de données n'est pas une erreur : la plupart
// des tours n'ont rien à lire. Une lecture courte ne l'est pas non plus — le noyau rend un
// nombre entier d'événements, et la boucle avance par la longueur que chaque en-tête
// déclare plutôt que par la taille de la structure qu'elle a reconnue, ce qui lui permet de
// sauter un type qu'elle ne connaît pas sans se désynchroniser.
//
// **Les trois champs utiles se décodent un par un plutôt que par une relecture de la
// structure**, et ce n'est pas un détour : relire les octets comme un [drmEventVblank]
// demanderait une conversion de pointeur de plus, pour lire trois entiers dont les
// décalages sont figés par un cas. L'ordre d'octets est celui de la machine, l'événement
// venant du noyau et non d'une socket.
func (c *drmCard) drain() {
	var buf [256]byte

	for {
		n, err := syscall.Read(c.fd, buf[:])
		if n <= 0 || err != nil {
			return
		}

		for at := 0; at+drmEventHeaderSize <= n; {
			kind := binary.NativeEndian.Uint32(buf[at:])
			size := int(binary.NativeEndian.Uint32(buf[at+4:]))
			if size < drmEventHeaderSize || at+size > n {
				return
			}

			if kind == drmEventFlipComplete && size >= drmEventVblankSize {
				data := binary.NativeEndian.Uint64(buf[at+drmEventVblankDataAt:])
				if i := int(data); i >= 0 && i < len(c.frames) {
					c.frames[i].pending = false
				}
			}
			at += size
		}
	}
}

// fail retient ce qui a mis fin à la présentation autrement que par la volonté du joueur.
//
// Seule la première cause est gardée, comme les deux autres familles le font : ce qui suit
// un affichage perdu n'en est que la conséquence.
func (c *drmCard) fail(err error) {
	if c.err == nil {
		c.err = err
	}
	c.closed = true
}
