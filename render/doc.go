// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package render porte les structures que le noyau produit et que tout backend
// consomme : le quad projeté, la liste ordonnée qui les rassemble, et l'image dans
// laquelle ils se peignent.
//
// Il ne connaît ni Windows, ni X11, ni aucun format d'image. C'est ce qui permet au
// rastériseur — celui qui peint les quads, pixel par pixel, sans carte graphique — et
// aux backends système de consommer exactement la même chose, et donc à la suite de
// non-régression de juger le rendu sans jamais ouvrir de fenêtre.
//
// [Quad] et [DrawList], [Overlap] et [OverlapList] gardent une disposition compatible
// C : champs de taille fixe, tableau contigu, ni chaîne ni interface ni slice
// imbriquée. Un cas du paquet fige la taille et les décalages des deux structures, et
// la contiguïté des deux listes — sans quoi l'énumération ci-dessus serait la seule
// garde, et la taille d'`Overlap` a doublé entre deux versions sans que rien ne le
// voie. La contrainte vient de
// ce que ces structures seront un jour lues par du code écrit dans un autre langage,
// selon la convention d'appel du C, et elle vaut avant même que la bibliothèque
// native existe — une chaîne dans un quad imposerait une conversion par quad et par
// image, ce qui rendrait cette exposition inutilisable.
//
// Elle ne porte pas sur [Buffer], qui n'est pas de la donnée par quad : un hôte en
// reçoit les dimensions et le pointeur des pixels, une fois par image.
package render
