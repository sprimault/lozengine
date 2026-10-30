// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les vingt-deux cas du dessin, et c'est le fichier le plus long du dépôt pour une
// raison : chaque variante du blit — copie, modulation, mélange, aplat, miroir — a son
// propre découpage, et un découpage faux d'un pixel ne se voit pas à l'œil.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// Les couleurs des cas de dessin, nommées pour que les attentes se lisent en clair :
// un test qui attend « red, none, blue » dit ce qu'il vérifie, là où trois
// structures littérales ne disent rien. Les quatre premières sont opaques, donc
// recopiées telles quelles par le rastériseur.
var (
	red   = render.Color{R: 255, A: 255}
	green = render.Color{G: 255, A: 255}
	blue  = render.Color{B: 255, A: 255}
	white = render.Color{R: 255, G: 255, B: 255, A: 255}

	// halfRed est la seule à alpha partiel, et donc la seule dont le mélange se
	// calcule au lieu de se recopier. Prémultipliée : sa composante égale son alpha.
	halfRed = render.Color{R: 128, A: 128}

	// none est la valeur zéro, entièrement transparente, que le dessin doit laisser
	// passer sans toucher à la destination.
	none = render.Color{}
)

// atlasOf range une planche remplie des pixels donnés, ligne par ligne, et rend
// l'atlas dans lequel un quad la désigne par l'index 0.
func atlasOf(t *testing.T, width, height uint16, pixels ...render.Color) *Atlas {
	t.Helper()

	p := NewSheet(width, height)
	if len(pixels) != len(p.Pixels) {
		t.Fatalf("planche de %d×%d : %d pixels donnés", width, height, len(pixels))
	}
	copy(p.Pixels, pixels)

	var a Atlas
	if _, err := a.Add(p); err != nil {
		t.Fatalf("ajout : %v", err)
	}
	return &a
}

// quadOf rend un quad à teinte neutre, celle qui laisse la source intacte.
func quadOf(x, y int16, width, height uint16) render.Quad {
	return render.Quad{X: x, Y: y, Width: width, Height: height, Tint: render.White()}
}

// pixel rend le pixel du tampon, pour une lecture qui se lit en coordonnées.
func pixel(t *render.Buffer, x, y int) render.Color {
	return t.Pixels[y*int(t.Width)+x]
}

// painted compte les pixels non transparents, ce qui suffit à dire qu'un découpage
// n'a pas dessiné plus que sa part.
func painted(t *render.Buffer) int {
	n := 0
	for _, c := range t.Pixels {
		if c.A != 0 {
			n++
		}
	}
	return n
}

// TestDrawPlaces vérifie qu'un quad atterrit à sa position et que ses pixels
// gardent l'ordre de la planche.
func TestDrawPlaces(t *testing.T) {
	a := atlasOf(t, 2, 2, red, green, blue, white)
	buffer := render.NewBuffer(4, 4)

	Draw(buffer, render.DrawList{quadOf(1, 1, 2, 2)}, a)

	want := map[[2]int]render.Color{
		{1, 1}: red, {2, 1}: green,
		{1, 2}: blue, {2, 2}: white,
	}
	for position := range want {
		if got := pixel(buffer, position[0], position[1]); got != want[position] {
			t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, want[position])
		}
	}
	if n := painted(buffer); n != 4 {
		t.Errorf("pixels peints : %d, attendu 4", n)
	}
}

// TestDrawClips éprouve le découpage sur les quatre bords : ce qui sort du
// tampon ne se dessine pas, et ce qui reste ne se décale pas.
func TestDrawClips(t *testing.T) {
	cases := []struct {
		name string
		x, y int16
		want map[[2]int]render.Color
	}{
		{"coin haut-gauche", -1, -1, map[[2]int]render.Color{{0, 0}: white}},
		{"coin bas-droit", 3, 3, map[[2]int]render.Color{{3, 3}: red}},
		{"bord gauche", -1, 0, map[[2]int]render.Color{{0, 0}: green, {0, 1}: white}},
		{"bord bas", 0, 3, map[[2]int]render.Color{{0, 3}: red, {1, 3}: green}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 2, 2, red, green, blue, white)
			buffer := render.NewBuffer(4, 4)

			Draw(buffer, render.DrawList{quadOf(c.x, c.y, 2, 2)}, a)

			for position, want := range c.want {
				if got := pixel(buffer, position[0], position[1]); got != want {
					t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, want)
				}
			}
			if n := painted(buffer); n != len(c.want) {
				t.Errorf("pixels peints : %d, attendu %d", n, len(c.want))
			}
		})
	}
}

