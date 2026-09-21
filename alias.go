// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import "github.com/sprimault/lozengine/internal/rendu"

// Quad est la primitive unique du moteur : un rectangle de planche posé dans le
// tampon, déjà projeté, cullé et trié. Sa disposition est compatible C.
type Quad = rendu.Quad

// Rendu est la suite ordonnée des quads d'une image : l'ordre du tableau est
// l'ordre de dessin.
type Rendu = rendu.Rendu

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
// celle-ci disparaît à l'inlining.
func Blanc() Couleur { return rendu.Blanc() }
