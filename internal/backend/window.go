// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le contrat qu'une famille de plateformes remplit, et ce qu'un appelant règle avant
// d'ouvrir. Les deux vivent ensemble parce qu'ils se lisent ensemble : la boucle pose
// des options, reçoit une fenêtre, et ne connaît rien d'autre d'une plateforme.

package backend

import (
	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
)

// Window est ce qu'une famille de plateformes doit savoir faire, et rien de plus.
//
// Cinq méthodes, parce que la boucle n'a besoin que de cela : lire le clavier,
// présenter une image, savoir dans quoi elle présente, savoir si on lui a demandé de
// partir, et partir.
type Window interface {
	// Input rend l'état du clavier pour le pas qui commence, les touches tenues
	// comme celles qui viennent d'être enfoncées.
	Input() input.Input

	// Present pose le tampon à l'écran, tel quel.
	//
	// Sans erreur : le chemin d'une image n'en rend pas, et il n'y a personne à qui
	// se plaindre soixante fois par seconde. Une présentation qui ne peut pas avoir
	// lieu — fenêtre minimisée, par exemple — ne fait rien, ce qui est le bon
	// comportement et non un échec. Ce qui est terminal se lit par Closed.
	Present(t *render.Buffer)

	// Size rend les dimensions de la zone utile, en pixels, hors bordures et
	// barre de titre. C'est d'elle que la boucle tire le facteur d'agrandissement.
	Size() (width, height uint16)

	// Closed dit que la fenêtre ne présentera plus rien : le joueur l'a fermée, ou
	// le système l'a reprise.
	Closed() bool

	// Close rend au système ce qu'Open a pris. Appeler deux fois est sans effet.
	Close()
}

// Options porte ce qu'une fenêtre admet de régler à l'ouverture.
//
// Sa valeur zéro ne convient pas ici, contrairement au reste du moteur : une fenêtre
// de zéro pixel n'a pas de défaut raisonnable, et le titre d'un jeu n'en a pas non
// plus. C'est le paquet racine qui pose les siens avant d'appeler.
type Options struct {
	Title         string
	Width, Height uint16

	// Icons porte les icônes de la fenêtre, une par taille que le jeu fournit, en
	// pixels prémultipliés comme le reste de ce que le moteur peint. Vide, la
	// fenêtre porte l'icône du système.
	//
	// Plusieurs tailles et non une, parce que les plateformes en veulent plusieurs :
	// Windows emploie une grande icône et une petite, et la propriété d'X11 accepte
	// elle-même une suite d'images de tailles différentes. Chacune choisit, et
	// aucune ne redimensionne — la règle d'échelle entière l'interdit au moteur.
	Icons []*render.Buffer
}
