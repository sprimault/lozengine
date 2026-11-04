// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le seul endroit du moteur qui adresse une requête au pilote d'affichage, et la seule
// conversion de pointeur de cette famille.

package backend

import (
	"syscall"
	"unsafe"
)

// drmIoctl adresse une requête au pilote et rend ce qu'il a refusé.
//
// **C'est l'unique site de conversion de pointeur de la famille, et c'est une décision de
// périmètre.** Un `ioctl` ne se fait pas sans passer l'adresse de son argument, donc la
// conversion est inévitable ; ce qui se choisissait est son nombre. Une par requête en
// aurait mis seize dans la liste close de `conversions_test.go`, chacune à relire
// séparément alors qu'elles ne diffèrent que par la structure. Ici il y en a une, et toute
// la relecture tient dans cette fonction.
//
// **L'appelant garde la propriété de l'argument**, qui doit être un pointeur vers une
// structure du relevé — la fonction ne le retient pas au-delà de l'appel, donc rien n'a à
// être épinglé. Le pointeur traverse par `uintptr` dans le même appel, ce que la
// documentation d'`unsafe` autorise nommément pour `syscall.Syscall` : la conversion et
// l'appel forment une seule expression, donc le ramasse-miettes ne peut pas déplacer la
// valeur entre les deux.
//
// **Le réessai sur `EINTR` n'est pas de la défensive** : le noyau interrompt un `ioctl`
// d'affichage quand un signal arrive, et un client qui ne réessaie pas perd un réglage de
// mode ou une bascule pour une raison étrangère à l'affichage. `EAGAIN` n'est pas réessayé
// ici, lui : sur une bascule de page il veut dire qu'une bascule est déjà en vol, ce que le
// contrat de fenêtre traite comme un tour à sauter et non comme une attente.
func drmIoctl(fd int, request uintptr, arg unsafe.Pointer) error {
	for {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, uintptr(arg))
		switch errno {
		case 0:
			return nil
		case syscall.EINTR:
			continue
		default:
			return errno
		}
	}
}