// TestDrawOffScreen vérifie qu'un quad entièrement dehors ne coûte et ne peint
// rien, y compris quand il touche le bord sans le franchir.
func TestDrawOffScreen(t *testing.T) {
	cases := []struct {
		name string
		x, y int16
	}{
		{"à droite", 4, 0},
		{"à gauche", -2, 0},
		{"en bas", 0, 4},
		{"en haut", 0, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 2, 2, red, green, blue, white)
			buffer := render.NewBuffer(4, 4)

			Draw(buffer, render.DrawList{quadOf(c.x, c.y, 2, 2)}, a)

			if n := painted(buffer); n != 0 {
				t.Errorf("pixels peints : %d, attendu aucun", n)
			}
		})
	}
}

// TestDrawOrder vérifie que le dernier quad recouvre le premier : l'ordre du
// tableau est la seule vérité du résultat.
func TestDrawOrder(t *testing.T) {
	a := atlasOf(t, 1, 1, white)
	buffer := render.NewBuffer(2, 1)

	first, second := quadOf(0, 0, 1, 1), quadOf(0, 0, 1, 1)
	first.Tint, second.Tint = red, green

	Draw(buffer, render.DrawList{first, second}, a)

	if got := pixel(buffer, 0, 0); got != green {
		t.Errorf("pixel recouvert : %v, attendu %v", got, green)
	}
}

// TestDrawSkipsTransparent vérifie qu'un pixel transparent de la planche laisse le fond en
// place : c'est ce que le pré-encodage promet au blit.
func TestDrawSkipsTransparent(t *testing.T) {
	a := atlasOf(t, 2, 1, none, red)
	buffer := render.NewBuffer(2, 1)
	buffer.Pixels[0], buffer.Pixels[1] = blue, blue

	Draw(buffer, render.DrawList{quadOf(0, 0, 2, 1)}, a)

	if got := pixel(buffer, 0, 0); got != blue {
		t.Errorf("fond sous un pixel transparent : %v, attendu %v", got, blue)
	}
	if got := pixel(buffer, 1, 0); got != red {
		t.Errorf("pixel opaque : %v, attendu %v", got, red)
	}
}

// TestDrawBlends vérifie la composition d'une source à demi transparente sur
// un fond opaque.
func TestDrawBlends(t *testing.T) {
	a := atlasOf(t, 1, 1, halfRed)
	buffer := render.NewBuffer(1, 1)
	buffer.Pixels[0] = white

	Draw(buffer, render.DrawList{quadOf(0, 0, 1, 1)}, a)

	if want := (render.Color{R: 255, G: 127, B: 127, A: 255}); pixel(buffer, 0, 0) != want {
		t.Errorf("mélange : %v, attendu %v", pixel(buffer, 0, 0), want)
	}
}

// TestDrawTint vérifie les trois régimes de modulation, dont celui où la
// teinte rend translucide un pixel que la planche donnait opaque.
func TestDrawTint(t *testing.T) {
	cases := []struct {
		name string
		tint render.Color
		want render.Color
	}{
		{"neutre", white, white},
		{"couleur, opaque", red, red},
		{"demi-opacité", render.Color{R: 128, G: 128, B: 128, A: 128}, render.Color{R: 128, G: 128, B: 128, A: 128}},
		{"teinte nulle", none, none},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 1, 1, white)
			buffer := render.NewBuffer(1, 1)

			q := quadOf(0, 0, 1, 1)
			q.Tint = c.tint
			Draw(buffer, render.DrawList{q}, a)

			if got := pixel(buffer, 0, 0); got != c.want {
				t.Errorf("pixel : %v, attendu %v", got, c.want)
			}
		})
	}
}

