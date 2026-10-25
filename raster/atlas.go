// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"fmt"

	"github.com/sprimault/lozengine/render"
)

// Atlas est la suite des planches qu'un quad désigne par index.
//
// Il ne traverse pas la frontière C : un hôte le tient par un handle opaque, et
// c'est ce qui autorise ici une structure Go ordinaire là où le quad, lui, est
// contraint.
type Atlas struct {
	Sheets []Sheet
}

// Add encode les lignes de la planche, la range, et rend l'index qui la
// désigne dans un quad.
//
// L'encodage a lieu ici parce que c'est le seul passage obligé : une planche
// devient dessinable en entrant dans un atlas, et l'oubli n'est donc pas possible.
// Le blit, lui, ne peut ni encoder ni s'en plaindre — il ne renvoie pas d'erreur et
// n'alloue pas.
//
// L'erreur au-delà de [render.MaxSheets] arrive ici pour la même raison : un index
// que l'ordre de dessin ne sait plus distinguer se lirait comme une autre planche.
// Refuser au chargement vaut mieux que ramener au plafond à l'image, ce qui
// rangerait deux planches ensemble sans rien dire.
func (a *Atlas) Add(p Sheet) (uint16, error) {
	if len(a.Sheets) >= render.MaxSheets {
		return 0, fmt.Errorf("adding a sheet: %d at most, what a quad can address", render.MaxSheets)
	}

	p.encode()
	a.Sheets = append(a.Sheets, p)
	return uint16(len(a.Sheets) - 1), nil
}
