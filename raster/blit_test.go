// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// Les couleurs des cas de dessin, nommées pour que les attentes se lisent en clair :
// un test qui attend « rouge, rien, bleu » dit ce qu'il vérifie, là où trois
// structures littérales ne disent rien. Les quatre premières sont opaques, donc
// recopiées telles quelles par le rastériseur.
var (
	rouge = rendu.Couleur{R: 255, A: 255}
	vert  = rendu.Couleur{V: 255, A: 255}
	bleu  = rendu.Couleur{B: 255, A: 255}
	blanc = rendu.Couleur{R: 255, V: 255, B: 255, A: 255}

	// demiRge est la seule à alpha partiel, et donc la seule dont le mélange se
	// calcule au lieu de se recopier. Prémultipliée : sa composante égale son alpha.
	demiRge = rendu.Couleur{R: 128, A: 128}

	// rien est la valeur zéro, entièrement transparente, que le dessin doit laisser
	// passer sans toucher à la destination.
	rien = rendu.Couleur{}
)

// atlasOf range une planche remplie des pixels donnés, ligne par ligne, et rend
// l'atlas dans lequel un quad la désigne par l'index 0.
func atlasOf(t *testing.T, width, height uint16, pixels ...rendu.Couleur) *Atlas {
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
func quadOf(x, y int16, width, height uint16) rendu.Quad {
	return rendu.Quad{X: x, Y: y, Largeur: width, Hauteur: height, Teinte: rendu.Blanc()}
}

// pixel rend le pixel du tampon, pour une lecture qui se lit en coordonnées.
func pixel(t *rendu.Tampon, x, y int) rendu.Couleur {
	return t.Pixels[y*int(t.Largeur)+x]
}

// painted compte les pixels non transparents, ce qui suffit à dire qu'un découpage
// n'a pas dessiné plus que sa part.
func painted(t *rendu.Tampon) int {
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
	a := atlasOf(t, 2, 2, rouge, vert, bleu, blanc)
	buffer := rendu.NouveauTampon(4, 4)

	Draw(buffer, rendu.Rendu{quadOf(1, 1, 2, 2)}, a)

	want := map[[2]int]rendu.Couleur{
		{1, 1}: rouge, {2, 1}: vert,
		{1, 2}: bleu, {2, 2}: blanc,
	}
	for position, veut := range want {
		if got := pixel(buffer, position[0], position[1]); got != veut {
			t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, veut)
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
		want map[[2]int]rendu.Couleur
	}{
		{"coin haut-gauche", -1, -1, map[[2]int]rendu.Couleur{{0, 0}: blanc}},
		{"coin bas-droit", 3, 3, map[[2]int]rendu.Couleur{{3, 3}: rouge}},
		{"bord gauche", -1, 0, map[[2]int]rendu.Couleur{{0, 0}: vert, {0, 1}: blanc}},
		{"bord bas", 0, 3, map[[2]int]rendu.Couleur{{0, 3}: rouge, {1, 3}: vert}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := rendu.NouveauTampon(4, 4)

			Draw(tampon, rendu.Rendu{quadOf(c.x, c.y, 2, 2)}, a)

			for position, veut := range c.want {
				if got := pixel(tampon, position[0], position[1]); got != veut {
					t.Errorf("pixel (%d, %d) : %v, attendu %v", position[0], position[1], got, veut)
				}
			}
			if n := painted(tampon); n != len(c.want) {
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
			a := atlasOf(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := rendu.NouveauTampon(4, 4)

			Draw(tampon, rendu.Rendu{quadOf(c.x, c.y, 2, 2)}, a)

			if n := painted(tampon); n != 0 {
				t.Errorf("pixels peints : %d, attendu aucun", n)
			}
		})
	}
}

// TestDrawOrder vérifie que le dernier quad recouvre le premier : l'ordre du
// tableau est la seule vérité du résultat.
func TestDrawOrder(t *testing.T) {
	a := atlasOf(t, 1, 1, blanc)
	buffer := rendu.NouveauTampon(2, 1)

	premier, second := quadOf(0, 0, 1, 1), quadOf(0, 0, 1, 1)
	premier.Teinte, second.Teinte = rouge, vert

	Draw(buffer, rendu.Rendu{premier, second}, a)

	if got := pixel(buffer, 0, 0); got != vert {
		t.Errorf("pixel recouvert : %v, attendu %v", got, vert)
	}
}

// TestDrawSkipsTransparent vérifie qu'un pixel transparent de la planche laisse le fond en
// place : c'est ce que le pré-encodage promet au blit.
func TestDrawSkipsTransparent(t *testing.T) {
	a := atlasOf(t, 2, 1, rien, rouge)
	buffer := rendu.NouveauTampon(2, 1)
	buffer.Pixels[0], buffer.Pixels[1] = bleu, bleu

	Draw(buffer, rendu.Rendu{quadOf(0, 0, 2, 1)}, a)

	if got := pixel(buffer, 0, 0); got != bleu {
		t.Errorf("fond sous un pixel transparent : %v, attendu %v", got, bleu)
	}
	if got := pixel(buffer, 1, 0); got != rouge {
		t.Errorf("pixel opaque : %v, attendu %v", got, rouge)
	}
}

// TestDrawBlends vérifie la composition d'une source à demi transparente sur
// un fond opaque.
func TestDrawBlends(t *testing.T) {
	a := atlasOf(t, 1, 1, demiRge)
	buffer := rendu.NouveauTampon(1, 1)
	buffer.Pixels[0] = blanc

	Draw(buffer, rendu.Rendu{quadOf(0, 0, 1, 1)}, a)

	if veut := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); pixel(buffer, 0, 0) != veut {
		t.Errorf("mélange : %v, attendu %v", pixel(buffer, 0, 0), veut)
	}
}

// TestDrawTint vérifie les trois régimes de modulation, dont celui où la
// teinte rend translucide un pixel que la planche donnait opaque.
func TestDrawTint(t *testing.T) {
	cases := []struct {
		name string
		tint rendu.Couleur
		want rendu.Couleur
	}{
		{"neutre", blanc, blanc},
		{"couleur, opaque", rouge, rouge},
		{"demi-opacité", rendu.Couleur{R: 255, V: 255, B: 255, A: 128}, rendu.Couleur{R: 128, V: 128, B: 128, A: 128}},
		{"teinte nulle", rien, rien},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 1, 1, blanc)
			tampon := rendu.NouveauTampon(1, 1)

			q := quadOf(0, 0, 1, 1)
			q.Teinte = c.tint
			Draw(tampon, rendu.Rendu{q}, a)

			if got := pixel(tampon, 0, 0); got != c.want {
				t.Errorf("pixel : %v, attendu %v", got, c.want)
			}
		})
	}
}

