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
// Six méthodes, parce que la boucle n'a besoin que de cela : lire le clavier, présenter
// une image, savoir dans quoi elle présente, savoir si on lui a demandé de partir, dire
// ce qui l'a fait partir, et partir.
//
// **L'ordre d'appel fait partie du contrat, et Input vient en premier** : c'est elle qui
// vide la file d'événements, donc tout ce que les autres rapportent d'un pas a été
// appris là. Les interroger avant elle rend l'état du pas précédent, ce qui fait une
// fenêtre en retard d'une image.
//
// **Tout se passe sur le thread qui a ouvert la fenêtre**, et aucune méthode n'est sûre
// depuis une autre goroutine. La boucle du paquet racine verrouille le sien pour cela.
//
// **Size et Closed sont des accesseurs purs** : elles rendent ce que la dernière Input a
// appris, sans interroger le système. Une famille qui parle par une socket paierait
// autrement un aller-retour complet par image pour une valeur que le serveur lui a déjà
// annoncée par un événement.
type Window interface {
	// Input rend les entrées du pas qui commence : les touches et les boutons tenus
	// comme ceux qui viennent d'être enfoncés, la position du pointeur et la molette.
	//
	// C'est elle qui vide la file d'événements de la plateforme, donc elle s'appelle
	// une fois par pas et en premier.
	//
	// **La position est en pixels de fenêtre**, celle que le système rapporte. C'est
	// à la boucle de la ramener en pixels de l'image : elle seule connaît le facteur
	// d'agrandissement, qu'un backend n'a aucune raison de calculer.
	Input() input.Input

	// Present pose le tampon à l'écran, tel quel.
	//
	// Sans erreur : le chemin d'une image n'en rend pas, et il n'y a personne à qui
	// se plaindre soixante fois par seconde. Une présentation qui ne peut pas avoir
	// lieu — fenêtre minimisée, par exemple — ne fait rien, ce qui est le bon
	// comportement et non un échec. Ce qui est terminal se lit par Closed, et sa
	// cause par Err.
	Present(t *render.Buffer)

	// Size rend les dimensions de la zone utile, en pixels, hors bordures et
	// barre de titre. C'est d'elle que la boucle tire le facteur d'agrandissement.
	Size() (width, height uint16)

	// Closed dit que la fenêtre ne présentera plus rien : le joueur l'a fermée, ou
	// le système l'a reprise.
	Closed() bool

	// Err dit ce qui a mis fin à la fenêtre, et nil quand c'est le joueur.
	//
	// Elle sépare des sorties que la boucle voit sinon identiques : la croix
	// cliquée, qui est le régime normal, une erreur de protocole, un lien coupé.
	// Sans elle, un serveur d'affichage qui meurt se rapporte comme un arrêt
	// propre — et c'est le régime courant du développement d'un backend.
	//
	// Elle se lit à la sortie de la boucle, quelle qu'en soit la cause. Ce qui
	// échoue avant que la fenêtre existe est rendu par Open.
	Err() error

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

	// HideCursor retire le curseur du système sur la zone de jeu.
	//
	// C'est ce qu'un jeu demande quand il dessine le sien dans son image : sans cela
	// deux pointeurs se suivraient, celui du jeu agrandi avec l'image et celui du
	// système net et décalé d'un pixel sur sept. Les bordures et la barre de titre
	// gardent les leurs, qui disent ce qu'on peut y faire.
	HideCursor bool
}
