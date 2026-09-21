// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// Scenario décrit une scène scriptée, ses images et ce qu'il faut pour les rendre.
//
// Les deux fonctions séparent ce qui se fait une fois de ce qui se fait par image.
// Atlas construit ses planches par code et jamais depuis un fichier : la suite de
// non-régression ne doit dépendre d'aucune ressource qu'il faudrait savoir refaire.
//
// Quads est une fonction pure de son numéro d'image et de son entrée. Pas d'état
// gardé d'une image à la suivante, pas d'horloge, pas d'aléatoire. C'est plus
// contraignant qu'une mise à jour de jeu, et c'est voulu : une image doit pouvoir se
// rendre seule, deux fois, dans n'importe quel ordre, et donner les mêmes pixels. Un
// déplacement se décrit donc comme une position calculée, pas comme une accumulation.
//
// La tranche passée à Quads est celle de l'image précédente, vidée : la remplir par
// append évite d'allouer à chaque image.
//
// La résolution appartient au scénario. Un scénario de découpage sur les bords en veut
// une petite, et `raster` n'a de toute façon aucune résolution en constante.
type Scenario struct {
	Nom              string
	Largeur, Hauteur uint16
	Images           int
	Fond             rendu.Couleur
	Atlas            func() (*raster.Atlas, error)
	Quads            func(numero int, e Entrees, reserve rendu.Rendu) rendu.Rendu

	// References désigne les images que la suite de non-régression compare. Vide,
	// elles le sont toutes — ce qui convient aux scénarios courts, faits pour tenir
	// en une image ou deux. Un scénario long n'en verse que les moments qui décident.
	References []int
}

// ImagesDeReference rend les numéros d'image que la suite compare.
func (s Scenario) ImagesDeReference() []int {
	if len(s.References) > 0 {
		return s.References
	}

	tous := make([]int, s.Images)
	for i := range tous {
		tous[i] = i
	}
	return tous
}
