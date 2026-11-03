// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le clavier et le pointeur d'un compositeur : ce qu'un siège accorde, ce que ses deux
// objets rapportent, et les crans qu'une molette n'envoie pas toujours sous forme de
// crans.
//
// **La différence de fond avec X11 n'est pas le protocole, c'est que rien n'est acquis.**
// Là-bas une fenêtre reçoit le clavier parce qu'elle a demandé les événements dans son
// masque ; ici un siège annonce ce qu'il porte, le client crée un objet par capacité
// annoncée, et l'annonce revient quand un périphérique se branche ou se débranche.

package backend

import (
	"fmt"
	"math"
	"syscall"
)

// wlSeat porte ce que le siège a accordé et ce que la molette a accumulé.
//
// Les deux identifiants sont gardés séparément de celui du siège parce que les trois
// objets reçoivent des événements et qu'un opcode seul ne dit pas de qui il vient : le
// `leave` du clavier et celui du pointeur valent tous deux deux, et le premier oublie
// l'état des touches là que le second sort le pointeur de la fenêtre.
type wlSeat struct {
	id      uint32
	version uint32

	// keyboard et pointer sont les objets accordés, et zéro dit « pas de capacité ».
	// C'est contre eux que se juge une annonce de capacités, et non contre le masque
	// précédent : ce qui compte est ce qui existe, pas ce qui a été annoncé.
	keyboard uint32
	pointer  uint32

	// enterSerial est le numéro du dernier `enter` du pointeur.
	//
	// Il date la requête de curseur, qui est la seule du backend à porter un numéro
	// de série en dehors de l'acquittement d'une configuration. Une requête qui en
	// porterait un périmé est ignorée en silence, donc le garder est la condition
	// pour que le curseur disparaisse.
	enterSerial uint32

	axis wlAxis
}

// Ce qu'un cran de molette vaut dans les deux unités où le protocole exprime un
// défilement.
//
// **La première est normative, la seconde est une convention de fait.**
// `axis_value120` compte « as fraction of 120 », donc cent vingt est le cran lui-même.
// L'événement `axis`, lui, donne « length of vector in surface-local coordinate space »
// et ne dit nulle part ce qu'un cran y vaut : dix unités est ce que les pilotes emploient,
// comme les cent pixels par cran d'une page, et la valeur attend une vérification sur une
// molette réelle.
const (
	wlValue120PerNotch int32 = 120
	wlFixedPerNotch    int32 = 10 << wlFixedShift
)

// wlAxis accumule ce qu'une trame de défilement rapporte, et n'en rend que des crans
// entiers.
//
// **Deux sources disent la même chose et il ne faut en compter qu'une.** Un compositeur
// qui envoie `axis_value120` envoie aussi `axis` pour le même geste — sa description le
// dit, « in addition to the wl_pointer.axis event » —, donc additionner les deux
// doublerait chaque cran de molette. Mais `axis` arrive **seul** quand le geste n'a pas de
// cran, ce qui est le régime d'un pavé tactile : l'écarter d'office rendrait le
// défilement au doigt muet. C'est donc la trame qui tranche, et c'est à cela que sert
// `wl_pointer.frame`.
//
// **Le reste survit au geste, et c'est voulu** : deux gestes courts dans le même sens
// font un cran, là que le jeter en ferait deux fois rien. C'est le même choix que le
// backend web, pour la même raison — le système crante avant de rapporter sur une
// molette, jamais sur un doigt.
type wlAxis struct {
	// pending est la distance que la trame en cours a rapportée, en virgule fixe, et
	// stepped dit que la même trame porte des crans — auquel cas cette distance en
	// est le doublon et non la mesure.
	pending int32
	stepped bool

	// notches et residue sont ce qui n'a pas encore fait un cran entier, chacun dans
	// l'unité de sa source. Deux restes et non un : un geste de molette et un geste de
	// doigt n'ont pas la même granularité, et mélanger leurs restes ferait dépendre le
	// premier cran de l'un de ce que l'autre avait laissé.
	notches int32
	residue int32
}

// notch ajoute ce qu'une source crantée rapporte, en cent-vingtièmes de cran.
func (a *wlAxis) notch(value120 int32) {
	a.notches += value120
	a.stepped = true
}

