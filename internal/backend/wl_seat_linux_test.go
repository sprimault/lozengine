// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la lecture des entrées doit tenir : n'ouvrir que les capacités annoncées, rendre
// les codes du noyau tels quels, fermer la carte de clavier qu'on ne lit pas, et ne
// compter qu'une fois le cran qu'un compositeur annonce deux fois.

package backend

import (
	"os"
	"strings"
	"testing"

	"github.com/sprimault/lozengine/input"
)

// Les codes evdev dont les cas de ce fichier se servent, nommés pour que la lecture ne
// demande pas les en-têtes du noyau.
const (
	evdevKeyA    uint32 = 30
	evdevKeyB    uint32 = 48
	evdevKeyLeft uint32 = 105
)

// wlSigned met une valeur signée dans le mot que le protocole transporte.
//
// Elle existe parce qu'une conversion de constante négative vers `uint32` est refusée à la
// compilation : les distances et les crans sont signés dans la description, et [wlEvent]
// compose des mots. Passer par un argument rend la conversion permise, et c'est exactement
// ce que [wlReader.int] fait en sens inverse.
func wlSigned(v int32) uint32 {
	return uint32(v)
}

// wlSeatMount monte une fenêtre sur un compositeur de pacotille et stabilise l'ouverture.
//
// L'aller-retour est fait ici parce que tous les cas en ont besoin : le dernier envoi du
// montage part après la boucle d'attente, donc ce que le compositeur a vu ne se lit pas
// avant qu'une barrière le garantisse.
func wlSeatMount(t *testing.T, f *wlFake, o Options) *wlWindow {
	t.Helper()

	w, err := wlMount(t, f, o)
	if err != nil {
		t.Fatalf("montage de la fenêtre : %v", err)
	}
	wlSettle(t, w.c)
	return w
}

// wlSeatStep pousse des événements vers le client et rend l'image d'entrées qui en sort.
//
// Les deux gestes vont ensemble dans tous les cas : une entrée n'existe pour le jeu qu'une
// fois que le pas l'a relevée, et c'est [wlWindow.Input] qui vide le flux.
func wlSeatStep(t *testing.T, w *wlWindow, f *wlFake, messages ...[]byte) input.Input {
	t.Helper()

	f.emit(t, messages...)
	in := w.Input()
	if err := w.Err(); err != nil {
		t.Fatalf("la fenêtre est tombée pendant la lecture : %v", err)
	}
	return in
}

// TestWLSeatOpensOnlyWhatTheCompositorAnnounces garde le masque de capacités.
//
// **C'est le cas qui porte le lot, et il rougit sur les deux défauts qui comptent.** Si le
// registre n'écoutait pas `wl_seat.capabilities` à l'ouverture, le masque serait nul et le
// premier régime ne créerait rien : un siège stable n'envoie cette annonce qu'une fois,
// juste après la liaison, donc l'entendre tard revient à ne jamais l'entendre. Et si le
// backend créait ses objets sans regarder le masque, le dernier régime en ouvrirait deux
// là que la description appelle cela une violation du protocole — « it is a protocol
// violation to issue this request on a seat that has never had the … capability ».
func TestWLSeatOpensOnlyWhatTheCompositorAnnounces(t *testing.T) {
	cases := []struct {
		name     string
		caps     uint32
		wants    []string
		notWants []string
	}{
		{
			name:  "un siège ordinaire, clavier et pointeur",
			caps:  wlSeatKeyboard | wlSeatPointer,
			wants: []string{"get_keyboard", "get_pointer"},
		},
		{
			name:     "un clavier sans souris, ce qu'une machine sans pointeur annonce",
			caps:     wlSeatKeyboard,
			wants:    []string{"get_keyboard"},
			notWants: []string{"get_pointer"},
		},
		{
			name:     "un tactile seul, dont le moteur ne prend rien",
			caps:     wlSeatTouch,
			notWants: []string{"get_keyboard", "get_pointer"},
		},
		{
			name:     "un siège sans aucune capacité",
			caps:     0,
			notWants: []string{"get_keyboard", "get_pointer"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
			f.caps = c.caps
			w := wlSeatMount(t, f, wlTestOptions())

			seen := strings.Join(f.saw(), "\n")
			for _, want := range c.wants {
				if !strings.Contains(seen, want) {
					t.Errorf("le compositeur n'a pas vu %q, et le masque %#x l'annonçait", want, c.caps)
				}
			}
			for _, not := range c.notWants {
				if strings.Contains(seen, not) {
					t.Errorf("le compositeur a vu %q, que le masque %#x n'annonçait pas", not, c.caps)
				}
			}

			// L'identifiant côté client doit suivre le même masque : router sur un
			// objet qu'on n'a pas demandé enverrait les événements d'un autre
			// client au mauvais traitement.
			if has := c.caps&wlSeatKeyboard != 0; has != (w.seat.keyboard != 0) {
				t.Errorf("le clavier du client vaut %d pour un masque %#x", w.seat.keyboard, c.caps)
			}
			if has := c.caps&wlSeatPointer != 0; has != (w.seat.pointer != 0) {
				t.Errorf("le pointeur du client vaut %d pour un masque %#x", w.seat.pointer, c.caps)
			}
		})
	}
}