// TestDrawMirror vérifie les deux miroirs et leur combinaison, par lesquels un
// fournisseur d'apparence tire l'est de l'ouest sans doubler son atlas.
func TestDrawMirror(t *testing.T) {
	cases := []struct {
		name  string
		flags render.Flags
		want  [4]render.Color // rouge, vert, bleu, blanc dans l'ordre du tampon
	}{
		{"sans miroir", 0, [4]render.Color{red, green, blue, white}},
		{"miroir horizontal", render.MirrorX, [4]render.Color{green, red, white, blue}},
		{"miroir vertical", render.MirrorY, [4]render.Color{blue, white, red, green}},
		{"les deux", render.MirrorX | render.MirrorY, [4]render.Color{white, blue, green, red}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 2, 2, red, green, blue, white)
			buffer := render.NewBuffer(2, 2)

			q := quadOf(0, 0, 2, 2)
			q.Flags = c.flags
			Draw(buffer, render.DrawList{q}, a)

			for i, want := range c.want {
				if buffer.Pixels[i] != want {
					t.Errorf("pixel %d : %v, attendu %v", i, buffer.Pixels[i], want)
				}
			}
		})
	}
}

// TestDrawMirrorClipped éprouve le croisement des deux : le bord gauche du
// tampon découpe la droite de la source quand le miroir est posé.
func TestDrawMirrorClipped(t *testing.T) {
	a := atlasOf(t, 2, 1, red, green)
	buffer := render.NewBuffer(2, 1)

	q := quadOf(-1, 0, 2, 1)
	q.Flags = render.MirrorX
	Draw(buffer, render.DrawList{q}, a)

	if got := pixel(buffer, 0, 0); got != red {
		t.Errorf("pixel visible : %v, attendu %v", got, red)
	}
	if n := painted(buffer); n != 1 {
		t.Errorf("pixels peints : %d, attendu 1", n)
	}
}

// TestDrawSource vérifie qu'un quad ne dessine que son rectangle de source,
// celui qu'une découpe en grille lui a donné.
func TestDrawSource(t *testing.T) {
	a := atlasOf(t, 2, 2, red, green, blue, white)
	buffer := render.NewBuffer(1, 1)

	q := quadOf(0, 0, 1, 1)
	q.SourceX, q.SourceY = 1, 1
	Draw(buffer, render.DrawList{q}, a)

	if got := pixel(buffer, 0, 0); got != white {
		t.Errorf("pixel : %v, attendu %v", got, white)
	}
}

// TestDrawFlat vérifie qu'un aplat se peint sans lire l'atlas — l'atlas est
// vide ici — et qu'il se découpe comme les autres.
func TestDrawFlat(t *testing.T) {
	var a Atlas
	buffer := render.NewBuffer(3, 2)

	q := quadOf(1, 0, 5, 1)
	q.Tint = green
	q.Flags = render.Flat
	Draw(buffer, render.DrawList{q}, &a)

	for x := 1; x < 3; x++ {
		if got := pixel(buffer, x, 0); got != green {
			t.Errorf("pixel (%d, 0) : %v, attendu %v", x, got, green)
		}
	}
	if n := painted(buffer); n != 2 {
		t.Errorf("pixels peints : %d, attendu 2", n)
	}
}

// TestDrawFlatTranslucent vérifie qu'un aplat à demi transparent se mélange au
// fond, ce qu'un fondu au noir demande.
func TestDrawFlatTranslucent(t *testing.T) {
	var a Atlas
	buffer := render.NewBuffer(1, 1)
	buffer.Pixels[0] = white

	q := quadOf(0, 0, 1, 1)
	q.Tint = halfRed
	q.Flags = render.Flat
	Draw(buffer, render.DrawList{q}, &a)

	if want := (render.Color{R: 255, G: 127, B: 127, A: 255}); pixel(buffer, 0, 0) != want {
		t.Errorf("aplat mélangé : %v, attendu %v", pixel(buffer, 0, 0), want)
	}
}

// TestDrawEmptyQuad vérifie que la valeur zéro d'un quad ne dessine rien : c'est
// l'échec sûr sur lequel compte un hôte qui remplit la structure de zéros.
func TestDrawEmptyQuad(t *testing.T) {
	a := atlasOf(t, 1, 1, red)
	buffer := render.NewBuffer(2, 2)

	Draw(buffer, render.DrawList{{}}, a)

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels peints : %d, attendu aucun", n)
	}
}

