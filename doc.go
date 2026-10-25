// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package lozengine est la surface publique du moteur, et la seule.
//
// Tout le reste vit sous internal/, que le compilateur rend impossible à importer
// depuis un autre module — sauf [github.com/sprimault/lozengine/rendu], dont un hôte
// doit pouvoir lire les champs : ses structures seront un jour lues par du code écrit
// dans un autre langage, selon la convention d'appel du C. Publier demande donc de
// l'écrire, au lieu de l'être par défaut : c'est ce qui rend tenable la règle voulant
// que toute signature publique soit définitive.
//
// Les types du noyau sont republiés par alias et non par enveloppe. Un alias
// désigne le même type, sans conversion ni copie — ce qui compte pour des
// structures qu'un hôte reçoit par milliers à chaque image et qui doivent garder
// leur disposition compatible C.
//
// Deux portes d'entrée. Pour écrire un jeu en Go, ce paquet tiendra la fenêtre, la
// boucle à pas fixe et les entrées : il ne restera qu'une méthode à écrire, l'ordre
// de dessin venant de la clé de tri et l'échelle du facteur entier. Pour intégrer
// le moteur dans un jeu qui a déjà sa boucle, le noyau s'appellera directement, sans
// réclamer ni fenêtre ni thread. La première porte ajoute du confort, jamais de
// capacité : tout ce qu'elle permet se fait aussi par la seconde.
//
// Aucune des deux n'est encore atteignable. Ce paquet ne publie que la projection et
// les structures du rendu : le moteur n'a ni fenêtre, ni boucle, et rien n'y suffit
// à écrire un jeu.
package lozengine
