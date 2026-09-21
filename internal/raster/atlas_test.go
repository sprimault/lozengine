// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import "testing"

// TestAjouterRendLIndex vérifie que l'index rendu est bien celui qu'un quad doit
// porter pour désigner la planche qu'on vient d'ajouter.
func TestAjouterRendLIndex(t *testing.T) {
	var a Atlas

	for attendu := uint16(0); attendu < 3; attendu++ {
		index, err := a.Ajouter(NouvellePlanche(8, 8))
		if err != nil {
			t.Fatalf("ajout : %v", err)
		}
		if index != attendu {
			t.Errorf("index : %d, attendu %d", index, attendu)
		}
	}
	if len(a.Planches) != 3 {
		t.Errorf("planches rangées : %d, attendu 3", len(a.Planches))
	}
}

// TestAjouterAuDela vérifie le refus de la planche qui déborderait du champ que la
// clé de tri réserve à la planche. L'atlas est rempli de planches vides : c'est le
// compte qui est éprouvé, pas les pixels.
func TestAjouterAuDela(t *testing.T) {
	a := Atlas{Planches: make([]Planche, planchesMax)}

	if _, err := a.Ajouter(NouvellePlanche(8, 8)); err == nil {
		t.Errorf("planche acceptée au-delà de %d", planchesMax)
	}
	if len(a.Planches) != planchesMax {
		t.Errorf("planches rangées : %d, attendu %d", len(a.Planches), planchesMax)
	}
}
