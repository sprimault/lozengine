// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la republication doit garantir, et qu'aucun autre test ne verrait : qu'un
// alias désigne bien le type du noyau, qu'une enveloppe rende bien ce que la fonction
// qu'elle cache rend, et qu'un jeu puisse tout écrire sans importer autre chose.

package lozengine

import (
	"bytes"
	"testing"

	"github.com/sprimault/lozengine/appearance"
	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// TestConstantsRepublished vérifie que chaque constante republiée vaut celle du
// noyau. Un bloc const Go répète l'expression précédente quand une ligne n'en
// porte pas : l'oubli ne se compile pas en erreur, il se compile en valeur
// fausse.
func TestConstantsRepublished(t *testing.T) {
	cases := []struct {
		name       string
		root, core Flags
	}{
		{"MirrorX", MirrorX, render.MirrorX},
		{"MirrorY", MirrorY, render.MirrorY},
		{"Flat", Flat, render.Flat},
		{"Additive", Additive, render.Additive},
	}
	for _, c := range cases {
		if c.root != c.core {
			t.Errorf("%s republié : %#b, attendu %#b", c.name, c.root, c.core)
		}
	}
}

// TestKeysRepublished vérifie la **valeur** des quatre-vingt-six places de clavier que
// la racine republie, et c'est la seule chose qu'aucun autre contrôle ne regarde.
//
// `make surface` compare les identifiants exportés, donc il verrait un nom qui
// disparaît ; il ne voit pas une valeur fausse. Et le régime qui la produit est celui
// que la godoc du cas précédent nomme : un bloc `const` Go répète l'expression
// précédente quand une ligne n'en porte pas, donc un `KeyF13` ajouté sans expression
// prendrait la valeur de la ligne d'avant — et `KeyNumpad7 = input.KeyNumpad1` compile
// aussi bien que la bonne.
//
// La surface est juste aujourd'hui, comparée nom à nom en écrivant ce cas. Ce qui le
// rend utile est le prochain ajout, pas celui-ci.
func TestKeysRepublished(t *testing.T) {
	for _, c := range []struct {
		name       string
		root, core Key
	}{
		{"KeyA", KeyA, input.KeyA}, {"KeyB", KeyB, input.KeyB},
		{"KeyC", KeyC, input.KeyC}, {"KeyD", KeyD, input.KeyD},
		{"KeyE", KeyE, input.KeyE}, {"KeyF", KeyF, input.KeyF},
		{"KeyG", KeyG, input.KeyG}, {"KeyH", KeyH, input.KeyH},
		{"KeyI", KeyI, input.KeyI}, {"KeyJ", KeyJ, input.KeyJ},
		{"KeyK", KeyK, input.KeyK}, {"KeyL", KeyL, input.KeyL},
		{"KeyM", KeyM, input.KeyM}, {"KeyN", KeyN, input.KeyN},
		{"KeyO", KeyO, input.KeyO}, {"KeyP", KeyP, input.KeyP},
		{"KeyQ", KeyQ, input.KeyQ}, {"KeyR", KeyR, input.KeyR},
		{"KeyS", KeyS, input.KeyS}, {"KeyT", KeyT, input.KeyT},
		{"KeyU", KeyU, input.KeyU}, {"KeyV", KeyV, input.KeyV},
		{"KeyW", KeyW, input.KeyW}, {"KeyX", KeyX, input.KeyX},
		{"KeyY", KeyY, input.KeyY}, {"KeyZ", KeyZ, input.KeyZ},

		{"Key1", Key1, input.Key1}, {"Key2", Key2, input.Key2},
		{"Key3", Key3, input.Key3}, {"Key4", Key4, input.Key4},
		{"Key5", Key5, input.Key5}, {"Key6", Key6, input.Key6},
		{"Key7", Key7, input.Key7}, {"Key8", Key8, input.Key8},
		{"Key9", Key9, input.Key9}, {"Key0", Key0, input.Key0},

		{"KeyUp", KeyUp, input.KeyUp}, {"KeyLeft", KeyLeft, input.KeyLeft},
		{"KeyRight", KeyRight, input.KeyRight}, {"KeyDown", KeyDown, input.KeyDown},

		{"KeyEscape", KeyEscape, input.KeyEscape},
		{"KeyBackspace", KeyBackspace, input.KeyBackspace},
		{"KeyTab", KeyTab, input.KeyTab},
		{"KeyEnter", KeyEnter, input.KeyEnter},
		{"KeySpace", KeySpace, input.KeySpace},

		{"KeyLeftShift", KeyLeftShift, input.KeyLeftShift},
		{"KeyRightShift", KeyRightShift, input.KeyRightShift},
		{"KeyLeftControl", KeyLeftControl, input.KeyLeftControl},
		{"KeyRightControl", KeyRightControl, input.KeyRightControl},
		{"KeyLeftAlt", KeyLeftAlt, input.KeyLeftAlt},
		{"KeyRightAlt", KeyRightAlt, input.KeyRightAlt},

		{"KeyF1", KeyF1, input.KeyF1}, {"KeyF2", KeyF2, input.KeyF2},
		{"KeyF3", KeyF3, input.KeyF3}, {"KeyF4", KeyF4, input.KeyF4},
		{"KeyF5", KeyF5, input.KeyF5}, {"KeyF6", KeyF6, input.KeyF6},
		{"KeyF7", KeyF7, input.KeyF7}, {"KeyF8", KeyF8, input.KeyF8},
		{"KeyF9", KeyF9, input.KeyF9}, {"KeyF10", KeyF10, input.KeyF10},
		{"KeyF11", KeyF11, input.KeyF11}, {"KeyF12", KeyF12, input.KeyF12},

		{"KeyHome", KeyHome, input.KeyHome}, {"KeyEnd", KeyEnd, input.KeyEnd},
		{"KeyPageUp", KeyPageUp, input.KeyPageUp},
		{"KeyPageDown", KeyPageDown, input.KeyPageDown},
		{"KeyInsert", KeyInsert, input.KeyInsert},
		{"KeyDelete", KeyDelete, input.KeyDelete},

		{"KeyNumpad0", KeyNumpad0, input.KeyNumpad0},
		{"KeyNumpad1", KeyNumpad1, input.KeyNumpad1},
		{"KeyNumpad2", KeyNumpad2, input.KeyNumpad2},
		{"KeyNumpad3", KeyNumpad3, input.KeyNumpad3},
		{"KeyNumpad4", KeyNumpad4, input.KeyNumpad4},
		{"KeyNumpad5", KeyNumpad5, input.KeyNumpad5},
		{"KeyNumpad6", KeyNumpad6, input.KeyNumpad6},
		{"KeyNumpad7", KeyNumpad7, input.KeyNumpad7},
		{"KeyNumpad8", KeyNumpad8, input.KeyNumpad8},
		{"KeyNumpad9", KeyNumpad9, input.KeyNumpad9},
		{"KeyNumpadDecimal", KeyNumpadDecimal, input.KeyNumpadDecimal},
		{"KeyNumpadMultiply", KeyNumpadMultiply, input.KeyNumpadMultiply},
		{"KeyNumpadSubtract", KeyNumpadSubtract, input.KeyNumpadSubtract},
		{"KeyNumpadAdd", KeyNumpadAdd, input.KeyNumpadAdd},
		{"KeyNumpadDivide", KeyNumpadDivide, input.KeyNumpadDivide},
		{"KeyNumpadEnter", KeyNumpadEnter, input.KeyNumpadEnter},
		{"KeyNumLock", KeyNumLock, input.KeyNumLock},
	} {
		if c.root != c.core {
			t.Errorf("%s republié : %d, attendu %d", c.name, c.root, c.core)
		}
	}
}

// TestButtonsAndBoundsRepublished vérifie la valeur des républications qui restaient
// sans garde : les cinq boutons, les trois alignements et le nombre de directions.
//
// Même régime que les touches, et les boutons y sont le plus exposés : cinq lignes
// voisines dans un seul bloc `const`.
func TestButtonsAndBoundsRepublished(t *testing.T) {
	for _, c := range []struct {
		name       string
		root, core Button
	}{
		{"ButtonLeft", ButtonLeft, input.ButtonLeft},
		{"ButtonRight", ButtonRight, input.ButtonRight},
		{"ButtonMiddle", ButtonMiddle, input.ButtonMiddle},
		{"ButtonBack", ButtonBack, input.ButtonBack},
		{"ButtonForward", ButtonForward, input.ButtonForward},
	} {
		if c.root != c.core {
			t.Errorf("%s republié : %d, attendu %d", c.name, c.root, c.core)
		}
	}

	for _, c := range []struct {
		name       string
		root, core Alignment
	}{
		{"AlignCenter", AlignCenter, scene.AlignCenter},
		{"AlignStart", AlignStart, scene.AlignStart},
		{"AlignEnd", AlignEnd, scene.AlignEnd},
	} {
		if c.root != c.core {
			t.Errorf("%s republié : %d, attendu %d", c.name, c.root, c.core)
		}
	}

	if Directions != appearance.Directions {
		t.Errorf("Directions republié : %d, attendu %d", Directions, appearance.Directions)
	}
}

// TestWrappersPassTheirArgumentsThrough garde les cinq enveloppes de la racine qu'aucun
// cas n'appelait, et deux d'entre elles portent des paramètres interchangeables.
//
// **Ce qui les rend dangereuses est l'absence de refus** : `Premultiply(r, g, b, a)` a
// quatre paramètres du même type, `NewSheetFromStraight(width, height, …)` deux. Écrire
// `render.Premultiply(b, g, r, a)` dans l'enveloppe compile, et laissait tout le dépôt
// vert.
//
// **Deux conditions sans lesquelles l'inversion survit au cas**, et c'est pourquoi elles
// sont écrites ici : quatre valeurs **distinctes** pour la couleur, et une planche **non
// carrée**. Avec `(10, 10, 10, 255)` ou un carré, toute permutation passerait.
func TestWrappersPassTheirArgumentsThrough(t *testing.T) {
	// Opaque, donc la prémultiplication laisse les trois composantes intactes et la
	// comparaison porte bien sur leur ordre.
	if got, want := Premultiply(10, 20, 30, 255), render.Premultiply(10, 20, 30, 255); got != want {
		t.Errorf("Premultiply : %+v, attendu %+v", got, want)
	}

	straight := make([]uint8, 4*8*2)
	sheet, err := NewSheetFromStraight(8, 2, straight)
	if err != nil {
		t.Fatalf("NewSheetFromStraight : %v", err)
	}
	if sheet.Width != 8 || sheet.Height != 2 {
		t.Errorf("planche %d×%d, attendu 8×2 — les deux dimensions sont inversées",
			sheet.Width, sheet.Height)
	}

	// Les trois autres n'ont pas de paramètre interchangeable : un appel qui aboutit
	// suffit à garder le câblage.
	if _, err := LoadSheet(bytes.NewReader(nil)); err == nil {
		t.Error("LoadSheet a accepté un flux vide")
	}
	if got, want := Turn(1, QuarterTurn, Directions), appearance.Turn(1, scene.QuarterTurn, appearance.Directions); got != want {
		t.Errorf("Turn : %d, attendu %d", got, want)
	}
	if NewSheets() == nil {
		t.Error("NewSheets rend nil")
	}
}

// TestBufferRepublished vérifie que le tampon traverse la surface publique sans
// conversion, et que sa borne republiée est bien celle du noyau.
//
// L'annotation de type explicite est le test lui-même : elle ne compile que si Buffer
// désigne le type du noyau. Une enveloppe, ou un type distinct de même forme,
// obligerait un hôte à convertir l'image à chaque passage de frontière.
func TestBufferRepublished(t *testing.T) {
	var root *Buffer = render.NewBuffer(2, 1)

	if MaxBufferSide != render.MaxBufferSide {
		t.Errorf("MaxBufferSide republié : %d, attendu %d", MaxBufferSide, render.MaxBufferSide)
	}
	if got := NewBuffer(2, 1); got.Width != root.Width || len(got.Pixels) != len(root.Pixels) {
		t.Errorf("NewBuffer republié rend %d×%d et %d pixels", got.Width, got.Height, len(got.Pixels))
	}
}

// TestConstructorsRepublished vérifie que chaque enveloppe rend la matrice du noyau.
//
// Ce qu'elle est seule à pouvoir attraper : une enveloppe qui appellerait le mauvais
// constructeur, ou qui échangerait ses deux paramètres, se compilerait sans broncher
// et ne se verrait qu'à l'image. La comparaison elle-même prouve au passage que
// Matrix est bien un alias et non un type distinct — deux types différents ne se
// compareraient pas.
func TestConstructorsRepublished(t *testing.T) {
	cases := []struct {
		name       string
		root, core Matrix
	}{
		{"Dimetric", Dimetric(32, 16), geometry.Dimetric(32, 16)},
		{"Isometric", Isometric(32), geometry.Isometric(32)},
		{"Grille", Grid(32), geometry.Grid(32)},
		{"Brute", Raw(2, 3, 5, 7), geometry.Raw(2, 3, 5, 7)},
	}
	for _, c := range cases {
		if c.root != c.core {
			t.Errorf("%s republié ne rend pas la matrice du noyau", c.name)
		}
	}
}

// TestPresentationRepublished vérifie que les trois voies de présentation s'atteignent
// depuis la racine, et dans le bon ordre d'arguments.
//
// **C'est l'ordre qui est gardé ici, pas la présence.** Les deux tampons de ScaleFactor
// comme d'Upscale ont le même type, donc une enveloppe qui les échangerait se
// compilerait sans un mot — c'est le défaut même que leur godoc décrit. Le cas rougit
// alors de deux façons : le facteur tombe à 1, et la sortie ne reçoit aucun pixel.
//
// Une image de deux sur deux au facteur quatre remplit exactement une sortie de huit,
// donc tout pixel qui n'est pas de la teinte est une bande qui n'aurait pas dû être
// peinte.
func TestPresentationRepublished(t *testing.T) {
	tint := Color{R: 200, G: 100, B: 50, A: 255}
	inner := NewBuffer(2, 2)
	for i := range inner.Pixels {
		inner.Pixels[i] = tint
	}
	out := NewBuffer(8, 8)

	if got := ScaleFactor(inner, out); got != 4 {
		t.Errorf("ScaleFactor republié rend %d, attendu 4", got)
	}

	Upscale(inner, out, 4, Color{A: 255})
	painted := 0
	for _, p := range out.Pixels {
		if p == tint {
			painted++
		}
	}
	if painted != len(out.Pixels) {
		t.Errorf("Upscale republié peint %d pixels sur %d", painted, len(out.Pixels))
	}

	var png bytes.Buffer
	if err := WritePNG(out, &png); err != nil {
		t.Fatalf("WritePNG republié : %v", err)
	}
	if png.Len() == 0 {
		t.Error("WritePNG republié n'a rien écrit")
	}
}

// TestOrientationsRepublished vérifie que les quatre quarts de tour valent ceux du
// noyau, et les plafonds de couche et d'étage avec eux.
//
// Même piège que pour les drapeaux, plus un second : une ligne qui recopierait la
// mauvaise constante — QuarterTurn valant le demi-tour — se compile sans broncher et ne se
// voit qu'à l'image, où elle ressemble à un défaut de caméra.
func TestOrientationsRepublished(t *testing.T) {
	cases := []struct {
		name       string
		root, core Orientation
	}{
		{"NoRotation", NoRotation, scene.NoRotation},
		{"QuarterTurn", QuarterTurn, scene.QuarterTurn},
		{"HalfTurn", HalfTurn, scene.HalfTurn},
		{"ThreeQuarters", ThreeQuarters, scene.ThreeQuarters},
	}
	for _, c := range cases {
		if c.root != c.core {
			t.Errorf("%s republié : %d, attendu %d", c.name, c.root, c.core)
		}
	}

	if MaxLayer != render.MaxLayer {
		t.Errorf("MaxLayer republié : %d, attendu %d", MaxLayer, render.MaxLayer)
	}
	if MaxFloor != render.MaxFloor {
		t.Errorf("MaxFloor republié : %d, attendu %d", MaxFloor, render.MaxFloor)
	}
	if MaxSide != render.MaxSide {
		t.Errorf("MaxSide republié : %d, attendu %d", MaxSide, render.MaxSide)
	}
	if MaxTiers != scene.MaxTiers {
		t.Errorf("MaxTiers republié : %d, attendu %d", MaxTiers, scene.MaxTiers)
	}
	if MaxSheets != render.MaxSheets {
		t.Errorf("MaxSheets republié : %d, attendu %d", MaxSheets, render.MaxSheets)
	}
}

// TestInputRepublished vérifie que l'état du clavier traverse la surface publique sans
// conversion, et que les deux régimes se lisent depuis ce paquet seul.
//
// L'annotation de type explicite est le test : elle ne compile que si Input désigne le
// type du noyau. Un jeu reçoit cette valeur à chaque pas, donc une enveloppe lui
// coûterait une conversion par image.
func TestInputRepublished(t *testing.T) {
	var state input.State
	state.Press(Key(17))

	var received Input = state.Frame()
	if !received.Held(17) || !received.JustPressed(17) {
		t.Errorf("touche tenue %v, montée %v, attendu les deux",
			received.Held(17), received.JustPressed(17))
	}
}

// TestSheetsUsableFromTheRoot suit le chemin complet d'une image de jeu : une
// planche, sa découpe en grille, son entrée dans un atlas, et l'index qu'un quad
// désignera — sans rien importer d'autre que ce paquet.
//
// Le rectangle est comparé à un littéral, ce qui ne compile que si ses champs sont
// atteignables depuis ici. C'est précisément ce qu'un alias vers un paquet interne ne
// donnait pas.
func TestSheetsUsableFromTheRoot(t *testing.T) {
	sheet := NewSheet(64, 32)
	if err := sheet.CutGrid(32, 16); err != nil {
		t.Fatalf("découpe : %v", err)
	}

	if n := len(sheet.Rectangles); n != 4 {
		t.Fatalf("%d rectangles, attendu 4", n)
	}
	if got := sheet.Rectangles[3]; got != (Rectangle{X: 32, Y: 16, Width: 32, Height: 16}) {
		t.Errorf("dernière case %+v", got)
	}

	var atlas Atlas
	index, err := atlas.Add(sheet)
	if err != nil {
		t.Fatalf("ajout à l'atlas : %v", err)
	}
	if index != 0 {
		t.Errorf("index %d, attendu 0", index)
	}
}

// TestSceneUsableFromTheRoot vérifie qu'un jeu décrit une scène entière sans rien
// importer d'autre que ce paquet : grille, entités, décor, caméra.
//
// C'est la porte que ce lot ouvre, et le seul cas qui la démontre de bout en bout — un
// alias manquant, ou un constructeur qui échangerait ses paramètres, ne se verrait
// nulle part ailleurs.
func TestSceneUsableFromTheRoot(t *testing.T) {
	s, err := NewScene(16, 16, Dimetric(32, 16), SceneOptions{Entities: 8})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	p, ok := s.Place(Entity{X: 2, Y: 3, Aspect: Aspect{Width: 32, Height: 16, Tint: White()}})
	if !ok {
		t.Fatal("entité refusée")
	}
	if !s.Move(p, 4, 5, 1) {
		t.Fatal("déplacement refusé")
	}

	// Aux deux bornes, pour que le cas démontre aussi qu'elles sont employables.
	top := Aspect{Width: 32, Height: 16, Tint: White(), Layer: MaxLayer, Floor: MaxFloor}
	if !s.PlaceCell(1, 1, 0, top) {
		t.Fatal("cellule refusée aux bornes")
	}
	if got := s.Cell(1, 1, 0); got.Layer != MaxLayer || got.Floor != MaxFloor {
		t.Errorf("cellule relue en couche %d étage %d", got.Layer, got.Floor)
	}

	s.Orient(HalfTurn)
	if got := s.Orientation(); got != HalfTurn {
		t.Errorf("orientation %d, attendu %d", got, HalfTurn)
	}

	s.Focus(4, 5, 480, 270)
	if _, _, width, height := s.View(); width != 480 || height != 270 {
		t.Errorf("vue de %d×%d, attendu 480×270", width, height)
	}
}

// TestMatrixUsableFromTheRoot vérifie qu'une matrice obtenue de la racine
// s'emploie sans rien importer d'autre : c'est tout ce qu'un consommateur peut en
// faire au sortir de ce jalon, et le seul cas qui le démontre.
func TestMatrixUsableFromTheRoot(t *testing.T) {
	m := Dimetric(32, 16).WithFloorHeight(8)

	ex, ey, depth, height := m.Project(1, 1, 1)
	if ex != 0 || ey != 8 || depth != 8 || height != 4 {
		t.Errorf("projeté (%d, %d), profondeur %d, hauteur %d — attendu (0, 8), 8 et 4", ex, ey, depth, height)
	}

	if x, y := m.Unproject(ex, ey); x != 0.5 || y != 0.5 {
		t.Errorf("inversé (%g, %g), attendu (0.5, 0.5)", x, y)
	}
}
