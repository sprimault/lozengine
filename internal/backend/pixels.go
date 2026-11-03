// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Où les octets d'un pixel du moteur doivent aller pour que la destination les lise
// comme on les a peints, et la relecture en octets pour les destinations qui n'ont rien
// à faire.
//
// C'est ici que la frontière du noyau se paie, et c'était prévu : le tampon que le noyau
// rend est du RVB à alpha prémultiplié et cette disposition ne se négocie pas, donc
// chaque backend pose ce tampon comme sa plateforme l'exige et la conversion est à lui.
// Ce fichier porte la conversion elle-même, que les familles se partagent ; ce qui reste
// propre à chacune est la façon d'apprendre quelle disposition sa plateforme veut.
//
// **Windows fait le même échange, et la tête de l'ancien fichier X11 annonçait le
// contraire jusqu'au 29 octobre 2026** : « Windows n'en a aucune à faire — il décrit ses
// masques au système, qui lit le tampon là où il est. » La description est permise et la
// copie bien épargnée, mais GDI convertit alors pixel par pixel, pour quarante-sept fois
// le prix.
//
// **Il ne convertit pas par la même voie pour autant, et l'affirmer était une
// déduction** — « composer le mot depuis les composantes » décrivait Windows et non ce
// chemin, où les trois boucles écrivaient trois octets. Ce qui les sépare est
// structurel : Windows écrit dans un `[]uint32`, donc composer un mot est le geste
// naturel ; ici la destination est un `[]byte` dont l'ordre vient de la plateforme, donc
// poser un mot demande de le sérialiser. Mesuré le 31 octobre 2026, cette sérialisation
// gagne tout de même 11 à 18 % sur les deux dispositions rapides, et c'est elle qu'elles
// emploient. La disposition générale garde ses trois octets : elle lit les index au lieu
// de les tenir en clair, donc le mot ne s'y compose pas à coût constant.
//
// Et c'est pourquoi ce fichier n'abrite pas la conversion de Windows : elle écrit
// ailleurs, pas autrement.

package backend

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// pixelKind dit quelle boucle de conversion convient à la destination.
//
// Trois valeurs. [pixelDirect] quand les trois composantes tombent déjà dans l'ordre du
// moteur ; [pixelSwapped] quand il suffit d'échanger la première et la troisième, qui
// est le cas de tout serveur X11 petit-boutien à visuel RVB — donc celui qu'on rencontre
// en pratique de ce côté-là ; [pixelMapped] pour le reste, où chaque composante va où la
// plateforme la veut.
type pixelKind uint8

// Les trois dispositions reconnues, de la moins coûteuse à la plus générale.
const (
	pixelDirect pixelKind = iota
	pixelSwapped
	pixelMapped
)

// pixelOrder porte la disposition d'un pixel de la destination, calculée une fois à
// l'ouverture.
//
// **Elle se dérive de ce que la plateforme annonce, jamais de l'architecture de la
// machine.** Un échange codé en dur serait juste sur la destination qu'on a sous les yeux
// et faux ailleurs sans que rien ne le dise : les couleurs sortiraient permutées, ce
// qu'aucune erreur de protocole ne signale et qu'un job sans écran ne verrait pas.
//
// **Deux voies y mènent, et c'est ce qui place la structure ici plutôt que dans l'une
// des deux.** [x11PixelOrderOf] la tire des masques du visuel et de l'ordre d'octets
// qu'un serveur X11 déclare ; [pixelOrderOfFourCC] la tire du code à quatre caractères
// que Wayland et DRM/KMS emploient tous les deux. Ce que la structure décrit est la
// **destination** et non la source, et c'est pourquoi les deux s'y projettent : où va
// chaque composante dans les quatre octets.
//
// **Le brief du jalon 3 s'est trompé sur un cas, et c'est le calcul qui l'a montré.** Il
// annonçait qu'un serveur gros-boutien aux mêmes masques recevrait le tampon tel quel.
// C'est faux : avec les masques courants, un serveur gros-boutien attend les octets dans
// l'ordre `_RVB`, soit un décalage et non un échange. Le cas « tel quel » existe, mais
// c'est celui d'un serveur petit-boutien à visuel BVR — ou, côté codes à quatre
// caractères, celui de `XB24`.
type pixelOrder struct {
	kind pixelKind

	// at porte l'index d'octet où vont le rouge, le vert et le bleu.
	at [3]int

	// size est la place d'un pixel à destination, en octets. Toujours quatre, ce que
	// les deux constructeurs vérifient, et les boucles s'appuient dessus.
	size int
}

