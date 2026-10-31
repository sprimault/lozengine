// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le placement en pixels de l'image doit tenir : ne pas bouger avec la caméra,
// sortir dans l'ordre de pose, et se ranger par sa couche comme tout le reste.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// TestScreenPieceDoesNotMoveWithTheCamera garde la raison d'être de cette porte.
//
// **C'est le seul cas à pouvoir la faire échouer**, et le défaut qu'il attrape est
// exactement celui qu'on aurait en posant l'interface en coordonnées du monde : le
// panneau suivrait la caméra d'un pixel par cran de déplacement, et une ligne de texte
// tremblerait pendant la marche du personnage.
//
// Il déplace la vue d'un nombre de cellules qui ne tombe pas juste en pixels, pour
// qu'un arrondi qui se glisserait dans la compensation se voie.
func TestScreenPieceDoesNotMoveWithTheCamera(t *testing.T) {
	s := testScene(t, 32, 1)

	a := figure()
	a.Layer = 1
	if !s.PlaceScreen(37, 11, a) {
		t.Fatal("pièce d'écran refusée")
	}

	at := func(focusX, focusY float64) (int16, int16) {
		s.Focus(focusX, focusY, 480, 270)

		image := New(s).Frame()
		if len(image) != 1 {
			t.Fatalf("%d quads, attendu 1", len(image))
		}
		return image[0].X, image[0].Y
	}

	x0, y0 := at(16, 16)
	x1, y1 := at(19.3, 22.7)

	if x0 != x1 || y0 != y1 {
		t.Errorf("la pièce a bougé de (%d, %d) à (%d, %d) quand la caméra a bougé",
			x0, y0, x1, y1)
	}
	if x0 != 37 || y0 != 11 {
		t.Errorf("pièce en (%d, %d), attendue au pixel posé (37, 11)", x0, y0)
	}
}

// TestScreenPiecesComeOutInOrderOfPlacement garde ce qu'une ligne de texte demande :
// ce qui est posé après recouvre ce qui précède.
//
// Les pièces partagent couche, planche et hauteur, donc seul le rang de pose les
// sépare. S'il ne passait pas dans la clé, elles tomberaient toutes dans le même
// groupe de départage — un tri par insertion, quadratique sur le nombre de glyphes.
func TestScreenPiecesComeOutInOrderOfPlacement(t *testing.T) {
	s := testScene(t, 32, 1)

	tints := []render.Color{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
	}
	for i, tint := range tints {
		a := figure()
		a.Layer, a.Tint = 1, tint
		if !s.PlaceScreen(int32(i*4), 0, a) {
			t.Fatalf("pièce %d refusée", i)
		}
	}

	image := New(s).Frame()
	if len(image) != len(tints) {
		t.Fatalf("%d quads, attendu %d", len(image), len(tints))
	}
	for i, want := range tints {
		if image[i].Tint != want {
			t.Errorf("rang %d : teinte %v, attendue %v", i, image[i].Tint, want)
		}
	}
}

// TestScreenPieceRanksByItsLayer vérifie qu'une pièce d'écran se range par sa couche
// comme n'importe quoi d'autre, et n'échappe pas à l'ordre commun.
//
// C'est ce qui fait que le placement en pixels n'est pas un second chemin de rendu :
// en couche basse, la pièce passe sous le monde ; en couche haute, au-dessus. Le jeu
// décide, et rien n'est câblé.
func TestScreenPieceRanksByItsLayer(t *testing.T) {
	for _, c := range []struct {
		name  string
		layer uint8
		first bool
	}{
		{"couche basse : sous le monde", 0, true},
		{"couche haute : au-dessus", 1, false},
	} {
		s := testScene(t, 32, 1)
		if _, ok := s.Place(scene.Entity{X: 16.5, Y: 16.5, Aspect: figure()}); !ok {
			t.Fatal("entité refusée")
		}

		piece := figure()
		piece.Layer, piece.Tint = c.layer, render.Color{R: 255, A: 255}
		if !s.PlaceScreen(0, 0, piece) {
			t.Fatal("pièce d'écran refusée")
		}

		image := New(s).Frame()
		if len(image) != 2 {
			t.Fatalf("%s : %d quads, attendu 2", c.name, len(image))
		}

		if got := image[0].Tint == piece.Tint; got != c.first {
			t.Errorf("%s : la pièce est peinte en premier = %v, attendu %v",
				c.name, got, c.first)
		}
	}
}