// TestWLSeatReleasesWhatACapabilityTakesAway garde le débranchement d'un périphérique.
//
// Deux choses s'y jouent et la seconde est celle qu'on oublierait : l'objet est rendu au
// compositeur, et l'état est relâché. Sans le relâchement, une touche tenue au moment où
// un clavier se débranche le resterait indéfiniment — c'est le défaut que la perte du
// focus pose déjà, et un débranchement le pose de la même façon.
func TestWLSeatReleasesWhatACapabilityTakesAway(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())

	held := wlSeatStep(t, w, f,
		wlEvent(w.seat.keyboard, wlKeyboardKey, 1, 0, evdevKeyA, wlPressed))
	if !held.Held(input.KeyA) {
		t.Fatal("la touche A devait être tenue avant le débranchement")
	}

	after := wlSeatStep(t, w, f, wlEvent(w.seat.id, wlSeatCapabilities, 0))
	if after.Held(input.KeyA) {
		t.Error("la touche A reste tenue après le retrait du clavier, et elle le resterait pour de bon")
	}
	if w.seat.keyboard != 0 || w.seat.pointer != 0 {
		t.Errorf("le client garde clavier %d et pointeur %d après un masque nul",
			w.seat.keyboard, w.seat.pointer)
	}

	wlSettle(t, w.c)
	seen := strings.Join(f.saw(), "\n")
	for _, want := range []string{"release_keyboard", "release_pointer"} {
		if !strings.Contains(seen, want) {
			t.Errorf("le compositeur n'a pas vu %q : son entrée de table reste prise", want)
		}
	}
}

