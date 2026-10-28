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
// de l'aspect s'y ajoute comme ailleurs, pour qui veut décaler.
type ScreenPiece struct {
	X, Y int32
	Aspect
}

// PlaceScreen pose un aspect à un pixel de l'image, et rend faux si sa couche ou son
// étage dépasse ce que l'ordre de dessin distingue.
//
// **Les coordonnées sont celles de l'image, pas du monde**, et c'est la seule voie du
// moteur où il en va ainsi. Une interface doit tenir au même endroit pendant que la
// caméra bouge ; l'exprimer en cellules obligerait le jeu à défaire la projection à
// chaque image, et l'aller-retour n'étant pas exact au pixel près, le panneau
// tremblerait pendant la marche du personnage.
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
func (s *Scene) PlaceScreen(ex, ey int32, a Aspect) bool {
	if !a.fitsInKey() {
		return false
	}

	s.screen = append(s.screen, ScreenPiece{X: ex, Y: ey, Aspect: a})
	return true
}

// ClearScreen retire tout ce que [Scene.PlaceScreen] a posé, en gardant la capacité.
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
