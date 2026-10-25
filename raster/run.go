// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'encodage d'une planche en séquences de pixels de même nature — transparents,
// opaques, translucides. Calculé une fois au chargement, il permet au dessin de sauter
// les transparents et de recopier les opaques en bloc au lieu de tester chaque pixel.

package raster

// kind dit ce que le blit a à faire d'une suite de pixels.
type kind uint8

const (
	transparent kind = iota // entièrement transparents : rien à écrire
	opaque                  // entièrement opaques : le tampon se remplace
	translucent             // partiellement transparents : un mélange par pixel
)

// run est une suite de pixels consécutifs de même genre dans une ligne.
//
// Elle ne porte pas sa colonne de départ : les séquences d'une ligne la pavent
// exactement, de la première colonne à la dernière, si bien que la position se
// déduit en avançant. Une donnée redondante finirait par diverger de celle qu'on
// déduit, et c'est ce pavage que le blit tient pour acquis.
type run struct {
	count uint16
	kind  kind
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
func (p *Sheet) encode() {
	p.rows = make([]uint32, int(p.Height)+1)
	p.runs = make([]run, 0, int(p.Height))

	for y := 0; y < int(p.Height); y++ {
		p.rows[y] = uint32(len(p.runs))

		row := p.Pixels[y*int(p.Width) : (y+1)*int(p.Width)]
		for x := 0; x < len(row); {
			k := kindOf(row[x].A)
			n := 1
			for x+n < len(row) && kindOf(row[x+n].A) == k {
				n++
			}
			p.runs = append(p.runs, run{uint16(n), k})
			x += n
		}
	}
	p.rows[p.Height] = uint32(len(p.runs))
}

// kindOf classe un pixel par son alpha. Les deux extrêmes sont les cas rapides,
// et ce sont eux qu'une planche porte presque partout.
func kindOf(alpha uint8) kind {
	switch alpha {
	case 0:
		return transparent
	case 255:
		return opaque
	default:
		return translucent
	}
}

// runsOf rend les séquences d'une ligne.
func (p *Sheet) runsOf(y uint16) []run {
	return p.runs[p.rows[y]:p.rows[y+1]]
}
