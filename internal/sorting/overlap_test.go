// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le relevé de recouvrement doit tenir : un compte exact et non un verdict, la
// poignée de l'entité qui l'a demandé, et rien pour celles qui n'ont rien demandé.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// measuredScene pose une entité marquée et, par `cover`, ce qui la recouvre.
//
// Les positions sont celles de [revealScene] : la seconde est plus profonde d'une
// demi-cellule sur chaque axe, donc elle se peint après et son quad mord sur le haut du
// premier.
func measuredScene(t *testing.T, cover func(*scene.Scene)) (*scene.Scene, scene.Handle) {
	t.Helper()

	s := testScene(t, 16, 4)

	h, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: figure(), Measure: true})
	if !ok {
		t.Fatal("entité suivie refusée")
	}
	if cover != nil {
		cover(s)
	}
	return s, h
}

// TestOverlapReportsAnExactCount garde ce qui distingue le relevé de la révélation : un
// nombre de pixels, et non le verdict d'un seuil.
//
// **C'est le seul cas à pouvoir faire échouer cette distinction.** Les cas de révélation
// ne peuvent constater qu'un quad de plus ou son absence, donc ils passeraient tous avec
// une mesure arrêtée dès le seuil franchi — ce que la mesure faisait avant ce lot. Ici le
// compte attendu est celui qu'[overlap] rend sur les deux quads, calculé à part par la
// même mesure élémentaire que [TestOverlapCountsSharedPixels] éprouve seule.
func TestOverlapReportsAnExactCount(t *testing.T) {
	s, h := measuredScene(t, func(s *scene.Scene) {
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 9.5, Aspect: figure()}); !ok {
			t.Fatal("entité qui cache refusée")
		}
	})

	sorter := New(s)
	image := sorter.Frame()
	list := sorter.Overlaps()

	if len(list) != 1 {
		t.Fatalf("%d relevés, attendu 1", len(list))
	}
	if list[0].Identity != uint32(h) {
		t.Errorf("poignée %d, attendu %d", list[0].Identity, uint32(h))
	}

	// Le compte attendu se dérive des quads rendus, et non d'une constante : il dépend
	// de la projection et de l'ancre, que ce cas n'a pas à figer.
	want := shared(image[0], image[1])
	if want == 0 {
		t.Fatal("les deux quads ne se recouvrent pas, le cas ne garde rien")
	}
	if int(list[0].Hidden) != want {
		t.Errorf("%d pixels cachés, attendu %d", list[0].Hidden, want)
	}
	if list[0].ByDecor != 0 {
		t.Errorf("%d pixels de décor, attendu aucun", list[0].ByDecor)
	}
}

// TestOverlapCountsPastTheThreshold garde que la mesure va jusqu'au bout au lieu de
// s'arrêter dès qu'un seuil est franchi.
//
// **C'est le seul cas de ce fichier qui puisse faire échouer cette propriété**, et les
// autres ne l'approchent pas : ils n'ont qu'un seul cachant, donc un total tronqué au
// premier égale le total exact et rien ne les distingue. Ici deux choses recouvrent, le
// décor en premier dans l'ordre de collecte — la révélation s'arrêtait là, puisque le
// décor seul suffisait à sa décision —, et le relevé doit porter les deux.
//
// Vu rougir en remettant cette sortie dans [Sorter.overlap] : le total retombe au
// recouvrement du décor seul, et la part de l'entité disparaît.
func TestOverlapCountsPastTheThreshold(t *testing.T) {
	s, _ := measuredScene(t, func(s *scene.Scene) {
		// **Un mur strictement devant, et les deux conditions comptent.** Une cellule
		// en (8, 9) a la profondeur de son coin, qui est exactement celle de l'entité
		// suivie : à égalité le décor se peint avant ce qui se tient dessus, donc il
		// ne le cache pas. Et une cellule plate en (9, 9), qui est bien devant,
		// tombe entièrement sous l'image de l'entité — mesuré, aucune des huit
		// voisines ne recouvre une figure plate ancrée par le pied. Il faut donc de la
		// hauteur, c'est-à-dire un mur, ce qui est aussi le seul cas qu'un jeu voit.
		if !s.PlaceCell(9, 9, 0, wall()) {
			t.Fatal("mur qui cache refusé")
		}
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 9.5, Aspect: figure()}); !ok {
			t.Fatal("entité qui cache refusée")
		}
	})

	sorter := New(s)
	sorter.Frame()
	list := sorter.Overlaps()

	if len(list) != 1 {
		t.Fatalf("%d relevés, attendu 1", len(list))
	}
	if list[0].ByDecor == 0 {
		t.Fatal("aucun pixel de décor, le cas ne garde rien")
	}
	if list[0].Hidden <= list[0].ByDecor {
		t.Errorf("%d pixels cachés pour %d de décor, attendu strictement plus : la mesure "+
			"s'est arrêtée au premier cachant", list[0].Hidden, list[0].ByDecor)
	}
}

