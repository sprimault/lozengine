// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'ouverture sous Windows : déclarer la densité, enregistrer la classe une fois,
// créer la fenêtre à la taille utile demandée, et la montrer.
//
// L'ordre n'est pas indifférent. La densité se déclare avant toute fenêtre, sans quoi
// le système en a déjà décidé pour le processus ; et la taille utile ne s'obtient
// qu'en ajustant celle qu'on demande, `CreateWindowExW` recevant la fenêtre entière.

package backend

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// windowClassName est le nom sous lequel la classe de fenêtre est enregistrée. Un seul
// enregistrement par processus, d'où le nom fixe.
const windowClassName = "LozengineWindow"

// registration enregistre la classe de fenêtre une fois pour le processus, et garde ce
// que cet enregistrement a donné.
var registration struct {
	once sync.Once
	name *uint16
	err  error
}

// Open monte une fenêtre et rend de quoi la piloter.
//
// Le premier geste est de déclarer la conscience de densité, et c'est une condition de
// l'échelle entière : sans elle, le système montre un écran plus petit que le vrai,
// puis étire l'image pour remplir le vrai — soit une mise à l'échelle fractionnaire
// que rien, dans les nombres lus, ne trahirait. L'échec est ignoré, et il ne signifie
// qu'une chose : un hôte a déjà décidé pour le processus, ce qui est son droit.
//
// Le déclarer par appel plutôt que par manifeste est un arbitrage assumé : Go
// n'embarque pas de manifeste sans ressource compilée par un outil de plus, et
// `go build` doit suffire.
func Open(o Options) (Window, error) {
	setProcessDpiAwarenessContext.Call(perMonitorAwareV2)

	name, err := registerClass()
	if err != nil {
		return nil, err
	}

	title, err := syscall.UTF16PtrFromString(o.Title)
	if err != nil {
		return nil, fmt.Errorf("window title %q: %w", o.Title, err)
	}

	// La taille demandée est celle de la zone utile ; celle qu'on passe au système
	// est la fenêtre entière.
	outerW, outerH := outerSize(o.Width, o.Height, wsOverlappedWindow)

	w := &window{
		lastW: o.Width,
		lastH: o.Height,
		info:  newBitmapInfo(),
		args:  newPresentArgs(),
	}

	hwnd, _, err := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow,
		cwUseDefault, cwUseDefault,
		uintptr(outerW), uintptr(outerH),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("creating the window: %w", err)
	}
	w.hwnd = hwnd

	windows.Lock()
	windows.byHandle[hwnd] = w
	windows.Unlock()

	// Le contexte est demandé une fois et gardé, la classe portant csOwnDC. Il n'est
	// pas découpé à la région de mise à jour, donc la présentation n'a ni à valider
	// ni à suivre quoi que ce soit.
	w.dc, _, _ = getDC.Call(hwnd)

	// Les icônes se posent avant de montrer la fenêtre, faute de quoi le système
	// afficherait la sienne le temps d'une image.
	w.setIcons(o.Icons)

	showWindow.Call(hwnd, swShow)

	// La taille utile réelle, qui peut différer de celle demandée si le système a
	// contraint la fenêtre.
	if width, height := clientSize(hwnd); width != 0 && height != 0 {
		w.lastW, w.lastH = width, height
	}

	return w, nil
}

// registerClass enregistre la classe de fenêtre au premier appel et rend son nom.
//
// Une seule fois par processus : un second enregistrement du même nom échouerait, et
// un jeu qui rouvrirait sa fenêtre après l'avoir fermée est un cas ordinaire.
func registerClass() (*uint16, error) {
	registration.once.Do(func() {
		name, err := syscall.UTF16PtrFromString(windowClassName)
		if err != nil {
			registration.err = fmt.Errorf("window class name: %w", err)
			return
		}

		instance, _, _ := getModuleHandleW.Call(0)

		// Le curseur se charge ici et non à chaque fenêtre : il appartient à la
		// classe, et `LoadCursorW` sur une ressource du système rend une poignée
		// partagée qu'il ne faut pas libérer.
		cursor, _, _ := loadCursorW.Call(0, idcArrow)

		c := windowClass{
			style:     csHRedraw | csVRedraw | csOwnDC,
			wndProc:   syscall.NewCallback(windowProc),
			instance:  instance,
			cursor:    cursor,
			className: name,
		}
		c.size = uint32(unsafe.Sizeof(c))

		if atom, _, err := registerClassExW.Call(uintptr(unsafe.Pointer(&c))); atom == 0 {
			registration.err = fmt.Errorf("registering the window class: %w", err)
			return
		}
		registration.name = name
	})
	return registration.name, registration.err
}
