// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

// genre dit ce que le blit a à faire d'une suite de pixels.
type genre uint8

const (
	saut    genre = iota // entièrement transparents : rien à écrire
	copie                // entièrement opaques : le tampon se remplace
	melange              // partiellement transparents : un mélange par pixel
)

// sequence est une suite de pixels consécutifs de même genre dans une ligne.
//
// Elle ne porte pas sa colonne de départ : les séquences d'une ligne la pavent
// exactement, de la première colonne à la dernière, si bien que la position se
// déduit en avançant. Une donnée redondante finirait par diverger de celle qu'on
// déduit, et c'est ce pavage que le blit tient pour acquis.
type sequence struct {
	nombre uint16
	genre  genre
}

// encoder calcule les séquences de chaque ligne de la planche.
//
// Le gain n'est pas d'éviter un test d'alpha par pixel, mais de donner au blit des
// segments sur lesquels il peut travailler en bloc : un saut n'écrit rien, une
// copie se ramène à un copy, et seul le mélange descend au pixel. Une planche de
// sprite étant surtout du vide et des aplats, l'essentiel d'une ligne se traite
// alors en trois décisions.
//
// Le coût est une allocation par planche, au chargement. Le pire cas — un pixel
// sur deux à demi transparent — rend une séquence par pixel et ne fait perdre que
// la mémoire ; aucune planche de ce genre ne se dessine.
func (p *Planche) encoder() {
	p.lignes = make([]uint32, int(p.Hauteur)+1)
	p.sequences = make([]sequence, 0, int(p.Hauteur))

	for y := 0; y < int(p.Hauteur); y++ {
		p.lignes[y] = uint32(len(p.sequences))

		ligne := p.Pixels[y*int(p.Largeur) : (y+1)*int(p.Largeur)]
		for x := 0; x < len(ligne); {
			g := genreDe(ligne[x].A)
			n := 1
			for x+n < len(ligne) && genreDe(ligne[x+n].A) == g {
				n++
			}
			p.sequences = append(p.sequences, sequence{uint16(n), g})
			x += n
		}
	}
	p.lignes[p.Hauteur] = uint32(len(p.sequences))
}

// genreDe classe un pixel par son alpha. Les deux extrêmes sont les cas rapides,
// et ce sont eux qu'une planche porte presque partout.
func genreDe(alpha uint8) genre {
	switch alpha {
	case 0:
		return saut
	case 255:
		return copie
	default:
		return melange
	}
}

// sequencesDe rend les séquences d'une ligne.
func (p *Planche) sequencesDe(y uint16) []sequence {
	return p.sequences[p.lignes[y]:p.lignes[y+1]]
}
