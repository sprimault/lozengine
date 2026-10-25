// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// scenePourTest monte une scène cadrée, prête à produire une image.
func scenePourTest(t *testing.T, cote, capacite int) *scene.Scene {
	t.Helper()

	s, err := scene.New(cote, cote, geometry.Dimetric(32, 16), scene.Options{Entities: capacite})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(float64(cote)/2, float64(cote)/2, 480, 270)
	return s
}

// figure rend un aspect visible de la taille d'une tuile.
func figure() scene.Aspect {
	return scene.Aspect{Width: 32, Height: 16, Tint: render.White()}
}

// TestImageOrdonneParProfondeur vérifie la propriété que tout le jalon sert : ce qui
// est plus loin se peint avant ce qui est plus près, quel que soit l'ordre de pose.
func TestImageOrdonneParProfondeur(t *testing.T) {
	s := scenePourTest(t, 16, 8)

	// Posées de la plus proche à la plus lointaine, pour que l'ordre de pose soit
	// l'inverse de l'ordre attendu.
	for _, d := range []float64{3, 2, 1} {
		a := figure()
		a.Sheet = uint16(d)
		if _, ok := s.Place(scene.Entity{X: 8 + d, Y: 8 + d, Aspect: a}); !ok {
			t.Fatal("entité refusée")
		}
	}

	image := Nouveau(s).Image()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}

	for i, attendu := range []uint16{1, 2, 3} {
		if image[i].Sheet != attendu {
			t.Errorf("rang %d : planche %d, attendu %d", i, image[i].Sheet, attendu)
		}
	}
}

// TestCeQuiSortDeLImageEstRetire vérifie que ce qui ne touche pas la vue n'entre pas
// dans la liste, et que ce qui dépasse d'un bord y entre encore.
func TestCeQuiSortDeLImageEstRetire(t *testing.T) {
	s := scenePourTest(t, 400, 8)

	proche := figure()
	proche.Sheet = 1
	s.Place(scene.Entity{X: 200, Y: 200, Aspect: proche})

	loin := figure()
	loin.Sheet = 2
	s.Place(scene.Entity{X: 10, Y: 390, Aspect: loin})

	image := Nouveau(s).Image()
	if len(image) != 1 {
		t.Fatalf("%d quads, attendu 1 — l'entité lointaine devait être écartée", len(image))
	}
	if image[0].Sheet != 1 {
		t.Errorf("planche %d, attendu celle du centre", image[0].Sheet)
	}
}

// TestIndexAttribueParProfondeur couvre le point que le brief a désigné comme le plus
// dangereux du jalon.
//
// Le champ ne porte que mille vingt-quatre valeurs. Numéroté sur toute l'image, il
// saturerait dès le millier et quelques, et la clé cesserait d'exprimer l'ordre sans
// que rien ne le dise. Numéroté par profondeur, deux mille choses réparties sur deux
// mille profondeurs reçoivent toutes le même index zéro.
func TestIndexAttribueParProfondeur(t *testing.T) {
	const n = 2000

	// Un quart de cellule d'écart donne une profondeur distincte à chacune — c'est
	// l'unité du champ — tout en les gardant assez groupées pour que la vue les
	// montre toutes — la moitié du cas serait sinon retirée comme hors de l'image.
	s := scenePourTest(t, 4096, n)
	for i := range n {
		if _, ok := s.Place(scene.Entity{X: float64(i) * 0.25, Aspect: figure()}); !ok {
			t.Fatalf("entité %d refusée", i)
		}
	}
	s.Focus(0, 0, 32767, 32767)

	trieur := Nouveau(s)
	trieur.Image()

	if len(trieur.cles) < n {
		t.Fatalf("%d clés relevées, attendu au moins %d", len(trieur.cles), n)
	}
	for i, c := range trieur.cles {
		if c.Index() != 0 {
			t.Fatalf("clé %d : index %d sur une profondeur qui lui est propre, attendu 0", i, c.Index())
		}
	}
}

// TestIndexNumeroteDansUneMemeProfondeur vérifie l'autre moitié : à profondeur égale,
// les index se suivent, et ils s'arrêtent au plafond du champ au lieu de déborder sur
// la planche.
func TestIndexNumeroteDansUneMemeProfondeur(t *testing.T) {
	const n = 40

	s := scenePourTest(t, 16, n)
	for range n {
		s.Place(scene.Entity{X: 8, Y: 8, Aspect: figure()})
	}

	trieur := Nouveau(s)
	trieur.Image()

	for i, c := range trieur.cles {
		if int(c.Index()) != i {
			t.Fatalf("clé %d : index %d, attendu %d", i, c.Index(), i)
		}
	}
}

