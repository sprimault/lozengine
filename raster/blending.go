// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'arithmétique d'un pixel : la multiplication arrondie, la modulation par une teinte
// et la composition sur le fond. Quatre fonctions courtes appelées des millions de fois
// par image, d'où l'absence de toute vérification ici.

package raster

import "github.com/sprimault/lozengine/render"

// product rend l'arrondi exact de a × b / 255 sans division.
//
// La division entière tronque, et l'erreur se voit : un blanc opaque modulé par un
// blanc opaque rendrait 254, et un dégradé se décalerait d'un cran sur toute sa
// longueur. Les deux décalages reconstituent la division par 255 à partir de
// celle par 256, que le processeur fait gratuitement.
func product(a, b uint8) uint8 {
	t := uint32(a)*uint32(b) + 128
	return uint8((t + t>>8) >> 8)
}

// modulate applique une teinte à un pixel source.
//
// L'alpha de la teinte agit sur les quatre composantes, celles de couleur étant
// prémultipliées : les amputer sans toucher l'alpha assombrirait le pixel au lieu
// de le rendre translucide.
func modulate(src, tint render.Color) render.Color {
	return render.Color{
		R: product(product(src.R, tint.R), tint.A),
		G: product(product(src.G, tint.G), tint.A),
		B: product(product(src.B, tint.B), tint.A),
		A: product(src.A, tint.A),
	}
}

// blend compose un pixel source sur un pixel de destination.
//
// C'est la seule opération que le prémultiplié rend aussi courte : source plus
// destination atténuée, sans division ni multiplication par l'alpha de la source.
func blend(dst *render.Color, src render.Color) {
	rest := 255 - src.A
	dst.R = src.R + product(dst.R, rest)
	dst.G = src.G + product(dst.G, rest)
	dst.B = src.B + product(dst.B, rest)
	dst.A = src.A + product(dst.A, rest)
}

// put écrit un pixel source sur un pixel de destination, teinte comprise.
//
// Les deux sorties rapides ne sont pas une redite du pré-encodage : une teinte peut
// rendre transparent un pixel opaque, ou opaque un pixel que la planche donnait
// déjà tel quel, et le genre de la séquence ne le sait pas.
func put(dst *render.Color, src, tint render.Color, neutral bool) {
	if !neutral {
		src = modulate(src, tint)
	}
	switch src.A {
	case 0:
		return
	case 255:
		*dst = src
	default:
		blend(dst, src)
	}
}
