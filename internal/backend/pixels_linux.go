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
// prix. Les deux backends convertissent donc.
//
// **Mais pas par la même voie, et l'affirmer était une déduction** — « composer le mot
// depuis les composantes » décrivait Windows et non ce fichier, où les trois boucles
// écrivaient trois octets. Ce qui les sépare est structurel : Windows écrit dans un
// `[]uint32`, donc composer un mot est le geste naturel ; ici la destination est un
// `[]byte` dont l'ordre vient du serveur, donc poser un mot demande de le sérialiser.
// Mesuré le 31 octobre 2026, cette sérialisation gagne tout de même 11 à 18 % sur les
// deux dispositions rapides, et c'est elle qu'elles emploient. La disposition générale
// garde ses trois octets : elle lit les index au lieu de les tenir en clair, donc le
// mot ne s'y compose pas à coût constant.

package backend

import (
	"encoding/binary"
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
// **Une copie est inévitable sauf dans un cas, et ce n'est pas un défaut de cette
// fonction** : un `[]render.Color` ne devient un `[]byte` que par une conversion de
// pointeur, que la liste close de la cible `vet` n'autorise pas. Le brief l'avait
// prévu — la conversion devient la passe de copie de toute façon inévitable vers le
// tampon d'émission.
//
// Ce que la conversion de pointeur achèterait, mesuré le 31 octobre 2026 et non plus
// supposé : elle ne vaut que pour [pixelDirect], seule disposition où l'ordre du
// serveur est déjà celui du moteur — soit le cas qu'on ne rencontre pas, le visuel
// courant étant échangé. Le chemin qui compte garderait sa copie, donc l'exception à
// la liste close n'achèterait rien là où le coût est.
//
// **Le choix de la boucle se fait une fois par appel, jamais par pixel.** C'est ce que
// les mesures du rastériseur ont établi : porter le cas jusqu'à la fonction de pixel
// coûte 2,8 % sur tous les chemins, là où un branchement par séquence est gratuit.
//
// **Les deux dispositions rapides composent le mot et le posent d'un coup**, ce qui
// écrit l'octet hors masques à zéro. Un rangement de trente-deux bits au lieu de trois
// de huit : mesuré sur la machine de contrôle, 18,1 % plus rapide en direct et 11,1 %
// en échangé à 1920×1080, en trois séries dont la base remesurée. Le godoc disait
// l'inverse jusqu'au 31 octobre 2026 — « les mettre à zéro coûterait une écriture de
// plus par pixel pour rien » —, et c'était une déduction : il n'y a pas d'écriture de
// plus, il y en a deux de moins. Voir [BenchmarkAppendPixels].
//
// La disposition générale, elle, écrit toujours trois octets aux index que les masques
// désignent, et laisse le quatrième tel quel. Ce qui en découle pour le lot qui relira
// l'image : sous cette disposition, comparer des mots bruts reste intermittent et il
// faut masquer des deux côtés.
//
// **Ce que cette fonction coûte, et c'est le maillon le plus chaud du backend X11** :
// 1,74 ms en direct et 1,90 ms en échangé à 1920×1080 sur la machine de contrôle, soit
// 10,5 et 11,4 % du budget d'une image à soixante par seconde. Elle est limitée par ses
// opérations et non par la bande passante — 8,3 Mo écrits, donc environ 4,4 Go/s sur une
// machine qui en porte deux cent cinquante —, donc sa part brute est déjà sa part et ne
// se pondère pas. Ce qui resterait à gagner demanderait `unsafe` dans le cœur, et le
// déclencheur est nommé dans les règles du dépôt : le jour où ce backend manque ses
// images.
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
	// **`LittleEndian` n'est pas une hypothèse sur le serveur, et ce serait la lire à
	// l'envers.** L'ordre d'octets du serveur est déjà replié dans `o.at` par
	// [pixelOrderOf], qui en a tiré la disposition ; ce qu'on demande ici est de poser
	// la composante d'index zéro en premier, ce que seul `LittleEndian` garantit — et
	// il le garantit sur n'importe quelle machine, là que `NativeEndian` en dépendrait.
	case pixelDirect:
		for i, c := range pixels {
			binary.LittleEndian.PutUint32(out[4*i:],
				uint32(c.R)|uint32(c.G)<<8|uint32(c.B)<<16)
		}

	case pixelSwapped:
		for i, c := range pixels {
			binary.LittleEndian.PutUint32(out[4*i:],
				uint32(c.B)|uint32(c.G)<<8|uint32(c.R)<<16)
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
