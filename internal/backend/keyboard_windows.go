// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La lecture du clavier : quatre messages, et le décodage du mot qui les accompagne.
//
// Le moteur nomme des places et non des caractères, donc c'est le code de balayage
// qu'on lit — pas le code virtuel, qui dépend de la disposition du joueur. Demander
// la traduction au système ne marcherait pas non plus : `MapVirtualKeyW` suit la
// disposition active, et sur un clavier français le code virtuel de A désigne la place
// du Q américain.

package backend

import "github.com/sprimault/lozengine/input"

// La disposition du mot qui accompagne un message de touche.
//
// Les champs qui nous concernent sont le code de balayage, le drapeau d'extension qui
// sépare deux places de même code, et le bit de transition qui dit si la touche monte
// ou descend. Le compte de répétition et le contexte n'ont pas d'usage ici.
const (
	scanCodeShift = 16
	scanCodeMask  = 0xFF

	extendedShift = 24
	extendedBit   = 1 << extendedShift
)

// keyOf rend la place que désigne le mot d'un message de touche.
//
// Le bit d'extension devient le neuvième bit de la place, ce qui sépare la flèche
// gauche du 4 du pavé numérique — tous deux 0x4B au clavier.
func keyOf(lParam uintptr) input.Key {
	key := input.Key(lParam>>scanCodeShift) & scanCodeMask
	if lParam&extendedBit != 0 {
		key |= extendedKey
	}
	return key
}

// extendedKey est le bit que le moteur ajoute à une place étendue. Il double la valeur
// publiée par le paquet des entrées, qui ne l'exporte pas : c'est un détail de codage
// des places, et un jeu nomme les places plutôt que de les composer.
const extendedKey input.Key = 0x100

// resync relit l'état du matériel et repose les touches déjà tenues.
//
// C'est le pendant de l'oubli au départ du focus, et il manquait : un doigt posé avant
// que la fenêtre devienne active a envoyé son appui à la fenêtre précédente, donc
// celle-ci ne verra que le relâchement et la touche restera morte jusqu'au doigt levé.
// Relevé le 25 octobre 2026 à la vérification manuelle, où des touches « ne
// répondaient pas » selon le moment où on les enfonçait.
//
// La relecture passe par les codes virtuels, seuls que le système sache interroger. La
// traduction d'une place vers un code virtuel suit la disposition du clavier, ce qui
// serait faux pour **nommer** une place et qui est exactement juste pour
// l'**interroger** : on demande le code virtuel que cette place porte sur ce clavier.
//
// **Un code virtuel ne suffit pas toujours à désigner une place.** Onze codes de
// balayage en portent deux — le pavé numérique partage les siens avec le bloc
// directionnel, et l'Entrée du pavé avec l'Entrée principale —, et la traduction rend
// alors le même code virtuel pour les deux. Ce que le matériel répond ne dit donc pas
// laquelle est tenue, et en poser une au hasard est pire que n'en poser aucune : le
// relâchement portera le code de l'autre, et celle qu'on a posée restera tenue pour de
// bon. Le verrou numérique tranche pour les chiffres, qui prennent leurs propres codes
// virtuels quand il est allumé ; ailleurs, la paire est laissée tranquille.
//
// Deux cent cinquante-six places sont parcourues, ce qui fait cinq cents appels. Cela
// n'arrive qu'à la prise de focus, jamais dans une image.
func (w *window) resync() {
	w.resyncWith(heldByHardware, numLockOn())
}

