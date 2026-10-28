// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la désignation doit tenir : retrouver la cellule qu'un pixel montre, sous les
// quatre orientations et à tous les étages.

package scene

import (
	"math"
	"testing"

	"github.com/sprimault/lozengine/geometry"
)

// TestAtFindsTheCellUnderAPixel vérifie que le point rendu par [Scene.At] est celui
// que la projection a envoyé sur ce pixel.
//
// **L'aller et le retour sont deux fonctions distinctes**, donc le cas ne se sert pas
// de ce qu'il éprouve pour dire ce qu'il attend : il projette le centre d'une cellule,
// retranche la vue comme le tri le fait, et redemande le point à la désignation.
//
// Les quatre orientations comptent parce que le remappage se défait dans ce chemin, et
// qu'une erreur de sens n'y est visible nulle part ailleurs : à orientation nulle les
// deux se confondent. Les étages comptent parce que la montée à l'écran s'y retranche,
// et qu'une case d'étage désignée sans elle renverrait celle d'en dessous.
func TestAtFindsTheCellUnderAPixel(t *testing.T) {
	const side = 12

	s, err := New(side, side, geometry.Dimetric(32, 16), Options{})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	for _, o := range []Orientation{NoRotation, QuarterTurn, HalfTurn, ThreeQuarters} {
		s.Orient(o)
		s.Focus(side/2, side/2, 480, 270)
		viewX, viewY, _, _ := s.View()

		for _, floor := range []uint8{0, 1, 3} {
			for _, cell := range [][2]int{{0, 0}, {3, 7}, {6, 6}, {11, 11}} {
				wantX, wantY := float64(cell[0])+0.5, float64(cell[1])+0.5

				rx, ry := s.Remap(wantX, wantY)
				ex, ey, _, _ := s.Projection().Project(rx, ry, float64(floor))

				gotX, gotY := s.At(ex-viewX, ey-viewY, floor)

				if math.Abs(gotX-wantX) > 0.05 || math.Abs(gotY-wantY) > 0.05 {
					t.Errorf("orientation %d, étage %d, cellule %v : (%.3f, %.3f), attendu (%.1f, %.1f)",
						o, floor, cell, gotX, gotY, wantX, wantY)
				}
			}
		}
	}
}

// TestAtSubtractsTheFloorRise vérifie qu'un étage change le point désigné, et du bon
// nombre de cellules.
//
// Sans ce cas, la désignation pourrait ignorer l'étage et passer le précédent : à
// orientation nulle et sans montée, un pixel désigne la même case quel que soit
// l'étage demandé, donc rien ne distinguerait les deux codes.
func TestAtSubtractsTheFloorRise(t *testing.T) {
	const side = 12

	s, err := New(side, side, geometry.Dimetric(32, 16), Options{})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(side/2, side/2, 480, 270)

	groundX, groundY := s.At(0, 0, 0)
	upperX, upperY := s.At(0, 0, 1)

	if groundX == upperX && groundY == upperY {
		t.Fatal("l'étage ne change rien au point désigné")
	}

	// Un étage monte par défaut de la hauteur d'une cellule, soit seize pixels
	// d'ordonnée ; et une cellule n'apporte que huit pixels d'ordonnée par axe, la
	// projection partageant la hauteur entre les deux. Redescendre de seize avance
	// donc d'une cellule **entière** sur chacun des deux axes, et non d'une demie.
	if dx, dy := upperX-groundX, upperY-groundY; math.Abs(dx-1) > 0.05 || math.Abs(dy-1) > 0.05 {
		t.Errorf("un étage décale de (%.3f, %.3f) cellule, attendu une sur chaque axe", dx, dy)
	}
}

// TestRiseFollowsTheFloorHeight garde la montée d'une hauteur, qui est la pièce que la
// désignation défait.
func TestRiseFollowsTheFloorHeight(t *testing.T) {
	m := geometry.Dimetric(32, 16)

	if got := m.Rise(1); got != 16 {
		t.Errorf("une hauteur monte de %d pixels, attendu 16", got)
	}
	if got := m.WithFloorHeight(24).Rise(2); got != 48 {
		t.Errorf("deux hauteurs ajustées montent de %d pixels, attendu 48", got)
	}
	if got := m.Rise(0); got != 0 {
		t.Errorf("une hauteur nulle monte de %d pixels, attendu 0", got)
	}
}
