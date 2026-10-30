// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'arithmétique d'un pixel : la multiplication arrondie, la modulation par une teinte
// prémultipliée et la composition sur le fond. Quatre fonctions courtes appelées des
// millions de fois par image, d'où l'absence de toute vérification ici.

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

// modulate applique une teinte à un pixel source, composante par composante.
//
// **Les deux couleurs sont prémultipliées, donc l'alpha de la teinte est déjà dans
// ses composantes** : le réappliquer les atténuerait deux fois, et une teinte
// translucide assombrirait le fond au lieu de le voiler. Une teinte qui module un
// blanc opaque rend donc exactement la teinte.
//
// Cette ligne a réappliqué `tint.A` jusqu'au 30 octobre 2026, ce qui traitait la
// teinte comme une couleur droite — composantes en filtre, alpha en opacité — là que
// [render.Color] promet le prémultiplié sur les deux chemins. L'écart était invisible
// à alpha plein, où le second produit est l'identité.
func modulate(src, tint render.Color) render.Color {
	return render.Color{
		R: product(src.R, tint.R),
		G: product(src.G, tint.G),
		B: product(src.B, tint.B),
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

// add ajoute un pixel source à un pixel de destination, chaque composante plafonnant à
// sa valeur pleine.
//
// Le plafond est le mélange, pas une précaution : sans lui la composante reboucle par
// le bas et un éclair noircit ce qu'il éclaire. La somme se fait en seize bits parce
// que c'est le seul endroit où elle peut déborder.
func add(dst *render.Color, src render.Color) {
	dst.R = saturate(dst.R, src.R)
	dst.G = saturate(dst.G, src.G)
	dst.B = saturate(dst.B, src.B)
	dst.A = saturate(dst.A, src.A)
}

// saturate rend la somme de deux composantes, plafonnée à sa valeur pleine.
func saturate(a, b uint8) uint8 {
	sum := uint16(a) + uint16(b)
	if sum > 255 {
		return 255
	}
	return uint8(sum)
}

// put écrit un pixel source sur un pixel de destination, teinte comprise.
//
// Les deux sorties rapides ne sont pas une redite du pré-encodage : une teinte peut
// rendre transparent un pixel opaque, ou opaque un pixel que la planche donnait
// déjà tel quel, et le genre de la séquence ne le sait pas.
//
// **Elle n'apprend rien de l'additif, et c'est voulu.** Le mélange se choisit par
// séquence, pas par pixel : lui passer un drapeau de plus ajoutait un argument et un
// test à chaque appel, pour deux pour cent sur un chemin parcouru des millions de fois
// par image — mesuré, puis rendu.
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

// putAdd ajoute un pixel source à un pixel de destination, teinte comprise.
//
// Le pendant additif de [put], séparé plutôt que paramétré. Il n'a qu'une sortie
// rapide : une source transparente n'apporte rien, ses composantes étant nulles avec
// son alpha en prémultiplié, mais une source opaque s'ajoute au lieu de remplacer.
func putAdd(dst *render.Color, src, tint render.Color, neutral bool) {
	if !neutral {
		src = modulate(src, tint)
	}
	if src.A == 0 {
		return
	}
	add(dst, src)
}
