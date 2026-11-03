// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La table des places que le noyau nomme : les codes de touche qu'`evdev` et le moteur
// partagent, les dix-huit qui divergent, et les boutons d'une souris.
//
// Elle sert les familles Linux ensemble. Wayland lui passe le code du noyau tel quel ;
// X11 retranche d'abord les huit que son protocole réserve, et ce décalage reste chez lui.

package backend

import "github.com/sprimault/lozengine/input"

// evdevShared est le dernier code evdev qui désigne la même place que la convention du
// moteur.
//
// La coïncidence sur toute cette plage est un fait mesuré plutôt qu'une chance : vérifiée
// le 27 octobre 2026 contre les en-têtes UAPI du noyau, sans une exception. Au-delà, les
// deux suites divergent — le jeu 1 atteignait ces touches par un préfixe d'échappement là
// où evdev leur a donné des codes à la suite, et c'est ce que [extendedKeys] rattrape.
const evdevShared = 88

// extendedKeys traduit les codes evdev des touches que le jeu 1 atteignait par un
// préfixe, et que le moteur désigne donc par un bit de plus.
//
// Dix-huit entrées, et le brief du jalon 3 en annonçait dix-neuf : l'écart est réel et la
// liste ci-dessous est celle que les codes relevés donnent. La dix-neuvième n'existe pas,
// à moins qu'on ait voulu compter la touche de pause, écartée dans [x11KeyOf].
//
// Une place vide vaut absence de correspondance. Zéro n'est pas une place du moteur — le
// jeu 1 n'a pas de code nul — donc la valeur sert de sentinelle sans ambiguïté.
//
// **Trois d'entre elles ne portent pas de nom dans le moteur**, et c'est voulu : la godoc
// de [input.Key] dit qu'une place sans nom se lit par sa valeur et désigne la même touche
// partout. Les traduire quand même est la condition de cette promesse — les laisser de
// côté rendrait la place muette sous Linux et vivante sous Windows.
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

// evdevKeyOf traduit un code de touche du noyau vers la place du moteur, et dit si elle
// existe.
//
// **Elle ne refuse pas les petits codes, et c'est ce qui la distingue de [x11KeyOf]** :
// en evdev, les codes 1 à 8 sont Échap et les chiffres 1 à 7. Le refus que le protocole
// X11 impose sous huit est donc propre à X11, et le déplacer ici perdrait huit touches en
// silence.
//
// Les codes inconnus sont écartés sans bruit. Un clavier exotique ou un pilote bavard est
// une donnée du monde extérieur, pas une faute de programmation.
func evdevKeyOf(code uint32) (input.Key, bool) {
	if code >= 1 && code <= evdevShared {
		return input.Key(code), true
	}
	if code < uint32(len(extendedKeys)) && extendedKeys[code] != 0 {
		return extendedKeys[code], true
	}
	return 0, false
}

// Les codes de bouton de souris du noyau, relevés le 3 novembre 2026 dans les en-têtes
// UAPI, de type `uint32` comme les codes que les deux protocoles transportent.
//
// `BTN_TASK`, à `0x117`, n'est pas de la liste : le moteur ne nomme aucune place pour un
// sixième bouton, et lui en attribuer une d'office dirait autre chose que ce que le code
// désigne.
const (
	evdevButtonLeft    uint32 = 0x110
	evdevButtonRight   uint32 = 0x111
	evdevButtonMiddle  uint32 = 0x112
	evdevButtonSide    uint32 = 0x113
	evdevButtonExtra   uint32 = 0x114
	evdevButtonForward uint32 = 0x115
	evdevButtonBack    uint32 = 0x116
)

// evdevButtonOf traduit un code de bouton du noyau, et dit s'il en porte un que le moteur
// nomme.
//
// **Quatre codes pour deux places, et c'est une correspondance assumée.** Le noyau nomme
// les boutons latéraux de deux façons — `SIDE` et `EXTRA` d'un côté, `BACK` et `FORWARD`
// de l'autre — et une souris émet l'une ou l'autre paire selon son micrologiciel. Aucune
// des quatre ne veut dire autre chose que « précédent » ou « suivant », donc les réunir ne
// crée aucune ambiguïté de sens, là où n'en prendre qu'une paire rendrait muette la moitié
// des souris. C'est la différence avec la touche de pause, qu'[x11KeyOf] refuse parce
// qu'elle ne désigne pas la même chose partout.
//
// **Ce qui n'a pas été vérifié, et qu'il faut dire** : quels numéros le pilote de X11
// donne à ces quatre codes. [x11ButtonOf] en lit deux, 8 et 9 ; si le pilote numérote
// `BACK` et `FORWARD` au-delà, la même souris serait lue ici et pas là-bas. Le relever
// demande un serveur X et une souris à cinq boutons, et la correction porterait alors sur
// les deux familles dans le même lot.
func evdevButtonOf(code uint32) (input.Button, bool) {
	switch code {
	case evdevButtonLeft:
		return input.ButtonLeft, true
	case evdevButtonRight:
		return input.ButtonRight, true
	case evdevButtonMiddle:
		return input.ButtonMiddle, true
	case evdevButtonSide, evdevButtonBack:
		return input.ButtonBack, true
	case evdevButtonExtra, evdevButtonForward:
		return input.ButtonForward, true
	}
	return 0, false
}
