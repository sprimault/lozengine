// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qui doit tenir d'une image à la suivante : l'ordre par profondeur, le retrait de
// ce qui ne touche pas la vue, et l'index de départage qui ne bouge pas quand une
// entité voisine disparaît — ce dernier défaut ne se voit qu'à l'image, où il
// ressemble à un défaut de tri.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// testScene monte une scène cadrée, prête à produire une image.
func testScene(t *testing.T, side, capacity int) *scene.Scene {
	t.Helper()

	s, err := scene.New(side, side, geometry.Dimetric(32, 16), scene.Options{Entities: capacity})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(float64(side)/2, float64(side)/2, 480, 270)
	return s
}

// figure rend un aspect visible de la taille d'une tuile.
func figure() scene.Aspect {
	return scene.Aspect{Width: 32, Height: 16, Tint: render.White()}
}

// TestCornerDepthMatchesProjection garde le raccourci par lequel le décor obtient la
// profondeur de son coin sans reprojeter.
//
// Le collecteur retranche [halfCellQuarters] à la profondeur du centre au lieu
// d'appeler la projection une seconde fois, ce qui coûtait huit pour cent du décor.
// L'égalité est exacte par construction — un demi en virgule fixe vaut la moitié du
// pas de quantification, et deux demis font exactement quatre quarts de cellule —,
// mais elle dépend de constantes que `geometry` n'exporte pas. Ce cas les relie : si
// la quantification changeait, le raccourci deviendrait faux en silence, et c'est le
// genre de faux qui se lit comme un défaut d'ordre.
//
// La vérification porte sur des cellules entières et sur des bords, le repli à zéro
// du coin de la première cellule compris.
func TestCornerDepthMatchesProjection(t *testing.T) {
	m := geometry.Dimetric(32, 16)

	for _, c := range [][2]float64{
		{0, 0}, {0, 1}, {1, 0}, {1, 1}, {3, 7}, {12, 5}, {99, 99}, {500, 500},
	} {
		cx, cy := c[0]+0.5, c[1]+0.5

		_, _, center, _ := m.Project(cx, cy, 0)
		_, _, corner, _ := m.Project(cx-0.5, cy-0.5, 0)

		if got := max(center, halfCellQuarters) - halfCellQuarters; got != corner {
			t.Errorf("cellule (%g, %g) : le raccourci rend %d, la projection du coin %d",
				c[0], c[1], got, corner)
		}
	}
}

// TestAnchorIsTheFootOfTheImage garde les trois faits que la godoc de [scene.Aspect]
// publie de l'ancre : le coin haut-gauche à (−largeur/2, −hauteur), la dernière ligne
// peinte une ligne au-dessus du point projeté, et le penchant d'un pixel vers la
// gauche d'une largeur paire.
//
// Le cas ne peut pas rougir sur le code d'aujourd'hui, et il n'en a pas l'objet : il
// n'y a pas de défaut à attraper, il y a une promesse publiée à tenir. Un jeu qui
// place une barre de vie au-dessus d'un personnage dérive sa position de ces trois
// faits, et les changer le décalerait sans qu'aucune image de référence ne le dise —
// `core` et `ground` emploient tous deux une largeur paire.
//
// Les deux largeurs sont voisines pour que la parité soit la seule différence.
func TestAnchorIsTheFootOfTheImage(t *testing.T) {
	const height = 24

	for _, c := range []struct {
		width       uint16
		left, right int32
	}{
		{10, 5, 4},
		{11, 5, 5},
	} {
		s := testScene(t, 16, 2)

		a := figure()
		a.Width, a.Height = c.width, height
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: a}); !ok {
			t.Fatal("entité refusée")
		}

		sorter := New(s)
		image := sorter.Frame()
		if len(image) != 1 {
			t.Fatalf("largeur %d : %d quads, attendu 1", c.width, len(image))
		}

		rx, ry := s.Remap(8.5, 8.5)
		px, py, _, _ := s.Projection().Project(rx, ry, 0)
		px, py = px-sorter.viewX, py-sorter.viewY

		q := image[0]
		if int32(q.X) != px-int32(c.width)/2 || int32(q.Y) != py-height {
			t.Errorf("largeur %d : coin en (%d, %d), attendu (%d, %d)",
				c.width, q.X, q.Y, px-int32(c.width)/2, py-height)
		}

		if last := int32(q.Y) + height - 1; last != py-1 {
			t.Errorf("largeur %d : dernière ligne peinte en %d, attendu %d",
				c.width, last, py-1)
		}

		left, right := px-int32(q.X), int32(q.X)+int32(c.width)-1-px
		if left != c.left || right != c.right {
			t.Errorf("largeur %d : %d colonnes à gauche et %d à droite, attendu %d et %d",
				c.width, left, right, c.left, c.right)
		}
	}
}

