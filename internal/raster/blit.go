// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "github.com/sprimault/lozengine/internal/rendu"

// Dessiner peint les quads dans l'ordre du tableau, sans rien allouer.
//
// L'ordre est la seule vérité du résultat : le rastériseur ne trie pas, ne projette
// pas et ne consulte aucune scène. Un quad qui désigne une planche absente de
// l'atlas est une erreur de programmation en amont ; l'index déborde et le
// programme s'arrête, ce qui vaut mieux qu'une image muette dont on chercherait la
// cause dans le tri.
func (t *Tampon) Dessiner(r rendu.Rendu, a *Atlas) {
	for _, q := range r {
		t.quad(q, a)
	}
}

// quad découpe un quad sur les bords du tampon, puis le peint.
//
// Le découpage se fait sur la destination et se reporte sur la source, de sorte
// qu'un sprite à moitié sorti de l'écran coûte la moitié de son travail et non un
// test par pixel. La source, elle, n'est pas vérifiée : un rectangle qui dépasse la
// planche dessine ce qu'il trouve et s'arrête à la dernière ligne, plutôt que de
// lire la ligne suivante.
func (t *Tampon) quad(q rendu.Quad, a *Atlas) {
	gauche, haut := int(q.X), int(q.Y)
	x0, y0 := max(gauche, 0), max(haut, 0)
	x1 := min(gauche+int(q.Largeur), int(t.Largeur))
	y1 := min(haut+int(q.Hauteur), int(t.Hauteur))
	if x0 >= x1 || y0 >= y1 {
		return
	}

	if q.Drapeaux&rendu.Aplat != 0 {
		t.aplat(x0, y0, x1, y1, q.Teinte)
		return
	}

	p := &a.Planches[q.Planche]
	neutre := q.Teinte == rendu.Blanc()
	miroirX := q.Drapeaux&rendu.MiroirX != 0
	sourceX, fin := int(q.SourceX), int(q.SourceX)+int(q.Largeur)

	for y := y0; y < y1; y++ {
		ligne := y - haut
		if q.Drapeaux&rendu.MiroirY != 0 {
			ligne = int(q.Hauteur) - 1 - ligne
		}
		source := int(q.SourceY) + ligne
		if source >= int(p.Hauteur) {
			continue
		}

		dst := t.Pixels[y*int(t.Largeur) : (y+1)*int(t.Largeur)]
		src := p.Pixels[source*int(p.Largeur) : (source+1)*int(p.Largeur)]

		colonne := 0
		for _, s := range p.sequencesDe(uint16(source)) {
			debut := colonne
			colonne += int(s.nombre)
			if s.genre == saut {
				continue
			}

			sa, sb := max(debut, sourceX), min(colonne, fin)
			if sa >= sb {
				continue
			}

			if miroirX {
				for sx := sa; sx < sb; sx++ {
					x := gauche + int(q.Largeur) - 1 - (sx - sourceX)
					if x < x0 || x >= x1 {
						continue
					}
					poser(&dst[x], src[sx], q.Teinte, neutre)
				}
				continue
			}

			da := max(gauche+sa-sourceX, x0)
			db := min(gauche+sb-sourceX, x1)
			if da >= db {
				continue
			}
			depart := sourceX + da - gauche

			if s.genre == copie && neutre {
				copy(dst[da:db], src[depart:depart+db-da])
				continue
			}
			for i := 0; i < db-da; i++ {
				poser(&dst[da+i], src[depart+i], q.Teinte, neutre)
			}
		}
	}
}

// aplat peint un rectangle déjà découpé en une seule couleur.
//
// Il ne touche pas à l'atlas : c'est ce qui permet à un panneau d'interface, à une
// bande ou à un fondu de se dessiner sans qu'aucune planche existe pour eux.
func (t *Tampon) aplat(x0, y0, x1, y1 int, c rendu.Couleur) {
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
			melanger(&dst[x], c)
		}
	}
}
