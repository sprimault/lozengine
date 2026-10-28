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
		Entities: 4, RevealAfter: after, RevealTint: tint,
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
