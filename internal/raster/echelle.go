// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "github.com/sprimault/lozengine/internal/rendu"

// Facteur rend le plus grand facteur entier qui fait tenir le tampon dans la sortie,
// et jamais moins de 1.
//
// C'est le défaut, pas une obligation : un jeu peut vouloir plafonner
// l'agrandissement plutôt que d'afficher des pixels de sept pixels de côté sur un
// écran 4K. Le choix lui appartient, [Tampon.Agrandir] prenant le facteur en
// paramètre.
func (t *Tampon) Facteur(sortie *Tampon) int {
	if t.Largeur == 0 || t.Hauteur == 0 {
		return 1
	}
	return max(min(int(sortie.Largeur)/int(t.Largeur), int(sortie.Hauteur)/int(t.Hauteur)), 1)
}

// Agrandir recopie le tampon dans la sortie, centré, au facteur entier donné, en
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
// Une sortie plus petite que le tampon ne fait pas échouer l'appel : le facteur
// tombe à 1 et l'image se découpe, toujours centrée. Un redimensionnement de fenêtre
// n'a pas à être un cas d'erreur.
func (t *Tampon) Agrandir(sortie *Tampon, facteur int, fond rendu.Couleur) {
	if facteur < 1 {
		facteur = 1
	}

	large, haut := int(t.Largeur)*facteur, int(t.Hauteur)*facteur
	if facteur == 1 && t.Largeur == sortie.Largeur && t.Hauteur == sortie.Hauteur {
		copy(sortie.Pixels, t.Pixels)
		return
	}

	// Rien à peindre autour quand la sortie tombe juste, ni quand elle est plus
	// petite : deux comparaisons plutôt que deux millions de pixels réécrits.
	if large < int(sortie.Largeur) || haut < int(sortie.Hauteur) {
		sortie.Effacer(fond)
	}

	decalageX := (int(sortie.Largeur) - large) / 2
	decalageY := (int(sortie.Hauteur) - haut) / 2
	bordA := max(decalageX, 0)
	bordB := min(decalageX+large, int(sortie.Largeur))
	if bordA >= bordB {
		return
	}

	for sy := 0; sy < int(t.Hauteur); sy++ {
		premiere := max(decalageY+sy*facteur, 0)
		derniere := min(decalageY+(sy+1)*facteur, int(sortie.Hauteur))
		if premiere >= derniere {
			continue
		}

		src := t.Pixels[sy*int(t.Largeur) : (sy+1)*int(t.Largeur)]
		dst := sortie.Pixels[premiere*int(sortie.Largeur) : (premiere+1)*int(sortie.Largeur)]

		for sx := range src {
			a := max(decalageX+sx*facteur, bordA)
			b := min(decalageX+(sx+1)*facteur, bordB)
			for x := a; x < b; x++ {
				dst[x] = src[sx]
			}
		}

		// Les lignes suivantes du même pixel source sont la première, à l'identique.
		for y := premiere + 1; y < derniere; y++ {
			ligne := sortie.Pixels[y*int(sortie.Largeur) : (y+1)*int(sortie.Largeur)]
			copy(ligne[bordA:bordB], dst[bordA:bordB])
		}
	}
}
