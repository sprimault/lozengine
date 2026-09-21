// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

// Rendu est la suite ordonnée des quads d'une image : l'ordre du tableau est
// l'ordre de dessin.
//
// C'est tout ce qu'un backend a le droit d'en savoir. S'il lui faut trier,
// projeter ou consulter la scène, la frontière est mal placée et le correctif
// appartient au noyau.
//
// Un tableau, et non une structure qui en porterait un : la contrainte de
// disposition C veut les quads contigus, et un hôte en reçoit le pointeur du
// premier avec leur nombre, sans rien à convertir ni aucune propriété mémoire à
// négocier.
//
// Les quads d'une même profondeur sortent groupés par planche, la clé de tri lui
// réservant un champ sous z. Le rastériseur peut n'en changer que par séries ; il
// ne doit pas en dépendre.
type Rendu []Quad
