// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package backend ouvre une fenêtre, lit le clavier et pose des pixels à l'écran.
//
// Il y a un backend par famille de plateformes — les appels système de Windows, le
// protocole X11, le canvas du navigateur — et tous remplissent le même contrat, celui
// de [Window]. Rien d'autre ne les distingue vu d'en haut, ce qui est la condition
// pour que la boucle ne connaisse aucune plateforme.
//
// **Un backend ne trie ni ne projette.** Il reçoit un tampon déjà peint et le présente
// tel quel, sans l'étirer : l'agrandissement entier et ses bandes sont faits en amont,
// par le rastériseur, qu'une image de référence couvre. S'il devait consulter la scène
// ou décider d'un ordre, la frontière serait mal placée et le correctif appartiendrait
// au noyau.
//
// Ce paquet reste sous internal/ et n'en sortira pas : ce qui s'expose aux autres
// langages est le noyau, jamais la fenêtre — un jeu hôte possède déjà la sienne et ne
// cédera pas son thread principal.
package backend
