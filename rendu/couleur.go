// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

// Couleur est une couleur RVBA sur quatre octets, à alpha prémultiplié.
//
// Le prémultiplié ramène le mélange d'un pixel à une multiplication et une
// addition par composante. Le PNG, lui, stocke ses composantes non
// prémultipliées : la conversion appartient au chargement de la planche, jamais
// au chemin de rendu.
type Couleur struct{ R, V, B, A uint8 }

// Blanc rend la teinte neutre, celle qui laisse la source intacte.
//
// Une fonction plutôt qu'une variable de paquet : la seconde se laisserait
// changer sous les pieds de tous ceux qui la lisent.
func Blanc() Couleur { return Couleur{255, 255, 255, 255} }
