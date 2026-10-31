// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'exposition à l'ABI rend irrévocable : la taille et la position de chaque
// champ d'un quad et d'un relevé de recouvrement, la contiguïté des deux listes, et
// les deux valeurs qu'un hôte compare sans lire notre documentation.

package render

import (
	"testing"
	"unsafe"
)

// TestLayout fige la disposition mémoire de Quad, que l'exposition à l'ABI C
// rend irrévocable : un champ déplacé ou élargi déplace tout ce qui suit et casse
// silencieusement un hôte déjà compilé. Le test échoue avant lui.
func TestLayout(t *testing.T) {
	var q Quad

	if got, want := unsafe.Sizeof(q), uintptr(20); got != want {
		t.Errorf("taille de Quad : %d, attendu %d", got, want)
	}
	if got, want := unsafe.Alignof(q), uintptr(2); got != want {
		t.Errorf("alignement de Quad : %d, attendu %d", got, want)
	}

	positions := []struct {
		field string
		got   uintptr
		want  uintptr
	}{
		{"X", unsafe.Offsetof(q.X), 0},
		{"Y", unsafe.Offsetof(q.Y), 2},
		{"SourceX", unsafe.Offsetof(q.SourceX), 4},
		{"SourceY", unsafe.Offsetof(q.SourceY), 6},
		{"Width", unsafe.Offsetof(q.Width), 8},
		{"Height", unsafe.Offsetof(q.Height), 10},
		{"Sheet", unsafe.Offsetof(q.Sheet), 12},
		{"Tint", unsafe.Offsetof(q.Tint), 14},
		{"Flags", unsafe.Offsetof(q.Flags), 18},
	}
	for _, p := range positions {
		if p.got != p.want {
			t.Errorf("position de %s : %d, attendu %d", p.field, p.got, p.want)
		}
	}

	if got, want := unsafe.Sizeof(Color{}), uintptr(4); got != want {
		t.Errorf("taille de Color : %d, attendu %d", got, want)
	}
}

// TestOverlapLayout fige la disposition mémoire d'Overlap, pour la raison qui vaut
// pour Quad : un hôte la reçoit par pointeur et longueur, donc un champ déplacé ou
// élargi casse du code déjà compilé sans qu'une signature bouge.
//
// **Elle n'était figée par rien, et sa taille a doublé entre deux versions publiées** —
// douze octets à la 0.5.3, vingt-quatre à la 0.5.4, le lendemain. Aucun contrôle ne
// l'a vu : ce fichier ne couvrait que Quad, et `make surface` **trie** sa sortie, donc
// permuter deux de ces six champs — tous du même type — déplace tous les décalages
// sans produire une seule ligne de différence.
//
// Les six champs sont des `uint32` consécutifs, donc leurs décalages sont ceux que la
// spécification impose sans remplissage. Les écrire ici les rend irrévocables, ce
// qu'ils sont de fait depuis que la godoc annonce le nombre.
func TestOverlapLayout(t *testing.T) {
	var o Overlap

	if got, want := unsafe.Sizeof(o), uintptr(24); got != want {
		t.Errorf("taille d'Overlap : %d, attendu %d", got, want)
	}
	if got, want := unsafe.Alignof(o), uintptr(4); got != want {
		t.Errorf("alignement d'Overlap : %d, attendu %d", got, want)
	}

	positions := []struct {
		field string
		got   uintptr
		want  uintptr
	}{
		{"Identity", unsafe.Offsetof(o.Identity), 0},
		{"Hidden", unsafe.Offsetof(o.Hidden), 4},
		{"ByDecor", unsafe.Offsetof(o.ByDecor), 8},
		{"Painted", unsafe.Offsetof(o.Painted), 12},
		{"Covered", unsafe.Offsetof(o.Covered), 16},
		{"CoveredByDecor", unsafe.Offsetof(o.CoveredByDecor), 20},
	}
	for _, p := range positions {
		if p.got != p.want {
			t.Errorf("position de %s : %d, attendu %d", p.field, p.got, p.want)
		}
	}
}

// TestContiguity vérifie que les deux listes qu'un hôte reçoit par pointeur et
// longueur sont bien des tableaux contigus, sans en-tête ni remplissage entre leurs
// éléments.
//
// Les deux et non la seule DrawList : OverlapList porte mot pour mot le même contrat
// dans sa documentation, et rien ne l'éprouvait.
func TestContiguity(t *testing.T) {
	t.Run("DrawList", func(t *testing.T) {
		r := DrawList{{}, {}, {}}

		gap := uintptr(unsafe.Pointer(&r[1])) - uintptr(unsafe.Pointer(&r[0]))
		if want := unsafe.Sizeof(r[0]); gap != want {
			t.Errorf("écart entre deux quads : %d, attendu %d", gap, want)
		}
	})

	t.Run("OverlapList", func(t *testing.T) {
		l := OverlapList{{}, {}, {}}

		gap := uintptr(unsafe.Pointer(&l[1])) - uintptr(unsafe.Pointer(&l[0]))
		if want := unsafe.Sizeof(l[0]); gap != want {
			t.Errorf("écart entre deux relevés : %d, attendu %d", gap, want)
		}
	})
}

// TestFlagsDistinct vérifie que les quatre variantes de dessin occupent des
// bits séparés, donc qu'elles se combinent.
func TestFlagsDistinct(t *testing.T) {
	if MirrorX|MirrorY|Flat|Additive != 0b1111 {
		t.Errorf("drapeaux chevauchants : %#b, %#b, %#b, %#b",
			MirrorX, MirrorY, Flat, Additive)
	}
}

// TestWhiteNeutral vérifie que la teinte neutre est blanche et opaque, valeur sur
// laquelle le rastériseur reconnaît une copie directe.
func TestWhiteNeutral(t *testing.T) {
	if got := White(); got != (Color{255, 255, 255, 255}) {
		t.Errorf("teinte neutre : %v", got)
	}
}
