// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la désignation au pointeur doit tenir : être l'exacte réciproque de
// l'agrandissement, mesurée sur ce que celui-ci dessine vraiment.

package lozengine

import "testing"

// marked rend un tampon noir portant un seul pixel blanc, à la place demandée.
func marked(width, height uint16, x, y int) *Buffer {
	b := NewBuffer(width, height)
	for i := range b.Pixels {
		b.Pixels[i] = Color{A: 255}
	}
	b.Pixels[y*int(width)+x] = White()
	return b
}

// TestAimIsTheReciprocalOfFit vérifie que le pixel désigné par une position de fenêtre
// est celui que l'agrandissement a peint à cette position.
//
// **Il ne recalcule pas la conversion, il la mesure sur le rendu.** Un pixel unique est
// posé dans l'image, celle-ci est agrandie comme la boucle l'agrandit, puis chaque
// pixel de la fenêtre qui porte sa couleur est donné à la désignation : elle doit
// rendre les coordonnées du pixel d'origine. Les deux chemins restent donc
// indépendants, et un centrage qui changerait d'un côté les séparerait.
//
// C'est le défaut que ce cas existe pour attraper, et il ne se verrait pas autrement :
// une désignation décalée d'un pixel au facteur un l'est de sept au facteur sept, et
// le jeu désigne alors une case voisine de celle qu'on voit sous le curseur.
func TestAimIsTheReciprocalOfFit(t *testing.T) {
	const innerW, innerH = 16, 9

	for _, c := range []struct {
		name           string
		outW, outH     uint16
		pixelX, pixelY int
	}{
		{"taille exacte", innerW, innerH, 3, 2},
		{"facteur trois, sans bande", innerW * 3, innerH * 3, 0, 0},
		{"facteur deux et des bandes", innerW*2 + 7, innerH*2 + 5, 15, 8},
		{"bandes impaires, reste à droite", innerW*4 + 3, innerH*4 + 1, 9, 4},

		// Plus petite que l'image, celle-ci est découpée autour de son centre : le
		// repère doit donc s'y trouver, sinon le cas ne peint rien et ne garde rien.
		{"fenêtre plus petite que l'image", innerW - 4, innerH - 2, 8, 4},
	} {
		inner := marked(innerW, innerH, c.pixelX, c.pixelY)
		out := NewBuffer(c.outW, c.outH)
		fit(inner, out, Color{R: 128, A: 255})

		found := 0
		for y := range int(c.outH) {
			for x := range int(c.outW) {
				if out.Pixels[y*int(c.outW)+x] != White() {
					continue
				}
				found++

				in := Input{}.WithPointer(int16(x), int16(y), true)
				gx, gy, inside := aim(in, inner, out).Pointer()

				if !inside || int(gx) != c.pixelX || int(gy) != c.pixelY {
					t.Errorf("%s : la fenêtre en (%d, %d) désigne (%d, %d) dedans=%v, attendu (%d, %d)",
						c.name, x, y, gx, gy, inside, c.pixelX, c.pixelY)
				}
			}
		}

		if found == 0 {
			t.Errorf("%s : le pixel repère n'a pas été peint, le cas ne garde rien", c.name)
		}
	}
}

// TestAimPutsTheBarsOutside vérifie qu'une position dans les bandes ne désigne aucun
// pixel de l'image.
//
// Sans cela, les bandes de gauche et du haut rendraient zéro — la division de Go
// arrondissant vers zéro et non vers le bas —, donc elles désigneraient la première
// case au lieu de rester dehors. Le défaut ne se verrait qu'au bord de l'écran, là où
// on ne le cherche pas.
func TestAimPutsTheBarsOutside(t *testing.T) {
	inner := NewBuffer(16, 9)
	out := NewBuffer(16*4+9, 9*4+7)

	for _, p := range [][2]int16{{0, 0}, {3, 20}, {40, 1}, {71, 42}} {
		in := Input{}.WithPointer(p[0], p[1], true)
		if _, _, inside := aim(in, inner, out).Pointer(); inside {
			t.Errorf("la fenêtre en (%d, %d) se dit sur l'image, attendu dehors", p[0], p[1])
		}
	}
}

// TestAimWithoutAWindowPointsAtNothing garde le cas d'une fenêtre réduite, où aucun
// tampon n'existe et où rien ne peut donc être désigné.
func TestAimWithoutAWindowPointsAtNothing(t *testing.T) {
	in := Input{}.WithPointer(40, 20, true)
	if _, _, inside := aim(in, NewBuffer(16, 9), nil).Pointer(); inside {
		t.Error("pointeur dans l'image sans fenêtre, attendu dehors")
	}
}
