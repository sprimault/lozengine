// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les quatre étapes d'une image mesurées séparément, et leur part d'un budget à
// soixante images par seconde sur une machine modeste. C'est ce rapport qui décide
// qu'on optimise ou non, jamais la mesure brute de la machine qui mesure.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// Une mesure brute dit ce que l'opération coûte sur la machine qui mesure. Ce qu'on
// veut savoir est la part d'image qu'elle prendrait sur une machine modeste : les
// chiffres de ces bancs se lisent donc pondérés, jamais bruts.
const (
	// weighting rapporte la mesure à une machine plus lente, pour ce qui dépend de
	// la bande passante mémoire.
	weighting = 4

	// frameAt60Hz est le budget d'une image, en nanosecondes.
	frameAt60Hz = 16_666_667
)

// framePart rapporte la part d'une image à 60 Hz que l'opération prendrait sur une
// machine modeste. Le ns/op brut reste rapporté à côté : c'est lui que `benchstat`
// compare, la pondération n'étant qu'une lecture.
//
// Elle n'a de sens que pour ce qui se fait une fois par image. Un banc à l'échelle du
// quad n'en reçoit pas : ce qu'on y lit est un rapport entre chemins, et un rapport
// n'a pas besoin d'être pondéré.
func framePart(b *testing.B) {
	perOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(perOp*weighting*100/frameAt60Hz, "%image-pondere")
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
		half := (height + 1) * 2

		for x := 16 - half; x < 16+half; x++ {
			color := render.Color{R: 90, G: 110, B: 90, A: 255}
			if x == 16-half || x == 16+half-1 {
				color = render.Color{R: 45, G: 55, B: 45, A: 128}
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
			color := render.Color{R: 160, G: 140, B: 120, A: 255}
			if x == 4 || x == 11 {
				color = render.Color{R: 80, G: 70, B: 60, A: 128}
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
//
// **Ce que ces lignes ont dit ensemble, et qui vaut pour la suite : la taille de
// cette fonction se paie en entier.** En ajoutant l'additif, porter le drapeau
// jusqu'à l'arithmétique d'un pixel a coûté 2,8 % en moyenne géométrique sur tous
// les chemins, additif ou non ; le choisir par séquence ne coûte rien de mesurable.
// Mais dupliquer une boucle interne pour sortir ce test a coûté 4,7 % sur le chemin
// visé et 2,4 % sur des chemins qui ne la traversent pas, la fonction passant de
// 2 196 à 2 789 octets compilés.
//
// D'où la règle qu'on relira ici : **un test par séquence est gratuit, du code en
// plus dans cette fonction ne l'est pas.** Les deux mesures se refont par
// `make bench PKG=./raster`, et ces chiffres-là périmeront.
func BenchmarkQuad(b *testing.B) {
	cases := []struct {
		name   string
		adjust func(*render.Quad)
	}{
		{"copie", func(*render.Quad) {}},
		{"teinte", func(q *render.Quad) { q.Tint = render.Color{R: 255, G: 160, B: 160, A: 255} }},
		{"demi-opacité", func(q *render.Quad) { q.Tint = render.Color{R: 128, G: 128, B: 128, A: 128} }},
		{"miroir", func(q *render.Quad) { q.Flags = render.MirrorX }},
		{"découpé de moitié", func(q *render.Quad) { q.X = -16 }},
		{"aplat", func(q *render.Quad) { q.Flags = render.Flat }},

		// L'additif perd la recopie d'une séquence opaque, qui est le chemin le plus
		// court du rastériseur : c'est contre « copie » qu'il se lit, et non contre
		// zéro.
		{"additif", func(q *render.Quad) { q.Flags = render.Additive }},
		{"aplat additif", func(q *render.Quad) { q.Flags = render.Flat | render.Additive }},
	}

	a := testAtlas(b)
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buffer := render.NewBuffer(480, 270)

			q := render.Quad{X: 100, Y: 100, Width: 32, Height: 16, Tint: render.White()}
			c.adjust(&q)
			r := render.DrawList{q}

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
		color render.Color
	}{
		{"transparent", render.Color{}},
		{"couleur", render.Color{R: 20, G: 30, B: 40, A: 255}},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buffer := render.NewBuffer(480, 270)

			b.ResetTimer()
			for range b.N {
				Clear(buffer, c.color)
			}
			framePart(b)
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
			inner := render.NewBuffer(480, 270)
			Clear(inner, render.Color{R: 60, G: 80, B: 60, A: 255})
			out := render.NewBuffer(c.outWidth, c.outHeight)

			b.ResetTimer()
			for range b.N {
				Upscale(inner, out, c.factor, render.Color{A: 255})
			}
			framePart(b)
		})
	}
}

// BenchmarkFrame mesure une image entière : un sol qui couvre la résolution interne
// et deux cents personnages dessus. C'est le seul chiffre qui réponde à la question
// qui compte — combien d'images par seconde le blit laisse au reste du moteur.
func BenchmarkFrame(b *testing.B) {
	a := testAtlas(b)
	buffer := render.NewBuffer(480, 270)

	var r render.DrawList
	for y := -16; y < 270; y += 8 {
		offset := 0
		if (y/8)%2 != 0 {
			offset = 16
		}
		for x := -32; x < 480; x += 32 {
			r = append(r, render.Quad{
				X: int16(x + offset), Y: int16(y),
				Width: 32, Height: 16, Tint: render.White(),
			})
		}
	}
	tiles := len(r)

	for i := range 200 {
		r = append(r, render.Quad{
			X: int16(i * 7 % 460), Y: int16(i * 13 % 240),
			Width: 16, Height: 24, Sheet: 1, Tint: render.White(),
		})
	}

	b.ResetTimer()
	for range b.N {
		Draw(buffer, r, a)
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "images/s")
	b.ReportMetric(float64(tiles), "tuiles/image")
	framePart(b)
}