// distance ajoute ce qu'une source sans cran rapporte, en virgule fixe.
func (a *wlAxis) distance(value int32) {
	a.pending += value
}

// frame clôt la trame et rend les crans entiers qu'elle a faits, positifs vers le haut.
//
// Le signe s'inverse : le protocole compte vers le bas de l'axe vertical, l'état d'entrée
// vers le haut.
func (a *wlAxis) frame() int {
	distance, stepped := a.pending, a.stepped
	a.pending, a.stepped = 0, false

	if stepped {
		steps := a.notches / wlValue120PerNotch
		a.notches -= steps * wlValue120PerNotch
		return -int(steps)
	}

	a.residue += distance
	steps := a.residue / wlFixedPerNotch
	a.residue -= steps * wlFixedPerNotch
	return -int(steps)
}

// wlSurfacePixel ramène une coordonnée de surface dans le type que l'état du pointeur
// porte.
//
// La borne n'est pas de la défensive : un pointeur capturé est rapporté hors de la
// fenêtre, et le protocole porte ses coordonnées sur vingt-quatre bits d'entier là que
// l'état d'entrée en garde seize. Sans elle, une position lointaine se replierait de
// signe et désignerait le coin opposé de l'image.
func wlSurfacePixel(v int32) int16 {
	return int16(min(max(v, math.MinInt16), math.MaxInt16))
}

// wlSeatEvent traite les événements du siège lui-même.
//
// `name` arrive ici aussi et ne sert à rien : c'est un libellé pour une interface
// d'utilisateur, pas une identité à retenir.
func (w *wlWindow) wlSeatEvent(opcode uint16, r *wlReader) error {
	if opcode != wlSeatCapabilities {
		return nil
	}
	caps := r.uint()
	if r.err != nil {
		return r.err
	}
	return w.wlApplyCaps(caps)
}

// wlApplyCaps crée ce que le siège annonce et rend ce qu'il retire.
//
// **Créer un objet pour une capacité non annoncée est une erreur de protocole**, et sa
// description le dit de `get_pointer` comme de `get_keyboard` : « it is a protocol
// violation to issue this request on a seat that has never had the … capability ». Le
// masque décide donc, et il revient à chaque branchement.
//
// Le retrait relâche l'état du périphérique parti. Sans cela, une touche tenue au moment
// où un clavier se débranche le resterait indéfiniment — c'est le défaut que la perte du
// focus pose déjà, et un débranchement le pose de la même façon.
func (w *wlWindow) wlApplyCaps(caps uint32) error {
	switch {
	case caps&wlSeatKeyboard != 0 && w.seat.keyboard == 0:
		id, err := w.c.newID()
		if err != nil {
			return err
		}
		w.seat.keyboard = id
		w.c.out.begin(w.seat.id, wlSeatGetKeyboard)
		w.c.out.uint(id)
		w.c.out.end()

	case caps&wlSeatKeyboard == 0 && w.seat.keyboard != 0:
		w.wlRelease(w.seat.keyboard, wlKeyboardRelease)
		w.seat.keyboard = 0
		w.state.Forget()
	}

	switch {
	case caps&wlSeatPointer != 0 && w.seat.pointer == 0:
		id, err := w.c.newID()
		if err != nil {
			return err
		}
		w.seat.pointer = id
		w.c.out.begin(w.seat.id, wlSeatGetPointer)
		w.c.out.uint(id)
		w.c.out.end()

	case caps&wlSeatPointer == 0 && w.seat.pointer != 0:
		w.wlRelease(w.seat.pointer, wlPointerRelease)
		w.seat.pointer = 0
		w.state.Leave()
	}
	return nil
}

// wlRelease rend un objet d'entrée au compositeur, quand la version liée le permet.
//
// La requête arrive en version 3 et l'émettre plus bas serait un opcode que l'interface
// n'a pas, donc une erreur de protocole et une connexion coupée. Ce que l'omission coûte
// est une entrée dans la table du compositeur et un identifiant que `delete_id` ne rendra
// pas — une fois par périphérique débranché, et sur un compositeur d'avant 2015.
func (w *wlWindow) wlRelease(id uint32, opcode uint16) {
	if w.seat.version < wlInputReleaseSince {
		return
	}
	w.c.out.begin(id, opcode)
	w.c.out.end()
}

