// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "fmt"

// planchesMax est le nombre de planches que la clé de tri sait distinguer : dix
// bits lui sont réservés. Au-delà, deux quads de planches différentes cesseraient
// de se grouper, et le rastériseur changerait de source à chaque quad.
const planchesMax = 1 << 10

// Atlas est la suite des planches qu'un quad désigne par index.
//
// Il ne traverse pas la frontière C : un hôte le tient par un handle opaque, et
// c'est ce qui autorise ici une structure Go ordinaire là où le quad, lui, est
// contraint.
type Atlas struct {
	Planches []Planche
}

// Ajouter range une planche et rend l'index qui la désigne dans un quad.
//
// L'erreur au-delà de la limite arrive ici parce qu'elle ne peut plus arriver
// ailleurs : le chemin de rendu ne renvoie rien, et un index qui déborde du champ
// de la clé se lirait comme une autre planche.
func (a *Atlas) Ajouter(p Planche) (uint16, error) {
	if len(a.Planches) >= planchesMax {
		return 0, fmt.Errorf("ajout d'une planche : %d au plus, limite des dix bits que la clé de tri leur réserve", planchesMax)
	}

	a.Planches = append(a.Planches, p)
	return uint16(len(a.Planches) - 1), nil
}