// TestOrdreTientQuandUneTroisiemeDisparait couvre le défaut pour lequel le départage
// existe, et aucun autre cas ne peut le faire échouer.
//
// Retirer une entité comble son trou avec la dernière : l'ordre de la tranche change
// sans que personne n'ait touché aux deux autres. Si l'index stable suivait cet ordre,
// deux choses de même profondeur se relaieraient au premier plan dès qu'une troisième
// meurt ailleurs — et ce scintillement ne se relie pas à sa cause quand on le voit.
func TestOrdreTientQuandUneTroisiemeDisparait(t *testing.T) {
	s := scenePourTest(t, 32, 8)

	// Posée en premier pour que son retrait déplace les deux suivantes. Sa
	// profondeur diffère, donc elle ne se mêle pas à leur départage.
	jetable, _ := s.Place(scene.Entity{X: 20, Y: 20, Aspect: figure()})

	// Même profondeur — la somme des deux axes vaut 28 — et deux abscisses écran
	// opposées. Même aspect, sans quoi la planche trancherait avant l'index.
	s.Place(scene.Entity{X: 16, Y: 12, Aspect: figure()})
	s.Place(scene.Entity{X: 12, Y: 16, Aspect: figure()})

	trieur := Nouveau(s)

	image := trieur.Image()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}
	if image[0].X > image[1].X {
		t.Errorf("abscisses %d puis %d : les deux d'une même profondeur ne vont pas de gauche à droite", image[0].X, image[1].X)
	}
	avant := [2]int16{image[0].X, image[1].X}

	s.Remove(jetable)

	image = trieur.Image()
	if len(image) != 2 {
		t.Fatalf("%d quads après le retrait, attendu 2", len(image))
	}
	if image[0].X != avant[0] || image[1].X != avant[1] {
		t.Errorf("ordre %d, %d après le retrait, attendu %d, %d", image[0].X, image[1].X, avant[0], avant[1])
	}
}

// TestDecorReleveParFenetre vérifie que le coût du décor suit la vue et non la carte.
//
// Sans cela, une carte de cinq cents cellules de côté en projetterait deux cent
// cinquante mille par image, là où la vue n'en montre que quelques centaines.
func TestDecorReleveParFenetre(t *testing.T) {
	const cote = 200

	s := scenePourTest(t, cote, 1)
	for v := range cote {
		for u := range cote {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	image := Nouveau(s).Image()
	if len(image) == 0 {
		t.Fatal("aucune cellule relevée")
	}
	if len(image) > 2000 {
		t.Errorf("%d quads pour une vue de 480×270, la carte entière en compte %d", len(image), cote*cote)
	}
}

// TestImageNAlloueApresLaPremiere vérifie la propriété dont le budget d'une image
// dépend : les tampons se montent une fois et se réemploient.
func TestImageNAlloueApresLaPremiere(t *testing.T) {
	s := scenePourTest(t, 64, 256)
	for i := range 256 {
		s.Place(scene.Entity{X: float64(i % 64), Y: float64(i / 64), Aspect: figure()})
	}
	for v := range 64 {
		for u := range 64 {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	trieur := Nouveau(s)
	trieur.Image()

	if allocs := testing.AllocsPerRun(5, func() { trieur.Image() }); allocs != 0 {
		t.Errorf("allocations : %v, attendu aucune après la première image", allocs)
	}
}

// TestLesQuatreOrientationsRendentAutantDeQuads vérifie qu'une rotation ne perd ni
// n'invente rien : elle change ce qu'on voit d'où, pas combien il y en a.
//
// Ce cas n'a aucun consommateur réel — cohue n'a pas d'orientation —, donc il est la
// seule chose qui garde les quatre vues.
func TestLesQuatreOrientationsRendentAutantDeQuads(t *testing.T) {
	s := scenePourTest(t, 24, 1)
	for v := range 24 {
		for u := range 24 {
			s.PlaceCell(u, v, 0, figure())
		}
	}

	trieur := Nouveau(s)
	comptes := make([]int, 0, 4)

	for _, o := range []scene.Orientation{scene.NoRotation, scene.QuarterTurn, scene.HalfTurn, scene.ThreeQuarters} {
		s.Orient(o)
		comptes = append(comptes, len(trieur.Image()))
	}

	for i, n := range comptes {
		if n != comptes[0] {
			t.Errorf("quart %d : %d quads, le premier en rend %d", i, n, comptes[0])
		}
	}
}
