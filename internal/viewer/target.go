// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La mire de démonstration, et le seul scénario qui lise une entrée simulée : elle
// exerce les quatre chemins du dessin d'un coup, et le découpage puisque le pion entre
// et sort par les bords.

package viewer

import (
	"fmt"
	"strings"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// Target est le scénario de démonstration : un sol en damier de losanges, un pion qui
// traverse l'image et en sort, et un voile translucide qui balaie le tout.
//
// Il est fait pour exercer les quatre chemins du blit d'un seul coup — copie pour le
// pion, modulation pour les tuiles, mélange pour leurs bords, aplat pour le voile — et
// le découpage, puisque le pion entre et sort par les bords.
func Target() Scenario {
	const (
		width, height = 128, 72
		tileW, tileH  = 32, 16
		pawnSide      = 8
	)

	// Deux teintes sur une planche blanche : la couleur du sol vient de la modulation,
	// ce qui évite deux planches pour un damier.
	ground := [2]render.Color{
		{R: 90, G: 130, B: 90, A: 255},
		{R: 70, G: 100, B: 75, A: 255},
	}

	return Scenario{
		Name:       "target",
		Width:      width,
		Height:     height,
		Frames:     24,
		Background: render.Color{R: 12, G: 12, B: 18, A: 255},

		// Quatre moments plutôt que vingt-quatre images : le pion hors champ à
		// gauche, puis dedans, puis le voile au milieu, puis le pion qui sort à
		// droite. Verser la totalité ferait vingt-quatre fichiers à relire à chaque
		// régénération, pour quatre cas distincts.
		References: []int{0, 8, 16, 23},

		Atlas: func() (*raster.Atlas, error) {
			var atlas raster.Atlas
			for _, p := range []raster.Sheet{tileSheet(), pawnSheet()} {
				if _, err := atlas.Add(p); err != nil {
					return nil, err
				}
			}
			return &atlas, nil
		},

		Quads: func(frame int, e InputLine, reserve render.DrawList) render.DrawList {
			quads := reserve

			for row := 0; row*tileH/2 < height+tileH; row++ {
				for column := 0; column*tileW < width+tileW; column++ {
					quads = append(quads, render.Quad{
						X:      int16(column*tileW - tileW + (row%2)*(tileW/2)),
						Y:      int16(row*tileH/2 - tileH),
						Width:  tileW,
						Height: tileH,
						Tint:   ground[(column+row)%2],
					})
				}
			}

			x, y := -pawnSide+frame*6, 30
			if px, py, ok := pawnPosition(e); ok {
				x, y = px, py
			}
			quads = append(quads, render.Quad{
				X: int16(x), Y: int16(y), Width: pawnSide, Height: pawnSide,
				Sheet: 1, Tint: render.White(),
			})

			if !strings.Contains(string(e), "no-veil") {
				quads = append(quads, render.Quad{
					X: int16(width - 4 - frame*5), Y: 0, Width: 24, Height: height,
					Tint: render.Color{A: 96}, Flags: render.Flat,
				})
			}
			return quads
		},
	}
}

// pawnPosition lit la directive « pawn X Y » d'une entrée.
//
// Une position absolue et non un déplacement : un scénario doit rester une fonction
// pure de son numéro d'image, donc rien ne s'accumule d'une image à la suivante.
func pawnPosition(e InputLine) (x, y int, ok bool) {
	if _, err := fmt.Sscanf(string(e), "pawn %d %d", &x, &y); err != nil {
		return 0, 0, false
	}
	return x, y, true
}

// tileSheet dessine un losange 32×16 blanc au bord adouci, que la teinte colore.
func tileSheet() raster.Sheet {
	p := raster.NewSheet(32, 16)
	for y := range 16 {
		rank := y
		if y >= 8 {
			rank = 15 - y
		}
		half := (rank + 1) * 2

		for x := 16 - half; x < 16+half; x++ {
			color := render.Color{R: 255, G: 255, B: 255, A: 255}
			if x == 16-half || x == 16+half-1 {
				color = render.Color{R: 128, G: 128, B: 128, A: 128}
			}
			p.Pixels[y*32+x] = color
		}
	}
	return p
}

// pawnSheet dessine un carré de 8 au contour translucide.
func pawnSheet() raster.Sheet {
	p := raster.NewSheet(8, 8)
	for y := range 8 {
		for x := range 8 {
			color := render.Color{R: 255, G: 255, B: 255, A: 255}
			if x == 0 || y == 0 || x == 7 || y == 7 {
				color = render.Color{R: 96, G: 96, B: 96, A: 96}
			}
			p.Pixels[y*8+x] = color
		}
	}
	return p
}
