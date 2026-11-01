// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la page attend du tampon du moteur : rien. Le fichier porte la relecture en
// octets, et il n'importe pas `syscall/js` pour que ses cas tournent sur l'hôte.

package backend

import (
	"unsafe"

	"github.com/sprimault/lozengine/render"
)

// bytesOf relit une tranche de pixels comme une tranche d'octets, sans rien copier ni
// convertir.
//
// **Le navigateur est la seule des trois familles qui n'ait aucune conversion à faire**, et
// c'est la disposition de [render.Color] qui le donne : quatre `uint8` nommés R, G, B et A
// dans cet ordre, soit exactement l'ordre que `ImageData.data` attend. Windows échange le
// rouge et le bleu, X11 suit l'ordre du visuel qu'il a choisi ; ici la tranche **est** déjà
// la disposition de destination.
//
// # Pourquoi une conversion de pointeur, là que les deux autres familles l'ont refusée
//
// Le refus de X11 est écrit dans son propre fichier, et son argument ne vaut pas ici : il
// doit permuter des composantes, donc il parcourt les pixels de toute façon et la
// conversion ne lui épargnerait qu'une copie — 0,45 ms sur un budget de 16,7. Le web n'a
// rien à permuter : la conversion **est** toute l'opération, et la boucle qu'elle remplace
// ne changerait pas un octet. Mesuré le 1er novembre 2026, elle vaut un facteur douze sur
// le chemin de présentation entier.
//
// Elle s'inscrit donc dans la liste close du Makefile, que `make conversions` tient par
// fichier et par compte. C'est ce qui rend la décision visible en relecture, et c'est
// nécessaire parce que `go vet` ne voit pas cette forme : son analyseur `unsafeptr` ne
// signale que les allers-retours par `uintptr`.
//
// # Ce que la fonction ne fait pas, et qui appartient à l'appelant
//
// Elle ne retient rien : la tranche rendue vit le temps de l'appel qui la consomme. Une
// vue tenue d'une image à l'autre est un autre sujet — elle se détache quand le tas de Go
// croît, et la réallocation d'une surface en déplace l'adresse sans la détacher —, donc
// c'est à l'appelant de la refaire, et d'épingler ce qu'il garde.
//
// La tranche vide rend une tranche vide plutôt que d'indexer un premier élément qui
// n'existe pas.
func bytesOf(pixels []render.Color) []byte {
	if len(pixels) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&pixels[0])), len(pixels)*4)
}
