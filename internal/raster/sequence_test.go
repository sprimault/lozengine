// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/internal/rendu"
)

// ligneDAlphas construit une planche d'une seule ligne dont les alphas sont donnés,
// et l'encode. Les composantes de couleur ne jouent aucun rôle dans le classement.
func ligneDAlphas(alphas ...uint8) Planche {
	p := NouvellePlanche(uint16(len(alphas)), 1)
	for i, a := range alphas {
		p.Pixels[i] = rendu.Couleur{R: a, V: a, B: a, A: a}
	}
	p.encoder()
	return p
}

// TestEncoderClasseLesLignes vérifie le découpage d'une ligne en séquences, cas par
// cas : c'est de lui que le blit tire ses trois chemins.
func TestEncoderClasseLesLignes(t *testing.T) {
	cas := []struct {
		nom    string
		alphas []uint8
		veut   []sequence
	}{
		{"ligne vide", []uint8{0, 0, 0}, []sequence{{3, saut}}},
		{"ligne opaque", []uint8{255, 255}, []sequence{{2, copie}}},
		{"ligne en demi-teinte", []uint8{128, 200}, []sequence{{2, melange}}},
		{
			"sprite : marges, corps, bord adouci",
			[]uint8{0, 0, 60, 255, 255, 255, 60, 0},
			[]sequence{{2, saut}, {1, melange}, {3, copie}, {1, melange}, {1, saut}},
		},
		{
			"alternance, le pire cas",
			[]uint8{0, 255, 0, 255},
			[]sequence{{1, saut}, {1, copie}, {1, saut}, {1, copie}},
		},
		{"alpha 254 n'est pas opaque", []uint8{254}, []sequence{{1, melange}}},
		{"alpha 1 n'est pas transparent", []uint8{1}, []sequence{{1, melange}}},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := ligneDAlphas(c.alphas...)

			got := p.sequencesDe(0)
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
	p := NouvellePlanche(5, 3)
	for i := range p.Pixels {
		p.Pixels[i] = rendu.Couleur{A: uint8(i % 3 * 127)}
	}
	p.encoder()

	for y := uint16(0); y < p.Hauteur; y++ {
		total := 0
		for _, s := range p.sequencesDe(y) {
			total += int(s.nombre)
		}
		if total != int(p.Largeur) {
			t.Errorf("ligne %d : %d pixels couverts, attendu %d", y, total, p.Largeur)
		}
	}
}

// TestEncoderSeparelesLignes vérifie qu'une ligne ne déborde pas sur la suivante :
// deux lignes identiques et uniformes doivent rendre deux séquences, pas une.
func TestEncoderSepareLesLignes(t *testing.T) {
	p := NouvellePlanche(4, 2)
	for i := range p.Pixels {
		p.Pixels[i] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}
	}
	p.encoder()

	if len(p.sequences) != 2 {
		t.Errorf("séquences : %v, attendu une par ligne", p.sequences)
	}
	for y := uint16(0); y < p.Hauteur; y++ {
		if got := p.sequencesDe(y); len(got) != 1 || got[0] != (sequence{4, copie}) {
			t.Errorf("ligne %d : %v", y, got)
		}
	}
}

// TestEncoderPlancheVide vérifie qu'une planche sans hauteur s'encode sans borne
// fausse : le tableau de bornes en porte toujours une de plus que de lignes.
func TestEncoderPlancheVide(t *testing.T) {
	p := NouvellePlanche(0, 0)
	p.encoder()

	if len(p.lignes) != 1 || p.lignes[0] != 0 {
		t.Errorf("bornes : %v", p.lignes)
	}
	if len(p.sequences) != 0 {
		t.Errorf("séquences : %v, attendu aucune", p.sequences)
	}
}

// TestAjouterEncode vérifie le passage obligé : une planche construite à la main
// ressort de l'atlas encodée, sans que l'appelant ait eu à le demander.
func TestAjouterEncode(t *testing.T) {
	var a Atlas

	p := NouvellePlanche(2, 1)
	p.Pixels[0] = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

	index, err := a.Ajouter(p)
	if err != nil {
		t.Fatalf("ajout : %v", err)
	}

	rangee := a.Planches[index]
	if got := rangee.sequencesDe(0); len(got) != 2 {
		t.Errorf("séquences de la planche rangée : %v, attendu une copie puis un saut", got)
	}
}
