// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La poignée qui désigne une entité : un rang et une génération dans un seul entier,
// pour qu'une poignée gardée trop longtemps ne finisse pas par désigner l'entité qui a
// repris sa place.

package scene

// Handle désigne une entité posée dans la scène, et vaut jusqu'à son retrait.
//
// Vingt bits de rang et douze de génération. La génération n'est pas du luxe : retirer
// une entité libère son rang, qu'une autre reprend ensuite, et une poignée gardée trop
// longtemps désignerait alors la nouvelle. Le moteur rendrait une image irréprochable
// pour une scène fausse, et la cause — la disparition d'une troisième entité, ailleurs,
// à un autre moment — ne se relie à rien quand on la voit à l'écran.
//
// Sa valeur zéro ne désigne rien : les générations commencent à un, donc un rang nul ne
// se confond pas avec une poignée qu'on a oublié de remplir.
type Handle uint32

const (
	rankBits = 20

	// maxRank et generationMax sont ce que les deux champs portent : un million
	// d'entités vivantes à la fois, et quatre mille réemplois d'un même rang avant
	// qu'une poignée périmée puisse se faire passer pour valide.
	maxRank       = 1<<rankBits - 1
	maxGeneration = 1<<(32-rankBits) - 1
)

// compose assemble une poignée depuis un rang et une génération.
func compose(rank, generation uint32) Handle {
	return Handle(generation<<rankBits | rank)
}

// rank rend la place que la poignée désigne dans les tables de la scène.
func (p Handle) rank() uint32 { return uint32(p) & maxRank }

// generation rend le numéro de réemploi du rang, zéro pour une poignée vide.
func (p Handle) generation() uint32 { return uint32(p) >> rankBits }
