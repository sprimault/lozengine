// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La forme d'une planche réduite à deux plans de bits, et la passe qui compte au pixel
// ce que le recouvrement de rectangles ne sait qu'approcher.

package sorting

import (
	"math/bits"

	"github.com/sprimault/lozengine/render"
)

// mask dit, pour une planche, quels pixels elle peint et lesquels sont opaques.
//
// **Un bit par pixel et deux plans, non les pixels eux-mêmes.** Les 651 planches du
// premier jeu pèsent 22,3 Mo de couleurs ; un plan en coûte 715 Ko et les deux 1,43 Mo,
// soit 6,4 %. Lire la couleur directement éviterait ce coût mémoire, mais la mesure
// devrait alors toucher un octet par pixel là où un mot en porte soixante-quatre.
//
// **Deux plans et non un, même si le premier jeu n'a aucun pixel translucide** —
// mesuré, 0,00 % de ses pixels ont un alpha strictement entre 0 et 255. Ce qu'un quad
// peint et ce qu'il cache ne sont pas la même question dès qu'un jeu emploie de la
// translucidité, et `raster` distingue déjà les deux de son côté.
type mask struct {
	width, height uint16

	// stride est le nombre de mots par ligne. Les lignes ne se chevauchent pas dans un
	// mot : une planche de 65 pixels de large en prend deux, dont le second ne porte
	// qu'un bit utile. C'est quelques octets perdus par planche contre un décalage de
	// plus à chaque lecture de ligne.
	stride int

	// painted porte les pixels dont l'alpha n'est pas nul, solid ceux dont il vaut
	// 255. Le second est inclus dans le premier.
	painted, solid []uint64
}

// newMask dérive les deux plans d'une planche.
//
// Le parcours est celui des pixels, une fois par planche et jamais à l'image : dériver
// les deux plans des 5,5 millions de pixels du premier jeu coûte 7,5 ms, à comparer aux
// 4,5 ms que l'encodage des séquences du rastériseur prend déjà sur les mêmes données.
func newMask(width, height uint16, pixels []render.Color) mask {
	stride := (int(width) + 63) / 64
	m := mask{
		width:   width,
		height:  height,
		stride:  stride,
		painted: make([]uint64, stride*int(height)),
		solid:   make([]uint64, stride*int(height)),
	}

	for y := range int(height) {
		row := y * stride
		for x := range int(width) {
			c := pixels[y*int(width)+x]
			if c.A == 0 {
				continue
			}
			word, bit := row+x/64, uint(x%64)
			m.painted[word] |= 1 << bit
			if c.A == 255 {
				m.solid[word] |= 1 << bit
			}
		}
	}
	return m
}

// empty dit qu'aucune planche n'a été donnée pour cet index.
func (m mask) empty() bool { return m.stride == 0 }

