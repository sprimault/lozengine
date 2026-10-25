// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// ligneDAlphas construit une planche d'une seule ligne dont les alphas sont donnés,
// et l'encode. Les composantes de couleur ne jouent aucun rôle dans le classement.
func ligneDAlphas(alphas ...uint8) Sheet {
	p := NewSheet(uint16(len(alphas)), 1)
	for i, a := range alphas {
		p.Pixels[i] = rendu.Couleur{R: a, V: a, B: a, A: a}
	}
	p.encode()
	return p
}

// TestEncoderClasseLesLignes vérifie le découpage d'une ligne en séquences, cas par
// cas : c'est de lui que le blit tire ses trois chemins.
func TestEncoderClasseLesLignes(t *testing.T) {
	cas := []struct {
		nom    string
		alphas []uint8
		veut   []run
	}{
		{"ligne vide", []uint8{0, 0, 0}, []run{{3, transparent}}},
		{"ligne opaque", []uint8{255, 255}, []run{{2, opaque}}},
		{"ligne en demi-teinte", []uint8{128, 200}, []run{{2, translucent}}},
		{
			"sprite : marges, corps, bord adouci",
			[]uint8{0, 0, 60, 255, 255, 255, 60, 0},
			[]run{{2, transparent}, {1, translucent}, {3, opaque}, {1, translucent}, {1, transparent}},
		},
		{
			"alternance, le pire cas",
			[]uint8{0, 255, 0, 255},
			[]run{{1, transparent}, {1, opaque}, {1, transparent}, {1, opaque}},
		},
		{"alpha 254 n'est pas opaque", []uint8{254}, []run{{1, translucent}}},
		{"alpha 1 n'est pas transparent", []uint8{1}, []run{{1, translucent}}},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := ligneDAlphas(c.alphas...)

			got := p.runsOf(0)
			if len(got) != len(c.veut) {
				t.Fatalf("séquences : %v, attendu %v", got, c.veut)
			}
			for i := range got {
				if got[i] != c.veut[i] {
					t.Errorf("séquence %d : %v, attendu %v", i, got[i], c.veut[i])
				}
			}
		})
	}
}

// TestEncoderPaveChaqueLigne vérifie l'invariant dont le blit dépend sans le
// vérifier : les séquences d'une ligne la couvrent exactement, ce qui autorise à
// déduire la colonne de départ en avançant plutôt qu'à la stocker.
func TestEncoderPaveChaqueLigne(t *testing.T) {
	p := NewSheet(5, 3)
	for i := range p.Pixels {
		p.Pixels[i] = rendu.Couleur{A: uint8(i % 3 * 127)}
	}
	p.encode()

	for y := uint16(0); y < p.Height; y++ {
		total := 0
		for _, s := range p.runsOf(y) {
			total += int(s.count)
		}
		if total != int(p.Width) {
			t.Errorf("ligne %d : %d pixels couverts, attendu %d", y, total, p.Width)
		}
	}
}

// TestEncoderSeparelesLignes vérifie qu'une ligne ne déborde pas sur la suivante :
// deux lignes identiques et uniformes doivent rendre deux séquences, pas une.
func TestEncoderSepareLesLignes(t *testing.T) {
	p := NewSheet(4, 2)
	for i := range p.Pixels {
		p.Pixels[i] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	}
	p.encode()

	if len(p.runs) != 2 {
		t.Errorf("séquences : %v, attendu une par ligne", p.runs)
	}
	for y := uint16(0); y < p.Height; y++ {
		if got := p.runsOf(y); len(got) != 1 || got[0] != (run{4, opaque}) {
			t.Errorf("ligne %d : %v", y, got)
		}
	}
}

// TestEncoderPlancheVide vérifie qu'une planche sans hauteur s'encode sans borne
// fausse : le tableau de bornes en porte toujours une de plus que de lignes.
func TestEncoderPlancheVide(t *testing.T) {
	p := NewSheet(0, 0)
	p.encode()

	if len(p.rows) != 1 || p.rows[0] != 0 {
		t.Errorf("bornes : %v", p.rows)
	}
	if len(p.runs) != 0 {
		t.Errorf("séquences : %v, attendu aucune", p.runs)
	}
}

// TestAjouterEncode vérifie le passage obligé : une planche construite à la main
// ressort de l'atlas encodée, sans que l'appelant ait eu à le demander.
func TestAjouterEncode(t *testing.T) {
	var a Atlas

	p := NewSheet(2, 1)
	p.Pixels[0] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

	index, err := a.Add(p)
	if err != nil {
		t.Fatalf("ajout : %v", err)
	}

	rangee := a.Sheets[index]
	if got := rangee.runsOf(0); len(got) != 2 {
		t.Errorf("séquences de la planche rangée : %v, attendu une copie puis un saut", got)
	}
}
