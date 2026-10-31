// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le texte doit tenir : une mesure qui s'accorde avec la pose, un caractère
// absent qui ne décale pas la suite, et le découpage de la planche de glyphes.

package appearance

import (
	"testing"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// ascii rend une police couvrant les vingt-six majuscules sur deux rangées.
func ascii() Font {
	return Font{
		Sheet: 5, Width: 6, Height: 10,
		Columns: 13, Count: 26, First: 'A',
		Tint: render.White(), Layer: 3,
	}
}

// textScene monte une scène vide, de quoi recevoir des pièces d'écran.
//
// Sur `testing.TB` et non sur `*testing.T` : les bancs du paquet veulent le même
// montage, et deux helpers qui divergeraient feraient mesurer autre chose que ce que
// les cas éprouvent.
func textScene(t testing.TB) *scene.Scene {
	t.Helper()

	s, err := scene.New(8, 8, geometry.Dimetric(32, 16), scene.Options{})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	s.Focus(4, 4, 480, 270)
	return s
}

// TestMeasureAgreesWithPlace garde l'accord entre les deux, qui est la raison d'être de
// la mesure.
//
// **Un jeu centre un texte avec la mesure puis le pose avec la pose** : si les deux
// divergeaient d'un pixel, tout ce qui est centré serait décalé, et le défaut se
// chercherait dans le jeu. Le cas les compare directement, sur un texte qui porte un
// caractère absent pour que le régime du trou soit couvert aussi.
// **Les deux polices comptent, et la seconde est celle qui garde quelque chose** : à
// chasse pleine, l'avance vaut la largeur de cellule et les deux chemins ne peuvent pas
// diverger, donc le cas passerait même si l'un ignorait l'avance. C'est en resserrant
// qu'il devient capable de rougir.
func TestMeasureAgreesWithPlace(t *testing.T) {
	tight := ascii()
	tight.Advance = 4

	for _, f := range []Font{ascii(), tight} {
		for _, text := range []string{"", "A", "CAB", "AZ.B", "???"} {
			width, height := f.Measure(text)
			if height != int(f.Height) {
				t.Errorf("avance %d, %q : hauteur %d, attendue %d",
					f.Advance, text, height, f.Height)
			}

			if placed := f.Place(textScene(t), 0, 0, text); placed != width {
				t.Errorf("avance %d, %q : posé sur %d pixels, mesuré %d",
					f.Advance, text, placed, width)
			}
		}
	}
}

// TestMissingGlyphKeepsItsPlace vérifie qu'un caractère absent ne pose rien et ne
// décale pas la suite de la ligne.
//
// C'est ce qui laisse deux lignes s'aligner quand l'une porte un signe que la planche
// n'a pas — un nom propre accentué dans une liste, par exemple.
func TestMissingGlyphKeepsItsPlace(t *testing.T) {
	f := ascii()

	s := textScene(t)
	f.Place(s, 0, 0, "A.B")

	pieces := scene.ScreenPieces(s)
	if len(pieces) != 2 {
		t.Fatalf("%d glyphes posés, attendu 2", len(pieces))
	}

	// Le B est au troisième rang de la ligne, donc à deux pas du A, bien que le
	// point n'ait rien posé entre eux.
	if got, want := pieces[1].X-pieces[0].X, int32(2*f.Width); got != want {
		t.Errorf("le B est à %d pixels du A, attendu %d", got, want)
	}
}

// TestGlyphCutsTheGrid vérifie que le rang d'un caractère choisit sa cellule, en
// colonnes puis en rangées.
func TestGlyphCutsTheGrid(t *testing.T) {
	f := ascii()

	for _, c := range []struct {
		name         string
		r            rune
		wantX, wantY uint16
	}{
		{"le premier est en haut à gauche", 'A', 0, 0},
		{"le troisième avance d'autant", 'C', 12, 0},
		{"le quatorzième passe à la rangée suivante", 'N', 0, 10},
		{"le dernier est en bas à droite", 'Z', 72, 10},
	} {
		a, ok := f.Glyph(c.r)
		if !ok {
			t.Errorf("%s : %q absent", c.name, c.r)
			continue
		}
		if a.SourceX != c.wantX || a.SourceY != c.wantY {
			t.Errorf("%s : source en (%d, %d), attendue (%d, %d)",
				c.name, a.SourceX, a.SourceY, c.wantX, c.wantY)
		}
	}
}

// TestGlyphRefusesWhatThePlateDoesNotCarry garde les trois façons de sortir de la
// planche : avant le premier caractère, après le dernier, et une police vide.
func TestGlyphRefusesWhatThePlateDoesNotCarry(t *testing.T) {
	f := ascii()

	for _, c := range []struct {
		name string
		f    Font
		r    rune
	}{
		{"avant le premier", f, '0'},
		{"après le dernier", f, 'a'},
		{"police vide", Font{}, 'A'},
	} {
		if a, ok := c.f.Glyph(c.r); ok || a != (scene.Aspect{}) {
			t.Errorf("%s : %q rendu %+v, attendu rien", c.name, c.r, a)
		}
	}
}

// TestAdvanceTightensTheLine vérifie qu'un pas plus serré que la cellule resserre la
// ligne sans changer le découpage de la planche.
func TestAdvanceTightensTheLine(t *testing.T) {
	f := ascii()
	f.Advance = 4

	if width, _ := f.Measure("ABC"); width != 12 {
		t.Errorf("largeur %d, attendue 12", width)
	}

	a, ok := f.Glyph('B')
	if !ok {
		t.Fatal("B absent")
	}
	if a.Width != f.Width || a.SourceX != f.Width {
		t.Errorf("glyphe de %d de large en %d, attendu %d et %d",
			a.Width, a.SourceX, f.Width, f.Width)
	}
}

// TestPlaceAreaMarksEveryGlyph garde que le repère choisi atteint bien **chaque** glyphe,
// et pas seulement le premier.
//
// C'est le défaut qu'une boucle écrite deux fois laisserait passer : un glyphe oublié ne
// se verrait qu'avec une marge, sur le seul caractère qui déborde de l'aire, et le mot
// paraîtrait simplement rogné d'une lettre. Les deux voies sont éprouvées ensemble, la
// valeur par défaut devant rester la surface.
func TestPlaceAreaMarksEveryGlyph(t *testing.T) {
	for _, c := range []struct {
		name   string
		place  func(Font, *scene.Scene)
		inArea bool
	}{
		{"Place vise la surface", func(f Font, s *scene.Scene) { f.Place(s, 10, 20, "ABC") }, false},
		{"PlaceArea vise l'aire", func(f Font, s *scene.Scene) { f.PlaceArea(s, 10, 20, "ABC") }, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := ascii()
			s := textScene(t)
			c.place(f, s)

			pieces := scene.ScreenPieces(s)
			if len(pieces) != 3 {
				t.Fatalf("%d glyphes posés, attendu 3", len(pieces))
			}
			for i, p := range pieces {
				if p.InArea != c.inArea {
					t.Errorf("glyphe %d posé dans l'aire = %v, attendu %v",
						i, p.InArea, c.inArea)
				}
			}
		})
	}
}

// TestPlacedTextCarriesTheFontLayer garde ce qui met le texte au-dessus du monde : sa
// couche vient de la police, et c'est le seul moyen d'y échapper au décor.
func TestPlacedTextCarriesTheFontLayer(t *testing.T) {
	f := ascii()

	s := textScene(t)
	f.Place(s, 10, 20, "AB")

	for i, p := range scene.ScreenPieces(s) {
		if p.Layer != f.Layer {
			t.Errorf("glyphe %d en couche %d, attendue %d", i, p.Layer, f.Layer)
		}
	}
}
