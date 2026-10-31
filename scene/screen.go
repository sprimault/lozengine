// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'on pose en pixels de l'image plutôt qu'en cellules du monde : une barre de vie,
// un panneau, une ligne de texte. La seule voie du moteur qui ne passe pas par la
// projection.

package scene

// ScreenPiece est un aspect posé à un pixel de l'image.
//
// La position est celle du coin haut-gauche, et non le pied centré d'une chose du
// monde : une interface se place par son coin, qui est ce qu'on sait mesurer. L'ancre
// de l'aspect s'ajoute à ce coin, qu'elle laisse en place quand elle est nulle.
type ScreenPiece struct {
	X, Y int32
	Aspect

	// InArea dit que le pixel visé est celui de l'aire de monde et non celui de la
	// surface, donc que la pièce se découpe sur l'aire. [Scene.PlaceScreen] le laisse
	// faux, [Scene.PlaceArea] le pose.
	InArea bool
}

// PlaceScreen pose un aspect à un pixel de la surface, et rend faux si sa couche, son
// étage ou sa couche de révélation dépasse ce que l'ordre de dessin distingue.
//
// **Le pixel est celui de la surface, marge comprise, et la pièce s'y découpe** : c'est
// la voie de ce qui n'appartient pas au monde — un bandeau, un score, une minicarte —,
// et la marge existe pour le porter. Ce qui appartient au monde mais vit dans une couche
// haute passe par [Scene.PlaceArea], qui vise l'aire et s'y découpe.
//
// **Les coordonnées sont celles de l'image, pas du monde**, et c'est la seule voie du
// moteur où il en va ainsi. Une interface doit tenir au même endroit pendant que la
// caméra bouge ; l'exprimer en cellules obligerait le jeu à défaire la projection à
// chaque image, et l'aller-retour n'étant pas exact au pixel près, le panneau
// tremblerait pendant la marche du personnage.
//
// **Le pixel donné est le coin haut-gauche de l'image**, là où [Scene.Place] et
// [Scene.PlaceCell] en visent le pied. Pour poser une image par son pied sur un
// pixel — une barre de vie au-dessus d'une créature, un chiffre de dégâts —,
// retrancher de ce pixel la moitié de sa largeur, arrondie vers le bas, et toute sa
// hauteur. [Aspect.AnchorX] et [Aspect.AnchorY] s'ajoutent d'eux-mêmes à ce coin :
// les ajouter au calcul les compterait deux fois, et l'image partirait du double de
// l'ancre.
//
// Ce n'est pas un second chemin de rendu pour autant : les quads produits sortent dans
// la même liste, rangés par la même clé, et c'est [Aspect.Layer] qui les met au-dessus
// du monde. Un aspect posé en couche zéro passera sous le décor plus proche que
// l'origine de la vue, ce qui est l'ordre juste et non une erreur — une interface
// nomme une couche haute.
//
// **L'ordre de pose est l'ordre de dessin** entre pièces d'écran de même couche : ce
// qui est posé après recouvre ce qui précède. C'est ce qu'une ligne de texte demande,
// ses glyphes partageant couche, planche et position d'écran voisine.
//
// **Le magasin n'a pas de plafond et ne se règle pas** : il croît par ajout jusqu'à sa
// plus forte occupation, puis n'alloue plus, [Scene.ClearScreen] gardant cette
// capacité. Un jeu n'alloue donc ici qu'à l'image où il pose plus de pièces qu'à aucune
// précédente — ce qui est la seule réserve à la promesse d'absence d'allocation que
// [Scene] et la boucle annoncent.
func (s *Scene) PlaceScreen(ex, ey int32, a Aspect) bool {
	return s.placePiece(ex, ey, a, false)
}

// PlaceArea pose un aspect à un pixel de l'aire de monde, et rend faux si sa couche,
// son étage ou sa couche de révélation dépasse ce que l'ordre de dessin distingue.
//
// **C'est la voie de ce qui accompagne le monde sans en être** : un chiffre de dégâts,
// l'annonce posée au-dessus d'un objet, une silhouette de rappel. Le pixel donné est
// relatif au coin haut-gauche de l'**aire** — celle que [Scene.View] dimensionne et que
// [Scene.WorldOrigin] situe dans la surface —, donc c'est exactement le repère où un jeu
// se trouve après avoir projeté un point du monde et retranché l'origine de la vue. Il
// n'a pas à ajouter cette origine lui-même : la pose s'en charge.
//
// **La pièce se découpe sur l'aire, et c'est la raison d'être de cette voie.** Dès
// qu'une marge existe, une chose située juste hors de l'aire reçoit une abscisse qui
// redevient positive une fois l'origine ajoutée : posée par [Scene.PlaceScreen], elle
// s'afficherait dans la marge, à côté du jeu. Un jeu peut refuser ce qui ne touche pas
// l'aire, mais pas **découper** une chose à cheval sur le bord — il lui faudrait refaire
// le calcul du moteur, miroirs compris.
//
// **Une entité en couche haute fait aussi l'affaire, et vaut mieux quand elle suffit** :
// [Scene.Place] projette, découpe sur l'aire et range par la couche de l'aspect, donc
// une chose qui vit à un point du monde s'y exprime sans passer par ici. Cette voie sert
// ce qui se pose en pixels sans être une entité — du texte surtout, dont chaque glyphe
// est un quad —, là qu'une poignée par glyphe posée et retirée à chaque image serait du
// gâchis.
//
// Le reste du contrat est celui de [Scene.PlaceScreen] : le pixel visé est le coin
// haut-gauche, l'ancre s'y ajoute, la couche décide du rang, et l'ordre de pose est
// l'ordre de dessin. **Cet ordre est commun aux deux voies** : les pièces vivent dans une
// seule liste, donc à couche égale ce qui est posé après recouvre ce qui précède, quel
// que soit le repère de chacune.
func (s *Scene) PlaceArea(ex, ey int32, a Aspect) bool {
	return s.placePiece(ex, ey, a, true)
}

// placePiece ajoute une pièce à la liste commune aux deux repères.
//
// Une seule liste, et c'est ce qui tient la promesse d'ordre : le rang de pose est
// l'indice ici, donc il ne repart pas à zéro d'un repère à l'autre. Deux listes
// feraient décider l'ordre de dessin par le repère à couche égale, ce qu'aucune godoc
// n'annonce et que personne n'irait chercher là.
func (s *Scene) placePiece(ex, ey int32, a Aspect, inArea bool) bool {
	if !a.fitsInKey() {
		return false
	}

	s.screen = append(s.screen, ScreenPiece{X: ex, Y: ey, Aspect: a, InArea: inArea})
	return true
}

// ClearScreen retire tout ce que [Scene.PlaceScreen] et [Scene.PlaceArea] ont posé, en
// gardant la capacité.
//
// **C'est au jeu de l'appeler**, et le moteur ne le fait jamais pour lui. Vider après
// chaque image paraîtrait commode et retirerait un choix : une interface qui ne change
// pas se reposerait pour rien soixante fois par seconde, et un jeu qui tient une liste
// stable n'aurait aucune façon de le dire.
func (s *Scene) ClearScreen() { s.screen = s.screen[:0] }

// ScreenPieces rend, dans l'ordre de pose, ce qui est posé en pixels de l'image.
//
// Fonction et non méthode, pour la raison qui vaut pour [Entities] et [Identities] :
// c'est une porte que le tri emprunte, et qu'on ne met pas sur le chemin d'un jeu qui
// cherche ce qu'une scène sait faire.
func ScreenPieces(s *Scene) []ScreenPiece { return s.screen }
