// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le fournisseur de référence doit tenir : le rectangle qu'il prend dans sa
// planche, le repli d'une demande sans réponse, et ce qu'il laisse au jeu.

package appearance

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// walking rend une entrée de marche, quatre images sur une planche de tuiles.
func walking() Entry {
	return Entry{
		Sheet: 3, Width: 24, Height: 32,
		Tint: render.White(),
		Clip: Clip{Frames: 4},
	}
}

// TestAspectCutsTheGrid vérifie que la direction choisit la colonne et l'image la
// ligne, dans l'ordre que le découpage d'une planche produit.
func TestAspectCutsTheGrid(t *testing.T) {
	s := NewSheets()
	s.Set(1, 0, walking())

	for _, c := range []struct {
		name         string
		facing       Facing
		frame        int
		wantX, wantY uint16
	}{
		{"sud, première image", 0, 0, 0, 0},
		{"troisième direction", 2, 0, 48, 0},
		{"deuxième image", 0, 1, 0, 32},
		{"les deux à la fois", 5, 3, 120, 96},
		{"une direction qui déborde revient", Facing(Directions), 0, 0, 0},
		{"une image qui déborde revient", 0, 4, 0, 0},
		{"une image négative revient aussi", 0, -1, 0, 96},
	} {
		a := s.Aspect(1, 0, c.facing, c.frame)
		if a.SourceX != c.wantX || a.SourceY != c.wantY {
			t.Errorf("%s : source en (%d, %d), attendu (%d, %d)",
				c.name, a.SourceX, a.SourceY, c.wantX, c.wantY)
		}
	}
}

// TestAspectCarriesTheEntry vérifie que ce qu'une entrée déclare arrive dans l'aspect.
func TestAspectCarriesTheEntry(t *testing.T) {
	e := walking()
	e.AnchorX, e.AnchorY = -3, 5
	e.Flags = render.MirrorX

	s := NewSheets()
	s.Set(7, 2, e)

	a := s.Aspect(7, 2, 0, 0)
	if a.Sheet != e.Sheet || a.Width != e.Width || a.Height != e.Height {
		t.Errorf("planche %d de %d×%d, attendu %d de %d×%d",
			a.Sheet, a.Width, a.Height, e.Sheet, e.Width, e.Height)
	}
	if a.AnchorX != e.AnchorX || a.AnchorY != e.AnchorY {
		t.Errorf("ancre (%d, %d), attendu (%d, %d)",
			a.AnchorX, a.AnchorY, e.AnchorX, e.AnchorY)
	}
	if a.Tint != e.Tint || a.Flags != e.Flags {
		t.Errorf("teinte %v drapeaux %v, attendu %v et %v",
			a.Tint, a.Flags, e.Tint, e.Flags)
	}
}

// TestAspectLeavesLayerAndFloorToTheGame garde la frontière entre ce qu'une chose
// montre et où elle se range.
//
// Un personnage qui grimpe change d'étage sans changer d'apparence : poser l'étage
// depuis l'entrée le figerait à ce que la planche a été déclarée valoir, et le jeu
// n'aurait aucun moyen de le corriger — l'aspect étant écrit en entier à chaque
// changement.
func TestAspectLeavesLayerAndFloorToTheGame(t *testing.T) {
	s := NewSheets()
	s.Set(1, 0, walking())

	if a := s.Aspect(1, 0, 3, 1); a.Layer != 0 || a.Floor != 0 {
		t.Errorf("couche %d étage %d, attendu zéro pour les deux", a.Layer, a.Floor)
	}
}

// TestAspectWithoutAnEntryDrawsNothing garde le repli d'une demande sans réponse.
//
// La valeur zéro plutôt que la planche d'autre chose : un trou se voit et se cherche,
// un mauvais sprite se discute. Et c'est le régime qu'un jeu atteint en se trompant
// d'état, ce qui arrive bien plus souvent qu'en se trompant de type.
func TestAspectWithoutAnEntryDrawsNothing(t *testing.T) {
	s := NewSheets()
	s.Set(1, 0, walking())

	if a := s.Aspect(1, 9, 0, 0); a != (scene.Aspect{}) {
		t.Errorf("%+v, attendu la valeur zéro", a)
	}
}

// TestSetReplacesWhatWasThere vérifie qu'une entrée déclarée deux fois garde la
// dernière, ce qu'un chargement qui rejoue son manifeste demande.
func TestSetReplacesWhatWasThere(t *testing.T) {
	s := NewSheets()
	s.Set(1, 0, walking())

	second := walking()
	second.Sheet = 9
	s.Set(1, 0, second)

	if a := s.Aspect(1, 0, 0, 0); a.Sheet != 9 {
		t.Errorf("planche %d, attendu 9", a.Sheet)
	}
}

// TestSheetsDeclaresItsDirections garde le nombre que le fournisseur annonce, qui est
// ce que [Turn] et le jeu lisent plutôt que de le supposer.
func TestSheetsDeclaresItsDirections(t *testing.T) {
	if got := NewSheets().Directions(); got != Directions {
		t.Errorf("%d directions, attendu %d", got, Directions)
	}
}

// TestSheetsIsAProvider garde que le fournisseur de référence remplit l'interface, ce
// que rien d'autre ne vérifierait : le paquet ne s'en sert nulle part.
func TestSheetsIsAProvider(t *testing.T) {
	var _ Provider = NewSheets()
}