// line rend les bits d'une ligne d'un plan, à partir d'une colonne de la planche et
// sur une largeur donnée, le bit de poids faible étant la première colonne rendue.
//
// **Les miroirs se lisent ici et nulle part ailleurs.** Un quad miroir désigne le même
// rectangle de planche et le peint à l'envers ; refléter les bits au moment de les lire
// évite de garder deux masques par planche, et le reflet d'un mot est une instruction
// sur les architectures qui l'ont.
//
// La largeur demandée ne dépasse pas soixante-quatre : c'est l'appelant qui découpe les
// quads plus larges, et ce découpage est le même pour les deux plans.
func (m mask) line(plane []uint64, q render.Quad, y, slice, width int) uint64 {
	if m.empty() {
		return 0
	}

	// **Le miroir reflète le rectangle du quad, pas la planche.** C'est la convention
	// que le rastériseur applique, et la confondre avec un reflet de la planche entière
	// décale l'image de tout ce qui sépare le rectangle du bord — sans rien casser sur
	// une planche d'une seule case, ce qui en fait un défaut qui attend sa planche
	// d'atlas pour se montrer.
	sy := int(q.SourceY) + y
	if q.Flags&render.MirrorY != 0 {
		sy = int(q.SourceY) + int(q.Height) - 1 - y
	}
	if sy < 0 || sy >= int(m.height) {
		return 0
	}

	sx := int(q.SourceX) + slice
	if q.Flags&render.MirrorX != 0 {
		sx = int(q.SourceX) + int(q.Width) - slice - width
	}

	// **Ce qui sort de la planche ne peint rien, et se tronque au lieu de refuser.**
	// Un rectangle source plus large que sa planche est une erreur de l'hôte, mais une
	// erreur qui doit rendre un quad partiellement vide et non une mesure nulle : la
	// première se voit à l'image, la seconde fait croire que rien ne recouvre.
	lo, hi := max(sx, 0), min(sx+width, int(m.width))
	if hi <= lo {
		return 0
	}

	// Une tranche de soixante-quatre colonnes chevauche au plus deux mots de la ligne.
	row := sy * m.stride
	word := row + lo/64
	got := plane[word] >> uint(lo%64)
	if shift := uint(lo % 64); shift != 0 && word+1 < row+m.stride {
		got |= plane[word+1] << (64 - shift)
	}

	if n := hi - lo; n < 64 {
		got &= 1<<uint(n) - 1
	}
	got <<= uint(lo - sx)
	if q.Flags&render.MirrorX != 0 {
		got = bits.Reverse64(got) >> uint(64-width)
	}
	return got
}

// at dit si un pixel d'un plan est posé, à une colonne et une ligne comptées dans le
// quad et non dans la planche.
//
// **Elle existe pour la référence du test et pour elle seule**, [mask.line] étant ce
// que la mesure emploie : lire un bit à la fois coûterait soixante-quatre fois plus.
// Les deux partagent la convention de miroir, donc leur accord ne la valide pas — c'est
// le rastériseur qui en est la vérité, et un cas distinct la confronte à lui.
func (m mask) at(plane []uint64, q render.Quad, x, y int) bool {
	if m.empty() {
		return false
	}

	sx, sy := int(q.SourceX)+x, int(q.SourceY)+y
	if q.Flags&render.MirrorX != 0 {
		sx = int(q.SourceX) + int(q.Width) - 1 - x
	}
	if q.Flags&render.MirrorY != 0 {
		sy = int(q.SourceY) + int(q.Height) - 1 - y
	}
	if sx < 0 || sy < 0 || sx >= int(m.width) || sy >= int(m.height) {
		return false
	}

	word := sy*m.stride + sx/64
	return word < len(plane) && plane[word]&(1<<uint(sx%64)) != 0
}

// hides dit qu'un quad cache ce qui est derrière lui, et non qu'il se contente de s'y
// ajouter ou de le teinter.
//
// Trois conditions, et chacune a son contre-exemple dans le premier jeu : un quad
// additif éclaircit sans masquer, une teinte d'alpha nul ne peint rien du tout, et une
// teinte partielle laisse voir au travers. Un quad plat vaut son rectangle entier sous
// ces mêmes conditions, n'ayant pas de source.
func hides(q render.Quad) bool {
	return q.Tint.A == 255 && q.Flags&render.Additive == 0
}

// paints dit qu'un quad peint quelque chose, ce qu'une teinte d'alpha nul défait quelle
// que soit sa source.
func paints(q render.Quad) bool { return q.Tint.A != 0 }

