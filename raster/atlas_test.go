// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

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
