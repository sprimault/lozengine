// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la séparation de la surface et de l'aire de monde doit tenir : la surface qui
// vaut la vue quand personne ne l'élargit, la place de l'aire selon l'alignement, et le
// repère du pixel désigné une fois qu'une marge existe.

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
)

// layoutScene monte une scène cadrée sur une aire de monde de la taille donnée.
func layoutScene(t *testing.T, width, height int32) *Scene {
	t.Helper()

	s, err := New(64, 64, geometry.Dimetric(32, 16), Options{Entities: 1})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(32, 32, width, height)
	return s
}

// TestSurfaceDefaultsToTheView garde ce qui rend le code d'un hôte d'avant la
// séparation encore juste : sans marge, il n'y a qu'un rectangle.
//
// C'est le seul cas à pouvoir attraper ce défaut-là, et il attraperait le pire des
// deux : une surface laissée à zéro donnerait une origine d'aire négative, donc des
// quads décalés hors de l'image sans qu'aucune erreur ne soit rendue.
func TestSurfaceDefaultsToTheView(t *testing.T) {
	s := layoutScene(t, 120, 80)

	if width, height := s.Surface(); width != 120 || height != 80 {
		t.Errorf("surface %d×%d, attendue égale à la vue 120×80", width, height)
	}
	if x, y := s.WorldOrigin(); x != 0 || y != 0 {
		t.Errorf("origine de l'aire (%d, %d), attendue (0, 0) sans marge", x, y)
	}
}

// TestWorldOriginFollowsTheAlignment garde les trois places qu'une aire peut prendre,
// et le partage du reste d'une marge impaire.
//
// La dernière ligne est celle qui compte le plus : [AlignCenter] sur une marge de
// soixante-dix-sept pixels doit rendre trente-huit, le pixel de reste allant après
// l'aire comme pour les bandes de l'agrandissement. Sans cette convention, un jeu porte
// deux centrages distants d'un pixel et cherche l'écart dans sa propre mise en page.
func TestWorldOriginFollowsTheAlignment(t *testing.T) {
	for _, c := range []struct {
		name           string
		alignX, alignY Alignment
		surfaceWidth   int32
		wantX, wantY   int32
	}{
		{"centrée, marges paires", AlignCenter, AlignCenter, 64, 24, 16},
		{"au début, toute la marge après", AlignStart, AlignStart, 64, 0, 0},
		{"à la fin, toute la marge avant", AlignEnd, AlignEnd, 64, 48, 32},
		{"un axe de chaque", AlignStart, AlignEnd, 64, 0, 32},
		{"centrée sur une marge impaire de 77, le reste va après", AlignCenter, AlignCenter, 93, 38, 16},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := layoutScene(t, 16, 32)
			s.ResizeSurface(c.surfaceWidth, 64)
			s.SetAlignment(c.alignX, c.alignY)

			if x, y := s.WorldOrigin(); x != c.wantX || y != c.wantY {
				t.Errorf("origine de l'aire (%d, %d), attendue (%d, %d)", x, y, c.wantX, c.wantY)
			}
		})
	}
}

// TestAtReadsASurfacePixel garde le repère de la désignation au pointeur, et c'est le
// point le plus exposé de la séparation.
//
// Un pixel arrive en coordonnées de surface, pas d'aire : la boucle a défait
// l'agrandissement et les bandes, rien de plus. Si [Scene.At] ne retranche pas la place
// de l'aire, tous les clics tombent décalés d'exactement la marge — sans erreur, sans
// trace, et avec l'apparence d'un défaut de projection qu'on ira chercher dans la
// matrice, où il n'est pas.
//
// Le cas l'éprouve par égalité plutôt que par un point recalculé : le même pixel de
// l'aire doit désigner le même point du monde, que l'aire remplisse la surface ou
// qu'elle y soit décalée. Reprojeter dans le test referait l'arithmétique mesurée.
func TestAtReadsASurfacePixel(t *testing.T) {
	flush := layoutScene(t, 16, 32)
	wantX, wantY := flush.At(5, 7, 0)

	for _, c := range []struct {
		name           string
		alignX, alignY Alignment
	}{
		{"centrée", AlignCenter, AlignCenter},
		{"au début", AlignStart, AlignStart},
		{"à la fin", AlignEnd, AlignEnd},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := layoutScene(t, 16, 32)
			s.ResizeSurface(64, 64)
			s.SetAlignment(c.alignX, c.alignY)

			originX, originY := s.WorldOrigin()
			x, y := s.At(5+originX, 7+originY, 0)

			if x != wantX || y != wantY {
				t.Errorf("le pixel (5, 7) de l'aire désigne (%g, %g), attendu (%g, %g) — "+
					"la place de l'aire dans la surface n'a pas été retranchée",
					x, y, wantX, wantY)
			}
		})
	}
}
