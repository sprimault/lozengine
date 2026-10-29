// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La mesure exacte confrontée au calcul pixel par pixel, qui est la seule référence
// qu'on puisse opposer à une manipulation de bits.

package sorting

import (
	"math/rand/v2"
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// shape rend une planche dont les pixels suivent un motif reproductible, de quoi
// distinguer une colonne d'une autre et un miroir d'un alignement.
func shape(width, height uint16, seed uint64) []render.Color {
	pixels := make([]render.Color, int(width)*int(height))
	r := rand.New(rand.NewPCG(seed, 7))
	for i := range pixels {
		switch r.IntN(3) {
		case 0: // transparent
		case 1:
			pixels[i] = render.Color{R: 255, G: 255, B: 255, A: 255}
		default:
			pixels[i] = render.Color{R: 128, G: 128, B: 128, A: 128}
		}
	}
	return pixels
}

// byPixel compte, pixel par pixel, ce que le quad de rang i peint et ce qui le couvre.
//
// **C'est la référence du lot, et elle est volontairement naïve** : un parcours par
// pixel et par quad, sans un seul décalage de bits. La mesure du moteur fait le même
// calcul par mots de soixante-quatre, et c'est l'accord des deux qui dit que les
// décalages, les miroirs et les tranches sont justes.
func byPixel(t *Sorter, i, n int) (painted, covered, byDecor int) {
	thing := t.raw[i]
	if !paints(thing) {
		return 0, 0, 0
	}
	m := t.maskOf(thing)
	if m.empty() {
		return 0, 0, 0
	}

	key := t.keys[i]
	for y := range int(thing.Height) {
		for x := range int(thing.Width) {
			if !m.at(m.painted, thing, x, y) {
				continue
			}
			painted++

			hidden, decor := false, false
			for j := range n {
				if t.uncounted[j] || j == i {
					continue
				}
				if key > t.keys[j] || (key == t.keys[j] && !t.before(uint32(i), uint32(j))) {
					continue
				}
				q := t.raw[j]
				if !hides(q) {
					continue
				}

				sx, sy := int(thing.X)+x-int(q.X), int(thing.Y)+y-int(q.Y)
				if sx < 0 || sy < 0 || sx >= int(q.Width) || sy >= int(q.Height) {
					continue
				}
				if q.Flags&render.Flat == 0 {
					qm := t.maskOf(q)
					if qm.empty() || !qm.at(qm.solid, q, sx, sy) {
						continue
					}
				}
				hidden = true
				if t.fromDecor[j] {
					decor = true
				}
			}
			if hidden {
				covered++
				if decor {
					byDecor++
				}
			}
		}
	}
	return painted, covered, byDecor
}

// TestCoverageMatchesThePixelByPixelCount confronte la mesure exacte à sa référence
// naïve sur des scènes tirées au sort.
//
// **C'est le seul cas qui puisse attraper une erreur de décalage, de miroir ou de
// tranche**, et il en faut beaucoup : un quad de soixante-cinq pixels de large, un
// miroir sur une largeur impaire, une source non alignée sur un mot sont des cas que
// trois scènes écrites à la main manqueraient. Les deux calculs n'ont en commun que
// la lecture d'un bit.
func TestCoverageMatchesThePixelByPixelCount(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))

	for round := range 200 {
		s := testScene(t, 16, 8)
		sorter := New(s)

		// Trois planches de largeurs qui exercent les bornes de mot : en dessous,
		// juste au-dessus, et à cheval.
		for i, w := range []uint16{32, 64, 65} {
			sorter.AddSheet(w, 40, shape(w, 40, uint64(i+1)))
		}

		for range 1 + r.IntN(4) {
			sheet := uint16(r.IntN(3))
			width := []uint16{32, 64, 65}[sheet]
			a := scene.Aspect{
				Sheet:   sheet,
				SourceX: uint16(r.IntN(3)),
				SourceY: uint16(r.IntN(4)),
				Width:   width - uint16(r.IntN(3)),
				Height:  uint16(8 + r.IntN(8)),
				Tint:    render.White(),
				Layer:   uint8(r.IntN(2)),
			}
			if r.IntN(4) == 0 {
				a.Flags |= render.MirrorX
			}
			if r.IntN(4) == 0 {
				a.Flags |= render.MirrorY
			}
			if r.IntN(5) == 0 {
				a.Flags |= render.Flat
			}
			if r.IntN(6) == 0 {
				a.Tint = render.Color{R: 128, G: 128, B: 128, A: 128}
			}
			x := 7 + r.Float64()*2
			y := 7 + r.Float64()*2
			e := scene.Entity{X: x, Y: y, Aspect: a, Measure: r.IntN(2) == 0}
			if _, ok := s.Place(e); !ok {
				t.Fatalf("ronde %d : entité refusée", round)
			}
		}

		sorter.Frame()

		n := len(sorter.raw)
		for i := range n {
			wp, wc, wd := byPixel(sorter, i, n)
			gp, gc, gd := sorter.coverage(i, n)
			if gp != wp || gc != wc || gd != wd {
				t.Fatalf("ronde %d, quad %d : la passe rend (%d, %d, %d), le pixel par pixel (%d, %d, %d)",
					round, i, gp, gc, gd, wp, wc, wd)
			}
		}
	}
}
