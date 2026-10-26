// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'index que l'ajout rend, et le plafond qu'il refuse de dépasser : au-delà, un quad
// ne saurait plus désigner la planche.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestAddReturnsTheIndex vérifie que l'index rendu est bien celui qu'un quad doit
// porter pour désigner la planche qu'on vient d'ajouter.
func TestAddReturnsTheIndex(t *testing.T) {
	var a Atlas

	for want := uint16(0); want < 3; want++ {
		index, err := a.Add(NewSheet(8, 8))
		if err != nil {
			t.Fatalf("ajout : %v", err)
		}
		if index != want {
			t.Errorf("index : %d, attendu %d", index, want)
		}
	}
	if len(a.Sheets) != 3 {
		t.Errorf("planches rangées : %d, attendu 3", len(a.Sheets))
	}
}

// TestAddTakesItsOwnCopy garde la moitié de la règle que la godoc d'[Atlas.Add]
// publie : l'atlas tient son propre encodage, ses propres dimensions et sa propre
// découpe, donc changer la planche d'origine après l'ajout ne touche pas ce qui se
// dessine.
//
// L'autre moitié — les pixels sont partagés — n'est pas gardée ici, et c'est délibéré :
// c'est un piège, pas une garantie. L'éprouver reviendrait à promettre le partage,
// alors que la godoc demande justement de ne pas s'en servir.
//
// Le cas ne peut pas rougir sur le code d'aujourd'hui. Il rougirait si Add prenait un
// pointeur, changement qui rendrait toute la règle fausse sans qu'une image bouge.
func TestAddTakesItsOwnCopy(t *testing.T) {
	var a Atlas

	p := NewSheet(2, 1)
	p.Pixels[0] = red
	if _, err := a.Add(p); err != nil {
		t.Fatalf("ajout : %v", err)
	}

	p.Width, p.Height = 7, 9
	p.Rectangles = []Rectangle{{Width: 1, Height: 1}}

	stored := a.Sheets[0]
	if stored.Width != 2 || stored.Height != 1 {
		t.Errorf("dimensions : %d×%d, attendu 2×1", stored.Width, stored.Height)
	}
	if len(stored.Rectangles) != 0 {
		t.Errorf("découpe : %d rectangles, attendu aucun", len(stored.Rectangles))
	}
	if len(stored.runs) == 0 {
		t.Error("aucune séquence encodée")
	}
}

// TestAddBeyondTheCeiling vérifie le refus de la planche qu'un quad ne saurait plus
// désigner. L'atlas est rempli de planches vides : c'est le compte qui est éprouvé,
// pas les pixels.
func TestAddBeyondTheCeiling(t *testing.T) {
	a := Atlas{Sheets: make([]Sheet, render.MaxSheets)}

	if _, err := a.Add(NewSheet(8, 8)); err == nil {
		t.Errorf("planche acceptée au-delà de %d", render.MaxSheets)
	}
	if len(a.Sheets) != render.MaxSheets {
		t.Errorf("planches rangées : %d, attendu %d", len(a.Sheets), render.MaxSheets)
	}
}
