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
