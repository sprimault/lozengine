// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la révélation doit tenir : un second quad quand l'image cache assez, aucun
// sinon, et rien du tout pour une chose qui ne l'a pas demandée.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// revealScene monte une scène où une chose marquée se fait cacher par une autre.
//
// Les deux positions sont choisies pour que le recouvrement soit réel et partiel : la
// seconde est plus profonde d'une demi-cellule sur chaque axe, donc elle se peint
// après — c'est la clé qui le dit — et son quad mord sur le haut du premier.
func revealScene(t *testing.T, after int, tint render.Color, reveal uint8) *scene.Scene {
	t.Helper()

	s, err := scene.New(16, 16, geometry.Dimetric(32, 16), scene.Options{
		Entities:  4,
		Revealing: scene.RevealOptions{After: after, Tint: tint},
	})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(8, 8, 480, 270)

	hidden := figure()
	hidden.Reveal = reveal
	if _, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: hidden}); !ok {
		t.Fatal("entité cachée refusée")
	}

	if _, ok := s.Place(scene.Entity{X: 8.5, Y: 9.5, Aspect: figure()}); !ok {
		t.Fatal("entité qui cache refusée")
	}
	return s
}

// TestRevealEmitsASecondQuad vérifie qu'une chose marquée que l'image cache se
// redessine dans la couche demandée.
//
// **C'est le seul cas à pouvoir faire échouer la révélation**, et il tient par deux
// faits ensemble : le second quad existe, et il est peint après celui qui cachait.
// Vérifier le seul compte laisserait passer une couche mal posée, qui est le défaut
// le plus probable — le quad serait là, et toujours dessous.
func TestRevealEmitsASecondQuad(t *testing.T) {
	const layer = 2

	image := New(revealScene(t, 0, render.Color{}, layer)).Frame()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}

	// Le dernier peint est la révélation : sa couche domine celle des deux autres.
	first, revealed := image[0], image[2]
	if first.X != revealed.X || first.Y != revealed.Y {
		t.Errorf("révélation en (%d, %d), attendu (%d, %d)",
			revealed.X, revealed.Y, first.X, first.Y)
	}
	if first.Sheet != revealed.Sheet || first.Width != revealed.Width {
		t.Errorf("révélation sur la planche %d de %d de large, attendu %d et %d",
			revealed.Sheet, revealed.Width, first.Sheet, first.Width)
	}
}

// TestRevealCountsAnyOrigin garde que la révélation compte tout ce qui se peint après,
// décor comme entité.
//
// **C'est le cas que la godoc avait manqué**, et il montre à quel point une
// documentation plus étroite que son mécanisme coûte cher : elle disait « quand le
// décor la cache », alors que le relevé porte sur la clé et ignore l'origine du quad.
// Un intégrateur qui la lisait concluait que l'encerclement par des créatures n'était
// pas couvert — c'est-à-dire précisément le cas que la justification du champ promet —
// et réécrivait chez lui un mécanisme qu'il possédait déjà. Relevé le 28 octobre 2026
// depuis le module publié, pas depuis le dépôt.
//
// Les deux origines sont donc nommées ici, pour qu'un resserrement de la condition ne
// puisse plus passer pour une précision.
func TestRevealCountsAnyOrigin(t *testing.T) {
	for _, c := range []struct {
		name  string
		cover func(*scene.Scene)
	}{
		{"une entité qui s'interpose", func(s *scene.Scene) {
			if _, ok := s.Place(scene.Entity{X: 8.5, Y: 9.5, Aspect: figure()}); !ok {
				t.Fatal("entité qui cache refusée")
			}
		}},
		{"une cellule de décor", func(s *scene.Scene) {
			if !s.PlaceCell(8, 9, 0, figure()) {
				t.Fatal("cellule qui cache refusée")
			}
		}},
	} {
		s := testScene(t, 16, 2)
		s.SetRevealing(scene.RevealOptions{})

		hidden := figure()
		hidden.Reveal = 2
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: hidden}); !ok {
			t.Fatal("entité cachée refusée")
		}
		c.cover(s)

		revealed := 0
		for _, q := range New(s).Frame() {
			if q.Tint == hidden.Tint && q.Width == hidden.Width {
				revealed++
			}
		}

		// Deux quads de la chose : l'original et sa révélation. La cellule de décor
		// en porte un de plus, de même taille, d'où la comparaison sur la teinte.
		if revealed < 2 {
			t.Errorf("%s : %d quads de la chose, attendu deux", c.name, revealed)
		}
	}
}

