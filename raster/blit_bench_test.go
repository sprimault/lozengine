// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// Une mesure brute dit ce que l'opération coûte sur la machine qui mesure. Ce qu'on
// veut savoir est la part d'image qu'elle prendrait sur une machine modeste : les
// chiffres de ces bancs se lisent donc pondérés, jamais bruts.
const (
	// ponderation rapporte la mesure à une machine plus lente, pour ce qui dépend de
	// la bande passante mémoire.
	ponderation = 4

	// imageA60Hz est le budget d'une image, en nanosecondes.
	imageA60Hz = 16_666_667
)

// imagePart rapporte la part d'une image à 60 Hz que l'opération prendrait sur une
// machine modeste. Le ns/op brut reste rapporté à côté : c'est lui que `benchstat`
// compare, la pondération n'étant qu'une lecture.
//
// Elle n'a de sens que pour ce qui se fait une fois par image. Un banc à l'échelle du
// quad n'en reçoit pas : ce qu'on y lit est un rapport entre chemins, et un rapport
// n'a pas besoin d'être pondéré.
func imagePart(b *testing.B) {
	parOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(parOp*ponderation*100/imageA60Hz, "%image-pondere")
}

// tile rend une tile de sol en losange 32×16 : coins transparents, corps opaque,
// bord à demi transparent. Les trois genres de séquence y sont représentés dans les
// proportions d'un vrai sprite, ce qui est tout ce qu'on demande à une mire de
// mesure.
func tile() Sheet {
	p := NewSheet(32, 16)
	for y := 0; y < 16; y++ {
		height := y
		if y >= 8 {
			height = 15 - y
		}
		demi := (height + 1) * 2

		for x := 16 - demi; x < 16+demi; x++ {
			color := rendu.Couleur{R: 90, V: 110, B: 90, A: 255}
			if x == 16-demi || x == 16+demi-1 {
				color = rendu.Couleur{R: 45, V: 55, B: 45, A: 128}
			}
			p.Pixels[y*32+x] = color
		}
	}
	return p
}

// character rend une silhouette debout de 16×24, marges comprises : c'est la
// deuxième forme que le moteur dessine par milliers.
func character() Sheet {
	p := NewSheet(16, 24)
	for y := 2; y < 24; y++ {
		for x := 4; x < 12; x++ {
			color := rendu.Couleur{R: 160, V: 140, B: 120, A: 255}
			if x == 4 || x == 11 {
				color = rendu.Couleur{R: 80, V: 70, B: 60, A: 128}
			}
			p.Pixels[y*16+x] = color
		}
	}
	return p
}

// testAtlas range les deux formes et rend l'atlas, la tuile en 0 et le personnage
// en 1.
func testAtlas(b *testing.B) *Atlas {
	b.Helper()

	var a Atlas
	for _, p := range []Sheet{tile(), character()} {
		if _, err := a.Add(p); err != nil {
			b.Fatalf("ajout : %v", err)
		}
	}
	return &a
}

// BenchmarkQuad mesure un quad seul par chemin du blit. Les écarts entre ces
// lignes sont ce qui dira si une optimisation a servi : la copie est le cas qu'il
// faut garder rapide, le miroir celui qui perd la copie en bloc.
func BenchmarkQuad(b *testing.B) {
	cases := []struct {
		name   string
		regler func(*rendu.Quad)
	}{
		{"copie", func(*rendu.Quad) {}},
		{"teinte", func(q *rendu.Quad) { q.Teinte = rendu.Couleur{R: 255, V: 160, B: 160, A: 255} }},
		{"demi-opacité", func(q *rendu.Quad) { q.Teinte = rendu.Couleur{R: 128, V: 128, B: 128, A: 128} }},
		{"miroir", func(q *rendu.Quad) { q.Drapeaux = rendu.MiroirX }},
		{"découpé de moitié", func(q *rendu.Quad) { q.X = -16 }},
		{"aplat", func(q *rendu.Quad) { q.Drapeaux = rendu.Aplat }},
	}

	a := testAtlas(b)
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buffer := rendu.NouveauTampon(480, 270)

			q := rendu.Quad{X: 100, Y: 100, Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc()}
			c.regler(&q)
			r := rendu.Rendu{q}

			b.ResetTimer()
			for range b.N {
				Draw(buffer, r, a)
			}
		})
	}
}

// BenchmarkClear mesure la première opération de chaque image, à la résolution
// interne. Elle se compare au blit qui la suit : une remise à zéro qui coûterait
// autant que le dessin serait le premier endroit à regarder.
func BenchmarkClear(b *testing.B) {
	cases := []struct {
		name  string
		color rendu.Couleur
	}{
		{"transparent", rendu.Couleur{}},
		{"couleur", rendu.Couleur{R: 20, V: 30, B: 40, A: 255}},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buffer := rendu.NouveauTampon(480, 270)

			b.ResetTimer()
			for range b.N {
				Clear(buffer, c.color)
			}
			imagePart(b)
		})
	}
}

// BenchmarkUpscale mesure la dernière opération de l'image, celle qu'un backend sans
// mise à l'échelle matérielle doit faire lui-même. Les deux tailles de sortie séparent
// ce que coûtent les bandes de ce que coûte l'agrandissement.
func BenchmarkUpscale(b *testing.B) {
	cases := []struct {
		name                string
		outWidth, outHeight uint16
		factor              int
	}{
		{"sans bande, facteur 4", 1920, 1080, 4},
		{"avec bandes, facteur 3", 1920, 1080, 3},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			inner := rendu.NouveauTampon(480, 270)
			Clear(inner, rendu.Couleur{R: 60, V: 80, B: 60, A: 255})
			out := rendu.NouveauTampon(c.outWidth, c.outHeight)

			b.ResetTimer()
			for range b.N {
				Upscale(inner, out, c.factor, rendu.Couleur{A: 255})
			}
			imagePart(b)
		})
	}
}

// BenchmarkFrame mesure une image entière : un sol qui couvre la résolution interne
// et deux cents personnages dessus. C'est le seul chiffre qui réponde à la question
// qui compte — combien d'images par seconde le blit laisse au reste du moteur.
func BenchmarkFrame(b *testing.B) {
	a := testAtlas(b)
	buffer := rendu.NouveauTampon(480, 270)

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
		Draw(buffer, r, a)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "images/s")
	b.ReportMetric(float64(tuiles), "tuiles/image")
	imagePart(b)
}
