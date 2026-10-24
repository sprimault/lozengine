// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"slices"
	"testing"
)

// TestRangerOrdonne vérifie que le rangement suit la clé et non l'ordre d'entrée, en
// mêlant deux champs : une couche haute passe après une profondeur lointaine.
func TestRangerOrdonne(t *testing.T) {
	cles := []Cle{
		Composer(0, 0, 5, 0, 0, 0),
		Composer(1, 0, 0, 0, 0, 0),
		Composer(0, 0, 1, 0, 0, 0),
		Composer(0, 0, 3, 0, 0, 0),
	}
	sources := []uint32{10, 11, 12, 13}

	NouveauRangeur(len(cles)).Ranger(cles, sources)

	if !slices.IsSorted(cles) {
		t.Fatalf("clés non ordonnées : %v", cles)
	}
	if attendu := []uint32{12, 13, 10, 11}; !slices.Equal(sources, attendu) {
		t.Errorf("indices %v, attendu %v", sources, attendu)
	}
}

// TestRangerEstStable vérifie que deux choses de même clé sortent dans l'ordre où
// elles sont entrées.
//
// Les deux clés alternent, pour que le rangement ait réellement à déplacer : une
// séquence déjà groupée passerait sans rien faire et ne prouverait rien. Sans cette
// stabilité, deux sprites superposés se relaieraient au premier plan d'une image à
// l'autre, et le scintillement se voit tout de suite.
func TestRangerEstStable(t *testing.T) {
	const n = 64

	premiere, seconde := Composer(0, 0, 1, 0, 0, 0), Composer(0, 0, 2, 0, 0, 0)
	cles := make([]Cle, n)
	sources := make([]uint32, n)
	for i := range cles {
		cles[i] = premiere
		if i%2 == 1 {
			cles[i] = seconde
		}
		sources[i] = uint32(i)
	}

	NouveauRangeur(n).Ranger(cles, sources)

	for i, s := range sources {
		attendu := uint32(2 * i)
		if i >= n/2 {
			attendu = uint32(2*(i-n/2) + 1)
		}
		if s != attendu {
			t.Fatalf("rang %d : indice %d, attendu %d", i, s, attendu)
		}
	}
}

// TestRangerCasCourts vérifie qu'une séquence vide ou d'un seul élément ne touche à
// rien : le rangement part d'un premier élément qu'il faut bien avoir.
func TestRangerCasCourts(t *testing.T) {
	r := NouveauRangeur(4)

	r.Ranger(nil, nil)

	cles, sources := []Cle{42}, []uint32{7}
	r.Ranger(cles, sources)
	if cles[0] != 42 || sources[0] != 7 {
		t.Errorf("un seul élément dérangé : %v, %v", cles, sources)
	}
}

// TestRangerCroitAuBesoin vérifie qu'une séquence plus longue que prévu se range
// quand même, en allouant une fois — ce qui arrive au montage d'une scène, jamais en
// cours de partie.
func TestRangerCroitAuBesoin(t *testing.T) {
	const n = 100

	r := NouveauRangeur(0)
	cles := make([]Cle, n)
	sources := make([]uint32, n)
	for i := range cles {
		cles[i] = Composer(0, 0, uint32(n-i), 0, 0, 0)
		sources[i] = uint32(i)
	}

	r.Ranger(cles, sources)

	if !slices.IsSorted(cles) {
		t.Error("clés non ordonnées après croissance des tampons")
	}
}

// TestRangerNAllouePas vérifie la propriété dont le chemin d'une image dépend : une
// fois les tampons montés, ranger ne touche plus au tas.
func TestRangerNAllouePas(t *testing.T) {
	const n = 1024

	r := NouveauRangeur(n)
	cles := make([]Cle, n)
	sources := make([]uint32, n)
	for i := range cles {
		cles[i] = Composer(uint8(i%4), uint8(i%8), uint32(i*7%4096), uint16(i), uint16(i%64), uint16(i%1024))
		sources[i] = uint32(i)
	}

	if allocs := testing.AllocsPerRun(5, func() { r.Ranger(cles, sources) }); allocs != 0 {
		t.Errorf("allocations : %v, attendu aucune", allocs)
	}
	if !slices.IsSorted(cles) {
		t.Error("clés non ordonnées")
	}
}
