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

import "github.com/sprimault/lozengine/rendu"

// Aspect est ce qu'une chose montre : de quoi remplir un quad, sans dire où.
//
// Séparé de la position parce que le décor n'en a pas besoin — une cellule de grille
// est située par son rang, et lui recopier des coordonnées doublerait ce que la grille
// sait déjà. Sur une carte de cinq cents cellules de côté, le doublon pèserait six
// mégaoctets par couche.
//
// Sa valeur zéro ne montre rien, et c'est ainsi qu'une cellule vide se reconnaît : une
// largeur ou une hauteur nulle ne couvre aucun pixel, donc aucune table de présence
// n'a à exister à côté.
type Aspect struct {
	// La planche où lire, et le rectangle qu'on y prend.
	Planche          uint16
	SourceX, SourceY uint16
	Largeur, Hauteur uint16

	// Teinte multiplie la source, et [rendu.Blanc] est la valeur qui la laisse
	// intacte. Attention : la valeur zéro d'une couleur est transparente, donc elle
	// ne dessine rien — une cellule qu'on veut voir porte une teinte explicite.
	Teinte   rendu.Couleur
	Drapeaux rendu.Drapeaux

	// Couche et Etage ouvrent les deux champs les plus forts de la clé de tri. La
	// couche range ce qui doit passer devant tout le reste, une interface par exemple ;
	// l'étage sépare les niveaux d'un décor. Combien un jeu en emploie ne regarde que
	// lui — la clé sait en distinguer seize et soixante-quatre.
	//
	// Ni l'une ni l'autre n'est la strate de [Scene.PoserCellule], qui dit seulement
	// ce qui s'empile sur une même cellule. Une couche passe devant toute l'image, une
	// strate ne passe devant que ce qu'elle recouvre.
	Couche uint8
	Etage  uint8
}

// Entite est un aspect posé à un point du monde.
//
// Les coordonnées sont des cellules, et elles sont flottantes : une entité se tient en
// x = 3,5 sans que rien ne l'arrondisse ici. La conversion vers les entiers du rendu se
// fait à l'entrée de la projection, une fois.
//
// z est la hauteur au-dessus du sol et reste un champ à part. L'écran la retranche de
// l'ordonnée, la clé de tri la range dans son propre champ : les fondre perdrait
// l'information dès l'entrée et rendrait l'ordre de dessin irréparable.
type Entite struct {
	X, Y, Z float64

	Aspect
}
