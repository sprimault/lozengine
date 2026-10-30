// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'agrandissement vers la fenêtre : un facteur entier, et des bandes autour quand la
// taille ne tombe pas juste. Jamais de mise à l'échelle fractionnaire, qui brouillerait
// des pixels dessinés un par un.

package raster

import "github.com/sprimault/lozengine/render"

// ScaleFactor rend le plus grand facteur entier qui fait tenir le tampon dans la sortie,
// et jamais moins de 1.
//
// C'est le défaut, pas une obligation : un jeu peut vouloir plafonner
// l'agrandissement plutôt que d'afficher des pixels de sept pixels de côté sur un
// écran 4K. Le choix lui appartient, [Upscale] prenant le facteur en paramètre.
//
// Une fonction et non une méthode : le tampon vit dans [render], que l'ABI expose, et
// Go veut qu'une méthode soit déclarée dans le paquet de son type. Vaut pour
// [Upscale] comme pour celle-ci.
func ScaleFactor(t, out *render.Buffer) int {
	if t.Width == 0 || t.Height == 0 {
		return 1
	}
	return max(min(int(out.Width)/int(t.Width), int(out.Height)/int(t.Height)), 1)
}

// Upscale recopie le tampon dans la sortie, centré, au facteur entier donné, en
// peignant de fond ce qui reste autour.
//
// Jamais de facteur fractionnaire : à cette résolution, un pixel et demi donne des
// colonnes inégales que l'œil suit d'un bout à l'autre de l'écran. D'où les bandes,
// qui sont le prix assumé de cette règle.
//
// Le reste d'un centrage impair va à droite et en bas. Ce n'est pas indifférent :
// une image de référence doit être reproductible au pixel, donc l'arbitraire se
// documente au lieu de se découvrir.
//
// Une sortie plus petite que le tampon ne fait pas échouer l'appel : **c'est le
// découpage qui opère, pas le facteur**, qui reste celui qu'on a donné. L'image est
// rognée par les bords de la sortie, toujours centrée. Un redimensionnement de fenêtre
// n'a pas à être un cas d'erreur.
//
// Seul un facteur inférieur à 1 est remonté à 1 : il n'y a pas de réduction ici, la
// mise à l'échelle entière ne sachant qu'agrandir.
func Upscale(t, out *render.Buffer, factor int, background render.Color) {
	if factor < 1 {
		factor = 1
	}

	scaledW, scaledH := int(t.Width)*factor, int(t.Height)*factor
	if factor == 1 && t.Width == out.Width && t.Height == out.Height {
		copy(out.Pixels, t.Pixels)
		return
	}

	offsetX := (int(out.Width) - scaledW) / 2
	offsetY := (int(out.Height) - scaledH) / 2
	leftBound := max(offsetX, 0)
	rightBound := min(offsetX+scaledW, int(out.Width))
	topBound := max(offsetY, 0)
	bottomBound := min(offsetY+scaledH, int(out.Height))

	if leftBound >= rightBound || topBound >= bottomBound {
		fill(out.Pixels, background)
		return
	}

	// Rien à peindre autour quand la sortie tombe juste, ni quand elle est plus
	// petite : deux comparaisons plutôt qu'une sortie entière réécrite.
	if scaledW < int(out.Width) || scaledH < int(out.Height) {
		paintBars(out, leftBound, topBound, rightBound, bottomBound, background)
	}

	for sy := 0; sy < int(t.Height); sy++ {
		first := max(offsetY+sy*factor, 0)
		last := min(offsetY+(sy+1)*factor, int(out.Height))
		if first >= last {
			continue
		}

		src := t.Pixels[sy*int(t.Width) : (sy+1)*int(t.Width)]
		dst := out.Pixels[first*int(out.Width) : (first+1)*int(out.Width)]

		for sx := range src {
			a := max(offsetX+sx*factor, leftBound)
			b := min(offsetX+(sx+1)*factor, rightBound)
			for x := a; x < b; x++ {
				dst[x] = src[sx]
			}
		}

		// Les lignes suivantes du même pixel source sont la première, à l'identique.
		for y := first + 1; y < last; y++ {
			row := out.Pixels[y*int(out.Width) : (y+1)*int(out.Width)]
			copy(row[leftBound:rightBound], dst[leftBound:rightBound])
		}
	}
}

// paintBars peint ce qui reste autour de l'image, aux quatre bords, et rien
// d'autre. Les quatre bornes délimitent l'image dans le tampon.
//
// Effacer toute la sortie avant d'écrire l'image serait plus court de dix lignes et
// coûterait 225 µs par image sur une sortie 1920×1080 : le centre, huit
// mégaoctets, y serait écrit deux fois. L'opération est limitée par la bande
// passante mémoire, donc ce qu'on n'écrit pas est gagné tel quel.
//
// Les bandes latérales se recopient d'une ligne modèle au lieu de se remplir chacune :
// sur des segments courts, le coût d'appel de copy dépasse ce qu'il écrit.
func paintBars(t *render.Buffer, x0, y0, x1, y1 int, background render.Color) {
	width := int(t.Width)

	fill(t.Pixels[:y0*width], background)
	fill(t.Pixels[y1*width:], background)

	if x0 <= 0 && x1 >= width {
		return
	}

	model := t.Pixels[y0*width : (y0+1)*width]
	fill(model[:x0], background)
	fill(model[x1:], background)
	for y := y0 + 1; y < y1; y++ {
		row := t.Pixels[y*width : (y+1)*width]
		copy(row[:x0], model[:x0])
		copy(row[x1:], model[x1:])
	}
}