// TestDrawMirror vérifie les deux miroirs et leur combinaison, par lesquels un
// fournisseur d'apparence tire l'est de l'ouest sans doubler son atlas.
func TestDrawMirror(t *testing.T) {
	cases := []struct {
		name     string
		drapeaux rendu.Drapeaux
		want     [4]rendu.Couleur // rouge, vert, bleu, blanc dans l'ordre du tampon
	}{
		{"sans miroir", 0, [4]rendu.Couleur{rouge, vert, bleu, blanc}},
		{"miroir horizontal", rendu.MiroirX, [4]rendu.Couleur{vert, rouge, blanc, bleu}},
		{"miroir vertical", rendu.MiroirY, [4]rendu.Couleur{bleu, blanc, rouge, vert}},
		{"les deux", rendu.MiroirX | rendu.MiroirY, [4]rendu.Couleur{blanc, bleu, vert, rouge}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := atlasOf(t, 2, 2, rouge, vert, bleu, blanc)
			tampon := rendu.NouveauTampon(2, 2)

			q := quadOf(0, 0, 2, 2)
			q.Drapeaux = c.drapeaux
			Draw(tampon, rendu.Rendu{q}, a)

			for i, veut := range c.want {
				if tampon.Pixels[i] != veut {
					t.Errorf("pixel %d : %v, attendu %v", i, tampon.Pixels[i], veut)
				}
			}
		})
	}
}

// TestDrawMirrorClipped éprouve le croisement des deux : le bord gauche du
// tampon découpe la droite de la source quand le miroir est posé.
func TestDrawMirrorClipped(t *testing.T) {
	a := atlasOf(t, 2, 1, rouge, vert)
	buffer := rendu.NouveauTampon(2, 1)

	q := quadOf(-1, 0, 2, 1)
	q.Drapeaux = rendu.MiroirX
	Draw(buffer, rendu.Rendu{q}, a)

	if got := pixel(buffer, 0, 0); got != rouge {
		t.Errorf("pixel visible : %v, attendu %v", got, rouge)
	}
	if n := painted(buffer); n != 1 {
		t.Errorf("pixels peints : %d, attendu 1", n)
	}
}

// TestDrawSource vérifie qu'un quad ne dessine que son rectangle de source,
// celui qu'une découpe en grille lui a donné.
func TestDrawSource(t *testing.T) {
	a := atlasOf(t, 2, 2, rouge, vert, bleu, blanc)
	buffer := rendu.NouveauTampon(1, 1)

	q := quadOf(0, 0, 1, 1)
	q.SourceX, q.SourceY = 1, 1
	Draw(buffer, rendu.Rendu{q}, a)

	if got := pixel(buffer, 0, 0); got != blanc {
		t.Errorf("pixel : %v, attendu %v", got, blanc)
	}
}