// TestRevealHoldsUnderItsThreshold vérifie qu'un recouvrement sous le seuil ne
// déclenche rien.
//
// Sans seuil, un chevauchement d'un pixel suffirait et la chose serait réémise presque
// toujours : un quad de plus par image pour un résultat identique.
func TestRevealHoldsUnderItsThreshold(t *testing.T) {
	image := New(revealScene(t, 1_000_000, render.Color{}, 2)).Frame()
	if len(image) != 2 {
		t.Errorf("%d quads, attendu 2", len(image))
	}
}

// TestRevealLeavesUnmarkedThingsAlone garde le cas courant, où rien n'est à révéler et
// où la passe ne doit rien coûter ni rien changer.
func TestRevealLeavesUnmarkedThingsAlone(t *testing.T) {
	image := New(revealScene(t, 0, render.Color{}, 0)).Frame()
	if len(image) != 2 {
		t.Errorf("%d quads, attendu 2", len(image))
	}
}

// TestRevealTintReplacesTheAspectTint vérifie qu'une teinte de scène s'applique à la
// chose révélée et à elle seule, et que sa valeur zéro laisse celle de l'aspect.
func TestRevealTintReplacesTheAspectTint(t *testing.T) {
	sombre := render.Color{R: 40, G: 40, B: 40, A: 255}

	image := New(revealScene(t, 0, sombre, 2)).Frame()
	if len(image) != 3 {
		t.Fatalf("%d quads, attendu 3", len(image))
	}
	if image[2].Tint != sombre {
		t.Errorf("révélation teintée %v, attendu %v", image[2].Tint, sombre)
	}
	if image[0].Tint == sombre {
		t.Error("la teinte de révélation a débordé sur la chose elle-même")
	}

	plain := New(revealScene(t, 0, render.Color{}, 2)).Frame()
	if plain[2].Tint != plain[0].Tint {
		t.Errorf("sans teinte de scène : %v, attendu celle de l'aspect %v",
			plain[2].Tint, plain[0].Tint)
	}
}

// tall rend un aspect haut, de quoi recouvrir franchement ce qui le précède.
func tall(height uint16) scene.Aspect {
	a := figure()
	a.Height = height
	return a
}

// TestTintSwitchesOnTheThresholdNotOnPresence garde la règle de bascule entre les deux
// teintes, qui est la part la plus facile à écrire de travers.
//
// **La teinte du décor s'applique quand le décor seul atteindrait le seuil, et non dès
// qu'un pixel de décor se mêle au recouvrement.** C'est la raison même du seuil, portée
// une seconde fois : un pixel caché ne dit rien de la profondeur, puisqu'on ne voit pas
// qu'il y a quelque chose devant. La formulation naïve — « dès qu'une part vient du
// décor » — ne diffère de celle-ci que sur un cas, et c'est le plus courant : un
// personnage encerclé qui tourne autour d'un pilier verrait sa teinte battre à chaque
// frôlement alors que l'essentiel du recouvrement vient de la mêlée. Un signal qui
// clignote sur un écran chargé est pire que pas de signal.
//
// Le troisième cas est exactement celui-là, et il est le seul que la formulation naïve
// ferait rougir.
func TestTintSwitchesOnTheThresholdNotOnPresence(t *testing.T) {
	const threshold = 300

	plain := render.Color{R: 255, G: 255, B: 255, A: 255}
	muted := render.Color{R: 40, G: 40, B: 40, A: 255}

	for _, c := range []struct {
		name  string
		cover func(*scene.Scene)
		want  render.Color
	}{
		{"le décor seul suffit", func(s *scene.Scene) {
			s.PlaceCell(9, 9, 0, tall(48))
		}, muted},

		{"les entités seules", func(s *scene.Scene) {
			s.Place(scene.Entity{X: 9.5, Y: 9.5, Aspect: tall(48)})
		}, plain},

		{"un frôlement de décor dans une mêlée", func(s *scene.Scene) {
			s.PlaceCell(9, 9, 0, tall(20))
			s.Place(scene.Entity{X: 9.5, Y: 9.5, Aspect: tall(48)})
		}, plain},
	} {
		s := testScene(t, 16, 3)
		s.SetRevealing(scene.RevealOptions{
			After: threshold, Tint: plain, TintByDecor: muted,
		})

		hidden := figure()
		hidden.Reveal = 2
		if _, ok := s.Place(scene.Entity{X: 8.5, Y: 8.5, Aspect: hidden}); !ok {
			t.Fatal("entité cachée refusée")
		}
		c.cover(s)

		image := New(s).Frame()
		revealed := image[len(image)-1]

		if revealed.Tint != c.want {
			t.Errorf("%s : teinte %v, attendue %v", c.name, revealed.Tint, c.want)
		}
	}
}

