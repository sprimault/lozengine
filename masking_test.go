// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la nature déclarée d'un masquant change au relevé de recouvrement à l'échelle du
// pixel, que le noyau de tri ne peut pas éprouver faute d'atlas.

package lozengine

import "testing"

// maskedByAnEntity monte une chose suivie qu'une entité recouvre, et rend son relevé.
//
// L'entité qui cache est plus profonde d'une demi-cellule, donc elle se peint après ; elle
// est trois fois plus haute que la chose, donc elle en couvre réellement des pixels.
func maskedByAnEntity(t *testing.T, masks Masking) Overlap {
	t.Helper()

	s, err := NewScene(16, 16, Isometric(32), SceneOptions{})
	if err != nil {
		t.Fatalf("montage de la scène : %v", err)
	}

	// Une planche pleine et opaque : la mesure au pixel ne compte que ce qui peint, et
	// un trou y ferait dépendre le compte du dessin plutôt que du classement.
	sheet := NewSheet(32, 48)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = White()
	}
	var atlas Atlas
	if _, err := atlas.Add(sheet); err != nil {
		t.Fatalf("ajout de la planche : %v", err)
	}

	if _, ok := s.Place(Entity{X: 8.5, Y: 8.5, Measure: true,
		Aspect: Aspect{Width: 32, Height: 16, Tint: White()}}); !ok {
		t.Fatal("entité suivie refusée")
	}
	if _, ok := s.Place(Entity{X: 8.5, Y: 9.5, Aspect: Aspect{
		Width: 32, Height: 48, Tint: White(), Masks: masks}}); !ok {
		t.Fatal("entité qui cache refusée")
	}

	e, err := NewEngine(s, Options{Atlas: &atlas, Width: 256, Height: 256,
		MaxWidth: 256, MaxHeight: 256, Background: Color{A: 255}})
	if err != nil {
		t.Fatalf("montage du moteur : %v", err)
	}
	e.Step(nil, Input{})

	list := e.Overlaps()
	if len(list) != 1 {
		t.Fatalf("%d relevés, attendu 1", len(list))
	}
	if list[0].Covered == 0 {
		t.Fatal("rien n'est couvert au pixel, le montage ne garde rien")
	}
	return list[0]
}

// TestMaskingReachesThePixelScale garde que la nature déclarée d'un masquant porte
// jusqu'à la mesure au pixel, et pas seulement jusqu'à celle par rectangles.
//
// **C'est ici et pas dans le noyau de tri que ce cas peut vivre** : la mesure au pixel lit
// les pixels des planches, donc elle rend zéro sans atlas, et ce paquet est le seul à en
// monter un. Les deux échelles lisent le même drapeau mais par deux chemins différents, et
// celui du pixel n'a aucun autre gardien.
//
// **La forme à deux voies de pose que le constat d'origine décrit est gardée à l'échelle de
// la boîte**, dans le noyau de tri. Elle ne se transpose pas ici : au pixel chaque pixel
// couvert compte une fois, donc deux masquants qui se superposent couvrent exactement les
// mêmes pixels et les deux colonnes restent égales quel que soit le classement. Mesuré en
// écrivant ce cas, qui passait alors sans le correctif.
//
// Les deux régimes se mesurent sur le même montage, et la première comparaison est ce qui
// prouve que l'entité couvre bel et bien : sans elle, un montage où rien ne se recouvre
// vérifierait la seconde trivialement.
func TestMaskingReachesThePixelScale(t *testing.T) {
	placed := maskedByAnEntity(t, MaskAsPlaced)
	if placed.CoveredByDecor != 0 {
		t.Fatalf("%d pixels couverts par le décor alors qu'une entité seule recouvre : "+
			"la voie de pose ne décide plus rien, donc le cas ne peut rien garder",
			placed.CoveredByDecor)
	}

	declared := maskedByAnEntity(t, MaskAsDecor)
	if declared.CoveredByDecor != declared.Covered {
		t.Errorf("%d pixels couverts par le décor sur %d couverts : la nature déclarée "+
			"n'atteint pas la mesure au pixel", declared.CoveredByDecor, declared.Covered)
	}
}
