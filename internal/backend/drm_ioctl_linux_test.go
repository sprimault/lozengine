// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'appel au pilote doit tenir avant qu'aucune carte ne soit ouverte.

package backend

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

// TestDRMIoctlReachesTheKernelWithItsArgumentsInPlace garde la mécanique de l'appel sans
// aucun matériel.
//
// **Ce qu'il éprouve n'est pas le pilote mais la place des trois arguments**, et c'est ce
// qui le rend possible partout : un fichier ordinaire n'a pas d'interface d'affichage,
// donc le noyau refuse la requête par `ENOTTY` — « inappropriate ioctl for device ». Ce
// refus **est** la preuve que l'appel a porté : il vient du noyau, après que le numéro
// d'appel système, le descripteur et le mot de requête ont été lus à leur place.
//
// **Ce qui le fait rougir** : permuter le descripteur et la requête, ce qui rend `EBADF`
// puisque le mot de requête ne désigne aucun descripteur ouvert ; et passer le pointeur
// autrement qu'en troisième position, ce qui rend `EFAULT` ou `EINVAL` selon la requête.
// Un cas qui se contenterait de vérifier « une erreur est rendue » ne distinguerait aucun
// des trois, d'où l'assertion sur l'erreur exacte.
//
// Le second régime vérifie qu'un descripteur fermé remonte tel quel, sans être avalé par
// la boucle de réessai : seul `EINTR` y boucle, et confondre les deux ferait tourner une
// boucle sans fin sur une carte qui vient de disparaître.
func TestDRMIoctlReachesTheKernelWithItsArgumentsInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pas-une-carte")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("création du fichier d'essai : %v", err)
	}
	t.Cleanup(func() { file.Close() })

	var res drmModeCardRes
	err = drmIoctl(int(file.Fd()), drmModeGetResources, unsafe.Pointer(&res))
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("sur un fichier ordinaire, la requête rend %v, attendu ENOTTY", err)
	}

	if err := drmIoctl(-1, drmModeGetResources, unsafe.Pointer(&res)); !errors.Is(err, syscall.EBADF) {
		t.Errorf("sur un descripteur invalide, la requête rend %v, attendu EBADF", err)
	}
}

// TestDRMIoctlLeavesItsArgumentUntouchedOnRefusal garde que l'argument ne soit pas
// corrompu quand le pilote refuse.
//
// **C'est la propriété dont les deux appels d'une énumération dépendent** : le premier
// rend les comptes, l'appelant alloue, le second remplit. Si un refus écrivait dans la
// structure, un appelant qui réessaie après une erreur transitoire partirait de comptes
// faux et allouerait trop peu — un dépassement que rien ne signale, puisque le noyau
// écrirait dans la mémoire qu'on lui a décrite.
//
// Le cas est posé ici et non chez l'appelant parce que c'est la fonction d'appel qui
// décide de ce qui traverse : un jour où elle recopierait son argument dans un tampon
// intermédiaire, c'est elle qui devrait tenir cette garantie.
func TestDRMIoctlLeavesItsArgumentUntouchedOnRefusal(t *testing.T) {
	res := drmModeCardRes{countConnectors: 7, maxWidth: 4096}
	before := res

	if err := drmIoctl(-1, drmModeGetResources, unsafe.Pointer(&res)); err == nil {
		t.Fatal("la requête sur un descripteur invalide a réussi")
	}
	if res != before {
		t.Errorf("l'argument a changé pendant un refus : %+v, attendu %+v", res, before)
	}
}
