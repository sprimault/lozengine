// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La suite ordonnée des quads d'une image. Un tableau nu, parce qu'un hôte la reçoit
// par pointeur et longueur.

package render

// DrawList est la suite ordonnée des quads d'une image : l'ordre du tableau est
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
// **Les quads ne sont pas groupés par planche, et aucun hôte ne doit le supposer.**
// Ils l'étaient jusqu'à la 0.5.3, la clé de tri réservant un champ à l'index de
// planche sous la hauteur. Ce champ décidait l'ordre de dessin là où il n'avait rien à
// décider : à profondeur égale, c'est l'ordre de chargement des images qui tranchait.
// Il porte depuis le rang d'empilement, et le groupement n'était de toute façon qu'un
// effet de bord — le rastériseur relit sa planche à chaque quad, sans garder d'état de
// l'un à l'autre.
type DrawList []Quad