// TestWLKeyboardPlaysTheKernelCodes garde la traduction et les deux régimes du focus.
//
// Les codes sont ceux du noyau et non ceux d'X11 : le code 30 est la lettre A ici, et
// Échap là-bas. Un cas qui passerait avec le décalage serait le signe que la traduction a
// été reprise de travers.
func TestWLKeyboardPlaysTheKernelCodes(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())
	kb := w.seat.keyboard

	pressed := wlSeatStep(t, w, f,
		wlEvent(kb, wlKeyboardKey, 1, 0, evdevKeyA, wlPressed),
		wlEvent(kb, wlKeyboardKey, 2, 0, evdevKeyLeft, wlPressed))
	if !pressed.Held(input.KeyA) || !pressed.JustPressed(input.KeyA) {
		t.Error("la lettre A devait être tenue et montante")
	}
	if !pressed.Held(input.KeyLeft) {
		t.Error("la flèche de gauche devait être tenue : c'est une place étendue de la table")
	}

	released := wlSeatStep(t, w, f,
		wlEvent(kb, wlKeyboardKey, 3, 0, evdevKeyA, wlReleased))
	if released.Held(input.KeyA) {
		t.Error("la lettre A reste tenue après son relâchement")
	}

	// La prise de focus porte la liste des touches déjà enfoncées, et elle remplace
	// l'état au lieu de s'y ajouter : la flèche tenue avant doit disparaître.
	entered := wlSeatStep(t, w, f,
		wlEvent(kb, wlKeyboardEnter, 4, w.surface, 4, evdevKeyB))
	if !entered.Held(input.KeyB) {
		t.Error("la touche que la prise de focus annonce tenue ne l'est pas")
	}
	if entered.Held(input.KeyLeft) {
		t.Error("la prise de focus s'est ajoutée à l'état au lieu de le remplacer")
	}

	left := wlSeatStep(t, w, f, wlEvent(kb, wlKeyboardLeave, 5, w.surface))
	if left.Held(input.KeyB) {
		t.Error("une touche tenue survit à la perte du focus, et le personnage marcherait tout seul")
	}
}

// TestWLKeyboardTakesTheKeymapDescriptorOutOfTheQueue garde le geste qui décale tout.
//
// **Rien ne lie un descripteur au message qui l'annonce**, donc en laisser un dans la file
// décale tous les suivants : le prochain à être réclamé serait celui du clavier. Le cas
// rougit si l'événement est ignoré au lieu d'être traité, ce qui est tentant puisque la
// carte elle-même ne sert à rien.
func TestWLKeyboardTakesTheKeymapDescriptorOutOfTheQueue(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())

	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("ouverture de %s : %v", os.DevNull, err)
	}
	defer null.Close()

	f.emitFD(t, int(null.Fd()),
		wlEvent(w.seat.keyboard, wlKeyboardKeymap, 1, 64))
	w.Input()
	if err := w.Err(); err != nil {
		t.Fatalf("la fenêtre est tombée sur une carte de clavier : %v", err)
	}

	if n := len(w.c.fds); n != 0 {
		t.Errorf("%d descripteur(s) restent dans la file : le prochain réclamé serait celui du clavier", n)
	}
}

// TestWLKeyboardDoesNotLeakKeymapDescriptors garde la seconde moitié du même geste.
//
// Retirer le descripteur de la file et le fermer sont deux choses, et seule la première se
// voit sur le protocole : un client qui retire sans fermer parle juste et fuit un
// descripteur par prise de focus. Soixante-quatre cartes suffisent à rendre la fuite
// flagrante, là qu'une seule se perdrait dans le bruit d'un processus de test.
//
// La marge est explicite parce que le compte d'un processus n'est pas stable au
// descripteur près : le runtime en ouvre et en ferme pour son compte.
func TestWLKeyboardDoesNotLeakKeymapDescriptors(t *testing.T) {
	const maps, margin = 64, 8

	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())

	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("ouverture de %s : %v", os.DevNull, err)
	}
	defer null.Close()

	open := func() int {
		t.Helper()
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Skipf("le compte de descripteurs n'est pas lisible ici : %v", err)
		}
		return len(entries)
	}

	before := open()
	for range maps {
		f.emitFD(t, int(null.Fd()), wlEvent(w.seat.keyboard, wlKeyboardKeymap, 1, 64))
		w.Input()
		if err := w.Err(); err != nil {
			t.Fatalf("la fenêtre est tombée sur une carte de clavier : %v", err)
		}
	}

	if after := open(); after > before+margin {
		t.Errorf("%d descripteurs ouverts après %d cartes de clavier, %d avant : la carte n'est pas fermée",
			after, maps, before)
	}
}

