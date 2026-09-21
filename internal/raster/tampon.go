// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "github.com/sprimault/lozengine/internal/rendu"

// Tampon est l'image dans laquelle le rastériseur écrit : des pixels contigus,
// ligne par ligne, sans remplissage, prémultipliés comme ceux d'une planche.
//
// Aucune taille n'est imposée ici. La résolution interne du moteur est un
// paramètre du paquet racine, et la même structure sert au tampon de sortie
// qu'un backend présente.
type Tampon struct {
	Largeur, Hauteur uint16
	Pixels           []rendu.Couleur
}

// NouveauTampon rend un tampon transparent aux dimensions demandées.
func NouveauTampon(largeur, hauteur uint16) *Tampon {
	return &Tampon{
		Largeur: largeur,
		Hauteur: hauteur,
		Pixels:  make([]rendu.Couleur, int(largeur)*int(hauteur)),
	}
}

// Effacer remet tout le tampon à une couleur, transparente comprise.
//
// C'est la première opération de chaque image, et elle n'alloue rien : le tampon se
// réutilise d'une image à la suivante. Le cas transparent passe par clear, que le
// compilateur ramène à une mise à zéro en bloc.
//
// La couleur, elle, passe par un doublement : on pose le premier pixel, puis on
// recopie sur lui-même en doublant la portion écrite. Une boucle d'affectation ne se
// vectorise pas et coûtait douze fois plus, mesuré — 74 µs contre 6 sur la
// résolution interne, soit près d'un tiers du temps de dessin d'une image entière.
func (t *Tampon) Effacer(c rendu.Couleur) {
	remplir(t.Pixels, c)
}

// remplir écrit une couleur sur toute une tranche de pixels.
//
// Le doublement vaut pour une tranche d'un seul tenant et de bonne taille. Sur des
// segments courts et nombreux — les bandes latérales d'une image mise à l'échelle —
// le coût d'appel de copy l'emporte, et l'appelant recopie plutôt une ligne modèle.
func remplir(pixels []rendu.Couleur, c rendu.Couleur) {
	if c == (rendu.Couleur{}) {
		clear(pixels)
		return
	}
	if len(pixels) == 0 {
		return
	}

	pixels[0] = c
	for ecrit := 1; ecrit < len(pixels); ecrit *= 2 {
		copy(pixels[ecrit:], pixels[:ecrit])
	}
}
