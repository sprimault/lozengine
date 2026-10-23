// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

import (
	"testing"
	"unsafe"
)

// TestDisposition fige la disposition mémoire de Quad, que l'exposition à l'ABI C
// rend irrévocable : un champ déplacé ou élargi déplace tout ce qui suit et casse
// silencieusement un hôte déjà compilé. Le test échoue avant lui.
func TestDisposition(t *testing.T) {
	var q Quad

	if got, veut := unsafe.Sizeof(q), uintptr(20); got != veut {
		t.Errorf("taille de Quad : %d, attendu %d", got, veut)
	}
	if got, veut := unsafe.Alignof(q), uintptr(2); got != veut {
		t.Errorf("alignement de Quad : %d, attendu %d", got, veut)
	}

	positions := []struct {
		champ string
		got   uintptr
		veut  uintptr
	}{
		{"X", unsafe.Offsetof(q.X), 0},
		{"Y", unsafe.Offsetof(q.Y), 2},
		{"SourceX", unsafe.Offsetof(q.SourceX), 4},
		{"SourceY", unsafe.Offsetof(q.SourceY), 6},
		{"Largeur", unsafe.Offsetof(q.Largeur), 8},
		{"Hauteur", unsafe.Offsetof(q.Hauteur), 10},
		{"Planche", unsafe.Offsetof(q.Planche), 12},
		{"Teinte", unsafe.Offsetof(q.Teinte), 14},
		{"Drapeaux", unsafe.Offsetof(q.Drapeaux), 18},
	}
	for _, p := range positions {
		if p.got != p.veut {
			t.Errorf("position de %s : %d, attendu %d", p.champ, p.got, p.veut)
		}
	}

	if got, veut := unsafe.Sizeof(Couleur{}), uintptr(4); got != veut {
		t.Errorf("taille de Couleur : %d, attendu %d", got, veut)
	}
}

// TestContiguite vérifie qu'un Rendu est bien le tableau contigu de quads qu'un
// hôte reçoit par pointeur et longueur, sans en-tête ni remplissage entre eux.
func TestContiguite(t *testing.T) {
	r := Rendu{{}, {}, {}}

	ecart := uintptr(unsafe.Pointer(&r[1])) - uintptr(unsafe.Pointer(&r[0]))
	if veut := unsafe.Sizeof(r[0]); ecart != veut {
		t.Errorf("écart entre deux quads : %d, attendu %d", ecart, veut)
	}
}

// TestDrapeauxDistincts vérifie que les trois variantes de dessin occupent des
// bits séparés, donc qu'elles se combinent.
func TestDrapeauxDistincts(t *testing.T) {
	if MiroirX|MiroirY|Aplat != 0b111 {
		t.Errorf("drapeaux chevauchants : %#b, %#b, %#b", MiroirX, MiroirY, Aplat)
	}
}

// TestBlancNeutre vérifie que la teinte neutre est blanche et opaque, valeur sur
// laquelle le rastériseur reconnaît une copie directe.
func TestBlancNeutre(t *testing.T) {
	if got := Blanc(); got != (Couleur{255, 255, 255, 255}) {
		t.Errorf("teinte neutre : %v", got)
	}
}
