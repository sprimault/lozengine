// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Où les octets d'un pixel du moteur doivent aller pour que le serveur les lise comme
// on les a peints.
//
// C'est ici que la frontière du noyau se paie, et c'était prévu : le tampon que le
// noyau rend est du RVB à alpha prémultiplié et cette disposition ne se négocie pas,
// donc chaque backend pose ce tampon comme sa plateforme l'exige et la conversion est
// à lui. X11 impose les siens, et la spécification est sans ambiguïté : les images sont
// transmises dans le format, ordre des octets compris, que le serveur spécifie.
//
// **Windows fait le même échange, et cette tête annonçait le contraire jusqu'au
// 29 octobre 2026** : « Windows n'en a aucune à faire — il décrit ses masques au
// système, qui lit le tampon là où il est. » La description est permise et la copie
// bien épargnée, mais GDI convertit alors pixel par pixel, pour quarante-sept fois le
// prix. Les deux backends convertissent donc, et par la même voie — composer le mot
// depuis les composantes, sans conversion de pointeur.

package backend

import (
	"fmt"
	"math/bits"

	"github.com/sprimault/lozengine/render"
)

// pixelKind dit quelle boucle de conversion convient au visuel du serveur.
//
// Trois valeurs. [pixelDirect] quand les trois composantes tombent déjà dans l'ordre
// du moteur ; [pixelSwapped] quand il suffit d'échanger la première et la troisième,
// qui est le cas de tout serveur petit-boutien à visuel RVB — donc celui qu'on
// rencontre en pratique ; [pixelMapped] pour le reste, où chaque composante va où son
// masque la veut.
type pixelKind uint8

// Les trois dispositions reconnues, de la moins coûteuse à la plus générale.
const (
	pixelDirect pixelKind = iota
	pixelSwapped
	pixelMapped
)

// pixelOrder porte la disposition d'un pixel du serveur, calculée une fois à
// l'ouverture.
//
// **Elle se dérive des masques du visuel et d'`image-byte-order`, jamais de
// l'architecture de la machine.** Un échange codé en dur serait juste sur le serveur
// qu'on a sous les yeux et faux ailleurs sans que rien ne le dise : les couleurs
// sortiraient permutées, ce qu'aucune erreur de protocole ne signale et qu'un job sans
// écran ne verrait pas.
//
// **Le brief du jalon s'est trompé sur un cas, et c'est le calcul qui l'a montré.** Il
// annonçait qu'un serveur gros-boutien aux mêmes masques recevrait le tampon tel quel.
// C'est faux : avec les masques courants, un serveur gros-boutien attend les octets
// dans l'ordre `_RVB`, soit un décalage et non un échange. Le cas « tel quel » existe,
// mais c'est celui d'un serveur petit-boutien à visuel BVR.
type pixelOrder struct {
	kind pixelKind

	// at porte l'index d'octet où vont le rouge, le vert et le bleu.
	at [3]int

	// size est la place d'un pixel sur la connexion, en octets. Toujours quatre, ce
	// que [pixelOrderOf] vérifie, et les boucles s'appuient dessus.
	size int
}

// pixelOrderOf calcule la disposition qu'un serveur attend, d'après son handshake.
//
// Les deux refus qu'elle porte nomment ce qu'ils ont trouvé. Ils sont préférables à
// une conversion approximative : une image aux couleurs permutées ressemble à un
// défaut du rastériseur, et on la cherche là où elle n'est pas.
func pixelOrderOf(s setup) (pixelOrder, error) {
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

	switch o.at {
	case [3]int{0, 1, 2}:
		o.kind = pixelDirect
	case [3]int{2, 1, 0}:
		o.kind = pixelSwapped
	default:
		o.kind = pixelMapped
	}
	return o, nil
}

// appendPixels écrit les pixels à la suite de dst, dans l'ordre que le serveur attend,
// et rend dst agrandi.
//
// **Une copie est inévitable**, et ce n'est pas un défaut de cette fonction : un
// `[]render.Color` ne devient un `[]byte` que par une conversion de pointeur, et la
// liste close que la cible `vet` exige ne s'ouvre pas pour une optimisation qu'aucune
// mesure n'a demandée. Le brief l'avait prévu — la conversion devient la passe de
// copie de toute façon inévitable vers le tampon d'émission.
//
// **Le choix de la boucle se fait une fois par appel, jamais par pixel.** C'est ce que
// les mesures du rastériseur ont établi : porter le cas jusqu'à la fonction de pixel
// coûte 2,8 % sur tous les chemins, là où un branchement par séquence est gratuit.
//
// **Le quatrième octet n'est pas écrit.** En profondeur 24 sur trente-deux bits, les
// huit bits hors masques sont indéfinis et le serveur les ignore ; les mettre à zéro
// coûterait une écriture de plus par pixel pour rien. Ce qui en découle pour le lot
// qui relira l'image : comparer des mots bruts serait intermittent, il faut masquer
// des deux côtés.
func (o pixelOrder) appendPixels(dst []byte, pixels []render.Color) []byte {
	at := len(dst)

	// Le tampon d'émission est réutilisé d'une image à l'autre : on l'étend sans
	// passer par une tranche temporaire, que `append` d'une tranche de zéros
	// allouerait à chaque image.
	need := at + o.size*len(pixels)
	if cap(dst) < need {
		grown := make([]byte, need, 2*need)
		copy(grown, dst)
		dst = grown
	} else {
		dst = dst[:need]
	}
	out := dst[at:]

	// La taille d'un pixel est quatre, vérifiée à l'ouverture. L'écrire en clair
	// dans les boucles vaut mieux que de lire un champ : le compilateur en tire les
	// décalages, et c'est la boucle la plus parcourue du backend.
	switch o.kind {
	case pixelDirect:
		for i, c := range pixels {
			j := 4 * i
			out[j], out[j+1], out[j+2] = c.R, c.G, c.B
		}

	case pixelSwapped:
		for i, c := range pixels {
			j := 4 * i
			out[j], out[j+1], out[j+2] = c.B, c.G, c.R
		}

	default:
		r, g, b := o.at[0], o.at[1], o.at[2]
		for i, c := range pixels {
			j := 4 * i
			out[j+r], out[j+g], out[j+b] = c.R, c.G, c.B
		}
	}
	return dst
}
