// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le dessin d'une liste de quads : pour chacun, le découpage à l'image puis la recopie
// de ses pixels. C'est le chemin le plus chaud du moteur, et le seul que la suite de
// référence juge au pixel près.

package raster

import "github.com/sprimault/lozengine/render"

// Draw peint les quads dans l'ordre du tableau, sans rien allouer.
//
// L'ordre est la seule vérité du résultat : le rastériseur ne trie pas, ne projette
// pas et ne consulte aucune scène. Un quad qui désigne une planche absente de
// l'atlas est une erreur de programmation en amont ; l'index déborde et le
// programme s'arrête, ce qui vaut mieux qu'une image muette dont on chercherait la
// cause dans le tri.
//
// Une fonction et non une méthode : le tampon vit dans [render], que l'ABI expose, et
// Go veut qu'une méthode soit déclarée dans le paquet de son type.
func Draw(t *render.Buffer, r render.DrawList, a *Atlas) {
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
func quad(t *render.Buffer, q render.Quad, a *Atlas) {
	left, top := int(q.X), int(q.Y)
	x0, y0 := max(left, 0), max(top, 0)
	x1 := min(left+int(q.Width), int(t.Width))
	y1 := min(top+int(q.Height), int(t.Height))
	if x0 >= x1 || y0 >= y1 {
		return
	}

	if q.Flags&render.Flat != 0 {
		fillRect(t, x0, y0, x1, y1, q.Tint)
		return
	}

	p := &a.Sheets[q.Sheet]
	neutral := q.Tint == render.White()
	mirrorX := q.Flags&render.MirrorX != 0
	sourceX, end := int(q.SourceX), int(q.SourceX)+int(q.Width)

	for y := y0; y < y1; y++ {
		row := y - top
		if q.Flags&render.MirrorY != 0 {
			row = int(q.Height) - 1 - row
		}
		source := int(q.SourceY) + row
		if source >= int(p.Height) {
			continue
		}

		dst := t.Pixels[y*int(t.Width) : (y+1)*int(t.Width)]
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
					x := left + int(q.Width) - 1 - (sx - sourceX)
					if x < x0 || x >= x1 {
						continue
					}
					put(&dst[x], src[sx], q.Tint, neutral)
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
				put(&dst[da+i], src[from+i], q.Tint, neutral)
			}
		}
	}
}

// fillRect peint un rectangle déjà découpé en une seule couleur.
//
// Il ne touche pas à l'atlas : c'est ce qui permet à un panneau d'interface, à une
// bande ou à un fondu de se dessiner sans qu'aucune planche existe pour eux.
func fillRect(t *render.Buffer, x0, y0, x1, y1 int, c render.Color) {
	if c.A == 0 {
		return
	}
	for y := y0; y < y1; y++ {
		dst := t.Pixels[y*int(t.Width) : (y+1)*int(t.Width)]
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