// resyncWith est la boucle seule, la lecture du matériel et l'état du verrou numérique
// lui étant donnés.
//
// Séparer les deux est ce qui rend la traduction vérifiable : un cas de test fournit
// une lecture qui répond « tenue » pour un code virtuel choisi, et contrôle que la
// place posée est la bonne. Sans cette couture, il faudrait quelqu'un pour tenir une
// touche pendant que la suite tourne — et quelqu'un pour basculer le verrou.
func (w *window) resyncWith(held func(vk uintptr) bool, numLockOn bool) {
	w.state.Forget()

	numpad := numpadPlaces(numLockOn)

	// Les deux places d'un code se traitent ensemble, parce que c'est de la paire
	// que dépend la décision : onze codes en ont deux, et le matériel ne les
	// distingue pas toujours.
	for scan := uintptr(1); scan < 0x80; scan++ {
		plain, _, _ := mapVirtualKeyW.Call(scan, mapvkVscToVkEx)
		extended, _, _ := mapVirtualKeyW.Call(scan|extendedPrefix, mapvkVscToVkEx)

		if twinned := plain != 0 && plain == extended; twinned {
			pad, separable := numpad[scan]
			if !separable {
				continue
			}
			plain = pad
		}

		if plain != 0 && held(plain) {
			w.state.Press(input.Key(scan))
		}
		if extended != 0 && held(extended) {
			w.state.Press(input.Key(scan) | extendedKey)
		}
	}
}

// numpadPlaces rend la place de chaque chiffre du pavé numérique, indexée par code de
// balayage, et rien du tout quand le verrou numérique est éteint.
//
// C'est ce qui sépare une paire de places jumelles. Verrou allumé, le pavé produit ses
// propres codes virtuels et la distinction est exacte ; verrou éteint, il produit ceux
// du bloc directionnel et rien ne le distingue de lui.
//
// La correspondance est demandée au système plutôt qu'écrite ici : seuls les codes
// virtuels du pavé sont posés, et c'est lui qui dit quelle place chacun occupe.
func numpadPlaces(numLockOn bool) map[uintptr]uintptr {
	if !numLockOn {
		return nil
	}

	places := make(map[uintptr]uintptr, len(numpadKeys))
	for _, vk := range numpadKeys {
		if scan, _, _ := mapVirtualKeyW.Call(vk, mapvkVkToVsc); scan != 0 {
			places[scan] = vk
		}
	}
	return places
}

// numpadKeys porte les codes virtuels que le pavé numérique produit quand le verrou
// est allumé : les dix chiffres et le séparateur décimal.
//
// L'Entrée du pavé n'y est pas, et c'est la raison pour laquelle sa paire reste
// indistinguable : elle produit le code de l'Entrée principale dans les deux états du
// verrou. Les opérateurs non plus, n'ayant pas de jumelle à départager.
var numpadKeys = [11]uintptr{
	0x60, 0x61, 0x62, 0x63, 0x64, // pavé 0 à 4
	0x65, 0x66, 0x67, 0x68, 0x69, // pavé 5 à 9
	0x6E, // séparateur décimal
}

// heldByHardware dit si le matériel rapporte la touche enfoncée.
//
// Seul le bit de poids fort compte : le bit zéro dit « a été enfoncée depuis le dernier
// appel » et se consomme à la lecture, donc deux appels de suite ne rendent pas la
// même valeur.
func heldByHardware(vk uintptr) bool {
	state, _, _ := getAsyncKeyState.Call(vk)
	return uint16(state)&0x8000 != 0
}

// extendedPrefix est ce que le clavier envoie devant le code d'une place étendue, et
// que la traduction attend pour la distinguer de sa jumelle.
const extendedPrefix = 0xE000

// keyMessage traduit un message de touche en appui ou en relâchement, et dit s'il
// s'agissait bien d'un message de touche.
//
// Les quatre messages comptent, et pas seulement les deux évidents : le système envoie
// les variantes « système » quand la touche Alt est tenue, ou pour F10 toute seule.
// Ne traiter que les deux premières laisserait un déplacement coincé dès qu'un joueur
// garde Alt enfoncé.
func keyMessage(msg uint32, lParam uintptr, state *input.State) bool {
	switch msg {
	case wmKeyDown, wmSysKeyDown:
		// La répétition automatique renvoie des appuis tant que le doigt reste
		// posé ; l'état les absorbe, donc il n'y a rien à filtrer ici.
		state.Press(keyOf(lParam))
		return true

	case wmKeyUp, wmSysKeyUp:
		state.Release(keyOf(lParam))
		return true
	}
	return false
}

// numLockOn dit si le verrou numérique est allumé.
func numLockOn() bool {
	state, _, _ := getKeyState.Call(vkNumLock)
	return state&1 != 0
}
