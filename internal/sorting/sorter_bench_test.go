// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La mesure du chemin complet d'une image, à la cible de contrôle du jalon. Les chiffres
// se lisent en part d'une image, que [bench.Part] rapporte : une mesure brute dit ce que
// vaut la machine qui mesure, pas si l'étape mérite qu'on y touche.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/internal/bench"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// benchScene monte une scène de la taille et du contenu demandés, vue par une
// fenêtre à la résolution interne du moteur.
//
// Les entités sont éparpillées sur toute la carte par deux pas premiers entre eux
// plutôt que posées côte à côte : groupées, elles tomberaient toutes dans la fenêtre
// ou toutes en dehors, et le banc mesurerait un cas qu'aucune partie ne produit.
func benchScene(b *testing.B, side, entities, tiers int) *scene.Scene {
	b.Helper()

	s, err := scene.New(side, side, geometry.Dimetric(32, 16), scene.Options{Entities: entities + 1})
	if err != nil {
		b.Fatalf("montage : %v", err)
	}
	s.Focus(float64(side)/2, float64(side)/2, 480, 270)

	ground := scene.Aspect{Width: 32, Height: 16, Tint: render.White()}
	for tier := range tiers {
		for v := range side {
			for u := range side {
				s.PlaceCell(u, v, uint8(tier), ground)
			}
		}
	}

	figure := scene.Aspect{Width: 12, Height: 24, Tint: render.White()}
	for i := range entities {
		x := float64((i*37)%side) + 0.5
		y := float64((i*61)%side) + 0.5

		if _, ok := s.Place(scene.Entity{X: x, Y: y, Aspect: figure}); !ok {
			b.Fatalf("entité %d refusée", i)
		}
	}
	return s
}

// BenchmarkFrame mesure le chemin complet d'une image, à la cible de contrôle du
// projet et autour d'elle.
//
// Ce que les cas disent ensemble et qu'aucun ne dirait seul : le décor ne se relève que
// sur la fenêtre, donc son coût ne doit pas suivre la taille de la carte, là où les
// entités se parcourent toutes et le font suivre. Le rapport entre ces lignes est un
// fait du moteur ; leur valeur absolue n'est qu'un fait de la machine qui mesure.
//
// Le second fait sort de la troisième strate, et c'est le plus utile à qui dimensionne
// une scène : **une strate pleine de plus coûte environ dix fois ce que coûte le
// parcours de trois cents entités.** Elle ajoute un quad par cellule visible, là qu'une
// entité hors de la fenêtre ne coûte que son parcours. Le rapport s'obtient d'une seule
// série — la strate supplémentaire se déduit de l'écart entre une et trois, les entités
// de la ligne sans décor —, donc la dérive thermique ne le touche pas.
func BenchmarkFrame(b *testing.B) {
	cases := []struct {
		name                  string
		side, entities, tiers int
	}{
		{"cible du projet", 500, 5000, 1},
		{"la même carte sans entité", 500, 0, 1},
		{"les mêmes entités sans décor", 500, 5000, 0},
		{"carte dix fois plus petite", 50, 5000, 1},
		{"trois strates de décor", 500, 5000, 3},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			sorter := New(benchScene(b, c.side, c.entities, c.tiers))

			// Une première image monte les tampons, que les suivantes réemploient.
			// La mesurer avec les autres compterait une allocation qui n'a lieu
			// qu'une fois par partie.
			quads := len(sorter.Frame())

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sorter.Frame()
			}

			bench.Part(b)
			b.ReportMetric(float64(quads), "quads")
		})
	}
}
