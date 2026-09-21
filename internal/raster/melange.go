// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "github.com/sprimault/lozengine/internal/rendu"

// produit rend l'arrondi exact de a × b / 255 sans division.
//
// La division entière tronque, et l'erreur se voit : un blanc opaque modulé par un
// blanc opaque rendrait 254, et un dégradé se décalerait d'un cran sur toute sa
// longueur. Les deux décalages reconstituent la division par 255 à partir de
// celle par 256, que le processeur fait gratuitement.
func produit(a, b uint8) uint8 {
	t := uint32(a)*uint32(b) + 128
	return uint8((t + t>>8) >> 8)
}

// moduler applique une teinte à un pixel source.
//
// L'alpha de la teinte agit sur les quatre composantes, celles de couleur étant
// prémultipliées : les amputer sans toucher l'alpha assombrirait le pixel au lieu
// de le rendre translucide.
func moduler(src, teinte rendu.Couleur) rendu.Couleur {
	return rendu.Couleur{
		R: produit(produit(src.R, teinte.R), teinte.A),
		V: produit(produit(src.V, teinte.V), teinte.A),
		B: produit(produit(src.B, teinte.B), teinte.A),
		A: produit(src.A, teinte.A),
	}
}

// melanger compose un pixel source sur un pixel de destination.
//
// C'est la seule opération que le prémultiplié rend aussi courte : source plus
// destination atténuée, sans division ni multiplication par l'alpha de la source.
func melanger(dst *rendu.Couleur, src rendu.Couleur) {
	reste := 255 - src.A
	dst.R = src.R + produit(dst.R, reste)
	dst.V = src.V + produit(dst.V, reste)
	dst.B = src.B + produit(dst.B, reste)
	dst.A = src.A + produit(dst.A, reste)
}

// poser écrit un pixel source sur un pixel de destination, teinte comprise.
//
// Les deux sorties rapides ne sont pas une redite du pré-encodage : une teinte peut
// rendre transparent un pixel opaque, ou opaque un pixel que la planche donnait
// déjà tel quel, et le genre de la séquence ne le sait pas.
func poser(dst *rendu.Couleur, src, teinte rendu.Couleur, neutre bool) {
	if !neutre {
		src = moduler(src, teinte)
	}
	switch src.A {
	case 0:
		return
	case 255:
		*dst = src
	default:
		melanger(dst, src)
	}
}
