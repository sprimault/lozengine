// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la scène refuse et quand elle le dit, ce qu'une poignée désigne encore après
// un retrait, et la propriété dont le budget d'une image dépend : rien n'y alloue.

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
)

// testScene monte une petite scène, la capacité d'entités étant le seul réglage
// que les cas font varier.
func testScene(t *testing.T, capacity int) *Scene {
	t.Helper()

	s, err := New(8, 8, geometry.Dimetric(32, 16), Options{Entities: capacity})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	return s
}

// solid rend un aspect visible, la valeur zéro d'un aspect ne montrant rien.
func solid() Aspect {
	return Aspect{Width: 32, Height: 16, Tint: render.White()}
}

// TestNewRefuses vérifie que ce qui ne peut pas marcher se dit au montage, le seul
// endroit où une scène a quelqu'un à qui le dire.
func TestNewRefuses(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		o             Options
	}{
		{"largeur nulle", 0, 8, Options{}},
		{"hauteur négative", 8, -1, Options{}},
		{"au-delà de ce que la clé ordonne", render.MaxSide + 1, 8, Options{}},
		{"capacité au-delà du champ de rang", 8, 8, Options{Entities: maxRank + 1}},
		{"capacité négative", 8, 8, Options{Entities: -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := New(c.width, c.height, geometry.Dimetric(32, 16), c.o); err == nil {
				t.Error("scène acceptée")
			}
		})
	}
}

// TestNewAcceptsTheLimit vérifie que la garde s'arrête là où la clé de tri cesse
// de distinguer, et pas une cellule avant.
func TestNewAcceptsTheLimit(t *testing.T) {
	s, err := New(render.MaxSide, 1, geometry.Dimetric(32, 16), Options{Entities: 1})
	if err != nil {
		t.Fatalf("grille de %d de côté refusée : %v", render.MaxSide, err)
	}

	if l, h := s.Dimensions(); l != render.MaxSide || h != 1 {
		t.Errorf("dimensions %d×%d, attendu %d×1", l, h, render.MaxSide)
	}
}

// TestRemoveKeepsOtherHandles couvre le piège pour lequel les poignées
// existent.
//
// Retirer comble le trou avec la dernière entité, donc une entité que personne n'a
// touchée change de place. Sans la table qui suit ce déplacement, sa poignée
// désignerait une place morte : le moteur rendrait une image irréprochable pour une
// scène fausse, et la cause serait la disparition d'une troisième entité ailleurs.
// Aucun autre cas ne peut faire échouer celui-là.
func TestRemoveKeepsOtherHandles(t *testing.T) {
	s := testScene(t, 4)

	first, _ := s.Place(Entity{X: 1, Aspect: solid()})
	middle, _ := s.Place(Entity{X: 2, Aspect: solid()})
	last, _ := s.Place(Entity{X: 3, Aspect: solid()})

	s.Remove(middle)

	if !s.Move(first, 10, 0, 0) {
		t.Fatal("la première ne se désigne plus")
	}
	if !s.Move(last, 30, 0, 0) {
		t.Fatal("la dernière ne se désigne plus après avoir comblé le trou")
	}

	seen := map[float64]bool{}
	for _, e := range Entities(s) {
		seen[e.X] = true
	}
	if len(seen) != 2 || !seen[10] || !seen[30] {
		t.Errorf("positions %v, attendu 10 et 30", seen)
	}
}

// TestStaleHandle vérifie les trois façons qu'une poignée a de ne plus rien
// désigner, dont la plus traître : son rang a été rendu puis repris par une autre
// entité.
func TestStaleHandle(t *testing.T) {
	s := testScene(t, 4)

	alone, _ := s.Place(Entity{X: 1, Aspect: solid()})
	s.Remove(alone)

	if s.Move(alone, 5, 0, 0) {
		t.Error("une poignée retirée désigne encore")
	}

	reused, _ := s.Place(Entity{X: 2, Aspect: solid()})
	if reused == alone {
		t.Error("le rang repris rend la même poignée, donc la génération n'avance pas")
	}
	if s.Move(alone, 5, 0, 0) {
		t.Error("l'ancienne poignée désigne l'entité qui a repris son rang")
	}

	if s.Move(0, 5, 0, 0) {
		t.Error("la poignée vide désigne quelque chose")
	}
}