// fourcc compose un code de format à partir de ses quatre caractères.
//
// C'est la macro `fourcc_code` des en-têtes UAPI du noyau, relevée telle quelle. Les
// deux familles qui s'en servent partagent ce registre — DRM/KMS par ses `DRM_FORMAT_…`
// et Wayland par `wl_shm.format` —, et une valeur y désigne la même disposition de part
// et d'autre : c'est ce qui fait qu'un seul constructeur sert les deux.
func fourcc(a, b, c, d byte) uint32 {
	return uint32(a) | uint32(b)<<8 | uint32(c)<<16 | uint32(d)<<24
}

// Les quatre codes de format que le moteur sait écrire, tels que les en-têtes UAPI les
// définissent, de type `uint32`.
//
// Le commentaire de chacun donne d'abord la disposition du mot de trente-deux bits du
// bit 31 au bit 0, comme la source l'écrit, puis l'ordre des octets en mémoire sur une
// machine petit-boutienne — qui est ce dont la conversion a besoin. Les deux se lisent à
// l'envers l'un de l'autre, et c'est le piège de cette table.
var (
	formatXRGB8888 = fourcc('X', 'R', '2', '4') // x:R:G:B, soit B, V, R, x en mémoire
	formatXBGR8888 = fourcc('X', 'B', '2', '4') // x:B:G:R, soit R, V, B, x en mémoire
	formatRGBX8888 = fourcc('R', 'X', '2', '4') // R:G:B:x, soit x, B, V, R en mémoire
	formatBGRX8888 = fourcc('B', 'X', '2', '4') // B:G:R:x, soit x, R, V, B en mémoire
)

// pixelOrderOfFourCC rend la disposition qu'un code de format demande.
//
// **Les formats à alpha sont refusés, et ce n'est pas de la prudence : ils seraient
// faux.** Les deux dispositions rapides d'[pixelOrder.appendPixels] composent le mot
// entier, donc elles écrivent zéro dans l'octet que les composantes ne couvrent pas.
// C'est sans effet quand cet octet est ignoré, et cela rend la surface **entièrement
// transparente** quand il porte l'alpha — un défaut qu'aucune erreur de protocole ne
// signale. Le repli existe toujours : Wayland garantit `xrgb8888` à côté d'`argb8888`.
//
// Le refus nomme le code en clair plutôt qu'en nombre, un intégrateur ne reconnaissant
// pas un format à son hexadécimal.
func pixelOrderOfFourCC(code uint32) (pixelOrder, error) {
	o := pixelOrder{size: 4}

	switch code {
	case formatXBGR8888:
		o.at = [3]int{0, 1, 2}
	case formatXRGB8888:
		o.at = [3]int{2, 1, 0}
	case formatRGBX8888:
		o.at = [3]int{3, 2, 1}
	case formatBGRX8888:
		o.at = [3]int{1, 2, 3}
	default:
		return pixelOrder{}, fmt.Errorf(
			"format %s (%#08x) is not one this backend writes: it needs a 32-bit format whose remaining byte is unused",
			fourccName(code), code)
	}

	o.kind = pixelKindOf(o.at)
	return o, nil
}

