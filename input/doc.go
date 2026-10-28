// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package input porte ce qu'un jeu lit de ses entrées à l'image courante : quelles
// touches et quels boutons sont enfoncés, lesquels viennent de l'être, où se trouve le
// pointeur et de combien la molette a tourné.
//
// Une position, jamais un caractère. Une [Key] désigne l'endroit où la touche se trouve
// sur le clavier, indépendamment de la disposition active : la touche nommée W est
// celle marquée Z sur un clavier français, et les quatre touches de déplacement gardent
// leur losange partout. C'est ce que la documentation du système appelle un code de
// balayage. Un caractère, lui, dépendrait de la langue, et le jeu qui s'y fierait
// demanderait à un joueur français d'écarter les doigts.
//
// Deux régimes, parce que la différence compte. L'état maintenu sert au déplacement :
// tant que la touche est tenue, le personnage avance. Le front montant sert à ce qui se
// consomme — tirer, ramasser, valider : au maintien, l'action partirait à l'image où le
// doigt se pose et se répéterait soixante fois par seconde, et le joueur ne saurait
// jamais s'il l'a dépensée exprès.
//
// Deux types, parce que deux côtés. [State] est celui du backend, qui relève les
// événements du système et les accumule d'une image à l'autre. [Input] est ce qu'un jeu
// reçoit : une valeur, qu'il lit et ne peut pas modifier.
package input