// TestSetRevealingChangesTheConventionsMidGame garde qu'un réglage de lisibilité se
// change en cours de partie, et non seulement à la construction.
//
// Les figer pour la durée d'une partie était une limite sans raison : un jeu baisse sa
// teinte la nuit, relève son seuil quand il agrandit ses sprites, ou l'annule le temps
// d'une cinématique. Le cas éprouve les deux sens, puisqu'un réglage qui ne saurait
// que durcir vaudrait à peine mieux qu'un réglage figé.
func TestSetRevealingChangesTheConventionsMidGame(t *testing.T) {
	s := revealScene(t, 0, render.Color{}, 2)

	if image := New(s).Frame(); len(image) != 3 {
		t.Fatalf("au départ : %d quads, attendu 3", len(image))
	}

	// Seuil hors d'atteinte : la révélation s'arrête.
	s.SetRevealing(scene.RevealOptions{After: 1_000_000})
	if image := New(s).Frame(); len(image) != 2 {
		t.Errorf("seuil relevé : %d quads, attendu 2", len(image))
	}

	// Et elle repart, teinte comprise.
	sombre := render.Color{R: 40, G: 40, B: 40, A: 255}
	s.SetRevealing(scene.RevealOptions{Tint: sombre})

	image := New(s).Frame()
	if len(image) != 3 {
		t.Fatalf("seuil rabaissé : %d quads, attendu 3", len(image))
	}
	if image[2].Tint != sombre {
		t.Errorf("révélation teintée %v, attendu %v", image[2].Tint, sombre)
	}
}

// TestSetRevealingRefusesANegativeThreshold garde le repli d'un seuil négatif, qu'une
// soustraction de jeu produit sans qu'on l'ait voulu.
func TestSetRevealingRefusesANegativeThreshold(t *testing.T) {
	s := revealScene(t, 0, render.Color{}, 2)
	s.SetRevealing(scene.RevealOptions{After: -5})

	if o := scene.Revealing(s); o.After != 0 {
		t.Errorf("seuil de %d, attendu zéro", o.After)
	}
}

// TestOverlapCountsSharedPixels éprouve la mesure seule, que les cas ci-dessus
// n'atteignent qu'à travers le seuil.
func TestOverlapCountsSharedPixels(t *testing.T) {
	for _, c := range []struct {
		name string
		a, b render.Quad
		want int
	}{
		{"disjoints en abscisse",
			render.Quad{X: 0, Y: 0, Width: 10, Height: 10},
			render.Quad{X: 10, Y: 0, Width: 10, Height: 10}, 0},
		{"disjoints en ordonnée",
			render.Quad{X: 0, Y: 0, Width: 10, Height: 10},
			render.Quad{X: 0, Y: 10, Width: 10, Height: 10}, 0},
		{"un coin commun",
			render.Quad{X: 0, Y: 0, Width: 10, Height: 10},
			render.Quad{X: 6, Y: 7, Width: 10, Height: 10}, 12},
		{"l'un dans l'autre",
			render.Quad{X: 0, Y: 0, Width: 10, Height: 10},
			render.Quad{X: 2, Y: 2, Width: 4, Height: 4}, 16},
		{"coordonnées négatives",
			render.Quad{X: -8, Y: -4, Width: 10, Height: 10},
			render.Quad{X: 0, Y: 0, Width: 10, Height: 10}, 12},
	} {
		if got := overlap(c.a, c.b); got != c.want {
			t.Errorf("%s : %d pixels, attendu %d", c.name, got, c.want)
		}
	}
}