// TestScreenPieceAnchorAddsToTheCorner garde l'ancre sur la voie en pixels.
//
// **Rien ne la gardait**, et les deux défauts qu'elle laisse passer sont silencieux :
// mesuré le 29 octobre 2026, faire ignorer l'ancre à cette voie, ou centrer une
// largeur impaire d'un pixel de l'autre côté, laissait la suite complète verte. Les
// trois cas qui précèdent posent une figure de largeur paire à ancre nulle, donc ni
// l'ancre ni la parité n'y changent quoi que ce soit.
//
// C'est aussi le cas qui tient le texte de [scene.Scene.PlaceScreen] : il dit que
// l'ancre s'ajoute au coin d'elle-même, et un jeu qui l'ajoute aussi la compte deux
// fois.
func TestScreenPieceAnchorAddsToTheCorner(t *testing.T) {
	for _, c := range []struct {
		name             string
		width            uint16
		anchorX, anchorY int16
		wantX, wantY     int16
	}{
		{"largeur paire, sans ancre", 10, 0, 0, 37, 11},
		{"largeur impaire, sans ancre", 11, 0, 0, 37, 11},
		{"largeur paire, ancre", 10, 3, -2, 40, 9},
		{"largeur impaire, ancre", 11, -5, 7, 32, 18},
	} {
		s := testScene(t, 32, 1)

		a := figure()
		a.Width, a.Height = c.width, 7
		a.AnchorX, a.AnchorY = c.anchorX, c.anchorY
		a.Layer = 1
		if !s.PlaceScreen(37, 11, a) {
			t.Fatalf("%s : pièce d'écran refusée", c.name)
		}

		image := New(s).Frame()
		if len(image) != 1 {
			t.Fatalf("%s : %d quads, attendu 1", c.name, len(image))
		}
		if image[0].X != c.wantX || image[0].Y != c.wantY {
			t.Errorf("%s : coin en (%d, %d), attendu (%d, %d)",
				c.name, image[0].X, image[0].Y, c.wantX, c.wantY)
		}
	}
}

// marginScene monte une scène dont l'aire de monde est plus petite que la surface, donc
// décalée dedans : c'est la seule configuration où les deux repères se distinguent.
//
// L'aire est centrée, donc l'origine est non nulle sur les deux axes — un repère confondu
// avec l'autre se verrait alors sur les deux, là qu'un alignement au début cacherait le
// défaut derrière une origine à zéro.
func marginScene(t *testing.T, capacity int) *scene.Scene {
	t.Helper()

	s := testScene(t, 32, capacity)
	s.Focus(16, 16, 200, 120)
	s.ResizeSurface(480, 270)
	s.SetAlignment(scene.AlignCenter, scene.AlignCenter)
	return s
}

