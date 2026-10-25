// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les cas où l'agrandissement ne tombe pas juste : une sortie plus petite que l'entrée,
// un centrage impair, des bandes à découper. C'est là que le calcul des bornes se
// trompe, pas sur le cas qui divise exactement.

package raster

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// bufferOf rend un tampon rempli des pixels donnés, ligne par ligne.
func bufferOf(t *testing.T, width, height uint16, pixels ...render.Color) *render.Buffer {
	t.Helper()

	buffer := render.NewBuffer(width, height)
	if len(pixels) != len(buffer.Pixels) {
		t.Fatalf("tampon de %d×%d : %d pixels donnés", width, height, len(pixels))
	}
	copy(buffer.Pixels, pixels)
	return buffer
}

// TestScaleFactor vérifie le facteur retenu, y compris quand la sortie ne peut pas
// contenir la résolution interne.
func TestScaleFactor(t *testing.T) {
	cases := []struct {
		name                string
		width, height       uint16
		outWidth, outHeight uint16
		want                int
	}{
		{"tombe juste", 480, 270, 1920, 1080, 4},
		{"limité par la hauteur", 480, 270, 1920, 700, 2},
		{"limité par la largeur", 480, 270, 1000, 1080, 2},
		{"sortie trop petite", 480, 270, 100, 100, 1},
		{"sortie identique", 480, 270, 480, 270, 1},
		{"tampon vide", 0, 0, 800, 600, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inner := render.NewBuffer(c.width, c.height)
			out := render.NewBuffer(c.outWidth, c.outHeight)

			if got := ScaleFactor(inner, out); got != c.want {
				t.Errorf("facteur : %d, attendu %d", got, c.want)
			}
		})
	}
}

// TestUpscaleWithoutBars vérifie qu'au facteur deux chaque pixel devient un carré de
// quatre, et qu'une sortie qui tombe juste ne garde rien du fond.
func TestUpscaleWithoutBars(t *testing.T) {
	inner := bufferOf(t, 2, 2, rouge, vert, bleu, blanc)
	out := render.NewBuffer(4, 4)

	Upscale(inner, out, 2, demiRge)

	want := [16]render.Color{
		rouge, rouge, vert, vert,
		rouge, rouge, vert, vert,
		bleu, bleu, blanc, blanc,
		bleu, bleu, blanc, blanc,
	}
	for i, veut := range want {
		if out.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, out.Pixels[i], veut)
		}
	}
}

// TestUpscaleBars vérifie que ce qui reste autour de l'image est peint en fond, et
// que l'image est bien centrée dedans.
func TestUpscaleBars(t *testing.T) {
	inner := bufferOf(t, 1, 1, rouge)
	out := render.NewBuffer(3, 3)

	Upscale(inner, out, 1, bleu)

	for i, got := range out.Pixels {
		want := bleu
		if i == 4 {
			want = rouge
		}
		if got != want {
			t.Errorf("pixel %d : %v, attendu %v", i, got, want)
		}
	}
}

// TestUpscaleOddCentering fige le partage du reste : à droite et en bas. Une image
// de référence qui basculerait d'un pixel d'une version à l'autre ne vaudrait rien.
func TestUpscaleOddCentering(t *testing.T) {
	inner := bufferOf(t, 1, 1, rouge)
	out := render.NewBuffer(5, 2)

	Upscale(inner, out, 2, bleu)

	// Deux colonnes de reste : une à gauche, deux à droite.
	for x := 0; x < 5; x++ {
		for y := 0; y < 2; y++ {
			want := bleu
			if x == 1 || x == 2 {
				want = rouge
			}
			if got := out.Pixels[y*5+x]; got != want {
				t.Errorf("pixel (%d, %d) : %v, attendu %v", x, y, got, want)
			}
		}
	}
}

// TestUpscaleSmallerOutput vérifie le découpage centré quand la fenêtre est plus
// petite que la résolution interne, cas d'un redimensionnement et non d'une erreur.
func TestUpscaleSmallerOutput(t *testing.T) {
	inner := bufferOf(t, 4, 2,
		rouge, vert, bleu, blanc,
		blanc, bleu, vert, rouge,
	)
	out := render.NewBuffer(2, 2)

	Upscale(inner, out, 1, demiRge)

	want := [4]render.Color{vert, bleu, bleu, vert}
	for i, veut := range want {
		if out.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, out.Pixels[i], veut)
		}
	}
}