// TestWLPointerReportsItsPositionAndButtons garde la position et les codes de bouton.
//
// La position arrive en virgule fixe et les boutons en codes evdev : `BTN_LEFT` vaut
// `0x110`, donc un cas qui passerait avec les numéros de un à neuf d'X11 dirait que la
// traduction a été reprise de la mauvaise famille.
func TestWLPointerReportsItsPositionAndButtons(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())
	ptr := w.seat.pointer

	f.entered(7)
	in := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerEnter, 7, w.surface, 12<<wlFixedShift, 34<<wlFixedShift))
	if x, y, inside := in.Pointer(); x != 12 || y != 34 || !inside {
		t.Errorf("la prise de focus a rendu (%d, %d, %v), attendu (12, 34, true)", x, y, inside)
	}

	// Un demi-pixel ne fait pas un pixel : la troncature est ce que le déterminisme
	// du moteur demande, et la moitié haute d'un pixel appartient à ce pixel.
	moved := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerMotion, 0, 5<<wlFixedShift|1<<(wlFixedShift-1), 9<<wlFixedShift))
	if x, y, _ := moved.Pointer(); x != 5 || y != 9 {
		t.Errorf("le mouvement a rendu (%d, %d), attendu (5, 9)", x, y)
	}

	clicked := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerButton, 8, 0, evdevButtonLeft, wlPressed),
		wlEvent(ptr, wlPointerButton, 9, 0, evdevButtonSide, wlPressed))
	if !clicked.ButtonHeld(input.ButtonLeft) || !clicked.ButtonJustPressed(input.ButtonLeft) {
		t.Error("le bouton principal devait être tenu et montant")
	}
	if !clicked.ButtonHeld(input.ButtonBack) {
		t.Error("le latéral nommé côté devait rendre le bouton précédent")
	}

	released := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerButton, 10, 0, evdevButtonLeft, wlReleased))
	if released.ButtonHeld(input.ButtonLeft) {
		t.Error("le bouton principal reste tenu après son relâchement")
	}

	// La position reste celle du dernier mouvement après une sortie : elle dit par où
	// le pointeur est sorti, ce qu'un jeu qui suit le curseur veut savoir.
	out := wlSeatStep(t, w, f, wlEvent(ptr, wlPointerLeave, 11, w.surface))
	if x, y, inside := out.Pointer(); inside || x != 5 || y != 9 {
		t.Errorf("la sortie a rendu (%d, %d, %v), attendu (5, 9, false)", x, y, inside)
	}
}

// TestWLPointerCountsANotchOnceWhenBothSourcesSpeak garde le doublon du protocole.
//
// **C'est le cas qui porte la molette, et il rougit sur l'écriture naïve.** Un compositeur
// qui envoie `axis_value120` envoie **aussi** `axis` pour le même geste — « in addition to
// the wl_pointer.axis event » —, donc additionner les deux doublerait chaque cran. Mais
// l'écarter d'office rendrait muet le défilement au doigt, qui n'envoie qu'`axis` : c'est
// la trame qui tranche, et c'est à cela que sert `wl_pointer.frame`.
//
// Le signe est gardé du même geste : le protocole compte vers le bas de l'axe vertical,
// l'état d'entrée vers le haut.
func TestWLPointerCountsANotchOnceWhenBothSourcesSpeak(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())
	ptr := w.seat.pointer

	// Une trame de molette telle qu'un compositeur la compose : la distance, le cran,
	// puis la clôture.
	down := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, wlAxisVertical, wlSigned(wlFixedPerNotch)),
		wlEvent(ptr, wlPointerAxisValue120, wlAxisVertical, wlSigned(wlValue120PerNotch)),
		wlEvent(ptr, wlPointerFrame))
	if got := down.Wheel(); got != -1 {
		t.Errorf("un cran vers le bas a rendu %d crans, attendu -1", got)
	}

	up := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, wlAxisVertical, wlSigned(-wlFixedPerNotch)),
		wlEvent(ptr, wlPointerAxisValue120, wlAxisVertical, wlSigned(-wlValue120PerNotch)),
		wlEvent(ptr, wlPointerFrame))
	if got := up.Wheel(); got != 1 {
		t.Errorf("un cran vers le haut a rendu %d crans, attendu 1", got)
	}

	// Un défilement latéral n'est pas une molette verticale : il est écarté plutôt que
	// confondu avec elle, comme X11 écarte ses boutons six et sept.
	sideways := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, 1, wlSigned(wlFixedPerNotch)),
		wlEvent(ptr, wlPointerAxisValue120, 1, wlSigned(wlValue120PerNotch)),
		wlEvent(ptr, wlPointerFrame))
	if got := sideways.Wheel(); got != 0 {
		t.Errorf("un défilement latéral a rendu %d crans, attendu 0", got)
	}
}

