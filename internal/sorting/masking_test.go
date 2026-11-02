// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le relevé doit classer d'après la nature déclarée d'un masquant plutôt que
// d'après la voie par laquelle il a été posé.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// masked rend le relevé de l'unique entité suivie d'une scène que `cover` recouvre.
func masked(t *testing.T, cover func(*scene.Scene)) render.Overlap {
	t.Helper()

	s, _ := measuredScene(t, cover)
	sorter := New(s)
	sorter.Frame()

	list := sorter.Overlaps()
	if len(list) != 1 {
		t.Fatalf("%d relevés, attendu 1", len(list))
	}
	return list[0]
}

// TestMaskingFollowsTheDeclaredNature garde les quatre combinaisons de la voie de pose et
// de la nature déclarée, et c'est le cas qui porte le lot.
//
// **Les deux premières lignes sont le comportement d'avant**, et elles doivent rester
// vraies : la valeur zéro laisse la voie décider, donc un jeu qui ne renseigne rien ne
// voit rien changer. Les deux suivantes sont ce que le lot ajoute, et elles rougissent
// sans lui — le relevé y rend l'inverse de ce qui est attendu, pas une valeur nulle.
func TestMaskingFollowsTheDeclaredNature(t *testing.T) {
	for _, c := range []struct {
		name    string
		masks   scene.Masking
		asCell  bool
		asDecor bool
	}{
		{"posé en cellule, rien de déclaré", scene.MaskAsPlaced, true, true},
		{"posé en entité, rien de déclaré", scene.MaskAsPlaced, false, false},
		{"posé en entité, déclaré décor", scene.MaskAsDecor, false, true},
		{"posé en cellule, déclaré entité", scene.MaskAsEntity, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := masked(t, func(s *scene.Scene) {
				a := wall()
				a.Masks = c.masks
				if c.asCell {
					if !s.PlaceCell(9, 9, 0, a) {
						t.Fatal("mur en cellule refusé")
					}
					return
				}
				if _, ok := s.Place(scene.Entity{X: 9, Y: 9, Aspect: a}); !ok {
					t.Fatal("mur en entité refusé")
				}
			})

			if got.Hidden == 0 {
				t.Fatal("rien n'est caché, le cas ne garde rien")
			}
			if (got.ByDecor != 0) != c.asDecor {
				t.Errorf("%d pixels de décor sur %d cachés, attendu %v",
					got.ByDecor, got.Hidden, map[bool]string{true: "du décor", false: "aucun"}[c.asDecor])
			}
		})
	}
}

// TestMaskingKeepsTheDecorLawAcrossBothPlacements garde l'égalité que le constat du
// consommateur a vue tomber : une chose cachée par les deux voies à la fois, toutes deux
// déclarées décor, doit voir tout son recouvrement compté comme décor.
//
// **C'est la forme observable du défaut**, et aucun des cas précédents ne la couvre : ils
// n'emploient qu'une voie, donc ils ne peuvent pas montrer deux colonnes en désaccord sur
// une même chose. Un hôte qui lit cette égalité pour choisir entre deux rendus basculait
// de branche sans l'avoir décidé.
//
// **À l'échelle de la boîte seulement, et ce n'est pas un choix** : la mesure au pixel lit
// les pixels des planches, donc elle rend zéro sans atlas et ce paquet n'en monte pas.
// L'égalité au pixel que le constat nomme — `CoveredByDecor == Covered` — se garde à la
// racine, où un atlas existe. Les deux échelles lisent le même drapeau, donc elles ne
// peuvent pas diverger.
func TestMaskingKeepsTheDecorLawAcrossBothPlacements(t *testing.T) {
	got := masked(t, func(s *scene.Scene) {
		a := wall()
		a.Masks = scene.MaskAsDecor

		if !s.PlaceCell(9, 9, 0, a) {
			t.Fatal("mur en cellule refusé")
		}
		if _, ok := s.Place(scene.Entity{X: 9, Y: 9, Aspect: a}); !ok {
			t.Fatal("mur en entité refusé")
		}
	})

	if got.Hidden == 0 {
		t.Fatal("rien n'est caché, le cas ne garde rien")
	}
	if got.ByDecor != got.Hidden {
		t.Errorf("%d pixels de décor sur %d cachés : la voie de pose décide encore, alors "+
			"que les deux masquants sont déclarés décor", got.ByDecor, got.Hidden)
	}
}
