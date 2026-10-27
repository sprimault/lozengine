// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qui se vérifie sans ouvrir de fenêtre, et c'est là que vivent les défauts : la
// traduction des messages en état, la taille qu'on rend quand le système n'en donne
// plus, et la disposition des structures que les appels système lisent.
//
// Aucun test n'ouvre de fenêtre, et c'est délibéré : l'intégration continue lance la
// suite sur une machine où rien ne garantit qu'une fenêtre puisse s'ouvrir. Les appels
// eux-mêmes ne sont donc couverts par aucun test, et la fenêtre se vérifie à la main
// une fois — ce qu'il faut dire plutôt que de laisser croire le contraire.

package backend

import (
	"testing"
	"unsafe"

	"github.com/sprimault/lozengine/input"
)

// deadHandle est une valeur qui n'est la poignée d'aucune fenêtre. Les appels
// système qui la reçoivent échouent proprement, ce qui est exactement ce qu'on veut
// éprouver.
const deadHandle = 0xDEAD

// track range une fenêtre dans la table pour la durée du test, comme Open le fait.
func track(t *testing.T, w *window) {
	t.Helper()

	w.hwnd = deadHandle
	windows.Lock()
	windows.byHandle[deadHandle] = w
	windows.Unlock()

	t.Cleanup(func() {
		windows.Lock()
		delete(windows.byHandle, deadHandle)
		windows.Unlock()
	})
}

// TestKillFocusForgetsHeldKeys couvre le défaut que la sonde a mis au jour, et c'est le
// seul cas à le pouvoir : au départ du focus, le relâchement d'une touche tenue est
// perdu et non différé, et l'état du système dit encore « enfoncée ».
//
// Sans l'oubli, une touche tenue le resterait pour toujours et le personnage
// marcherait tout seul au retour dans la fenêtre — un défaut qui ne se voit qu'à
// l'usage, et qu'aucune image de référence n'attraperait.
func TestKillFocusForgetsHeldKeys(t *testing.T) {
	var w window
	track(t, &w)

	// Un code de balayage brut, les noms de touches venant avec le lot qui lit le
	// clavier : 0x4D est la flèche right du bloc directionnel.
	const right = input.Key(0x4D)

	w.state.Press(right)
	if !w.state.Frame().Held(right) {
		t.Fatal("la touche n'est pas tenue avant le départ du focus")
	}

	windowProc(w.hwnd, wmKillFocus, 0, 0)

	if w.state.Frame().Held(right) {
		t.Error("la touche est encore tenue après le départ du focus")
	}
}

// TestSizeKeepsTheLastKnownSize vérifie que la boucle ne reçoit jamais zéro.
//
// Une fenêtre réduite a une zone utile nulle, et le facteur d'agrandissement s'en
// déduirait par une division par zéro. C'est au backend d'absorber le cas, puisque lui
// seul sait que la fenêtre est réduite.
func TestSizeKeepsTheLastKnownSize(t *testing.T) {
	cases := []struct {
		name          string
		w             window
		width, height uint16
	}{
		{"réduite", window{minimized: true, lastW: 480, lastH: 270}, 480, 270},
		{"poignée morte", window{lastW: 320, lastH: 180}, 320, 180},
		{"rien de connu", window{}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			width, height := c.w.Size()
			if width != c.width || height != c.height {
				t.Errorf("taille : %d×%d, attendu %d×%d", width, height, c.width, c.height)
			}
		})
	}
}

// TestSizeMessageTracksMinimization vérifie que la réduction se lit sur le message que
// le système envoie, et non en l'interrogeant à chaque image.
func TestSizeMessageTracksMinimization(t *testing.T) {
	w := window{lastW: 480, lastH: 270}
	track(t, &w)

	windowProc(w.hwnd, wmSize, sizeMinimized, 0)
	if !w.minimized {
		t.Error("la réduction n'a pas été relevée")
	}

	// Restaurée, la fenêtre reprend sa taille réelle ; la poignée étant morte, la
	// dernière connue doit survivre plutôt que de tomber à zéro.
	windowProc(w.hwnd, wmSize, 0, 0)
	if w.minimized {
		t.Error("la fenêtre est restée réduite")
	}
	if width, height := w.Size(); width != 480 || height != 270 {
		t.Errorf("taille après restauration : %d×%d, attendu 480×270", width, height)
	}
}

// TestCloseMessagesMarkTheWindowClosed vérifie les deux chemins par lesquels une
// fenêtre cesse de présenter : la fermeture demandée et la destruction.
func TestCloseMessagesMarkTheWindowClosed(t *testing.T) {
	for _, message := range []uint32{wmClose, wmDestroy} {
		var w window
		track(t, &w)

		if w.Closed() {
			t.Fatal("fenêtre déjà fermée avant tout message")
		}
		windowProc(w.hwnd, message, 0, 0)
		if !w.Closed() {
			t.Errorf("message %#04x : la fenêtre ne se dit pas fermée", message)
		}
	}
}

