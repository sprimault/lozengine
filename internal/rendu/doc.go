// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package rendu porte les structures que le noyau produit et que tout backend
// consomme : le quad projeté, et la liste ordonnée qui les rassemble.
//
// Il ne connaît ni Windows, ni X11, ni aucun format d'image. C'est ce qui permet
// au rastériseur logiciel et aux backends système de consommer exactement la même
// chose, et donc à la suite de non-régression de juger le rendu sans jamais
// ouvrir de fenêtre.
//
// Ses types gardent une disposition compatible C : champs de taille fixe, tableau
// contigu, ni chaîne ni interface ni slice imbriquée. La contrainte vient de
// l'export vers d'autres langages et vaut avant même que la bibliothèque native
// existe — une chaîne dans un quad imposerait une conversion par quad et par
// image, ce qui rendrait cette exposition inutilisable.
package rendu
