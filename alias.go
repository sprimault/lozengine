// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"github.com/sprimault/lozengine/internal/geometrie"
	"github.com/sprimault/lozengine/rendu"
)

// Quad est la primitive unique du moteur : un rectangle de planche posé dans le
// tampon, déjà placé à l'écran et rangé dans l'ordre de dessin. Sa disposition est
// compatible C.
type Quad = rendu.Quad

// Rendu est la suite ordonnée des quads d'une image : l'ordre du tableau est
// l'ordre de dessin.
type Rendu = rendu.Rendu

// Tampon est l'image dans laquelle le moteur écrit : des pixels contigus, ligne par
// ligne, prémultipliés comme ceux d'une planche.
//
// Un hôte peut l'allouer lui-même et laisser le moteur écrire dedans, ce qui évite
// d'avoir à négocier qui possède cette mémoire.
type Tampon = rendu.Tampon

// BorneTampon est le plus grand côté qu'un tampon puisse avoir : la plus grande
// coordonnée qu'un [Quad] sait désigner.
const BorneTampon = rendu.BorneTampon

// NouveauTampon rend un tampon transparent aux dimensions demandées, et panique
// au-delà de [BorneTampon] de côté — une taille pareille signale un calcul fautif en
// amont plutôt qu'une donnée du jeu.
//
// Une fonction et non un alias, pour la raison qui vaut pour [Blanc].
func NouveauTampon(largeur, hauteur uint16) *Tampon { return rendu.NouveauTampon(largeur, hauteur) }

// Couleur est une couleur RVBA sur quatre octets, à alpha prémultiplié.
type Couleur = rendu.Couleur

// Drapeaux porte les variantes de dessin d'un quad, un bit par variante.
type Drapeaux = rendu.Drapeaux

const (
	// MiroirX lit la source de droite à gauche.
	MiroirX = rendu.MiroirX

	// MiroirY lit la source de bas en haut.
	MiroirY = rendu.MiroirY

	// Aplat peint le rectangle en Teinte sans lire aucune source.
	Aplat = rendu.Aplat
)

// Blanc rend la teinte neutre, celle qui laisse la source intacte.
//
// Une fonction et non un alias : seuls les types se republient sans enveloppe, et
// celle-ci ne coûte rien — le compilateur la remplace par son corps.
func Blanc() Couleur { return rendu.Blanc() }

// Matrice convertit entre les cellules du monde et les pixels de l'écran, dans les
// deux sens, et garde l'inverse qu'elle a calculée une fois.
//
// Le calcul est entier de bout en bout, ce qui fait qu'une même scène rend les mêmes
// pixels d'une machine à l'autre : les nombres à virgule flottante ne le
// garantiraient pas, la spécification du langage autorisant à fusionner une
// multiplication suivie d'une addition en une instruction unique que les
// architectures ne pratiquent pas toutes.
//
// Une matrice s'obtient d'un des quatre constructeurs qui suivent, republiés en
// fonctions pour la raison qui vaut pour [Blanc], et s'emploie par ses méthodes.
// L'ordre de dessin n'en vient pas : c'est la clé de tri qui le porte.
type Matrice = geometrie.Matrice

// Dimetrique rend la projection d'une cellule en losange au ratio libre, celle d'un
// rendu isométrique de jeu : une cellule de 32 sur 16 avance de seize pixels vers le
// côté et de huit vers le bas.
//
// C'est le constructeur du cas courant, et le seul qui laisse la hauteur libre.
func Dimetrique(largeur, hauteur int) Matrice { return geometrie.Dimetrique(largeur, hauteur) }

// Isometrique rend la projection de l'isométrie vraie, dont la hauteur ne se choisit
// pas : elle vaut la largeur divisée par racine de trois, soit dix-huit pixels et
// demi pour une largeur de trente-deux là où le dimétrique courant en prend seize.
func Isometrique(largeur int) Matrice { return geometrie.Isometrique(largeur) }

// Grille rend la projection de la vue de dessus, où une cellule est un carré et non
// un losange. Les axes n'y tournent pas, ce qu'aucun ratio de [Dimetrique] ne donne.
func Grille(cote int) Matrice { return geometrie.Grille(cote) }

// Brute rend la projection de quatre coefficients donnés, pour ce qu'aucun des trois
// autres constructeurs n'exprime : l'abscisse écran vaut a·x + b·y et l'ordonnée
// c·x + d·y. Sans translation, la vue étant cadrée ailleurs.
func Brute(a, b, c, d float64) Matrice { return geometrie.Brute(a, b, c, d) }