// coverage compte, au pixel, ce que le quad de rang i peint et ce que les quads qui se
// peignent après lui en couvrent.
//
// **Chaque pixel ne compte qu'une fois**, là où [Sorter.overlap] additionne les
// rectangles de chaque couple : c'est toute la différence, et c'est elle qui empêche la
// mesure exacte de remplacer l'ancienne. Sur la charge du premier jeu, la même scène
// rend 5 534 en boîte, 144 en somme par paire et 63 en union.
//
// Les quads plus larges que soixante-quatre pixels se traitent par tranches de cette
// largeur, l'accumulateur portant une ligne de la chose à la fois.
func (t *Sorter) coverage(i, n int) (painted, covered, byDecor int) {
	thing := t.raw[i]
	if !paints(thing) {
		return 0, 0, 0
	}
	m := t.maskOf(thing)
	if m.empty() {
		return 0, 0, 0
	}

	// **Les candidats se trouvent une fois, pas une fois par ligne**, et c'est la
	// différence entre un lot qui tient le budget et un qui le mange. Chercher les
	// cachants à l'intérieur de la boucle des lignes coûtait 38,7 µs par entité
	// marquée — mesuré, 2,7 ms à soixante-cinq marquées, un sixième du budget d'une
	// image — parce que chaque ligne reparcourait les deux mille quads de la scène. Il
	// n'y en a qu'une quinzaine dont la boîte recouvre celle de la chose.
	t.candidates = t.candidates[:0]
	key := t.keys[i]
	for j := range n {
		if t.uncounted[j] || j == i {
			continue
		}
		if key > t.keys[j] || (key == t.keys[j] && !t.before(uint32(i), uint32(j))) {
			continue
		}
		if !hides(t.raw[j]) || shared(thing, t.raw[j]) == 0 {
			continue
		}
		t.candidates = append(t.candidates, uint32(j))
	}

	for slice := 0; slice < int(thing.Width); slice += 64 {
		width := min(64, int(thing.Width)-slice)

		for y := range int(thing.Height) {
			own := m.line(m.painted, thing, y, slice, width)
			if own == 0 {
				continue
			}
			painted += bits.OnesCount64(own)
			if len(t.candidates) == 0 {
				continue
			}

			var all, decor uint64
			for _, j := range t.candidates {
				hit := t.solidLine(int(j), int(thing.X)+slice, int(thing.Y)+y, width)
				all |= hit
				if t.fromDecor[j] {
					decor |= hit
				}
			}

			covered += bits.OnesCount64(own & all)
			byDecor += bits.OnesCount64(own & decor)
		}
	}
	return painted, covered, byDecor
}

// solidLine rend les colonnes opaques que le quad de rang j pose sur une ligne de
// l'écran, à partir d'une abscisse d'écran et sur une largeur donnée.
//
// Zéro quand la ligne tombe hors du quad, quand le quad ne cache pas, ou quand sa
// planche n'a pas de masque : dans ce dernier cas la mesure exacte n'a rien à dire, et
// c'est [Sorter.overlap] qui garde la réponse approchée.
func (t *Sorter) solidLine(j, screenX, screenY, width int) uint64 {
	q := t.raw[j]
	if !hides(q) {
		return 0
	}

	top := int(q.Y)
	if screenY < top || screenY >= top+int(q.Height) {
		return 0
	}

	// Les colonnes du quad qui tombent dans la fenêtre demandée, comptées dans le
	// quad : au plus la largeur de la fenêtre, donc une seule lecture de ligne.
	from := max(0, screenX-int(q.X))
	to := min(int(q.Width), screenX+width-int(q.X))
	if to <= from {
		return 0
	}

	var line uint64
	if q.Flags&render.Flat != 0 {
		line = 1<<uint(to-from) - 1
	} else {
		m := t.maskOf(q)
		if m.empty() {
			return 0
		}
		line = m.line(m.solid, q, screenY-top, from, to-from)
	}

	// Et son alignement sur la fenêtre, dont le bit de poids faible est screenX.
	return line << uint(int(q.X)+from-screenX)
}

// maskOf rend le masque de la planche d'un quad, vide si elle n'en a pas.
func (t *Sorter) maskOf(q render.Quad) mask {
	if int(q.Sheet) >= len(t.masks) {
		return mask{}
	}
	return t.masks[q.Sheet]
}
