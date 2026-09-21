// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

// Drapeaux porte les variantes de dessin d'un quad, un bit par variante.
type Drapeaux uint8

const (
	// MiroirX lit la source de droite à gauche. C'est ce qui permet à un
	// fournisseur d'apparence de ne dessiner que la moitié de ses directions :
	// l'est se tire de l'ouest sans rien ajouter à l'atlas.
	MiroirX Drapeaux = 1 << iota

	// MiroirY lit la source de bas en haut.
	MiroirY

	// Aplat peint le rectangle en Teinte sans lire aucune source ; Planche,
	// SourceX et SourceY sont alors ignorés.
	//
	// Sans lui, un panneau d'interface ou un fondu au noir demanderait d'étirer
	// un texel, c'est-à-dire la mise à l'échelle fractionnaire que le moteur
	// s'interdit. Le quad reste la primitive unique, et c'est son chemin le plus
	// court.
	Aplat
)

// Quad est la primitive unique du moteur : un rectangle de planche posé dans le
// tampon, déjà projeté, cullé et trié.
//
// Tout converge ici et rien n'en sort — les fonctions de confort produisent des
// quads, jamais un second chemin de dessin. Un quad ne porte pas sa clé de tri :
// un Rendu est ordonné par construction, et la répartition des bits de la clé n'a
// pas à traverser la frontière C, où elle deviendrait irrévocable.
//
// Teinte multiplie la source composante par composante, 255 étant neutre, et son
// alpha multiplie en plus l'opacité. À blanc opaque, le rastériseur retombe sur
// la copie directe : la modulation se décide une fois par quad et non par pixel.
//
// La disposition est celle qu'un hôte écrirait en C — champs de taille fixe,
// vingt octets, alignement de deux. D'où trois limites, documentées plutôt que
// vérifiées à chaque image : un tampon et une planche de 32 767 pixels de côté,
// et 1 024 planches, ce dernier chiffre venant des dix bits que la clé de tri
// réserve à la planche.
//
// La valeur zéro ne dessine rien, ce qui est l'échec sûr : une structure remplie
// de zéros par un hôte reste invisible au lieu de peindre n'importe quoi.
type Quad struct {
	X, Y             int16   // destination dans le tampon, coin haut-gauche
	SourceX, SourceY uint16  // coin haut-gauche dans la planche
	Largeur, Hauteur uint16  // taille, commune à la source et à la destination
	Planche          uint16  // index dans l'atlas que le rastériseur reçoit à côté
	Teinte           Couleur // modulation, voir ci-dessus
	Drapeaux         Drapeaux
	_                uint8 // complète à vingt octets ; explicite pour l'en-tête C
}