// TestPlaceRefusesWhenFull vérifie qu'une capacité atteinte se refuse au lieu de faire
// croître la tranche, ce qui ferait allouer en pleine partie.
func TestPlaceRefusesWhenFull(t *testing.T) {
	s := testScene(t, 2)

	for i := 0; i < 2; i++ {
		if _, ok := s.Place(Entity{Aspect: solid()}); !ok {
			t.Fatalf("entité %d refusée sous la capacité", i)
		}
	}

	if _, ok := s.Place(Entity{Aspect: solid()}); ok {
		t.Error("entité acceptée au-delà de la capacité")
	}
	if n := len(Entities(s)); n != 2 {
		t.Errorf("%d entités, attendu 2", n)
	}
}

// TestPlaceRefusesOutsideKeyBounds couvre le seul défaut de cette surface qui ne se
// verrait jamais : la composition de la clé ramène une couche ou un étage trop grand au
// plafond, donc un jeu qui en demande vingt en verrait six se confondre avec la
// quinzième, sur une image par ailleurs irréprochable.
//
// Les quatre portes sont éprouvées parce qu'elles ont quatre contrôles distincts, et
// qu'un aspect entre par n'importe laquelle. Chacune rend déjà un booléen : refuser ne
// coûte donc aucune signature.
func TestPlaceRefusesOutsideKeyBounds(t *testing.T) {
	outOfBounds := []struct {
		name   string
		aspect Aspect
	}{
		{"couche", Aspect{Width: 32, Height: 16, Tint: render.White(), Layer: render.MaxLayer + 1}},
		{"étage", Aspect{Width: 32, Height: 16, Tint: render.White(), Floor: render.MaxFloor + 1}},
	}
	for _, c := range outOfBounds {
		t.Run(c.name, func(t *testing.T) {
			s := testScene(t, 4)

			if _, ok := s.Place(Entity{Aspect: c.aspect}); ok {
				t.Error("Place a accepté")
			}
			if n := len(Entities(s)); n != 0 {
				t.Errorf("%d entités posées, attendu aucune", n)
			}

			p, ok := s.Place(Entity{Aspect: solid()})
			if !ok {
				t.Fatal("entité valide refusée")
			}
			if s.SetAspect(p, c.aspect) {
				t.Error("SetAspect a accepté")
			}
			if s.PlaceCell(1, 1, 0, c.aspect) {
				t.Error("PlaceCell a accepté")
			}
			if s.PlaceScreen(0, 0, c.aspect) {
				t.Error("PlaceScreen a accepté")
			}
			if n := len(ScreenPieces(s)); n != 0 {
				t.Errorf("%d pièces d'écran posées, attendu aucune", n)
			}
		})
	}
}

// TestPlaceAcceptsTheBounds vérifie que le refus s'arrête à la borne au lieu de la
// manger. Un hors-par-un ici retirerait la seizième couche et le soixante-quatrième
// étage, que le moteur annonce pourtant, et aucun autre cas ne le verrait.
func TestPlaceAcceptsTheBounds(t *testing.T) {
	s := testScene(t, 2)
	extremes := Aspect{Width: 32, Height: 16, Tint: render.White(), Layer: render.MaxLayer, Floor: render.MaxFloor}

	if _, ok := s.Place(Entity{Aspect: extremes}); !ok {
		t.Error("Place a refusé la borne")
	}
	if !s.PlaceCell(1, 1, 0, extremes) {
		t.Error("PlaceCell a refusé la borne")
	}
	if !s.PlaceScreen(0, 0, extremes) {
		t.Error("PlaceScreen a refusé la borne")
	}
}

