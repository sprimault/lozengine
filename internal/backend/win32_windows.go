// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le backend emprunte à Windows : les procédures, les constantes et les
// structures, et rien qui décide de quoi que ce soit.
//
// Les valeurs ne sont pas recopiées d'un en-tête ni d'un autre moteur : chacune a été
// éprouvée par son effet avant d'entrer ici — une opération raster par ce qu'elle
// peint, un numéro de message par le moment où il arrive. Les tailles de structures se
// prennent par `unsafe.Sizeof` plutôt que de s'écrire, parce qu'elles diffèrent entre
// `amd64` et `386` et que la seule façon de ne pas s'y tromper est de ne pas les
// connaître.

package backend

import (
	"syscall"
	"unsafe"
)

// Les deux bibliothèques du système, nommées sans chemin : elles figurent dans
// `KnownDLLs`, donc le chargeur prend la section déjà mappée et ne parcourt aucun
// chemin de recherche. Une bibliothèque plantée à côté du binaire ne peut pas s'y
// substituer, et il n'y a donc ni `LoadLibraryEx` ni chemin en dur à écrire.
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
)

// Les procédures employées, résolues au premier appel puis gardées.
//
// `LazyProc.Call` et non `SyscallN` : l'écart mesuré est de six nanosecondes sur
// soixante, et la pompe à messages en fait quelques dizaines par image, soit le
// millième de pour cent d'une image. Garder les adresses dans des variables
// n'apporterait rien non plus — un nanomètre de différence —, `Addr` étant un
// `sync.Once` déjà franchi.
var (
	registerClassExW = user32.NewProc("RegisterClassExW")
	createWindowExW  = user32.NewProc("CreateWindowExW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	defWindowProcW   = user32.NewProc("DefWindowProcW")
	showWindow       = user32.NewProc("ShowWindow")
	peekMessageW     = user32.NewProc("PeekMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessageW = user32.NewProc("DispatchMessageW")
	getClientRect    = user32.NewProc("GetClientRect")
	adjustWindowRect = user32.NewProc("AdjustWindowRect")
	getDC            = user32.NewProc("GetDC")
	releaseDC        = user32.NewProc("ReleaseDC")

	// Dans kernel32 et non user32, ce qu'un `LazyProc` ne dit qu'à l'appel : la
	// compilation et les tests sans écran laissaient passer l'erreur, et c'est la
	// vérification manuelle qui l'a trouvée.
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	setProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")

	setDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
)

// Les styles de classe et de fenêtre.
const (
	// csOwnDC donne à la fenêtre son propre contexte de périphérique, qui n'a donc
	// pas à être repris à chaque image.
	csVRedraw = 0x0001
	csHRedraw = 0x0002
	csOwnDC   = 0x0020

	// wsOverlappedWindow est la fenêtre ordinaire : bordure, titre, menu système,
	// boutons de réduction et d'agrandissement.
	wsOverlappedWindow = 0x00CF0000

	// cwUseDefault laisse le système placer la fenêtre.
	cwUseDefault = ^uintptr(0x7FFFFFFF)
)

// Les commandes de ShowWindow.
const (
	swShow = 5
)

// Les messages que le backend traite. Tous les autres vont à `DefWindowProcW` : un
// cycle de vie trivial en apporte trente-deux, dont deux enregistrés dynamiquement par
// le système, et aucun backend ne peut présumer de la liste.
const (
	wmDestroy   = 0x0002
	wmSize      = 0x0005
	wmKillFocus = 0x0008
	wmClose     = 0x0010
)

// sizeMinimized est le `wParam` de `wmSize` qui annonce la réduction. C'est par lui
// que la minimisation se lit, et non en interrogeant la fenêtre à chaque image.
const sizeMinimized = 1

// pmRemove retire de la file le message que `PeekMessageW` rend.
const pmRemove = 0x0001

// dibRGBColors dit que les couleurs du DIB sont littérales, sans passer par une
// palette.
const dibRGBColors = 0

// perMonitorAwareV2 est la poignée opaque de conscience de densité la plus complète.
//
// Sans elle, le système montre au processus un écran plus petit que le vrai et étire
// l'image pour remplir l'écran : le facteur entier serait calculé sur une taille
// fausse puis multiplié par une fraction, ce que l'échelle entière obligatoire existe
// pour interdire — et rien, dans les nombres que le moteur lit, ne le trahirait.
const perMonitorAwareV2 = ^uintptr(3)

// rect est le rectangle que rendent `GetClientRect` et qu'attend `AdjustWindowRect`.
type rect struct {
	left, top, right, bottom int32
}

// point est la position que portent les messages de souris, dont la pompe n'a pas
// l'usage mais que la structure d'un message contient.
type point struct {
	x, y int32
}

// message est ce que la pompe reçoit.
//
// Sa taille diffère entre les deux architectures — quarante-huit octets contre
// vingt-huit —, et c'est le compilateur qui le sait.
type message struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// windowClass est l'enregistrement d'une windowClass de fenêtre.
type windowClass struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

// bitmapInfoHeader décrit au système l'image que la présentation lui donne.
//
// Elle ne porte aucun pointeur, donc sa taille est la même sur les deux
// architectures : le chemin de présentation n'a aucune précaution à prendre de ce
// côté.
type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

// clientSize rend les dimensions de la zone utile, en pixels.
//
// Zéro quand la fenêtre est réduite, ce que l'appel système rend tel quel : c'est à
// l'appelant de ne pas en tirer un facteur d'agrandissement.
func clientSize(hwnd uintptr) (width, height uint16) {
	var r rect
	if ret, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ret == 0 {
		return 0, 0
	}
	return uint16(r.right - r.left), uint16(r.bottom - r.top)
}

// outerSize rend la taille de fenêtre qui donne la zone utile demandée.
//
// `CreateWindowExW` reçoit la taille de la fenêtre entière, bordures et barre de titre
// comprises : demander 480×270 donne une zone utile de 464×231 sur un thème ordinaire,
// et le facteur entier tomberait alors à un cran de moins que ce que l'écran permet.
// L'écart dépend du thème, donc il se calcule et ne s'écrit pas.
func outerSize(width, height uint16, style uintptr) (uint16, uint16) {
	r := rect{right: int32(width), bottom: int32(height)}
	if ret, _, _ := adjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), style, 0); ret == 0 {
		return width, height
	}
	return uint16(r.right - r.left), uint16(r.bottom - r.top)
}