// wlKeyboardEvent traite ce que le clavier rapporte.
//
// Les codes de `key` sont des codes evdev, que [evdevKeyOf] traduit sans décalage : c'est
// le protocole X11 qui ajoute huit aux siens, pas le noyau.
func (w *wlWindow) wlKeyboardEvent(opcode uint16, r *wlReader) error {
	switch opcode {
	case wlKeyboardKeymap:
		return w.wlDropKeymap(r)

	case wlKeyboardEnter:
		// Le numéro de série et la surface ne servent pas : aucune requête n'est
		// datée par une prise de focus, et la fenêtre n'a qu'une surface.
		r.uint()
		r.uint()
		keys := r.array()
		if r.err != nil {
			return r.err
		}
		w.wlRelearnKeys(keys)

	case wlKeyboardLeave:
		// Le relâchement d'une touche tenue est perdu, pas différé : le clavier ne
		// rapporte plus rien à cette surface. Oublier est la seule parade, sans
		// quoi le personnage marcherait tout seul au retour.
		w.state.Forget()

	case wlKeyboardKey:
		r.uint()
		r.uint()
		code, state := r.uint(), r.uint()
		if r.err != nil {
			return r.err
		}
		if k, ok := evdevKeyOf(code); ok {
			switch state {
			case wlPressed:
				w.state.Press(k)
			case wlReleased:
				w.state.Release(k)
			}
		}
	}
	return nil
}

// wlDropKeymap ferme la carte de clavier sans la lire.
//
// **Le descripteur doit être retiré de la file même si personne ne s'en sert**, et c'est
// le piège de cet événement : rien ne lie un descripteur au message qui l'annonce, donc
// en laisser un décale tous les suivants — le prochain tampon de mémoire partagée
// recevrait celui du clavier. Le fermer est la seconde moitié du geste, sans quoi un
// client fuirait un descripteur par prise de focus.
//
// Ce qu'on ne lit pas est la carte elle-même, et c'est la décision du jalon : une place de
// clavier n'est pas un caractère, donc le moteur n'a pas de table xkb à interpréter. Les
// deux valeurs lues ne servent qu'à nommer ce qu'on a reçu si la file est vide.
//
// L'échec de la fermeture est ignoré : un descripteur que le noyau vient d'installer se
// ferme, et il n'y a rien à faire d'un refus qui n'arrivera pas.
func (w *wlWindow) wlDropKeymap(r *wlReader) error {
	format, size := r.uint(), r.uint()
	if r.err != nil {
		return r.err
	}

	fd, ok := w.c.takeFD()
	if !ok {
		return fmt.Errorf("wl_keyboard.keymap announced %d bytes of format %d, and no file descriptor came with it",
			size, format)
	}
	syscall.Close(fd)
	return nil
}

// wlRelearnKeys reprend l'état du clavier depuis les codes que la prise de focus porte.
//
// **C'est l'équivalent du vecteur de bits d'X11, sous une autre forme** : là-bas le
// serveur envoie un bit par place possible, ici le compositeur envoie la liste des codes
// tenus. Les deux remplacent la relecture touche par touche du backend Windows.
//
// L'oubli préalable est voulu : ce que la liste ne nomme pas n'est pas enfoncé, et
// repartir de l'état réel est précisément l'objet de cet événement. Les touches tenues se
// rallument donc comme des appuis neufs, ce qui est aussi le choix des deux autres
// familles.
//
// Un tableau dont la longueur n'est pas un multiple de quatre voit son reste écarté : le
// protocole y met des mots de trente-deux bits, donc une queue plus courte qu'un mot est
// un message que le compositeur a mal composé, et en lire trois octets inventerait un
// code.
func (w *wlWindow) wlRelearnKeys(keys []byte) {
	w.state.Forget()

	for at := 0; at+4 <= len(keys); at += 4 {
		if k, ok := evdevKeyOf(wlOrder.Uint32(keys[at:])); ok {
			w.state.Press(k)
		}
	}
}

