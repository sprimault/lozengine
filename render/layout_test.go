// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que l'exposition à l'ABI rend irrévocable : la taille et la position de chaque
// champ d'un quad, la contiguïté de la liste, et les deux valeurs qu'un hôte compare
// sans lire notre documentation.

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

// TestContiguity vérifie qu'une DrawList est bien le tableau contigu de quads qu'un
// hôte reçoit par pointeur et longueur, sans en-tête ni remplissage entre eux.
func TestContiguity(t *testing.T) {
	r := DrawList{{}, {}, {}}

	gap := uintptr(unsafe.Pointer(&r[1])) - uintptr(unsafe.Pointer(&r[0]))
	if want := unsafe.Sizeof(r[0]); gap != want {
		t.Errorf("écart entre deux quads : %d, attendu %d", gap, want)
	}
}

// TestFlagsDistinct vérifie que les trois variantes de dessin occupent des
// bits séparés, donc qu'elles se combinent.
func TestFlagsDistinct(t *testing.T) {
	if MirrorX|MirrorY|Flat != 0b111 {
		t.Errorf("drapeaux chevauchants : %#b, %#b, %#b", MirrorX, MirrorY, Flat)
	}
}

// TestWhiteNeutral vérifie que la teinte neutre est blanche et opaque, valeur sur
// laquelle le rastériseur reconnaît une copie directe.
func TestWhiteNeutral(t *testing.T) {
	if got := White(); got != (Color{255, 255, 255, 255}) {
		t.Errorf("teinte neutre : %v", got)
	}
}
