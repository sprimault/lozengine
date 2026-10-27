// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La fenêtre Windows : ce qu'elle retient des messages du système, et les cinq
// réponses qu'elle doit à la boucle.
//
// Tout s'emploie depuis le thread qui a créé la fenêtre, ce que Windows impose pour la
// file de messages. C'est aussi ce que la boucle fait, puisqu'un hôte ne cède pas son
// thread principal : les champs n'ont donc pas de verrou, et en mettre un laisserait
// croire qu'un autre thread est permis.

package backend

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// windows relie une poignée de fenêtre à son état, pour que la procédure de fenêtre
// retrouve l'instance que le système ne lui passe pas.
//
// Une table et non `GWLP_USERDATA`, qui serait la voie canonique en C : un pointeur Go
// ne peut pas être gardé dans de la mémoire que Go ne gère pas, et le système
// conserverait celui-ci aussi longtemps que la fenêtre vit. Le verrou protège la table
// seule, la procédure pouvant être appelée pendant `CreateWindowExW`, c'est-à-dire
// avant que l'appel ait rendu la poignée.
var windows = struct {
	sync.Mutex
	byHandle map[uintptr]*window
}{
	byHandle: make(map[uintptr]*window),
}

// window est une fenêtre ouverte et ce que le système en a dit depuis la dernière
// image.
type window struct {
	hwnd uintptr
	dc   uintptr

	// state accumule ce que le clavier a fait. Le décodage des messages de touches
	// vient avec le lot qui lit le clavier ; ce qui est déjà câblé est l'oubli au
	// départ du focus, parce que sans lui une touche tenue le reste pour toujours.
	state input.State

	// closed retient que la fenêtre ne présentera plus rien. Il ne se relève jamais.
	closed bool

	// info décrit au système la disposition des pixels. Elle est montée une fois :
	// seules ses dimensions changent d'une présentation à l'autre, et la construire
	// par image allouerait dans le chemin de rendu.
	info bitmapInfoFields

	// args porte les douze arguments de la présentation, pour la même raison : une
	// liste variadique en alloue une par appel. Les cases constantes sont posées au
	// montage, et la présentation ne touche que celles qui changent.
	args []uintptr

	// minimized suit `wmSize`, et lastW et lastH gardent la dernière taille utile
	// connue.
	minimized bool
	lastW     uint16
	lastH     uint16

	// minW et minH sont le plancher de redimensionnement, en taille de fenêtre
	// entière et non en zone utile : c'est ce que `wmGetMinMaxInfo` attend. En
	// dessous, l'agrandissement entier ne tiendrait plus l'image et elle se
	// découperait sans que rien ne le dise.
	//
	// La mesure est prise à l'ouverture, bordures du thème comprises. Un écran à
	// autre densité en changerait l'épaisseur, et le plancher serait alors faux de
	// quelques pixels : le backend ne suit pas `WM_DPICHANGED`, et le découpage que
	// ces pixels donneraient coûte moins que de poursuivre une taille qui bouge.
	minW uint16
	minH uint16

	// icons porte les poignées d'icône posées sur la fenêtre, à rendre au système
	// avec elle. Le système ne les libère pas de lui-même : elles survivraient à la
	// fenêtre, et un jeu qui rouvre la sienne en fuirait deux par ouverture.
	icons []uintptr
}

