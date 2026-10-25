// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package backend ouvre une fenêtre, lit le clavier et pose des pixels à l'écran.
//
// Il y a un backend par famille de plateformes — les appels système de Windows, le
// protocole X11, le canvas du navigateur — et tous remplissent le même contrat, celui
// de [Fenetre]. Rien d'autre ne les distingue vu d'en haut, ce qui est la condition
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

import (
	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// Fenetre est ce qu'une famille de plateformes doit savoir faire, et rien de plus.
//
// Cinq méthodes, parce que la boucle n'a besoin que de cela : lire le clavier,
// présenter une image, savoir dans quoi elle présente, savoir si on lui a demandé de
// partir, et partir.
type Fenetre interface {
	// Entrees rend l'état du clavier pour le pas qui commence, les touches tenues
	// comme celles qui viennent d'être enfoncées.
	Entrees() input.Input

	// Presenter pose le tampon à l'écran, tel quel.
	//
	// Sans erreur : le chemin d'une image n'en rend pas, et il n'y a personne à qui
	// se plaindre soixante fois par seconde. Une présentation qui ne peut pas avoir
	// lieu — fenêtre minimisée, par exemple — ne fait rien, ce qui est le bon
	// comportement et non un échec. Ce qui est terminal se lit par Fermee.
	Presenter(t *render.Buffer)

	// Taille rend les dimensions de la zone utile, en pixels, hors bordures et
	// barre de titre. C'est d'elle que la boucle tire le facteur d'agrandissement.
	Taille() (largeur, hauteur uint16)

	// Fermee dit que la fenêtre ne présentera plus rien : le joueur l'a fermée, ou
	// le système l'a reprise.
	Fermee() bool

	// Fermer rend au système ce qu'Ouvrir a pris. Appeler deux fois est sans effet.
	Fermer()
}

// Options porte ce qu'une fenêtre admet de régler à l'ouverture.
//
// Sa valeur zéro ne convient pas ici, contrairement au reste du moteur : une fenêtre
// de zéro pixel n'a pas de défaut raisonnable, et le titre d'un jeu n'en a pas non
// plus. C'est le paquet racine qui pose les siens avant d'appeler.
type Options struct {
	Titre            string
	Largeur, Hauteur uint16
}
