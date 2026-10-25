// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'écriture d'un tampon en PNG, et la division par l'alpha qu'elle impose — le format
// veut des composantes non prémultipliées. C'est ce qui fait perdre un cran aux
// couleurs à alpha partiel, et pourquoi une image de référence est opaque.

package raster

import (
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/sprimault/lozengine/render"
)

// WritePNG écrit le tampon en PNG.
//
// C'est ce qui rend la suite de non-régression exécutable sans écran : une scène
// scriptée, un rendu, une image, et une comparaison au pixel avec une référence.
//
// Le PNG stocke ses composantes non prémultipliées et l'encodeur de la bibliothèque
// standard fait la conversion depuis image.RGBA, qui l'est. Ne pas la refaire à la
// main : la division par l'alpha est exactement l'endroit où deux arrondis
// divergent, et une référence versionnée n'a pas à dépendre du nôtre.
//
// L'aller-retour n'est donc pas l'identité : écrire puis recharger un pixel dont
// l'alpha est partiel lui coûte jusqu'à un cran par composante. Mesuré sur les
// 32 896 couples prémultipliés valides, un tiers le perdent, et jamais plus d'un
// cran. Un tampon opaque, lui, revient exact — c'est ce dont la suite de
// non-régression dépend, et pourquoi ses scénarios rendent des images opaques.
func WritePNG(t *render.Buffer, w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, int(t.Width), int(t.Height)))
	for i, c := range t.Pixels {
		o := i * 4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, c.A
	}

	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("writing PNG: %w", err)
	}
	return nil
}
