// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'un scénario doit fournir, et la frontière entre ce qui se fait une fois et ce
// qui se fait par image. C'est ici que vit la contrainte dont la comparaison au pixel
// près dépend : une image est une fonction pure de son numéro et de son entrée.

package viewer

import (
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
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
	Name          string
	Width, Height uint16
	Frames        int
	Background    render.Color
	Atlas         func() (*raster.Atlas, error)
	Quads         func(frame int, e InputLine, reserve render.DrawList) render.DrawList

	// References désigne les images que la suite de non-régression compare. Vide,
	// elles le sont toutes — ce qui convient aux scénarios courts, faits pour tenir
	// en une image ou deux. Un scénario long n'en verse que les moments qui décident.
	References []int
}

// ReferenceFrames rend les numéros d'image que la suite compare.
func (s Scenario) ReferenceFrames() []int {
	if len(s.References) > 0 {
		return s.References
	}

	all := make([]int, s.Frames)
	for i := range all {
		all[i] = i
	}
	return all
}