// TestFrameOrdersByDepth vérifie la propriété que tout le jalon sert : ce qui
// est plus loin se peint avant ce qui est plus près, quel que soit l'ordre de pose.
func TestFrameOrdersByDepth(t *testing.T) {
	s := testScene(t, 16, 8)

	// Posées de la plus proche à la plus lointaine, pour que l'ordre de pose soit
	// l'inverse de l'ordre attendu.
	for _, d := range []float64{3, 2, 1} {
		a := figure()
		a.Sheet = uint16(d)
		if _, ok := s.Place(scene.Entity{X: 8 + d, Y: 8 + d, Aspect: a}); !ok {
			t.Fatal("entité refusée")
		}
	}

	image := New(s).Frame()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}

	for i, want := range []uint16{1, 2, 3} {
		if image[i].Sheet != want {
			t.Errorf("rang %d : planche %d, attendu %d", i, image[i].Sheet, want)
		}
	}
}

// TestOffScreenIsRemoved vérifie que ce qui ne touche pas la vue n'entre pas
// dans la liste, et que ce qui dépasse d'un bord y entre encore.
func TestOffScreenIsRemoved(t *testing.T) {
	s := testScene(t, 400, 8)

	near := figure()
	near.Sheet = 1
	s.Place(scene.Entity{X: 200, Y: 200, Aspect: near})

	far := figure()
	far.Sheet = 2
	s.Place(scene.Entity{X: 10, Y: 390, Aspect: far})

	image := New(s).Frame()
	if len(image) != 1 {
		t.Fatalf("%d quads, attendu 1 — l'entité lointaine devait être écartée", len(image))
	}
	if image[0].Sheet != 1 {
		t.Errorf("planche %d, attendu celle du centre", image[0].Sheet)
	}
}

// TestIndexAssignedPerDepth couvre le point que le brief a désigné comme le plus
// dangereux du jalon.
//
// Le champ ne porte que mille vingt-quatre valeurs. Numéroté sur toute l'image, il
// saturerait dès le millier et quelques, et la clé cesserait d'exprimer l'ordre sans
// que rien ne le dise. Numéroté par profondeur, deux mille choses réparties sur deux
// mille profondeurs reçoivent toutes le même index zéro.
func TestIndexAssignedPerDepth(t *testing.T) {
	const n = 2000

	// Un quart de cellule d'écart donne une profondeur distincte à chacune — c'est
	// l'unité du champ — tout en les gardant assez groupées pour que la vue les
	// montre toutes — la moitié du cas serait sinon retirée comme hors de l'image.
	s := testScene(t, 4096, n)
	for i := range n {
		if _, ok := s.Place(scene.Entity{X: float64(i) * 0.25, Aspect: figure()}); !ok {
			t.Fatalf("entité %d refusée", i)
		}
	}
	s.Focus(0, 0, 32767, 32767)

	sorter := New(s)
	sorter.Frame()

	if len(sorter.keys) < n {
		t.Fatalf("%d clés relevées, attendu au moins %d", len(sorter.keys), n)
	}
	for i, c := range sorter.keys {
		if c.Index() != 0 {
			t.Fatalf("clé %d : index %d sur une profondeur qui lui est propre, attendu 0", i, c.Index())
		}
	}
}

// TestIndexNumbersWithinOneDepth vérifie l'autre moitié : à profondeur égale,
// les index se suivent, et ils s'arrêtent au plafond du champ au lieu de déborder sur
// la planche.
func TestIndexNumbersWithinOneDepth(t *testing.T) {
	const n = 40

	s := testScene(t, 16, n)
	for range n {
		s.Place(scene.Entity{X: 8, Y: 8, Aspect: figure()})
	}

	sorter := New(s)
	sorter.Frame()

	for i, c := range sorter.keys {
		if int(c.Index()) != i {
			t.Fatalf("clé %d : index %d, attendu %d", i, c.Index(), i)
		}
	}
}

