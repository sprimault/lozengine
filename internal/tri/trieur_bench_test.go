// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Une mesure brute dit ce que l'opération coûte sur la machine qui mesure. Ce qu'on
// veut savoir est la part d'image qu'elle prendrait sur une machine modeste : les
// chiffres de ce banc se lisent donc pondérés, jamais bruts.
//
// Les deux constantes et la fonction qui suit redisent celles du banc de `raster`. Ce
// n'est pas un oubli de factorisation : le tableau des frontières interdit à `tri`
// d'atteindre `raster`, et un paquet commun pour trois lignes de test coûterait une
// ligne de plus à ce tableau.
const (
	ponderation = 4
	imageA60Hz  = 16_666_667
)

// partDImage rapporte la part d'une image à 60 Hz que l'étape prendrait sur une machine
// modeste. Le ns/op brut reste rapporté à côté : c'est lui que `benchstat` compare, la
// pondération n'étant qu'une lecture.
func partDImage(b *testing.B) {
	parOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(parOp*ponderation*100/imageA60Hz, "%image-pondere")
}

// scenePourBanc monte une scène de la taille et du contenu demandés, vue par une
// fenêtre à la résolution interne du moteur.
//
// Les entités sont éparpillées sur toute la carte par deux pas premiers entre eux
// plutôt que posées côte à côte : groupées, elles tomberaient toutes dans la fenêtre
// ou toutes en dehors, et le banc mesurerait un cas qu'aucune partie ne produit.
func scenePourBanc(b *testing.B, cote, entites, strates int) *scene.Scene {
	b.Helper()

	s, err := scene.New(cote, cote, geometry.Dimetric(32, 16), scene.Options{Entities: entites + 1})
	if err != nil {
		b.Fatalf("montage : %v", err)
	}
	s.Focus(float64(cote)/2, float64(cote)/2, 480, 270)

	sol := scene.Aspect{Width: 32, Height: 16, Tint: render.White()}
	for strate := range strates {
		for v := range cote {
			for u := range cote {
				s.PlaceCell(u, v, uint8(strate), sol)
			}
		}
	}

	figure := scene.Aspect{Width: 12, Height: 24, Tint: render.White()}
	for i := range entites {
		x := float64((i*37)%cote) + 0.5
		y := float64((i*61)%cote) + 0.5

		if _, ok := s.Place(scene.Entity{X: x, Y: y, Aspect: figure}); !ok {
			b.Fatalf("entité %d refusée", i)
		}
	}
	return s
}

// BenchmarkImage mesure le chemin complet d'une image, à la cible de contrôle du
// projet et autour d'elle.
//
// Ce que les cas disent ensemble et qu'aucun ne dirait seul : le décor ne se relève que
// sur la fenêtre, donc son coût ne doit pas suivre la taille de la carte, là où les
// entités se parcourent toutes et le font suivre. Le rapport entre ces lignes est un
// fait du moteur ; leur valeur absolue n'est qu'un fait de la machine qui mesure.
func BenchmarkImage(b *testing.B) {
	cas := []struct {
		nom                    string
		cote, entites, strates int
	}{
		{"cible du projet", 500, 5000, 1},
		{"la même carte sans entité", 500, 0, 1},
		{"les mêmes entités sans décor", 500, 5000, 0},
		{"carte dix fois plus petite", 50, 5000, 1},
		{"trois strates de décor", 500, 5000, 3},
	}

	for _, c := range cas {
		b.Run(c.nom, func(b *testing.B) {
			trieur := Nouveau(scenePourBanc(b, c.cote, c.entites, c.strates))

			// Une première image monte les tampons, que les suivantes réemploient.
			// La mesurer avec les autres compterait une allocation qui n'a lieu
			// qu'une fois par partie.
			quads := len(trieur.Image())

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				trieur.Image()
			}

			partDImage(b)
			b.ReportMetric(float64(quads), "quads")
		})
	}
}
