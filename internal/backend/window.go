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
// Sept méthodes, parce que la boucle n'a besoin que de cela : lire le clavier, présenter
// une image, savoir dans quoi elle présente, savoir si on lui a demandé de partir, dire
// ce qui l'a fait partir, partir, et dire qui tient la cadence.
//
// **L'ordre d'appel fait partie du contrat, et Input vient en premier** : c'est elle qui
// vide la file d'événements, donc tout ce que les autres rapportent d'un pas a été
// appris là. Les interroger avant elle rend l'état du pas précédent, ce qui fait une
// fenêtre en retard d'une image.
//
// **Tout se passe sur le thread qui a ouvert la fenêtre**, et aucune méthode n'est sûre
// depuis une autre goroutine. La boucle du paquet racine verrouille le sien pour cela.
//
// **Size et Closed rendent ce que le pas courant sait, et ne traversent jamais le
// réseau** : une famille qui parle par une socket paierait autrement un aller-retour
// complet par image pour une valeur que le serveur lui a déjà annoncée par un événement.
//
// **Ce bloc exigeait des « accesseurs purs, sans interroger le système », et c'était faux
// de Windows depuis toujours** : Size y appelle clientSize à chaque image. Relevé le
// 1er novembre 2026. C'est la lettre qui était trop large, pas le code — un appel local
// n'est pas un aller-retour, et une taille mise en cache se désynchronise de l'événement
// qu'on n'a pas pensé à écouter. X11 rend un champ parce que le serveur le lui annonce ;
// une famille sans événement de taille peut interroger, tant que cela ne sort pas de la
// machine.
type Window interface {
	// Input rend les entrées du pas qui commence : les touches et les boutons tenus
	// comme ceux qui viennent d'être enfoncés, la position du pointeur et la molette.
	//
	// C'est elle qui vide la file d'événements de la plateforme, donc elle s'appelle
	// une fois par pas et en premier.
	//
	// **Elle a le droit d'attendre, et c'est une extension du contrat plutôt qu'une
	// tolérance.** Le navigateur y bloque depuis le jalon 5 sur l'image d'animation
	// que la page lui promet, parce que c'est le seul instant où une présentation est
	// certaine d'entrer dans l'image composée ; une famille dont la plateforme annonce
	// qu'elle veut une image peut faire de même. Le contrat disait « vide la file »,
	// ce qui a suffi tant que les deux familles natives rendaient aussitôt.
	//
	// **Ce qu'une attente doit respecter.** Elle se termine sur un événement que la
	// plateforme produit d'elle-même, jamais sur un délai : la boucle tient déjà une
	// horloge en visant une date plutôt qu'en dormant d'une durée fixe, et une seconde
	// horloge ici ferait dériver la première. Elle n'a donc pas d'autre borne que cet
	// événement — une plateforme qui cesse de composer arrête le jeu, ce qui est voulu,
	// un jeu que personne ne regarde n'ayant pas d'image à produire.
	//
	// **Attendre n'autorise pas Paced à rendre vrai** : sa condition porte sur ce qui
	// réarme le jeton de la plateforme, et non sur le fait d'attendre.
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

	// Paced dit que cette fenêtre cadence elle-même la boucle, en attendant dans
	// Input jusqu'à ce que sa plateforme veuille une image.
	//
	// **Une page l'impose, et c'est le cas qui a créé cette méthode.** Une page
	// n'offre pas d'horloge au jeu : elle appelle le module quand elle s'apprête à
	// composer une image, et c'est le seul instant où une présentation est certaine
	// d'entrer dans l'image qu'elle compose. Mesuré le 2 novembre 2026, une boucle qui
	// dort pour tenir son pas écrit **deux tiers de ses images hors de ce tour** à
	// quatre-vingt-dix pour cent de charge : le navigateur compose alors un tampon que
	// le moteur réécrit.
	//
	// **Ce qu'une réponse vraie exige, et ce n'est pas d'attendre** : que le jeton de
	// la plateforme se réarme **sans passer par Present**. Devant une fenêtre
	// cadencée, la boucle rend la main sans présenter quand un tour arrive en avance
	// sur l'échéance ; une famille dont la plateforme n'autorise l'image suivante
	// qu'en réponse à la présentation précédente n'a donc rien demandé ce tour-là, et
	// le tour suivant attend un jeton que personne n'a réclamé — ou rend aussitôt, et
	// la boucle tourne à plein régime, le sommeil étant interdit sous ce régime. Le
	// navigateur y échappe en demandant son image d'animation dans Input, pas dans
	// Present.
	//
	// C'est ce qui a fait répondre faux aux deux familles du jalon 6 : sous Wayland,
	// `wl_surface.frame` ne prend effet qu'au commit suivant, et sous DRM/KMS
	// l'événement de bascule répond au `PAGE_FLIP` qu'émet la présentation. Le piège
	// est gardé par un témoin de la boucle, qui rougit si celle-ci cesse de sauter la
	// présentation d'un tour en avance.
	//
	// **Ce que la boucle en fait, et pourquoi elle ne peut pas le deviner** : devant
	// une fenêtre cadencée, elle ne dort jamais et rend au contraire la main sans
	// faire de pas quand le tour arrive en avance sur l'échéance. Devant les autres,
	// elle dort, parce qu'un Input qui rend tout de suite ferait tourner la boucle à
	// plein régime. Les deux régimes ne se distinguent par rien d'observable — un
	// Input lent peut être une plateforme chargée —, d'où cette méthode plutôt qu'une
	// mesure.
	//
	// Elle ne varie pas pendant la vie d'une fenêtre : c'est une propriété de la
	// famille, pas de l'instant.
	Paced() bool
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