// TestDrawDoesNotAllocate vérifie l'invariant du chemin de rendu : aucune
// allocation par image. Un tampon réutilisé, une liste réutilisée, et rien qui
// échappe — sans quoi le ramasse-miettes s'invite à soixante hertz.
func TestDrawDoesNotAllocate(t *testing.T) {
	a := atlasOf(t, 2, 2, red, green, blue, halfRed)
	buffer := render.NewBuffer(8, 8)

	q := quadOf(1, 1, 2, 2)
	mirror := quadOf(-1, 5, 2, 2)
	mirror.Flags = render.MirrorX | render.MirrorY
	flat := quadOf(4, 4, 3, 3)
	flat.Flags = render.Flat
	r := render.DrawList{q, mirror, flat}

	if n := testing.AllocsPerRun(10, func() { Draw(buffer, r, a) }); n != 0 {
		t.Errorf("allocations par image : %v, attendu aucune", n)
	}
}

// TestDrawOverflowingSource vérifie qu'un rectangle de source plus grand que la
// planche s'arrête à la dernière ligne au lieu de lire la suivante.
func TestDrawOverflowingSource(t *testing.T) {
	a := atlasOf(t, 2, 1, red, green)
	buffer := render.NewBuffer(2, 2)

	Draw(buffer, render.DrawList{quadOf(0, 0, 2, 2)}, a)

	if got := pixel(buffer, 0, 0); got != red {
		t.Errorf("première ligne : %v, attendu %v", got, red)
	}
	if n := painted(buffer); n != 2 {
		t.Errorf("pixels peints : %d, attendu 2", n)
	}
}

// TestDrawStopsQuadAtItsWidth éprouve le découpage d'une suite de pixels
// plus longue que le quad qui la lit.
//
// Une planche entièrement opaque forme une seule suite, large de toute sa ligne, et
// un quad plus étroit ne doit en prendre que sa part. Aucun quad du dépôt ne
// parcourait ce cas : tous lisent une planche à leur propre taille.
func TestDrawStopsQuadAtItsWidth(t *testing.T) {
	a := atlasOf(t, 4, 1, red, green, blue, white)
	buffer := render.NewBuffer(4, 1)

	Draw(buffer, render.DrawList{quadOf(0, 0, 2, 1)}, a)

	if got := pixel(buffer, 0, 0); got != red {
		t.Errorf("pixel 0 : %v, attendu rouge", got)
	}
	if got := pixel(buffer, 1, 0); got != green {
		t.Errorf("pixel 1 : %v, attendu vert", got)
	}
	if n := painted(buffer); n != 2 {
		t.Errorf("pixels peints : %d, attendu 2 — la suite a débordé du quad", n)
	}
}

// TestDrawMirrorOnPartialSource éprouve le miroir quand la source ne commence
// pas au bord gauche de la planche.
//
// Le miroir calcule sa destination à partir de SourceX : une source décalée est donc
// le seul cas où ce calcul peut se tromper, et aucun quad miroité du dépôt n'avait de
// SourceX non nul.
func TestDrawMirrorOnPartialSource(t *testing.T) {
	a := atlasOf(t, 4, 1, red, green, blue, white)
	buffer := render.NewBuffer(2, 1)

	q := quadOf(0, 0, 2, 1)
	q.SourceX = 1
	q.Flags = render.MirrorX
	Draw(buffer, render.DrawList{q}, a)

	// La source lue est vert puis bleu ; le miroir la rend dans l'autre sens.
	if got := pixel(buffer, 0, 0); got != blue {
		t.Errorf("pixel 0 : %v, attendu bleu", got)
	}
	if got := pixel(buffer, 1, 0); got != green {
		t.Errorf("pixel 1 : %v, attendu vert", got)
	}
}

// TestDrawTransparentFlatPaintsNothing vérifie qu'un aplat à teinte entièrement
// transparente laisse le tampon intact. Il se rencontre dès qu'un fondu commence à
// zéro : sa première image demande un aplat qui n'a rien à écrire.
//
// Ce cas ne garde pas le court-circuit qui traite cet alpha nul, et il faut le dire
// plutôt que de le laisser croire : retirer ce raccourci ne changerait aucune image,
// puisque mélanger une couleur d'alpha zéro rend la destination inchangée. C'est une
// optimisation, pas une correction.
//
// Ce que le cas garde est le comportement lui-même : si le mélange venait à écrire
// quelque chose pour un alpha nul, l'image changerait et ce test le dirait.
func TestDrawTransparentFlatPaintsNothing(t *testing.T) {
	a := atlasOf(t, 1, 1, red)
	buffer := render.NewBuffer(2, 2)

	Draw(buffer, render.DrawList{{
		Width: 2, Height: 2, Tint: none, Flags: render.Flat,
	}}, a)

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels peints : %d, attendu 0", n)
	}
}