// windowProc est la procédure de fenêtre, appelée par le système pour chaque message.
//
// Elle ne traite que ce dont la boucle a besoin et passe tout le reste à
// `DefWindowProcW`. Un cycle de vie ordinaire apporte une trentaine de messages
// distincts, dont certains que le système enregistre dynamiquement : chercher à les
// connaître tous serait sans fin et sans objet.
func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	w := byHandle(hwnd)
	if w == nil {
		ret, _, _ := defWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	// Les touches d'abord : ce sont les messages les plus nombreux d'une partie, et
	// le seul groupe dont le traitement se délègue entièrement.
	if keyMessage(msg, lParam, &w.state) {
		return 0
	}

	switch msg {
	case wmGetMinMaxInfo:
		// Le système passe l'adresse de la structure dans le `lParam`, et c'est la
		// convention de la procédure de fenêtre : il n'y a pas d'autre façon de
		// l'atteindre. La valeur est lue dans le message et jamais gardée, donc le
		// pointeur ne survit pas à l'appel.
		//
		// Le plancher nul est le régime de la création : ce message ouvre la
		// séquence, avant que la poignée soit enregistrée ici, et `byHandle` rend
		// alors nil — le cas ne passe donc pas par ici. Un plancher nul resterait
		// sans effet de toute façon, le système n'imposant rien en dessous de ses
		// propres minimums.
		if w.minW == 0 || w.minH == 0 {
			break
		}
		info := (*minMaxInfo)(unsafe.Pointer(lParam))
		info.minTrack = point{x: int32(w.minW), y: int32(w.minH)}
		return 0

	case wmClose:
		// La fermeture demandée par le joueur n'est pas la destruction : on note, et
		// c'est la boucle qui décide quand s'arrêter.
		w.closed = true
		return 0

	case wmDestroy:
		w.closed = true
		return 0

	case wmSize:
		w.minimized = wParam == sizeMinimized
		if !w.minimized {
			if width, height := clientSize(hwnd); width != 0 && height != 0 {
				w.lastW, w.lastH = width, height
			}
		}
		return 0

	case wmSetFocus:
		// Ce que le clavier tient déjà n'est pas arrivé ici : l'appui est allé à la
		// fenêtre qui avait le focus avant. On relit le matériel plutôt que
		// d'attendre que le joueur lève le doigt.
		w.resync()
		return 0

	case wmKillFocus:
		// Le relâchement d'une touche tenue est perdu, pas différé : la fenêtre ne
		// reçoit plus rien après ce message, et relire l'état du système ne rattrape
		// rien puisqu'il dit encore « enfoncée ». Oublier est la seule parade, sans
		// quoi le personnage marcherait tout seul au retour.
		w.state.Forget()
		return 0
	}

	ret, _, _ := defWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// byHandle retrouve la fenêtre d'une poignée, ou nil si le système parle d'une
// fenêtre que nous ne tenons pas encore.
func byHandle(hwnd uintptr) *window {
	windows.Lock()
	defer windows.Unlock()
	return windows.byHandle[hwnd]
}

// Input vide la file des messages puis rend ce que le clavier dit au pas qui commence.
//
// C'est ici que la pompe tourne, parce que c'est le premier appel d'un pas : la boucle
// lit les entrées, met à jour, puis présente. Pomper ailleurs ferait répondre la
// fenêtre en retard d'une image.
func (w *window) Input() input.Input {
	var m message
	for {
		ret, _, _ := peekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
		if ret == 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return w.state.Frame()
}

// Present pose le tampon à l'écran, tel quel.
//
// Les pixels partent sans être recopiés ni convertis : les masques annoncés au système
// décrivent la disposition du moteur, donc il lit le tampon où il est. C'est ce qui
// épargne un parcours complet de l'image à chaque présentation, et c'est propre à
// cette plateforme — X11 impose l'ordre de son visuel, et le navigateur veut du non
// prémultiplié. La conversion, là où il en faudra une, appartiendra à ces backends.
//
// Sans erreur, et le contrat le veut ainsi pour une raison mesurée : sur une fenêtre
// réduite, l'appel rend le nombre de lignes demandé alors que rien n'est peint. Il n'y
// a donc pas d'échec à rapporter, seulement une présentation qui n'a pas eu lieu — et
// personne à qui s'en plaindre soixante fois par seconde.
func (w *window) Present(t *render.Buffer) {
	if w.dc == 0 || t == nil || len(t.Pixels) == 0 {
		return
	}

	w.info.header.width = int32(t.Width)
	w.info.header.height = -int32(t.Height)

	// Les arguments passent par une tranche gardée, et c'est le seul appel du backend
	// à le faire : douze arguments variadiques en allouent une à chaque présentation,
	// donc soixante fois par seconde. Ailleurs, `LazyProc.Call` reste préférable —
	// six nanosecondes de plus et rien sur le tas.
	w.args[0] = w.dc
	w.args[3] = uintptr(t.Width)
	w.args[4] = uintptr(t.Height)
	w.args[8] = uintptr(t.Height)
	w.args[9] = uintptr(unsafe.Pointer(&t.Pixels[0]))
	w.args[10] = uintptr(unsafe.Pointer(&w.info))

	syscall.SyscallN(setDIBitsToDevice.Addr(), w.args...)
}

// Size rend les dimensions de la zone utile, en pixels.
//
// Jamais zéro tant qu'une taille a été connue : une fenêtre réduite a une zone utile
// nulle, et la boucle en tirerait un facteur d'agrandissement par division par zéro.
// Le backend sait qu'elle est réduite, la boucle n'a pas à l'apprendre — elle
// présentera dans le vide, ce qui ne coûte rien et ne casse rien.
func (w *window) Size() (width, height uint16) {
	if w.minimized {
		return w.lastW, w.lastH
	}
	width, height = clientSize(w.hwnd)
	if width == 0 || height == 0 {
		return w.lastW, w.lastH
	}
	return width, height
}

// Closed dit que la fenêtre ne présentera plus rien.
func (w *window) Closed() bool {
	return w.closed
}

// Err rend toujours nil : sous Windows, une fenêtre ne meurt que parce qu'on la ferme.
//
// Rien de ce que le backend appelle après l'ouverture ne rapporte un échec dont la
// boucle pourrait faire quelque chose. La présentation sur une fenêtre réduite n'en est
// pas un, et la file de messages ne se coupe pas. Ce qui pouvait échouer l'a fait dans
// Open, qui rend son erreur.
//
// Le membre du contrat existe pour les familles qui parlent à un serveur par une
// socket, où le lien se perd sans que personne n'ait rien fermé.
func (w *window) Err() error {
	return nil
}

// Close rend au système ce qu'Open a pris. Appeler deux fois est sans effet.
func (w *window) Close() {
	// L'état se pose d'abord : une fenêtre fermée ne présentera plus rien, et c'est
	// vrai au second appel comme au premier. Court-circuiter avant laisserait
	// `Closed` mentir, ce qu'un cas de test a montré sur ce code même.
	w.closed = true

	if w.hwnd == 0 {
		return
	}

	if w.dc != 0 {
		releaseDC.Call(w.hwnd, w.dc)
		w.dc = 0
	}
	w.releaseIcons()
	destroyWindow.Call(w.hwnd)

	windows.Lock()
	delete(windows.byHandle, w.hwnd)
	windows.Unlock()

	w.hwnd = 0
}
