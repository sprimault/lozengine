// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La traduction d'un code de touche X11 vers la place que le moteur nomme.
//
// Elle est presque triviale, et c'est un fait mesuré plutôt qu'une chance : les codes
// evdev du noyau coïncident avec la convention du moteur sur ses quatre-vingt-huit
// premières places, sans une exception. Vérifié le 27 octobre 2026 contre les en-têtes
// UAPI du noyau — un code de touche est un fait d'interface, et la `Linux-syscall-note`
// existe précisément pour qu'un code non-GPL les emploie.
//
// **Il faut résister à la symétrie avec Windows, et c'est tout l'enjeu de ce fichier.**
// Deux dispositifs que le backend Windows a dû construire n'ont ici aucune
// contrepartie : le pavé numérique ne se devine pas d'après un verrou, parce qu'evdev
// le distingue nativement du bloc directionnel ; et l'état du clavier ne se relit pas
// touche par touche, parce que le serveur l'envoie entier de lui-même.

package backend

import "github.com/sprimault/lozengine/input"

// keycodeShift est l'écart entre un code de touche X11 et le code evdev du noyau.
//
// Le protocole réserve les codes 0 à 7 — il veut que les codes valides commencent à
// huit — et le pilote de clavier y pose les codes du noyau décalés d'autant. C'est le
// seul calcul de toute la traduction pour la grande majorité des touches.
const keycodeShift = 8

// evdevShared est le dernier code evdev qui désigne la même place que la convention du
// moteur.
//
// Au-delà, les deux suites divergent : le jeu 1 atteignait ces touches par un préfixe
// d'échappement là où evdev leur a donné des codes à la suite. C'est ce que
// [extendedKeys] rattrape.
const evdevShared = 88

// extendedKeys traduit les codes evdev des touches que le jeu 1 atteignait par un
// préfixe, et que le moteur désigne donc par un bit de plus.
//
// Dix-huit entrées, et le brief du jalon en annonçait dix-neuf : l'écart est réel et
// la liste ci-dessous est celle que les codes relevés donnent. La dix-neuvième n'existe
// pas, à moins qu'on ait voulu compter la touche de pause, écartée juste en dessous.
//
// Une place vide vaut absence de correspondance. Zéro n'est pas une place du moteur —
// le jeu 1 n'a pas de code nul — donc la valeur sert de sentinelle sans ambiguïté.
//
// **Trois d'entre elles ne portent pas de nom dans le moteur**, et c'est voulu : la
// godoc de [input.Key] dit qu'une place sans nom se lit par sa valeur et désigne la
// même touche partout. Les traduire quand même est la condition de cette promesse —
// les laisser de côté rendrait la place muette sous X11 et vivante sous Windows.
var extendedKeys = [...]input.Key{
	// Le pavé numérique et les modificateurs de droite.
	96:  input.KeyNumpadEnter,
	97:  input.KeyRightControl,
	98:  input.KeyNumpadDivide,
	99:  0x137, // impression d'écran
	100: input.KeyRightAlt,

	// Le bloc de navigation, puis les quatre flèches, qu'evdev sépare du pavé
	// numérique de lui-même.
	102: input.KeyHome,
	103: input.KeyUp,
	104: input.KeyPageUp,
	105: input.KeyLeft,
	106: input.KeyRight,
	107: input.KeyEnd,
	108: input.KeyDown,
	109: input.KeyPageDown,
	110: input.KeyInsert,
	111: input.KeyDelete,

	// Les trois touches qu'un clavier de bureau ajoute de part et d'autre de la
	// barre d'espace.
	125: 0x15B, // méta de gauche
	126: 0x15C, // méta de droite
	127: 0x15D, // menu contextuel
}

// keyOf traduit un code de touche X11 vers la place du moteur, et dit si elle existe.
//
// **La touche de pause n'y est pas, et c'est une correspondance refusée plutôt
// qu'oubliée.** Le jeu 1 l'envoyait en trois octets et Windows ne la rapporte pas
// uniformément : poser une équivalence ici la rendrait lisible sous X11 et pas sous
// Windows, pour la même valeur. Une place qui ne veut pas dire la même chose partout
// vaut moins qu'une place absente, puisque c'est l'universalité de la convention qui
// fait sa valeur.
//
// Les codes inconnus sont écartés sans bruit. Un clavier exotique ou un pilote bavard
// est une donnée du monde extérieur, pas une faute de programmation.
func keyOf(code byte) (input.Key, bool) {
	if code < keycodeShift {
		return 0, false
	}
	evdev := int(code) - keycodeShift

	if evdev >= 1 && evdev <= evdevShared {
		return input.Key(evdev), true
	}
	if evdev < len(extendedKeys) && extendedKeys[evdev] != 0 {
		return extendedKeys[evdev], true
	}
	return 0, false
}

// relearnKeys reprend l'état du clavier depuis le vecteur de bits que le serveur
// envoie.
//
// **C'est ce qui remplace la relecture touche par touche du backend Windows**, et
// l'écart de coût est le fait marquant du clavier X11 : là-bas la reprise du focus
// interroge le système pour chacune des places possibles, ici le serveur envoie l'état
// complet de lui-même après chaque prise de focus — à condition de l'avoir demandé, ce
// que le masque d'événements fait depuis le lot qui a créé la fenêtre.
//
// L'oubli préalable est voulu : ce que le vecteur ne nomme pas n'est pas enfoncé, et
// repartir de l'état réel est précisément l'objet de cet événement. Les touches tenues
// se rallument donc comme des appuis neufs, ce qui est aussi le choix de Windows.
//
// Le vecteur couvre les codes à partir de huit, un bit par touche : l'octet n porte
// les codes 8+8n à 15+8n, du bit de poids faible au bit de poids fort.
func (w *window) relearnKeys(bitmap []byte) {
	w.state.Forget()

	for at, mask := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if mask&(1<<bit) == 0 {
				continue
			}
			code := keycodeShift + 8*at + bit
			if code > 255 {
				// Un code de touche tient sur un octet, donc le vecteur ne peut
				// pas en porter plus de trente et un. Au-delà, c'est le serveur
				// qui parle d'autre chose que de ce protocole.
				return
			}
			if k, ok := keyOf(uint8(code)); ok {
				w.state.Press(k)
			}
		}
	}
}