// TestAreaPieceLandsWhereAnEntityLands garde ce que le lot ajoute, et c'est le seul cas
// à pouvoir le faire échouer.
//
// **Il faut que les deux chemins divergent pour qu'un recoupement serve à quelque
// chose** : celui de l'entité projette puis découpe sur l'aire, celui de la pièce reçoit
// un pixel déjà calculé. Ils ne partagent que le placement, donc ce que le recoupement
// éprouve est exactement ce que le lot décide — le repère de la pose. Un cas écrit sur
// une scène sans marge ne prouverait rien, les deux repères y étant confondus.
//
// La leçon est celle d'AJ2, payée : un recoupement n'attrape que ce qui **diverge** entre
// deux chemins, jamais ce qui est faux dans un chemin unique qui alimente les deux.
func TestAreaPieceLandsWhereAnEntityLands(t *testing.T) {
	a := figure()
	a.AnchorX, a.AnchorY = 3, -2

	world := marginScene(t, 1)
	if _, ok := world.Place(scene.Entity{X: 16.5, Y: 16.5, Aspect: a}); !ok {
		t.Fatal("entité refusée")
	}
	sorter := New(world)
	want := sorter.Frame()[0]

	// Le repère où un jeu se trouve après avoir projeté : relatif à l'aire, l'origine
	// de la vue retranchée et la place de l'aire dans la surface **non** ajoutée.
	rx, ry := world.Remap(16.5, 16.5)
	px, py, _, _ := world.Projection().Project(rx, ry, 0)
	px, py = px-sorter.viewX, py-sorter.viewY

	inArea := marginScene(t, 1)
	a.Layer = 1
	cx, cy := a.CornerFromFoot(px, py)
	if !inArea.PlaceArea(cx, cy, a) {
		t.Fatal("pièce d'aire refusée")
	}
	got := New(inArea).Frame()[0]

	if got.X != want.X || got.Y != want.Y {
		t.Errorf("pièce d'aire en (%d, %d), l'entité au même point du monde en (%d, %d) — "+
			"la place de l'aire dans la surface n'a pas été ajoutée", got.X, got.Y, want.X, want.Y)
	}
}

// TestAreaPieceIsClippedOnTheArea garde la raison d'être de la voie : une chose située
// juste hors de l'aire ne doit pas s'afficher dans la marge, à côté du jeu.
//
// **C'est le défaut observé sur le jeu**, et il est de la classe que rien ne signale :
// deux annonces d'objets hors champ posées aux bords extrêmes de la fenêtre, à la hauteur
// du jeu. Un hôte peut refuser ce qui ne touche pas l'aire, mais pas découper une chose à
// cheval sur le bord — il lui faudrait refaire le calcul du moteur.
//
// Les deux voies sont éprouvées sur la **même** pose, parce que c'est le contraste qui est
// la propriété : celle de surface doit atteindre la marge, celle d'aire non. Un cas qui ne
// tiendrait que la seconde passerait encore si les deux s'étaient mises à découper sur
// l'aire, ce qui retirerait au jeu l'endroit où il dessine son interface.
func TestAreaPieceIsClippedOnTheArea(t *testing.T) {
	const beyond = -8

	// La largeur ne suffit pas à distinguer les deux régimes — découpée sur l'aire ou
	// sur la surface, la pièce perd les mêmes huit colonnes —, donc l'abscisse exacte
	// est ce qui tranche : relative à l'aire, elle vaut son bord ; relative à la
	// surface, elle garde les huit colonnes de retard. C'est la question de `tests.md` :
	// quelle entrée ce cas est-il le seul à pouvoir faire échouer.
	for _, c := range []struct {
		name       string
		inArea     bool
		wantWidth  uint16
		wantOffset int32
	}{
		{"dans l'aire : la part hors de l'aire est retirée", true, 32 + beyond, 0},
		{"sur la surface : la marge est atteignable", false, 32, beyond},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := marginScene(t, 1)
			originX, originY := s.WorldOrigin()

			// Le même pixel de surface dans les deux cas, et il tombe dans la marge
			// gauche : seul le rectangle de découpe sépare les deux poses.
			a := figure()
			a.Layer = 1
			if c.inArea {
				if !s.PlaceArea(beyond, 20, a) {
					t.Fatal("pièce d'aire refusée")
				}
			} else if !s.PlaceScreen(beyond+originX, 20+originY, a) {
				t.Fatal("pièce d'écran refusée")
			}

			image := New(s).Frame()
			if len(image) != 1 {
				t.Fatalf("%d quads, attendu 1", len(image))
			}
			if image[0].Width != c.wantWidth {
				t.Errorf("largeur %d, attendue %d", image[0].Width, c.wantWidth)
			}

			if want := originX + c.wantOffset; int32(image[0].X) != want {
				t.Errorf("abscisse %d, attendue %d, l'aire commençant à %d — "+
					"la pièce n'est pas découpée sur le rectangle de sa voie",
					image[0].X, want, originX)
			}
		})
	}
}

