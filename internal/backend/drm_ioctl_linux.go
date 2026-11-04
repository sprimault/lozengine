// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le seul endroit du moteur qui adresse une requête au pilote d'affichage, et la seule
// conversion de pointeur de cette famille.

package backend

import (
	"runtime"
	"syscall"
	"unsafe"
)

// drmIoctl adresse une requête au pilote et rend ce qu'il a refusé.
//
// **Elle est générique, et c'est ce qui fait d'elle l'unique site de conversion de la
// famille.** La première forme prenait un `unsafe.Pointer`, donc chaque appelant
// convertissait le sien : la décision annonçait « un seul site » et le code en portait
// onze, ce que la liste close de `conversions_test.go` a refusé. Un paramètre de type
// déplace la conversion ici, une fois pour toutes, et retire l'`unsafe.Pointer` des mains
// de l'appelant — qui passe `&arg` comme il le ferait pour n'importe quelle fonction.
//
// **L'appelant garde la propriété de l'argument** : la fonction ne le retient pas au-delà
// de l'appel, donc rien n'a à être épinglé. Le pointeur traverse par `uintptr` dans la
// même expression que l'appel, ce que la documentation d'`unsafe` autorise nommément pour
// `syscall.Syscall` — la conversion et l'appel ne sont pas séparables, donc rien ne peut
// déplacer la valeur entre les deux.
//
// **Le réessai sur `EINTR` n'est pas de la défensive** : le noyau interrompt un `ioctl`
// d'affichage quand un signal arrive, et un client qui ne réessaie pas perd un réglage de
// mode ou une bascule pour une raison étrangère à l'affichage. `EAGAIN` n'est pas réessayé
// ici, lui : sur une bascule de page il veut dire qu'une bascule est déjà en vol, ce que le
// contrat de fenêtre traite comme un tour à sauter et non comme une attente.
func drmIoctl[T any](fd int, request uintptr, arg *T) error {
	for {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, uintptr(unsafe.Pointer(arg)))
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

// drmIoctlInto adresse une requête dont l'argument désigne des tableaux que le noyau
// remplit, et garde ces tableaux vivants le temps de l'appel.
//
// **C'est le second et dernier site de conversion de la famille, et il existe pour une
// raison précise.** Les énumérations du réglage de mode ne rendent pas leurs résultats
// dans leur structure : elles y lisent des **adresses** de tableaux, dans des champs de
// soixante-quatre bits. L'appelant écrit donc une adresse Go dans la structure avant
// l'appel, et cette adresse n'est plus un pointeur du point de vue du compilateur — rien
// ne relie plus le tableau à la structure qui le désigne.
//
// **Ce que `keep` tient**, et ce n'est pas de la précaution : la documentation d'`unsafe`
// pose qu'un `uintptr` n'est pas une référence, donc un tableau dont seule l'adresse
// subsiste peut être ramassé. En pratique le compilateur fait échapper au tas toute valeur
// dont l'adresse passe par `unsafe.Pointer`, et le ramasse-miettes de Go ne déplace pas ce
// qui est sur le tas — mais ni l'un ni l'autre n'est garanti par la spécification, et un
// `runtime.KeepAlive` après l'appel coûte zéro instruction.
//
// La forme retenue oblige l'appelant à **nommer** ce qu'il garde, ce qu'une conversion
// écrite sur place ne ferait pas : un tableau oublié dans la liste serait un défaut
// visible en relecture, là qu'une adresse posée dans un champ ne dit rien de ce qu'elle
// désigne.
func drmIoctlInto[T any](fd int, request uintptr, arg *T, keep ...any) error {
	err := drmIoctl(fd, request, arg)
	runtime.KeepAlive(keep)
	return err
}

// drmArrayAddr rend l'adresse du premier élément d'un tableau, telle qu'une structure du
// réglage de mode l'attend.
//
// Elle rend zéro pour un tableau vide, ce qui est exactement ce que le **premier** appel
// d'une énumération demande : des adresses nulles avec des comptes nuls, auxquelles le
// noyau répond par les comptes réels. Indexer un premier élément qui n'existe pas serait
// l'erreur naturelle ici, et elle ferait paniquer la découverte au lieu de la commencer.
//
// Le tableau que l'appelant passe doit survivre à l'appel, et c'est à quoi sert le dernier
// argument de [drmIoctlInto].
func drmArrayAddr[T any](a []T) uint64 {
	if len(a) == 0 {
		return 0
	}
	return uint64(uintptr(unsafe.Pointer(&a[0])))
}
