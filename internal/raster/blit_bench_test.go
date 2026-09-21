// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// tuile rend une tuile de sol en losange 32×16 : coins transparents, corps opaque,
// bord à demi transparent. Les trois genres de séquence y sont représentés dans les
// proportions d'un vrai sprite, ce qui est tout ce qu'on demande à une mire de
// mesure.
func tuile() Planche {
	p := NouvellePlanche(32, 16)
	for y := 0; y < 16; y++ {
		hauteur := y
		if y >= 8 {
			hauteur = 15 - y
		}
		demi := (hauteur + 1) * 2

		for x := 16 - demi; x < 16+demi; x++ {
			couleur := rendu.Couleur{R: 90, V: 110, B: 90, A: 255}
			if x == 16-demi || x == 16+demi-1 {
				couleur = rendu.Couleur{R: 45, V: 55, B: 45, A: 128}
			}
			p.Pixels[y*32+x] = couleur
		}
	}
	return p
}

// personnage rend une silhouette debout de 16×24, marges comprises : c'est la
// deuxième forme que le moteur dessine par milliers.
func personnage() Planche {
	p := NouvellePlanche(16, 24)
	for y := 2; y < 24; y++ {
		for x := 4; x < 12; x++ {
			couleur := rendu.Couleur{R: 160, V: 140, B: 120, A: 255}
			if x == 4 || x == 11 {
				couleur = rendu.Couleur{R: 80, V: 70, B: 60, A: 128}
			}
			p.Pixels[y*16+x] = couleur
		}
	}
	return p
}

// atlasEssai range les deux formes et rend l'atlas, la tuile en 0 et le personnage
// en 1.
func atlasEssai(b *testing.B) *Atlas {
	b.Helper()

	var a Atlas
	for _, p := range []Planche{tuile(), personnage()} {
		if _, err := a.Ajouter(p); err != nil {
			b.Fatalf("ajout : %v", err)
		}
	}
	return &a
}

// BenchmarkQuad mesure un quad seul par chemin du blit. Les écarts entre ces
// lignes sont ce qui dira si une optimisation a servi : la copie est le cas qu'il
// faut garder rapide, le miroir celui qui perd la copie en bloc.
func BenchmarkQuad(b *testing.B) {
	cas := []struct {
		nom    string
		regler func(*rendu.Quad)
	}{
		{"copie", func(*rendu.Quad) {}},
		{"teinte", func(q *rendu.Quad) { q.Teinte = rendu.Couleur{R: 255, V: 160, B: 160, A: 255} }},
		{"demi-opacité", func(q *rendu.Quad) { q.Teinte = rendu.Couleur{R: 128, V: 128, B: 128, A: 128} }},
		{"miroir", func(q *rendu.Quad) { q.Drapeaux = rendu.MiroirX }},
		{"découpé de moitié", func(q *rendu.Quad) { q.X = -16 }},
		{"aplat", func(q *rendu.Quad) { q.Drapeaux = rendu.Aplat }},
	}

	a := atlasEssai(b)
	for _, c := range cas {
		b.Run(c.nom, func(b *testing.B) {
			tampon := NouveauTampon(480, 270)

			q := rendu.Quad{X: 100, Y: 100, Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc()}
			c.regler(&q)
			r := rendu.Rendu{q}

			b.ResetTimer()
			for range b.N {
				tampon.Dessiner(r, a)
			}
		})
	}
}

// BenchmarkEffacer mesure la première opération de chaque image, à la résolution
// interne. Elle se compare au blit qui la suit : une remise à zéro qui coûterait
// autant que le dessin serait le premier endroit à regarder.
func BenchmarkEffacer(b *testing.B) {
	cas := []struct {
		nom     string
		couleur rendu.Couleur
	}{
		{"transparent", rendu.Couleur{}},
		{"couleur", rendu.Couleur{R: 20, V: 30, B: 40, A: 255}},
	}
	for _, c := range cas {
		b.Run(c.nom, func(b *testing.B) {
			tampon := NouveauTampon(480, 270)

			b.ResetTimer()
			for range b.N {
				tampon.Effacer(c.couleur)
			}
		})
	}
}

// BenchmarkImage mesure une image entière : un sol qui couvre la résolution interne
// et deux cents personnages dessus. C'est le seul chiffre qui réponde à la question
// qui compte — combien d'images par seconde le blit laisse au reste du moteur.
func BenchmarkImage(b *testing.B) {
	a := atlasEssai(b)
	tampon := NouveauTampon(480, 270)

	var r rendu.Rendu
	for y := -16; y < 270; y += 8 {
		decalage := 0
		if (y/8)%2 != 0 {
			decalage = 16
		}
		for x := -32; x < 480; x += 32 {
			r = append(r, rendu.Quad{
				X: int16(x + decalage), Y: int16(y),
				Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc(),
			})
		}
	}
	tuiles := len(r)

	for i := range 200 {
		r = append(r, rendu.Quad{
			X: int16(i * 7 % 460), Y: int16(i * 13 % 240),
			Largeur: 16, Hauteur: 24, Planche: 1, Teinte: rendu.Blanc(),
		})
	}

	b.ResetTimer()
	for range b.N {
		tampon.Dessiner(r, a)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "images/s")
	b.ReportMetric(float64(tuiles), "tuiles/image")
}