// TestUpscaleShorterOutput éprouve le découpage sur la hauteur, où le décalage
// de centrage devient négatif.
//
// C'est le pendant vertical de TestAgrandirSortiePlusPetite, qui ne rend négatif que
// le décalage horizontal : toutes les sorties du dépôt sont au moins aussi hautes que
// l'image, donc le clamp qui coupe les lignes du haut n'était parcouru par aucun cas.
func TestUpscaleShorterOutput(t *testing.T) {
	inner := bufferOf(t, 2, 4,
		rouge, rouge,
		vert, vert,
		bleu, bleu,
		blanc, blanc,
	)
	out := render.NewBuffer(2, 2)

	Upscale(inner, out, 1, demiRge)

	// Les deux lignes du milieu, le reste d'un centrage impair allant vers le bas.
	want := [4]render.Color{vert, vert, bleu, bleu}
	for i, veut := range want {
		if out.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, out.Pixels[i], veut)
		}
	}
}

// TestUpscaleBarsAndClipping éprouve le croisement des deux : la sortie est plus
// étroite que l'image et plus haute qu'elle, donc il faut découper sur un axe et
// peindre des bandes sur l'autre.
func TestUpscaleBarsAndClipping(t *testing.T) {
	inner := bufferOf(t, 4, 2,
		rouge, vert, bleu, blanc,
		blanc, bleu, vert, rouge,
	)
	out := render.NewBuffer(2, 4)

	Upscale(inner, out, 1, demiRge)

	want := [8]render.Color{
		demiRge, demiRge,
		vert, bleu,
		bleu, vert,
		demiRge, demiRge,
	}
	for i, veut := range want {
		if out.Pixels[i] != veut {
			t.Errorf("pixel %d : %v, attendu %v", i, out.Pixels[i], veut)
		}
	}
}

// TestUpscaleNothingFits vérifie qu'une sortie dans laquelle l'image ne tient pas du
// tout n'est pas laissée à moitié peinte : tout y est fond.
func TestUpscaleNothingFits(t *testing.T) {
	out := render.NewBuffer(2, 2)

	Upscale(render.NewBuffer(0, 0), out, 4, bleu)

	for i, got := range out.Pixels {
		if got != bleu {
			t.Errorf("pixel %d : %v, attendu %v", i, got, bleu)
		}
	}
}

// TestUpscaleSameSize vérifie le chemin rapide : une sortie de même taille au
// facteur un est une recopie.
func TestUpscaleSameSize(t *testing.T) {
	inner := bufferOf(t, 2, 1, rouge, vert)
	out := render.NewBuffer(2, 1)

	Upscale(inner, out, 1, bleu)

	if out.Pixels[0] != rouge || out.Pixels[1] != vert {
		t.Errorf("recopie : %v", out.Pixels)
	}
}

// TestUpscaleInvalidFactor vérifie qu'un facteur nul ou négatif vaut un, plutôt
// que de rendre une sortie vide sans rien dire.
func TestUpscaleInvalidFactor(t *testing.T) {
	for _, facteur := range []int{0, -3} {
		inner := bufferOf(t, 1, 1, rouge)
		sortie := render.NewBuffer(1, 1)

		Upscale(inner, sortie, facteur, bleu)

		if sortie.Pixels[0] != rouge {
			t.Errorf("facteur %d : %v, attendu %v", facteur, sortie.Pixels[0], rouge)
		}
	}
}

// TestUpscaleDoesNotAllocate vérifie que la dernière opération de l'image reste elle
// aussi hors du chemin d'allocation, bandes comprises.
func TestUpscaleDoesNotAllocate(t *testing.T) {
	inner := render.NewBuffer(32, 16)
	out := render.NewBuffer(100, 60)

	if n := testing.AllocsPerRun(10, func() { Upscale(inner, out, 3, bleu) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
