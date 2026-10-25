// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/sprimault/lozengine/rendu"
)

// EcrirePNG écrit le tampon en PNG.
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
func EcrirePNG(t *rendu.Tampon, w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, int(t.Largeur), int(t.Hauteur)))
	for i, c := range t.Pixels {
		o := i * 4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.V, c.B, c.A
	}

	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("écriture du PNG : %w", err)
	}
	return nil
}