// TestWLPointerAccumulatesWhatAFingerScrolls garde les fractions de cran.
//
// **Un pavé tactile défile par fractions**, et une division entière les perdrait toutes :
// le système crante avant de rapporter sur une molette, jamais sur un doigt. Le reste
// survit donc à la trame, et deux demi-crans en font un — là qu'un backend qui vide son
// accumulateur à chaque trame ne compterait jamais rien.
//
// La seconde moitié garde la même propriété sur une molette haute résolution, qui envoie
// des `axis_value120` plus petits qu'un cran.
func TestWLPointerAccumulatesWhatAFingerScrolls(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())
	ptr := w.seat.pointer

	half := wlSigned(wlFixedPerNotch / 2)
	first := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, wlAxisVertical, half),
		wlEvent(ptr, wlPointerFrame))
	if got := first.Wheel(); got != 0 {
		t.Errorf("un demi-cran au doigt a rendu %d crans, attendu 0", got)
	}

	second := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, wlAxisVertical, half),
		wlEvent(ptr, wlPointerFrame))
	if got := second.Wheel(); got != -1 {
		t.Errorf("le second demi-cran a rendu %d crans, attendu -1", got)
	}

	// Une molette haute résolution compte par fractions de cent vingt, et son reste
	// vit à part de celui du doigt : les deux granularités n'ont pas à se mêler.
	quarter := wlSigned(wlValue120PerNotch / 4)
	for i := range 3 {
		in := wlSeatStep(t, w, f,
			wlEvent(ptr, wlPointerAxisValue120, wlAxisVertical, quarter),
			wlEvent(ptr, wlPointerFrame))
		if got := in.Wheel(); got != 0 {
			t.Fatalf("le quart de cran %d a rendu %d crans, attendu 0", i+1, got)
		}
	}
	last := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxisValue120, wlAxisVertical, quarter),
		wlEvent(ptr, wlPointerFrame))
	if got := last.Wheel(); got != -1 {
		t.Errorf("le quatrième quart de cran a rendu %d crans, attendu -1", got)
	}
}

// TestWLPointerTakesADiscreteNotchFromAnOlderCompositor garde le repli de la molette.
//
// Un compositeur d'avant la huitième version envoie `axis_discrete` et jamais
// `axis_value120` — « this event is not sent to clients supporting version 8 or later » —,
// donc le traiter est la seule façon de servir une molette là-bas. Le cas le vérifie en
// l'émettant tel quel : le client lit ce qui arrive, et ce qui décide du régime est la
// version liée, non ce qu'on devine.
func TestWLPointerTakesADiscreteNotchFromAnOlderCompositor(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())
	ptr := w.seat.pointer

	in := wlSeatStep(t, w, f,
		wlEvent(ptr, wlPointerAxis, 0, wlAxisVertical, wlSigned(wlFixedPerNotch)),
		wlEvent(ptr, wlPointerAxisDiscrete, wlAxisVertical, 1),
		wlEvent(ptr, wlPointerFrame))
	if got := in.Wheel(); got != -1 {
		t.Errorf("un cran discret a rendu %d crans, attendu -1", got)
	}
}