// TestSetAspectKeepsThePosition vérifie que changer ce qu'une entité montre la
// laisse où elle est : c'est le chemin d'un changement de direction, qui touche des
// centaines d'entités sur une même image.
func TestSetAspectKeepsThePosition(t *testing.T) {
	s := testScene(t, 2)

	p, _ := s.Place(Entity{X: 3, Y: 5, Z: 1, Aspect: solid()})
	if !s.SetAspect(p, Aspect{Width: 8, Height: 8, Sheet: 7, Tint: render.White()}) {
		t.Fatal("habillage refusé")
	}

	e := Entities(s)[0]
	if e.X != 3 || e.Y != 5 || e.Z != 1 {
		t.Errorf("position (%g, %g, %g), attendu (3, 5, 1)", e.X, e.Y, e.Z)
	}
	if e.Sheet != 7 || e.Width != 8 {
		t.Errorf("aspect planche %d largeur %d, attendu 7 et 8", e.Sheet, e.Width)
	}
}

// TestDecorByTier vérifie qu'une cellule empile sans effacer, et qu'une strate
// n'existe qu'une fois qu'on y a posé quelque chose.
//
// C'est le besoin qui a donné les strates : garer un véhicule sur une chaussée sans
// perdre la chaussée, ni le passage piéton peint dessus. Un moteur qui n'offre qu'une
// chose par cellule oblige le jeu à choisir laquelle des trois il garde.
func TestDecorByTier(t *testing.T) {
	s := testScene(t, 1)

	if n := s.Tiers(); n != 0 {
		t.Errorf("%d strates sur une scène neuve, attendu aucune", n)
	}

	ground := solid()
	ground.Sheet = 1
	above := solid()
	above.Sheet = 2

	if !s.PlaceCell(2, 3, 0, ground) || !s.PlaceCell(2, 3, 3, above) {
		t.Fatal("pose refusée dans les bornes")
	}

	if got := s.Cell(2, 3, 0).Sheet; got != 1 {
		t.Errorf("strate 0 : planche %d, attendu 1 — la pose du dessus l'a effacée", got)
	}
	if got := s.Cell(2, 3, 3).Sheet; got != 2 {
		t.Errorf("strate 3 : planche %d, attendu 2", got)
	}
	if got := s.Cell(2, 3, 1); got != (Aspect{}) {
		t.Errorf("strate intermédiaire non posée : %v, attendu rien", got)
	}
	if n := s.Tiers(); n != 4 {
		t.Errorf("%d strates, attendu 4 — les rangs jusqu'à la plus haute posée", n)
	}
}

// TestCellOutOfBounds vérifie qu'un relevé hors grille rend le vide au lieu de
// paniquer : le tri relève une fenêtre qui dépasse les bords, et c'est voulu — une
// forme dont le sommet vient de sortir a encore son corps à l'image.
func TestCellOutOfBounds(t *testing.T) {
	s := testScene(t, 1)
	s.PlaceCell(0, 0, 0, solid())

	for _, c := range [][2]int{{-1, 0}, {0, -1}, {8, 0}, {0, 8}, {-5, 99}} {
		if got := s.Cell(c[0], c[1], 0); got != (Aspect{}) {
			t.Errorf("cellule (%d, %d) rend %v, attendu rien", c[0], c[1], got)
		}
	}
	if s.PlaceCell(8, 0, 0, solid()) {
		t.Error("pose acceptée hors grille")
	}
	if s.PlaceCell(0, 0, MaxTiers, solid()) {
		t.Error("pose acceptée au-delà des strates qu'une cellule empile")
	}
}

// TestFramePathDoesNotAllocate vérifie la propriété dont tout le reste dépend : ce
// qu'un jeu fait à chaque image — bouger ses entités, les rhabiller, les relire — ne
// touche pas au tas.
func TestFramePathDoesNotAllocate(t *testing.T) {
	s := testScene(t, 64)

	handles := make([]Handle, 0, 64)
	for i := 0; i < 64; i++ {
		p, _ := s.Place(Entity{X: float64(i), Aspect: solid()})
		handles = append(handles, p)
	}

	frame := func() {
		for i, p := range handles {
			s.Move(p, float64(i)+0.5, 0, 0)
			s.SetAspect(p, solid())
		}
		for _, e := range Entities(s) {
			_ = e.X
		}
	}

	if n := testing.AllocsPerRun(10, frame); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
