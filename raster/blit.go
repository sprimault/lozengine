// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "github.com/sprimault/lozengine/rendu"

// Draw peint les quads dans l'ordre du tableau, sans rien allouer.
//
// L'ordre est la seule vérité du résultat : le rastériseur ne trie pas, ne projette
// pas et ne consulte aucune scène. Un quad qui désigne une planche absente de
// l'atlas est une erreur de programmation en amont ; l'index déborde et le
// programme s'arrête, ce qui vaut mieux qu'une image muette dont on chercherait la
// cause dans le tri.
//
// Une fonction et non une méthode : le tampon vit dans [rendu], que l'ABI expose, et
// Go veut qu'une méthode soit déclarée dans le paquet de son type.
func Draw(t *rendu.Tampon, r rendu.Rendu, a *Atlas) {
	for _, q := range r {
		quad(t, q, a)
	}
}

// quad découpe un quad sur les bords du tampon, puis le peint.
//
// Le découpage se fait sur la destination et se reporte sur la source, de sorte
// qu'un sprite à moitié sorti de l'écran coûte la moitié de son travail et non un
// test par pixel. La source, elle, n'est pas vérifiée : un rectangle qui dépasse la
// planche dessine ce qu'il trouve et s'arrête à la dernière ligne, plutôt que de
// lire la ligne suivante.
func quad(t *rendu.Tampon, q rendu.Quad, a *Atlas) {
	left, top := int(q.X), int(q.Y)
	x0, y0 := max(left, 0), max(top, 0)
	x1 := min(left+int(q.Largeur), int(t.Largeur))
	y1 := min(top+int(q.Hauteur), int(t.Hauteur))
	if x0 >= x1 || y0 >= y1 {
		return
	}

	if q.Drapeaux&rendu.Aplat != 0 {
		fillRect(t, x0, y0, x1, y1, q.Teinte)
		return
	}

	p := &a.Sheets[q.Planche]
	neutral := q.Teinte == rendu.Blanc()
	mirrorX := q.Drapeaux&rendu.MiroirX != 0
	sourceX, end := int(q.SourceX), int(q.SourceX)+int(q.Largeur)

	for y := y0; y < y1; y++ {
		row := y - top
		if q.Drapeaux&rendu.MiroirY != 0 {
			row = int(q.Hauteur) - 1 - row
		}
		source := int(q.SourceY) + row
		if source >= int(p.Height) {
			continue
		}

		dst := t.Pixels[y*int(t.Largeur) : (y+1)*int(t.Largeur)]
		src := p.Pixels[source*int(p.Width) : (source+1)*int(p.Width)]

		column := 0
		for _, s := range p.runsOf(uint16(source)) {
			start := column
			column += int(s.count)
			if s.kind == transparent {
				continue
			}

			sa, sb := max(start, sourceX), min(column, end)
			if sa >= sb {
				continue
			}

			if mirrorX {
				for sx := sa; sx < sb; sx++ {
					x := left + int(q.Largeur) - 1 - (sx - sourceX)
					if x < x0 || x >= x1 {
						continue
					}
					put(&dst[x], src[sx], q.Teinte, neutral)
				}
				continue
			}

			da := max(left+sa-sourceX, x0)
			db := min(left+sb-sourceX, x1)
			if da >= db {
				continue
			}
			from := sourceX + da - left

			if s.kind == opaque && neutral {
				copy(dst[da:db], src[from:from+db-da])
				continue
			}
			for i := 0; i < db-da; i++ {
				put(&dst[da+i], src[from+i], q.Teinte, neutral)
			}
		}
	}
}

// fillRect peint un rectangle déjà découpé en une seule couleur.
//
// Il ne touche pas à l'atlas : c'est ce qui permet à un panneau d'interface, à une
// bande ou à un fondu de se dessiner sans qu'aucune planche existe pour eux.
func fillRect(t *rendu.Tampon, x0, y0, x1, y1 int, c rendu.Couleur) {
	if c.A == 0 {
		return
	}
	for y := y0; y < y1; y++ {
		dst := t.Pixels[y*int(t.Largeur) : (y+1)*int(t.Largeur)]
		if c.A == 255 {
			for x := x0; x < x1; x++ {
				dst[x] = c
			}
			continue
		}
		for x := x0; x < x1; x++ {
			blend(&dst[x], c)
		}
	}
}
