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
// Deux cent cinquante-six places sont parcourues, ce qui fait cinq cents appels. Cela
// n'arrive qu'à la prise de focus, jamais dans une image.
func (w *window) resync() {
	w.resyncWith(heldByHardware)
}

// resyncWith est la boucle seule, la lecture du matériel lui étant donnée.
//
// Séparer les deux est ce qui rend la traduction vérifiable : un cas de test fournit
// une lecture qui répond « tenue » pour un code virtuel choisi, et contrôle que la
// place posée est la bonne. Sans cette couture, il faudrait quelqu'un pour tenir une
// touche pendant que la suite tourne.
func (w *window) resyncWith(held func(vk uintptr) bool) {
	w.state.Forget()

	for scan := uintptr(1); scan < 0x80; scan++ {
		for _, extendedPlace := range [2]bool{false, true} {
			code := scan
			key := input.Key(scan)
			if extendedPlace {
				code |= extendedPrefix
				key |= extendedKey
			}

			vk, _, _ := mapVirtualKeyW.Call(code, mapvkVscToVkEx)
			if vk == 0 {
				continue
			}
			if held(vk) {
				w.state.Press(key)
			}
		}
	}
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