// TestOrderHoldsWhenAThirdDisappears couvre le défaut pour lequel le départage
// existe, et aucun autre cas ne peut le faire échouer.
//
// Retirer une entité comble son trou avec la dernière : l'ordre de la tranche change
// sans que personne n'ait touché aux deux autres. Si l'index stable suivait cet ordre,
// deux choses de même profondeur se relaieraient au premier plan dès qu'une troisième
// meurt ailleurs — et ce scintillement ne se relie pas à sa cause quand on le voit.
func TestOrderHoldsWhenAThirdDisappears(t *testing.T) {
	s := testScene(t, 32, 8)

	// Posée en premier pour que son retrait déplace les deux suivantes. Sa
	// profondeur diffère, donc elle ne se mêle pas à leur départage.
	toRemove, _ := s.Place(scene.Entity{X: 20, Y: 20, Aspect: figure()})

	// Même profondeur — la somme des deux axes vaut 28 — et deux abscisses écran
	// opposées. Même aspect, sans quoi la planche trancherait avant l'index.
	s.Place(scene.Entity{X: 16, Y: 12, Aspect: figure()})
	s.Place(scene.Entity{X: 12, Y: 16, Aspect: figure()})

	sorter := New(s)

	image := sorter.Frame()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}
	if image[0].X > image[1].X {
		t.Errorf("abscisses %d puis %d : les deux d'une même profondeur ne vont pas de gauche à droite", image[0].X, image[1].X)
	}
	previous := [2]int16{image[0].X, image[1].X}

	s.Remove(toRemove)

	image = sorter.Frame()
	if len(image) != 2 {
		t.Fatalf("%d quads après le retrait, attendu 2", len(image))
	}
	if image[0].X != previous[0] || image[1].X != previous[1] {
		t.Errorf("ordre %d, %d après le retrait, attendu %d, %d", image[0].X, image[1].X, previous[0], previous[1])
	}
}

// TestDecorCollectedByWindow vérifie que le coût du décor suit la vue et non la carte.
//
// Sans cela, une carte de cinq cents cellules de côté en projetterait deux cent
// cinquante mille par image, là où la vue n'en montre que quelques centaines.
func TestDecorCollectedByWindow(t *testing.T) {
	const side = 200

	s := testScene(t, side, 1)
	for v := range side {
		for u := range side {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	image := New(s).Frame()
	if len(image) == 0 {
		t.Fatal("aucune cellule relevée")
	}
	if len(image) > 2000 {
		t.Errorf("%d quads pour une vue de 480×270, la carte entière en compte %d", len(image), side*side)
	}
}

// TestFrameDoesNotAllocateAfterTheFirst vérifie la propriété dont le budget d'une image
// dépend : les tampons se montent une fois et se réemploient.
func TestFrameDoesNotAllocateAfterTheFirst(t *testing.T) {
	s := testScene(t, 64, 256)
	for i := range 256 {
		s.Place(scene.Entity{X: float64(i % 64), Y: float64(i / 64), Aspect: figure()})
	}
	for v := range 64 {
		for u := range 64 {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	sorter := New(s)
	sorter.Frame()

	if allocs := testing.AllocsPerRun(5, func() { sorter.Frame() }); allocs != 0 {
		t.Errorf("allocations : %v, attendu aucune après la première image", allocs)
	}
}

// TestFourOrientationsYieldAsManyQuads vérifie qu'une rotation ne perd ni
// n'invente rien : elle change ce qu'on voit d'où, pas combien il y en a.
//
// Ce cas n'a aucun consommateur réel — cohue n'a pas d'orientation —, donc il est la
// seule chose qui garde les quatre vues.
func TestFourOrientationsYieldAsManyQuads(t *testing.T) {
	s := testScene(t, 24, 1)
	for v := range 24 {
		for u := range 24 {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	sorter := New(s)
	counts := make([]int, 0, 4)

	for _, o := range []scene.Orientation{scene.NoRotation, scene.QuarterTurn, scene.HalfTurn, scene.ThreeQuarters} {
		s.Orient(o)
		counts = append(counts, len(sorter.Frame()))
	}

	for i, n := range counts {
		if n != counts[0] {
			t.Errorf("quart %d : %d quads, le premier en rend %d", i, n, counts[0])
		}
	}
}
