// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// TestAjouterRendLIndex vérifie que l'index rendu est bien celui qu'un quad doit
// porter pour désigner la planche qu'on vient d'ajouter.
func TestAjouterRendLIndex(t *testing.T) {
	var a Atlas

	for attendu := uint16(0); attendu < 3; attendu++ {
		index, err := a.Add(NewSheet(8, 8))
		if err != nil {
			t.Fatalf("ajout : %v", err)
		}
		if index != attendu {
			t.Errorf("index : %d, attendu %d", index, attendu)
		}
	}
	if len(a.Sheets) != 3 {
		t.Errorf("planches rangées : %d, attendu 3", len(a.Sheets))
	}
}

// TestAjouterAuDela vérifie le refus de la planche qu'un quad ne saurait plus
// désigner. L'atlas est rempli de planches vides : c'est le compte qui est éprouvé,
// pas les pixels.
func TestAjouterAuDela(t *testing.T) {
	a := Atlas{Sheets: make([]Sheet, rendu.PlanchesMax)}

	if _, err := a.Add(NewSheet(8, 8)); err == nil {
		t.Errorf("planche acceptée au-delà de %d", rendu.PlanchesMax)
	}
	if len(a.Sheets) != rendu.PlanchesMax {
		t.Errorf("planches rangées : %d, attendu %d", len(a.Sheets), rendu.PlanchesMax)
	}
}
