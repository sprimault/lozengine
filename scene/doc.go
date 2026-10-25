// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package scene décrit ce qu'une image contient : une grille de décor, des entités qui
// s'y déplacent, et la vue par laquelle on les regarde.
//
// Une scène se décrit, elle ne se dessine pas. Rien ici ne place quoi que ce soit à
// l'écran : c'est le tri qui s'en charge, et c'est ce qui permet au moteur de décider
// l'ordre de dessin au lieu de le demander au jeu.
//
// Deux chemins, parce que deux coûts. Le décor est posé par cellule et ne se relit que
// sur la fenêtre visible, donc une carte de cinq cents cellules de côté coûte autant
// qu'une de cinquante. Les entités, elles, se parcourent en entier, parce qu'elles sont
// presque toutes à l'image.
//
// Rien ici ne connaît la résolution ni la taille d'une tuile : la première arrive avec
// le cadrage, la seconde vit dans la matrice que l'appelant fournit.
package scene