// fourccName rend les quatre caractères d'un code de format.
//
// Un octet hors de l'imprimable devient un point d'interrogation : le but est qu'un
// message d'erreur reste lisible quand la plateforme a annoncé n'importe quoi, et non
// de restituer la valeur, que le message donne aussi en hexadécimal.
func fourccName(code uint32) string {
	b := [4]byte{byte(code), byte(code >> 8), byte(code >> 16), byte(code >> 24)}
	for i, c := range b {
		if c < ' ' || c > '~' {
			b[i] = '?'
		}
	}
	return string(b[:])
}

// pixelKindOf classe une disposition d'après les index où vont les composantes.
//
// Les deux constructeurs en font la même lecture, et c'est une copie et non une
// généralisation : ce qui est écrit une fois ici serait écrit à l'identique de part et
// d'autre.
func pixelKindOf(at [3]int) pixelKind {
	switch at {
	case [3]int{0, 1, 2}:
		return pixelDirect
	case [3]int{2, 1, 0}:
		return pixelSwapped
	default:
		return pixelMapped
	}
}

// appendPixels écrit les pixels à la suite de dst, dans l'ordre que la destination
// attend, et rend dst agrandi.
//
// **Une copie est inévitable sauf dans un cas, et ce n'est pas un défaut de cette
// fonction** : un `[]render.Color` ne devient un `[]byte` que par une conversion de
// pointeur, qui se décide en pull request et s'inscrit dans la liste close du Makefile.
// Le brief du jalon 3 l'avait prévu — la conversion devient la passe de copie de toute
// façon inévitable vers le tampon de destination.
//
// **Ce bloc attribuait cette garde à `vet`, et c'était faux** : son analyseur `unsafeptr`
// ne signale que les allers-retours par `uintptr`, donc il ne voit pas cette forme du
// tout. Mesuré le 1er novembre 2026, et la cible qui la garde depuis compte les sites par
// fichier.
//
// Ce que la conversion de pointeur achèterait, mesuré le 31 octobre 2026 et non plus
// supposé : elle ne vaut que pour [pixelDirect], seule disposition où l'ordre de la
// destination est déjà celui du moteur. Sous X11 c'est le cas qu'on ne rencontre pas, le
// visuel courant étant échangé ; c'est en revanche celui que les deux familles du jalon 6
// demandent, `XB24`, et le lot qui les mesure dira si le chemin change de nature.
//
// **Le choix de la boucle se fait une fois par appel, jamais par pixel.** C'est ce que
// les mesures du rastériseur ont établi : porter le cas jusqu'à la fonction de pixel
// coûte 2,8 % sur tous les chemins, là où un branchement par séquence est gratuit.
//
// **Les deux dispositions rapides composent le mot et le posent d'un coup**, ce qui écrit
// l'octet restant à zéro. Un rangement de trente-deux bits au lieu de trois de huit :
// mesuré sur la machine de contrôle, 18,1 % plus rapide en direct et 11,1 % en échangé à
// 1920×1080, en trois séries dont la base remesurée. Le godoc disait l'inverse jusqu'au
// 31 octobre 2026 — « les mettre à zéro coûterait une écriture de plus par pixel pour
// rien » —, et c'était une déduction : il n'y a pas d'écriture de plus, il y en a deux de
// moins. Voir [BenchmarkAppendPixels].
//
// **Et c'est cet octet à zéro qui interdit un format à alpha**, puisqu'il le mettrait à
// zéro aussi : [pixelOrderOfFourCC] les refuse pour cette raison, et un visuel X11 n'en
// propose pas, ses masques ne couvrant que trois composantes.
//
// La disposition générale, elle, écrit toujours trois octets aux index que la plateforme
// désigne, et laisse le quatrième tel quel. Ce qui en découle pour un lot qui relirait
// l'image : sous cette disposition, comparer des mots bruts reste intermittent et il faut
// masquer des deux côtés.
//
// **Ce que cette fonction coûte, et c'est le maillon le plus chaud du backend X11** :
// 1,74 ms en direct et 1,90 ms en échangé à 1920×1080 sur la machine de contrôle, soit
// 10,5 et 11,4 % du budget d'une image à soixante par seconde. Elle est limitée par ses
// opérations et non par la bande passante — 8,3 Mo écrits, donc environ 4,4 Go/s sur une
// machine qui en porte deux cent cinquante —, donc sa part brute est déjà sa part et ne
// se pondère pas. **Ces deux chiffres valent pour une destination X11**, c'est-à-dire un
// tampon d'émission du tas de Go ; une mémoire partagée projetée ou un tampon dumb du
// noyau sont une autre destination, et le lot qui présente les mesure. Ce qui resterait à
// gagner demanderait `unsafe` dans le cœur, et le déclencheur est nommé dans les règles
// du dépôt : le jour où un de ces backends manque ses images.
func (o pixelOrder) appendPixels(dst []byte, pixels []render.Color) []byte {
	at := len(dst)

	// Le tampon de destination est réutilisé d'une image à l'autre : on l'étend sans
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

	// La taille d'un pixel est quatre, vérifiée à l'ouverture. L'écrire en clair dans
	// les boucles vaut mieux que de lire un champ : le compilateur en tire les
	// décalages, et c'est la boucle la plus parcourue des backends.
	switch o.kind {
	// **`LittleEndian` n'est pas une hypothèse sur la destination, et ce serait la lire
	// à l'envers.** L'ordre d'octets qu'elle veut est déjà replié dans `o.at` par le
	// constructeur, qui en a tiré la disposition ; ce qu'on demande ici est de poser la
	// composante d'index zéro en premier, ce que seul `LittleEndian` garantit — et il le
	// garantit sur n'importe quelle machine, là que `NativeEndian` en dépendrait.
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

// bytesOf relit une tranche de pixels comme une tranche d'octets, sans rien copier ni
// convertir.
//
// **Elle sert les destinations qui prennent la disposition du moteur telle quelle**, et
// c'est celle de [render.Color] qui le permet : quatre `uint8` nommés R, G, B et A dans
// cet ordre. Le navigateur est dans ce cas par la spécification du canvas, puisque
// `ImageData.data` attend exactement cet ordre ; Wayland et DRM/KMS y sont dès qu'ils
// obtiennent `XB24`, le code de format que les deux demandent d'abord. Les destinations
// qui veulent un autre ordre passent par [pixelOrder.appendPixels], qui convertit.
//
// # Pourquoi une conversion de pointeur, là que X11 l'a refusée
//
// Le refus de X11 est écrit dans son propre fichier, et son argument ne vaut pas ici : il
// doit permuter des composantes, donc il parcourt les pixels de toute façon et la
// conversion ne lui épargnerait qu'une copie — 0,45 ms sur un budget de 16,7. Une
// destination qui n'a rien à permuter n'a au contraire que cela à faire : la conversion
// **est** toute l'opération, et la boucle qu'elle remplace ne changerait pas un octet.
// Mesuré le 1er novembre 2026, elle vaut un facteur douze sur le chemin de présentation
// du navigateur.
//
// Elle s'inscrit donc dans la liste close du Makefile, que `make conversions` tient par
// fichier et par compte. C'est ce qui rend la décision visible en relecture, et c'est
// nécessaire parce que `go vet` ne voit pas cette forme : son analyseur `unsafeptr` ne
// signale que les allers-retours par `uintptr`.
//
// # Ce que la fonction ne fait pas, et qui appartient à l'appelant
//
// Elle ne retient rien : la tranche rendue vit le temps de l'appel qui la consomme. Une
// vue tenue d'une image à l'autre est un autre sujet — elle se détache quand le tas de Go
// croît, et la réallocation d'une surface en déplace l'adresse sans la détacher —, donc
// c'est à l'appelant de la refaire, et d'épingler ce qu'il garde.
//
// La tranche vide rend une tranche vide plutôt que d'indexer un premier élément qui
// n'existe pas.
func bytesOf(pixels []render.Color) []byte {
	if len(pixels) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&pixels[0])), len(pixels)*4)
}