// TestOverlapSeparatesTheDecor garde que la part du décor se compte à part, puisque
// c'est sur elle qu'un hôte décide de la nature de son rappel.
//
// Le cachant est un mur strictement devant l'entité suivie, pour les deux raisons
// dites dans [TestOverlapCountsPastTheThreshold].
func TestOverlapSeparatesTheDecor(t *testing.T) {
	s, _ := measuredScene(t, func(s *scene.Scene) {
		if !s.PlaceCell(9, 9, 0, wall()) {
			t.Fatal("mur qui cache refusé")
		}
	})

	list := func() render.OverlapList { sorter := New(s); sorter.Frame(); return sorter.Overlaps() }()
	if len(list) != 1 {
		t.Fatalf("%d relevés, attendu 1", len(list))
	}
	if list[0].ByDecor == 0 || list[0].ByDecor != list[0].Hidden {
		t.Errorf("%d pixels de décor sur %d cachés, attendus égaux et non nuls",
			list[0].ByDecor, list[0].Hidden)
	}
}

// TestOverlapHoldsOnlyWhatWasAsked garde les trois régimes où le relevé doit rester
// vide, et c'est ce qui borne son coût : il ne porte que ce qu'un jeu a marqué.
func TestOverlapHoldsOnlyWhatWasAsked(t *testing.T) {
	for _, c := range []struct {
		name  string
		build func(*testing.T) *scene.Scene
	}{
		{"une entité qu'on ne suit pas", func(t *testing.T) *scene.Scene {
			s := testScene(t, 16, 4)
			if _, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: figure()}); !ok {
				t.Fatal("entité refusée")
			}
			return s
		}},
		// La grille est large à dessein : sur seize cellules, une vue de 480×270 montre
		// la carte entière et aucune position n'est hors champ.
		{"une entité suivie mais hors de la vue", func(t *testing.T) *scene.Scene {
			s := testScene(t, 200, 4)
			if _, ok := s.Place(scene.Entity{X: 1.5, Y: 1.5, Aspect: figure(), Measure: true}); !ok {
				t.Fatal("entité refusée")
			}
			return s
		}},
		{"une scène vide", func(t *testing.T) *scene.Scene { return testScene(t, 16, 4) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			sorter := New(c.build(t))
			sorter.Frame()

			if n := len(sorter.Overlaps()); n != 0 {
				t.Errorf("%d relevés, attendu aucun", n)
			}
		})
	}
}

// TestOverlapIsEmptyBeforeTheFirstFrame garde ce que la godoc promet à un hôte : la
// mesure est un produit du tri, donc elle n'existe pas avant qu'il ait eu lieu.
//
// Sans ce cas, un relevé rendu depuis une tranche non remise à zéro passerait inaperçu
// jusqu'à ce qu'un hôte lise des pixels d'une image qui n'a pas été calculée.
func TestOverlapIsEmptyBeforeTheFirstFrame(t *testing.T) {
	s, _ := measuredScene(t, nil)

	if n := len(New(s).Overlaps()); n != 0 {
		t.Errorf("%d relevés avant le premier pas, attendu aucun", n)
	}
}

// TestOverlapCostsNoAllocation garde l'invariant du chemin de rendu sur le relevé, que
// les deux gardes du paquet racine n'exercent pas : elles ne marquent aucune entité.
func TestOverlapCostsNoAllocation(t *testing.T) {
	s, _ := measuredScene(t, func(s *scene.Scene) {
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 9.5, Aspect: figure()}); !ok {
			t.Fatal("entité qui cache refusée")
		}
	})
	sorter := New(s)

	// Une première image alloue les tranches ; ce sont les suivantes qui doivent être
	// gratuites, puisque c'est ce qu'un jeu paie soixante fois par seconde.
	sorter.Frame()
	sorter.Overlaps()

	if n := testing.AllocsPerRun(20, func() { sorter.Frame(); sorter.Overlaps() }); n != 0 {
		t.Errorf("%g allocations par image, attendu aucune", n)
	}
}