// TestDrawFlat vérifie qu'un aplat se peint sans lire l'atlas — l'atlas est
// vide ici — et qu'il se découpe comme les autres.
func TestDrawFlat(t *testing.T) {
	var a Atlas
	buffer := rendu.NouveauTampon(3, 2)

	q := quadOf(1, 0, 5, 1)
	q.Teinte = vert
	q.Drapeaux = rendu.Aplat
	Draw(buffer, rendu.Rendu{q}, &a)

	for x := 1; x < 3; x++ {
		if got := pixel(buffer, x, 0); got != vert {
			t.Errorf("pixel (%d, 0) : %v, attendu %v", x, got, vert)
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
	buffer := rendu.NouveauTampon(1, 1)
	buffer.Pixels[0] = blanc

	q := quadOf(0, 0, 1, 1)
	q.Teinte = demiRge
	q.Drapeaux = rendu.Aplat
	Draw(buffer, rendu.Rendu{q}, &a)

	if veut := (rendu.Couleur{R: 255, V: 127, B: 127, A: 255}); pixel(buffer, 0, 0) != veut {
		t.Errorf("aplat mélangé : %v, attendu %v", pixel(buffer, 0, 0), veut)
	}
}

// TestDrawEmptyQuad vérifie que la valeur zéro d'un quad ne dessine rien : c'est
// l'échec sûr sur lequel compte un hôte qui remplit la structure de zéros.
func TestDrawEmptyQuad(t *testing.T) {
	a := atlasOf(t, 1, 1, rouge)
	buffer := rendu.NouveauTampon(2, 2)

	Draw(buffer, rendu.Rendu{{}}, a)

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels peints : %d, attendu aucun", n)
	}
}

// TestDrawDoesNotAllocate vérifie l'invariant du chemin de rendu : aucune
// allocation par image. Un tampon réutilisé, une liste réutilisée, et rien qui
// échappe — sans quoi le ramasse-miettes s'invite à soixante hertz.
func TestDrawDoesNotAllocate(t *testing.T) {
	a := atlasOf(t, 2, 2, rouge, vert, bleu, demiRge)
	buffer := rendu.NouveauTampon(8, 8)

	q := quadOf(1, 1, 2, 2)
	mirror := quadOf(-1, 5, 2, 2)
	mirror.Drapeaux = rendu.MiroirX | rendu.MiroirY
	aplat := quadOf(4, 4, 3, 3)
	aplat.Drapeaux = rendu.Aplat
	r := rendu.Rendu{q, mirror, aplat}

	if n := testing.AllocsPerRun(10, func() { Draw(buffer, r, a) }); n != 0 {
		t.Errorf("allocations par image : %v, attendu aucune", n)
	}
}

// TestDrawOverflowingSource vérifie qu'un rectangle de source plus grand que la
// planche s'arrête à la dernière ligne au lieu de lire la suivante.
func TestDrawOverflowingSource(t *testing.T) {
	a := atlasOf(t, 2, 1, rouge, vert)
	buffer := rendu.NouveauTampon(2, 2)

	Draw(buffer, rendu.Rendu{quadOf(0, 0, 2, 2)}, a)

	if got := pixel(buffer, 0, 0); got != rouge {
		t.Errorf("première ligne : %v, attendu %v", got, rouge)
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
	a := atlasOf(t, 4, 1, rouge, vert, bleu, blanc)
	buffer := rendu.NouveauTampon(4, 1)

	Draw(buffer, rendu.Rendu{quadOf(0, 0, 2, 1)}, a)

	if got := pixel(buffer, 0, 0); got != rouge {
		t.Errorf("pixel 0 : %v, attendu rouge", got)
	}
	if got := pixel(buffer, 1, 0); got != vert {
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
	a := atlasOf(t, 4, 1, rouge, vert, bleu, blanc)
	buffer := rendu.NouveauTampon(2, 1)

	q := quadOf(0, 0, 2, 1)
	q.SourceX = 1
	q.Drapeaux = rendu.MiroirX
	Draw(buffer, rendu.Rendu{q}, a)

	// La source lue est vert puis bleu ; le miroir la rend dans l'autre sens.
	if got := pixel(buffer, 0, 0); got != bleu {
		t.Errorf("pixel 0 : %v, attendu bleu", got)
	}
	if got := pixel(buffer, 1, 0); got != vert {
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
	a := atlasOf(t, 1, 1, rouge)
	buffer := rendu.NouveauTampon(2, 2)

	Draw(buffer, rendu.Rendu{{
		Largeur: 2, Hauteur: 2, Teinte: rien, Drapeaux: rendu.Aplat,
	}}, a)

	if n := painted(buffer); n != 0 {
		t.Errorf("pixels peints : %d, attendu 0", n)
	}
}
