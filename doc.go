// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package lozengine est la porte d'entrée du moteur : un seul import pour écrire un
// jeu, les types du noyau étant republiés ici.
//
// Cinq paquets l'accompagnent, et ils sont publics pour être lus : geometry porte la
// projection, scene la description du monde, render les structures qu'un quad
// remplit, raster les planches et le dessin, input le clavier. Un alias ne montre ni
// champ ni méthode, donc la documentation de [Matrix], de [Quad] ou de [Scene] se lit
// chez eux. Le reste vit sous internal/, que le compilateur rend impossible à
// importer depuis un autre module.
//
// Les types du noyau sont republiés par alias et non par enveloppe. Un alias
// désigne le même type, sans conversion ni copie — ce qui compte pour des
// structures qu'un hôte reçoit par milliers à chaque image et qui doivent garder
// leur disposition compatible C.
//
// Deux portes d'entrée. Pour écrire un jeu en Go, [Run] ouvre la fenêtre et tient la
// boucle à pas fixe : il ne reste qu'une méthode à écrire, [Game.Step], l'ordre de
// dessin venant de la clé de tri et l'échelle du facteur entier. Pour intégrer le
// moteur dans un jeu qui a déjà sa boucle, [NewEngine] rend la même capacité sans
// fenêtre ni horloge, un [Engine.Step] par image. La première porte ajoute du
// confort, jamais de capacité : tout ce qu'elle permet se fait aussi par la seconde.
//
// La fenêtre, elle, n'existe que sous Windows : ailleurs [Run] rend une erreur qui
// nomme ce qui manque, là où [NewEngine] calcule ses images partout.
//
// MIT ou Apache-2.0, au choix de qui emploie le moteur : l'une des deux suffit, elles
// ne se cumulent pas. Les annuaires de paquets les affichent côte à côte sans le
// « ou », ce qui est une façon de les indexer et non une exigence double.
package lozengine
