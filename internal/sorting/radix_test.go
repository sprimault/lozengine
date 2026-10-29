// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le rangement doit tenir — l'ordre, la stabilité, et le tas auquel il ne
// touche plus après la première image —, avec les deux tailles qui cassent une boucle
// par paquets : la séquence vide et celle d'un seul élément.

package sorting

import (
	"slices"
	"testing"
)

// TestSortOrders vérifie que le rangement suit la clé et non l'ordre d'entrée, en
// mêlant deux champs : une couche haute passe après une profondeur lointaine.
func TestSortOrders(t *testing.T) {
	keys := []Key{
		Compose(0, 0, 5, 0, 0, 0),
		Compose(1, 0, 0, 0, 0, 0),
		Compose(0, 0, 1, 0, 0, 0),
		Compose(0, 0, 3, 0, 0, 0),
	}
	sources := []uint32{10, 11, 12, 13}

	NewRadix(len(keys)).Sort(keys, sources)

	if !slices.IsSorted(keys) {
		t.Fatalf("clés non ordonnées : %v", keys)
	}
	if want := []uint32{12, 13, 10, 11}; !slices.Equal(sources, want) {
		t.Errorf("indices %v, attendu %v", sources, want)
	}
}

// TestSortIsStable vérifie que deux choses de même clé sortent dans l'ordre où
// elles sont entrées.
//
// Les deux clés alternent, pour que le rangement ait réellement à déplacer : une
// séquence déjà groupée passerait sans rien faire et ne prouverait rien. Sans cette
// stabilité, deux sprites superposés se relaieraient au premier plan d'une image à
// l'autre, et le scintillement se voit tout de suite.
func TestSortIsStable(t *testing.T) {
	const n = 64

	first, second := Compose(0, 0, 1, 0, 0, 0), Compose(0, 0, 2, 0, 0, 0)
	keys := make([]Key, n)
	sources := make([]uint32, n)
	for i := range keys {
		keys[i] = first
		if i%2 == 1 {
			keys[i] = second
		}
		sources[i] = uint32(i)
	}

	NewRadix(n).Sort(keys, sources)

	for i, s := range sources {
		want := uint32(2 * i)
		if i >= n/2 {
			want = uint32(2*(i-n/2) + 1)
		}
		if s != want {
			t.Fatalf("rang %d : indice %d, attendu %d", i, s, want)
		}
	}
}

// TestSortShortCases vérifie qu'une séquence vide ou d'un seul élément ne touche à
// rien : le rangement part d'un premier élément qu'il faut bien avoir.
func TestSortShortCases(t *testing.T) {
	r := NewRadix(4)

	r.Sort(nil, nil)

	keys, sources := []Key{42}, []uint32{7}
	r.Sort(keys, sources)
	if keys[0] != 42 || sources[0] != 7 {
		t.Errorf("un seul élément dérangé : %v, %v", keys, sources)
	}
}

// TestSortGrowsWhenNeeded vérifie qu'une séquence plus longue que prévu se range
// quand même, en allouant une fois — ce qui arrive au montage d'une scène, jamais en
// cours de partie.
func TestSortGrowsWhenNeeded(t *testing.T) {
	const n = 100

	r := NewRadix(0)
	keys := make([]Key, n)
	sources := make([]uint32, n)
	for i := range keys {
		keys[i] = Compose(0, 0, uint32(n-i), 0, 0, 0)
		sources[i] = uint32(i)
	}

	r.Sort(keys, sources)

	if !slices.IsSorted(keys) {
		t.Error("clés non ordonnées après croissance des tampons")
	}
}

// TestSortDoesNotAllocate vérifie la propriété dont le chemin d'une image dépend : une
// fois les tampons montés, ranger ne touche plus au tas.
func TestSortDoesNotAllocate(t *testing.T) {
	const n = 1024

	r := NewRadix(n)
	keys := make([]Key, n)
	sources := make([]uint32, n)
	for i := range keys {
		keys[i] = Compose(uint8(i%4), uint8(i%8), uint32(i*7%4096), uint16(i), uint8(i%32), uint16(i%1024))
		sources[i] = uint32(i)
	}

	if allocs := testing.AllocsPerRun(5, func() { r.Sort(keys, sources) }); allocs != 0 {
		t.Errorf("allocations : %v, attendu aucune", allocs)
	}
	if !slices.IsSorted(keys) {
		t.Error("clés non ordonnées")
	}
}
