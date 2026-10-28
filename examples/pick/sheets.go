// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les planches de l'exemple, dessinées par code : un sol, une plateforme, un repère de
// case, et une planche de glyphes assez large pour écrire ce que l'exemple affiche.

package main

import "github.com/sprimault/lozengine"

// Les caractères que la planche de glyphes porte, dans l'ordre où elle les range.
//
// Un jeu de caractères contigu est ce que [lozengine.Font] sait lire : le rang d'un
// signe s'y déduit de son écart avec le premier. L'exemple n'écrit que des majuscules,
// des chiffres et deux espacements, donc il couvre l'ASCII imprimable de l'espace à
// la lettre Z et s'arrête là.
const (
	glyphFirst = ' '
	glyphCount = 'Z' - glyphFirst + 1

	glyphW, glyphH  = 4, 6
	glyphColumns    = 16
	glyphRows       = (glyphCount + glyphColumns - 1) / glyphColumns
	glyphAdvanceGap = 1
)

// build monte l'atlas de l'exemple et la police qui va avec.
func build() (*lozengine.Atlas, lozengine.Font) {
	var atlas lozengine.Atlas

	for _, sheet := range []lozengine.Sheet{
		tile(32, 16, lozengine.Color{R: 76, G: 92, B: 76, A: 255}),
		tile(32, 32, lozengine.Color{R: 164, G: 150, B: 120, A: 255}),
		frame(32, 16, lozengine.Color{R: 240, G: 220, B: 120, A: 255}),
		glyphs(),
	} {
		if _, err := atlas.Add(sheet); err != nil {
			panic(err)
		}
	}

	return &atlas, lozengine.Font{
		Sheet: glyphSheet, Width: glyphW, Height: glyphH,
		Columns: glyphColumns, Count: glyphCount, First: glyphFirst,
		Advance: glyphW + glyphAdvanceGap,
		Tint:    lozengine.White(),

		// La couche la plus haute que l'exemple emploie : l'interface passe devant
		// le monde, et c'est la seule chose qui l'en distingue.
		Layer: 2,
	}
}

// tile rend une planche pleine, bordée d'un ton plus sombre pour que les cellules se
// distinguent les unes des autres.
func tile(width, height uint16, c lozengine.Color) lozengine.Sheet {
	border := lozengine.Color{R: c.R / 2, G: c.G / 2, B: c.B / 2, A: 255}

	s := lozengine.NewSheet(width, height)
	for y := range int(height) {
		for x := range int(width) {
			tint := c
			if x == 0 || y == 0 || x == int(width)-1 || y == int(height)-1 {
				tint = border
			}
			s.Pixels[y*int(width)+x] = tint
		}
	}
	return s
}

// frame rend un rectangle creux, qui sert de repère de case : il entoure la cellule
// désignée sans la masquer.
func frame(width, height uint16, c lozengine.Color) lozengine.Sheet {
	s := lozengine.NewSheet(width, height)
	for y := range int(height) {
		for x := range int(width) {
			if x == 0 || y == 0 || x == int(width)-1 || y == int(height)-1 {
				s.Pixels[y*int(width)+x] = c
			}
		}
	}
	return s
}

// glyphs rend la planche de caractères.
//
// **Elle est dessinée par code et non chargée**, pour que l'exemple tienne sans
// ressource à côté de lui. Un vrai jeu charge un atlas produit par sa chaîne de
// fabrication ; ce que le moteur lit est le même dans les deux cas — une planche de
// cellules de taille fixe, et rien d'autre.
//
// Chaque signe est décrit par six lignes de quatre bits, lues du haut vers le bas et
// de la gauche vers la droite. C'est illisible et c'est assumé : le sujet de l'exemple
// est la désignation, pas la typographie.
func glyphs() lozengine.Sheet {
	s := lozengine.NewSheet(glyphColumns*glyphW, glyphRows*glyphH)
	ink := lozengine.White()

	for index, rows := range glyphRowsOf() {
		originX := (index % glyphColumns) * glyphW
		originY := (index / glyphColumns) * glyphH

		for y, bits := range rows {
			for x := range glyphW {
				if bits&(1<<(glyphW-1-x)) == 0 {
					continue
				}
				s.Pixels[(originY+y)*glyphColumns*glyphW+originX+x] = ink
			}
		}
	}
	return s
}

// glyphRowsOf rend les motifs de chaque signe, dans l'ordre de la planche.
//
// Ce que l'exemple n'écrit pas reste vide : une cellule blanche est un glyphe qui ne
// dessine rien, ce qui vaut mieux qu'un signe approximatif. Seuls l'espace, les
// chiffres et les lettres employées par ses deux lignes sont tracés.
func glyphRowsOf() [glyphCount][glyphH]uint8 {
	var table [glyphCount][glyphH]uint8

	drawn := map[rune][glyphH]uint8{
		'0': {0x6, 0x9, 0x9, 0x9, 0x9, 0x6},
		'1': {0x2, 0x6, 0x2, 0x2, 0x2, 0x7},
		'2': {0x6, 0x9, 0x1, 0x2, 0x4, 0xF},
		'3': {0xE, 0x1, 0x6, 0x1, 0x1, 0xE},
		'4': {0x9, 0x9, 0xF, 0x1, 0x1, 0x1},
		'5': {0xF, 0x8, 0xE, 0x1, 0x1, 0xE},
		'6': {0x6, 0x8, 0xE, 0x9, 0x9, 0x6},
		'7': {0xF, 0x1, 0x2, 0x4, 0x4, 0x4},
		'8': {0x6, 0x9, 0x6, 0x9, 0x9, 0x6},
		'9': {0x6, 0x9, 0x9, 0x7, 0x1, 0x6},
		'A': {0x6, 0x9, 0x9, 0xF, 0x9, 0x9},
		'C': {0x6, 0x9, 0x8, 0x8, 0x9, 0x6},
		'D': {0xE, 0x9, 0x9, 0x9, 0x9, 0xE},
		'E': {0xF, 0x8, 0xE, 0x8, 0x8, 0xF},
		'G': {0x7, 0x8, 0xB, 0x9, 0x9, 0x7},
		'I': {0x7, 0x2, 0x2, 0x2, 0x2, 0x7},
		'N': {0x9, 0xD, 0xB, 0x9, 0x9, 0x9},
		'S': {0x7, 0x8, 0x6, 0x1, 0x1, 0xE},
		'U': {0x9, 0x9, 0x9, 0x9, 0x9, 0x6},
	}

	for r, rows := range drawn {
		table[r-glyphFirst] = rows
	}
	return table
}