// wlPointerEvent traite ce que le pointeur rapporte.
//
// Les boutons sont des codes evdev et non les numéros de un à neuf d'X11 : `BTN_LEFT` vaut
// `0x110`, donc [evdevButtonOf] est une seconde traduction et non celle d'X11.
func (w *wlWindow) wlPointerEvent(opcode uint16, r *wlReader) error {
	switch opcode {
	case wlPointerEnter:
		serial := r.uint()
		// La surface ne sert pas : la fenêtre n'en a qu'une, donc le compositeur
		// ne peut nommer que celle-là.
		r.uint()
		x, y := r.fixed(), r.fixed()
		if r.err != nil {
			return r.err
		}
		w.seat.enterSerial = serial
		w.state.Move(wlSurfacePixel(x), wlSurfacePixel(y))
		w.wlSetCursor()

	case wlPointerLeave:
		// La position reste celle du dernier mouvement : elle dit par où le
		// pointeur est sorti, ce qu'un jeu qui suit le curseur veut savoir.
		w.state.Leave()

	case wlPointerMotion:
		// L'horodatage ne sert pas : la boucle n'a pas de geste à suivre dans le
		// temps, et un pas voit la dernière position que la trame a portée.
		r.uint()
		x, y := r.fixed(), r.fixed()
		if r.err != nil {
			return r.err
		}
		w.state.Move(wlSurfacePixel(x), wlSurfacePixel(y))

	case wlPointerButton:
		// Le numéro de série et l'horodatage ne servent pas : aucune requête du
		// backend n'est datée par un appui.
		r.uint()
		r.uint()
		code, state := r.uint(), r.uint()
		if r.err != nil {
			return r.err
		}
		if b, ok := evdevButtonOf(code); ok {
			switch state {
			case wlPressed:
				w.state.PressButton(b)
			case wlReleased:
				w.state.ReleaseButton(b)
			}
		}

	case wlPointerAxis:
		r.uint()
		axis, value := r.uint(), r.int()
		if r.err != nil {
			return r.err
		}
		if axis == wlAxisVertical {
			w.seat.axis.distance(value)
		}
		// **Sous la cinquième version, aucun `frame` ne vient clore la trame**, et
		// attendre celui-ci rendrait la molette muette plutôt qu'imprécise. La
		// distance s'applique donc à l'arrivée, ce qui est exact : sans `frame`, un
		// compositeur n'a pas non plus de source cranté à annoncer.
		if w.seat.version < wlPointerFrameSince {
			w.state.Wheel(w.seat.axis.frame())
		}

	case wlPointerAxisDiscrete:
		axis, discrete := r.uint(), r.int()
		if r.err != nil {
			return r.err
		}
		if axis == wlAxisVertical {
			w.seat.axis.notch(discrete * wlValue120PerNotch)
		}

	case wlPointerAxisValue120:
		axis, value120 := r.uint(), r.int()
		if r.err != nil {
			return r.err
		}
		if axis == wlAxisVertical {
			w.seat.axis.notch(value120)
		}

	case wlPointerFrame:
		w.state.Wheel(w.seat.axis.frame())
	}
	return nil
}

// wlSetCursor retire le curseur du système quand le jeu dessine le sien.
//
// **C'est une requête datée et non un attribut de fenêtre, et c'est ce qui la distingue
// d'X11** : elle vaut pour la prise de focus que son numéro de série nomme, donc elle se
// renvoie à chaque `enter`. L'oublier fait reparaître le curseur **sans aucune erreur de
// protocole**, le compositeur ignorant en silence une requête dont le numéro est périmé.
//
// **Ne rien envoyer est ce qui s'applique quand le jeu garde le curseur du système**, et
// le protocole laisse alors l'image indéfinie : « the pointer image is undefined when
// focus enters a surface ». En poser une demanderait de lire un thème de curseur sur le
// disque, ce qui est un format de fichier entier pour un pixel qu'un jeu isométrique
// couvre de son propre pointeur ; les compositeurs posent le curseur de leur thème dans ce
// cas, et c'est ce qu'un joueur voit.
func (w *wlWindow) wlSetCursor() {
	if !w.hideCursor || w.seat.pointer == 0 {
		return
	}

	w.c.out.begin(w.seat.pointer, wlPointerSetCursor)
	w.c.out.uint(w.seat.enterSerial)
	// La surface nulle est ce qui cache l'image : « if surface is NULL, the pointer
	// image is hidden ». Les deux points chauds qui suivent sont signés sur le fil et
	// s'écrivent donc comme des entiers ordinaires, zéro ayant la même forme.
	w.c.out.uint(0)
	w.c.out.uint(0)
	w.c.out.uint(0)
	w.c.out.end()
}
