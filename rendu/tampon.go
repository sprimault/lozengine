// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

import (
	"fmt"
	"math"
)

// BorneTampon est le plus grand côté qu'un tampon puisse avoir : c'est la plus
// grande coordonnée qu'un [Quad] sait désigner, ses champs de position étant des
// int16. Au-delà, le tampon aurait une part qu'aucun quad ne peut atteindre.
const BorneTampon = math.MaxInt16

// Tampon est l'image dans laquelle le moteur écrit : des pixels contigus, ligne par
// ligne, sans remplissage, prémultipliés comme ceux d'une planche.
//
// Il vit dans ce paquet et non dans le rastériseur parce qu'un hôte en remplit les
// champs. Le contrat d'exposition aux autres langages est que l'hôte alloue l'image
// et que le moteur écrive dedans : il n'y a alors aucune propriété de mémoire à
// négocier. C'est aussi ce qui permet à un backend de nommer le type, lui qui n'a
// droit qu'à ce paquet.
//
// Seul le côté est borné, par BorneTampon. La résolution interne du moteur reste un
// paramètre du paquet racine, et la même structure sert au tampon de sortie qu'un
// backend présente.
type Tampon struct {
	Largeur, Hauteur uint16
	Pixels           []Couleur
}

// NouveauTampon rend un tampon transparent aux dimensions demandées.
//
// Au-delà de BorneTampon de côté, il panique, et c'est une violation d'invariant de
// programmation plutôt qu'une donnée du jeu : aucun écran n'en approche la moitié,
// donc un dépassement signale un calcul fautif en amont. Le laisser passer serait
// pire que s'arrêter — le tampon aurait une part qu'aucun quad ne sait désigner, où
// les quads lointains arriveraient repliés au milieu de l'image.
func NouveauTampon(largeur, hauteur uint16) *Tampon {
	if largeur > BorneTampon || hauteur > BorneTampon {
		panic(fmt.Sprintf("tampon de %d×%d : %d de côté au plus, la plus grande coordonnée qu'un quad désigne", largeur, hauteur, BorneTampon))
	}

	return &Tampon{
		Largeur: largeur,
		Hauteur: hauteur,
		Pixels:  make([]Couleur, int(largeur)*int(hauteur)),
	}
}