// TestPlacementOrderIsCommonToBothFrames garde la promesse d'ordre à travers les deux
// repères, et c'est elle qui a décidé qu'une seule liste les porte.
//
// Deux listes séparées coûteraient moins cher en bascules et feraient repartir le rang de
// pose à zéro pour la seconde : à couche égale, l'ordre de dessin viendrait alors du
// repère et non de la pose. Rien ne l'annoncerait, et une ligne de texte mêlée à un
// élément d'interface se relaierait au premier plan sans raison visible.
//
// **L'entrelacement n'est pas uniforme, et il ne peut pas l'être** : avec des repères
// qui alternent un sur deux, deux listes séparées rendent le même ordre par coïncidence —
// mesuré en simulant l'implémentation, la suite restait verte. Il faut que le compte des
// pièces diffère d'un repère à l'autre avant la fin pour que des rangs qui repartent à
// zéro se croisent.
//
// Les quatre pièces visent le même pixel dans leur repère respectif, ce qui leur donne la
// même abscisse de départage : le rang de pose est alors tout ce qui les sépare.
func TestPlacementOrderIsCommonToBothFrames(t *testing.T) {
	s := marginScene(t, 1)

	pieces := []struct {
		inArea bool
		tint   render.Color
	}{
		{true, render.Color{R: 255, A: 255}},
		{true, render.Color{G: 255, A: 255}},
		{false, render.Color{B: 255, A: 255}},
		{true, render.Color{R: 255, G: 255, A: 255}},
	}
	for i, p := range pieces {
		a := figure()
		a.Layer, a.Tint = 1, p.tint

		var placed bool
		if p.inArea {
			placed = s.PlaceArea(10, 10, a)
		} else {
			placed = s.PlaceScreen(10, 10, a)
		}
		if !placed {
			t.Fatalf("pièce %d refusée", i)
		}
	}

	image := New(s).Frame()
	if len(image) != len(pieces) {
		t.Fatalf("%d quads, attendu %d", len(image), len(pieces))
	}
	for i, p := range pieces {
		if image[i].Tint != p.tint {
			t.Errorf("rang %d : teinte %v, attendue %v — l'ordre de dessin vient du repère "+
				"et non de la pose", i, image[i].Tint, p.tint)
		}
	}
}

// TestScreenPieceByTheFootMatchesAnEntity garde la recette que [scene.Scene.PlaceScreen]
// donne pour poser une image par son pied sur un pixel.
//
// Elle relie les deux origines du moteur, et c'est ce qui en fait un cas et non une
// répétition du précédent : il rougit aussi si la compensation interne cesse d'être
// l'exact inverse de la pose par le pied, ce qu'aucun des deux ne verrait seul. La
// largeur impaire est là pour la division entière, qui est le seul endroit où les deux
// voies pourraient diverger d'un pixel.
func TestScreenPieceByTheFootMatchesAnEntity(t *testing.T) {
	for _, width := range []uint16{10, 11} {
		a := figure()
		a.Width, a.Height = width, 7
		a.AnchorX, a.AnchorY = 3, -2

		world := testScene(t, 32, 1)
		if _, ok := world.Place(scene.Entity{X: 16.5, Y: 16.5, Aspect: a}); !ok {
			t.Fatal("entité refusée")
		}
		sorter := New(world)
		want := sorter.Frame()[0]

		rx, ry := world.Remap(16.5, 16.5)
		px, py, _, _ := world.Projection().Project(rx, ry, 0)
		px, py = px-sorter.viewX, py-sorter.viewY

		onScreen := testScene(t, 32, 1)
		a.Layer = 1
		cx, cy := a.CornerFromFoot(px, py)
		if !onScreen.PlaceScreen(cx, cy, a) {
			t.Fatal("pièce d'écran refusée")
		}
		got := New(onScreen).Frame()[0]

		if got.X != want.X || got.Y != want.Y {
			t.Errorf("largeur %d : pièce en (%d, %d), l'entité en (%d, %d)",
				width, got.X, got.Y, want.X, want.Y)
		}
	}
}