// TestWLWindowHidesTheCursorOnEveryEnter garde la requête que rien ne rappelle.
//
// **C'est une requête datée et non un attribut de fenêtre**, et c'est ce qui la distingue
// d'X11 : elle vaut pour la prise de focus que son numéro de série nomme. L'oublier à la
// seconde prise de focus fait reparaître le curseur **sans aucune erreur de protocole**,
// donc rien d'autre ne l'attraperait — le compositeur de pacotille refuse pour cela un
// numéro qu'aucun `enter` n'a porté, là où un vrai compositeur ignorerait en silence.
func TestWLWindowHidesTheCursorOnEveryEnter(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	o := wlTestOptions()
	o.HideCursor = true
	w := wlSeatMount(t, f, o)
	ptr := w.seat.pointer

	for _, serial := range []uint32{41, 42} {
		f.entered(serial)
		wlSeatStep(t, w, f, wlEvent(ptr, wlPointerEnter, serial, w.surface, 0, 0))
	}
	wlSettle(t, w.c)

	seen := strings.Join(f.saw(), "\n")
	for _, want := range []string{"set_cursor 41 surface 0", "set_cursor 42 surface 0"} {
		if !strings.Contains(seen, want) {
			t.Errorf("le compositeur n'a pas vu %q : le curseur reparaîtrait à cette prise de focus", want)
		}
	}
}

// TestWLWindowLeavesTheCursorAloneWhenTheGameKeepsIt garde l'autre moitié du réglage.
//
// Le protocole laisse l'image du curseur indéfinie à la prise de focus, et en poser une
// demanderait de lire un thème sur le disque. Ne rien envoyer est donc ce qui s'applique,
// et un backend qui enverrait la surface nulle d'office cacherait le curseur d'un jeu qui
// ne l'a pas demandé.
func TestWLWindowLeavesTheCursorAloneWhenTheGameKeepsIt(t *testing.T) {
	f := wlNewFake(wlDefaultGlobals(), []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())

	f.entered(51)
	wlSeatStep(t, w, f, wlEvent(w.seat.pointer, wlPointerEnter, 51, w.surface, 0, 0))
	wlSettle(t, w.c)

	if seen := strings.Join(f.saw(), "\n"); strings.Contains(seen, "set_cursor") {
		t.Error("une requête de curseur est partie sans que le jeu l'ait demandée")
	}
}

// TestWLWindowOpensWithoutASeat garde le seul global que le backend lie sans l'exiger.
//
// Sans compositeur, sans mémoire partagée ou sans protocole de bureau il n'y a pas de
// fenêtre à montrer, et le refus est la seule réponse honnête. Sans siège il y a une
// fenêtre que personne ne pilote — ce qui est exactement un afficheur de kiosque, et un
// régime où un jeu a encore une image à produire.
func TestWLWindowOpensWithoutASeat(t *testing.T) {
	globals := []wlFakeGlobal{
		{1, "wl_compositor", 6},
		{2, "wl_shm", 2},
		{4, "xdg_wm_base", 7},
	}

	f := wlNewFake(globals, []uint32{wlShmXRGB8888})
	w := wlSeatMount(t, f, wlTestOptions())

	if w.seat.id != 0 {
		t.Errorf("le client a lié un siège %d qu'aucun global n'offrait", w.seat.id)
	}

	// La lecture d'un pas doit rester sûre : c'est le chemin que la boucle traverse à
	// chaque image, et il ne doit pas supposer qu'un siège existe.
	if in := w.Input(); in.Held(input.KeyA) {
		t.Error("une touche est tenue sans aucun clavier pour l'avoir enfoncée")
	}
	if err := w.Err(); err != nil {
		t.Errorf("la fenêtre est tombée sans siège : %v", err)
	}
}
