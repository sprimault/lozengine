// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Comment un serveur X11 annonce la disposition qu'il veut, et ce qu'on en tire. La
// conversion elle-même est partagée, dans `pixels.go` : ce fichier ne porte que la
// lecture du handshake.
//
// La spécification est sans ambiguïté sur ce qu'il y a à lire : les images sont
// transmises dans le format, ordre des octets compris, que le serveur spécifie. Ce
// format se déduit de deux choses qu'il déclare à la connexion — les masques de son
// visuel, et `image-byte-order`.

package backend

import (
	"fmt"
	"math/bits"
)

// x11PixelOrderOf calcule la disposition qu'un serveur attend, d'après son handshake.
//
// Les deux refus qu'elle porte nomment ce qu'ils ont trouvé. Ils sont préférables à une
// conversion approximative : une image aux couleurs permutées ressemble à un défaut du
// rastériseur, et on la cherche là où elle n'est pas.
//
// **Elle ne produit jamais de disposition à alpha**, et c'est ce qui la distingue de
// [pixelOrderOfFourCC], qui doit les refuser : un visuel X11 ne déclare que trois
// masques, donc le quatrième octet est libre par construction.
func x11PixelOrderOf(s setup) (pixelOrder, error) {
	var o pixelOrder

	// Trente-deux bits par pixel, et c'est une condition de la méthode plus qu'une
	// préférence : une permutation d'octets ne sait pas replier huit bits de
	// composante sur cinq, ce qu'un visuel de profondeur seize demanderait. Celui-là
	// est un autre chemin de rendu, pas un autre ordre.
	if s.bitsPerPixel != 32 {
		return o, fmt.Errorf("the display server wants %d bits per pixel, and this backend writes 32", s.bitsPerPixel)
	}
	o.size = int(s.bitsPerPixel) / 8

	for i, mask := range [3]uint32{s.redMask, s.greenMask, s.blueMask} {
		low := bits.TrailingZeros32(mask)
		if bits.OnesCount32(mask) != 8 || low%8 != 0 {
			return o, fmt.Errorf("the visual's channel masks are %#x/%#x/%#x, and this backend writes whole aligned bytes",
				s.redMask, s.greenMask, s.blueMask)
		}

		// Le masque dit où la composante vit dans le mot, l'ordre des octets dit
		// comment ce mot part sur la connexion. Sur un serveur gros-boutien, le
		// premier octet émis est celui de poids fort : l'index s'inverse.
		at := low / 8
		if s.imageOrder == msbFirst {
			at = o.size - 1 - at
		}
		o.at[i] = at
	}

	o.kind = pixelKindOf(o.at)
	return o, nil
}
