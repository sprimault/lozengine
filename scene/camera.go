// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package scene

// Orientation est l'un des quatre quarts de tour sous lesquels une scène se regarde.
//
// Quatre et non un angle libre : une apparence est dessinée pour un nombre fini de
// directions, et un angle quelconque demanderait de l'interpoler — ce qu'un rendu en
// pixels ne sait pas faire sans se brouiller.
type Orientation uint8

// Les quatre quarts de tour, dans le sens où la grille pivote sous la vue.
const (
	NoRotation Orientation = iota
	QuarterTurn
	HalfTurn
	ThreeQuarters
)

// Orient fait pivoter la vue d'un quart de tour ou d'un multiple, et recadre sur le
// point visé.
//
// La matrice n'y est pour rien et ne change pas : ce sont les coordonnées du monde qui
// sont remappées avant d'y entrer. Une rotation par changement de signe donnerait des
// coordonnées négatives, donc une profondeur signée, un biais à ajouter et la moitié
// de la plage perdue sur le champ que la clé de tri lui réserve. La scène connaissant
// l'étendue de sa grille, elle peut renverser un axe au lieu de l'inverser, et tout
// reste positif.
func (s *Scene) Orient(o Orientation) {
	s.orientation = o & 3
	s.reframe()
}

// Orientation rend le quart de tour courant.
func (s *Scene) Orientation() Orientation { return s.orientation }

// Focus centre la vue sur un point du monde, pour un tampon de la taille donnée.
//
// La taille vient de l'appelant et n'est jamais supposée ici : la résolution interne
// est le choix du jeu, et la mise à l'échelle vers une fenêtre appartient au backend.
func (s *Scene) Focus(x, y float64, width, height int32) {
	s.focusX, s.focusY = x, y
	s.viewWidth, s.viewHeight = width, height
	s.reframe()
}

// View rend l'origine de la vue en pixels absolus, et la taille du tampon.
//
// C'est ce que le tri retranche de chaque point projeté, au moment où il écarte ce qui
// tombe hors de l'image. Les deux gestes sont le même calcul.
func (s *Scene) View() (x, y, width, height int32) {
	return s.viewX, s.viewY, s.viewWidth, s.viewHeight
}

// Remap rend les coordonnées sous lesquelles l'orientation courante présente un
// point du monde.
//
// Un jeu pose ses entités et son décor dans son repère à lui, une fois pour toutes ;
// c'est la vue qui tourne. Le remappage renverse un axe autour de l'étendue de la
// grille plutôt que d'en changer le signe, ce qui garde tout positif.
func (s *Scene) Remap(x, y float64) (float64, float64) {
	width, height := float64(s.width), float64(s.height)

	switch s.orientation {
	case QuarterTurn:
		return height - y, x
	case HalfTurn:
		return width - x, height - y
	case ThreeQuarters:
		return y, width - x
	}
	return x, y
}

// Unremap fait le chemin inverse de [Scene.Remap].
//
// Le tri en a besoin pour relever le décor : il remonte des coins de l'image vers des
// coordonnées remappées, et doit retrouver la cellule que le jeu a posée.
func (s *Scene) Unremap(x, y float64) (float64, float64) {
	width, height := float64(s.width), float64(s.height)

	switch s.orientation {
	case QuarterTurn:
		return y, height - x
	case HalfTurn:
		return width - x, height - y
	case ThreeQuarters:
		return width - y, x
	}
	return x, y
}

// Extent rend les côtés de la grille tels que l'orientation courante les présente :
// un quart de tour les échange.
func (s *Scene) Extent() (width, height int) {
	if s.orientation == QuarterTurn || s.orientation == ThreeQuarters {
		return s.height, s.width
	}
	return s.width, s.height
}

// reframe replace l'origine de la vue après un déplacement ou une rotation.
//
// Le point visé se retrouve au centre du tampon. Aucune borne n'est appliquée aux
// bords de la grille : où la caméra s'arrête est une décision de jeu, pas de moteur —
// certains veulent buter sur le bord, d'autres montrer le vide autour.
func (s *Scene) reframe() {
	rx, ry := s.Remap(s.focusX, s.focusY)
	ex, ey, _, _ := s.projection.Project(rx, ry, 0)

	s.viewX = ex - s.viewWidth/2
	s.viewY = ey - s.viewHeight/2
}