// TestCloseTwiceIsHarmless vérifie ce que le contrat promet : appeler deux fois est
// sans effet. Un hôte qui ferme dans un defer et dans sa boucle le fera.
func TestCloseTwiceIsHarmless(t *testing.T) {
	var w window

	w.Close()
	w.Close()

	if !w.Closed() {
		t.Error("la fenêtre ne se dit pas fermée")
	}
}

// TestOrdinaryCloseIsNotAnError garde ce que la boucle en tire : une fenêtre que le
// joueur ferme ne rapporte aucune erreur, et c'est ce qui permet de rendre Err sans
// trier les causes de sortie.
//
// Le cas ne peut pas rougir sur le fond — cette plateforme n'a aucun échec terminal à
// rapporter, ce que la godoc d'Err dit — mais il rougirait si quelqu'un faisait un jour
// rendre une erreur à la fermeture ordinaire, et c'est le régime qu'il garde.
func TestOrdinaryCloseIsNotAnError(t *testing.T) {
	var w window
	track(t, &w)

	windowProc(w.hwnd, wmClose, 0, 0)

	if err := w.Err(); err != nil {
		t.Errorf("une fermeture ordinaire rapporte une erreur : %v", err)
	}
}

// TestUnknownWindowIsDelegated vérifie qu'un message portant une poignée inconnue ne
// fait rien de fâcheux : le système en envoie avant que CreateWindowExW ait rendu la
// poignée, donc le cas est normal et non défensif.
func TestUnknownWindowIsDelegated(t *testing.T) {
	windowProc(0xBEEF, wmSize, sizeMinimized, 0)
	windowProc(0xBEEF, wmKillFocus, 0, 0)
}

// TestOuterSizeExceedsTheClientArea vérifie l'ajustement sans lequel un jeu en 480×270
// n'aurait que 464×231 de zone utile, et un facteur d'agrandissement d'un cran trop
// bas.
//
// L'écart dépend du thème, donc le cas ne vérifie pas un nombre : il vérifie qu'il y a
// un écart, et que l'ajustement va dans le bon sens.
func TestOuterSizeExceedsTheClientArea(t *testing.T) {
	const width, height = 480, 270

	outer, outerHeight := outerSize(width, height, wsOverlappedWindow)

	if outer < width || outerHeight < height {
		t.Fatalf("fenêtre de %d×%d pour une zone utile de %d×%d", outer, outerHeight, width, height)
	}
	if outer == width && outerHeight == height {
		t.Error("l'ajustement n'a rien ajouté, alors que le style porte bordure et titre")
	}
}

// TestClientSizeOfADeadHandleIsZero vérifie que l'échec de l'appel se traduit par zéro
// et non par des valeurs prises au hasard dans la pile.
func TestClientSizeOfADeadHandleIsZero(t *testing.T) {
	if width, height := clientSize(deadHandle); width != 0 || height != 0 {
		t.Errorf("zone utile d'une poignée morte : %d×%d", width, height)
	}
}

// TestStructureSizes vérifie la disposition que les appels système lisent, et c'est le
// seul contrôle qui sépare `amd64` de `386`.
//
// Les tailles ne sont pas écrites dans le code — elles se prennent par `unsafe.Sizeof`
// —, mais un champ mal ordonné ou mal typé changerait la taille sans rien casser à la
// compilation, et l'appel lirait de travers. Les valeurs attendues viennent de la
// sonde du jalon, mesurées sur les deux architectures.
func TestStructureSizes(t *testing.T) {
	const pointer = unsafe.Sizeof(uintptr(0))

	cases := []struct {
		name       string
		size       uintptr
		on64, on32 uintptr
	}{
		{"classe", unsafe.Sizeof(windowClass{}), 80, 48},
		{"message", unsafe.Sizeof(message{}), 48, 28},
		{"rect", unsafe.Sizeof(rect{}), 16, 16},
		{"point", unsafe.Sizeof(point{}), 8, 8},
		{"bitmapInfoHeader", unsafe.Sizeof(bitmapInfoHeader{}), 40, 40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := c.on64
			if pointer == 4 {
				want = c.on32
			}
			if c.size != want {
				t.Errorf("%s fait %d octets, attendu %d sur un pointeur de %d",
					c.name, c.size, want, pointer)
			}
		})
	}
}

// TestClassSizeIsItsOwnField vérifie l'invariant qu'un enregistrement de classe porte
// sur lui-même : le système refuse la classe si le champ ne dit pas la taille réelle,
// et l'écrire en dur serait faux sur l'une des deux architectures.
func TestClassSizeIsItsOwnField(t *testing.T) {
	c := windowClass{}
	c.size = uint32(unsafe.Sizeof(c))

	if uintptr(c.size) != unsafe.Sizeof(windowClass{}) {
		t.Errorf("champ size : %d, taille réelle %d", c.size, unsafe.Sizeof(windowClass{}))
	}
}
