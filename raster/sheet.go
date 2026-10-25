// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La planche : les pixels d'une image chargée, et la découpe en grille qui donne les
// rectangles d'un sprite sans recopier quoi que ce soit.

package raster

import (
	"fmt"

	"github.com/sprimault/lozengine/render"
)

// Sheet est une page de pixels dans laquelle les quads puisent, ligne par ligne
// et sans remplissage : le pixel (x, y) est à l'index y×Width + x.
//
// Ses composantes sont prémultipliées, comme celles de [render.Color] : le
// mélange d'un pixel se ramène ainsi à une multiplication et une addition, et la
// conversion depuis un format qui ne l'est pas a lieu au chargement.
//
// Rectangles porte la découpe, remplie par [Sheet.CutGrid] ou à la main.
// Vide, la planche vaut pour elle-même — une tuile unique n'a rien à découper.
//
// Les séquences de lignes, elles, ne se remplissent pas à la main : [Atlas.Add]
// les calcule, et c'est le seul passage obligé avant qu'un quad puisse désigner la
// planche. Une planche construite pixel par pixel est donc encodée comme les
// autres, sans que personne ait à y penser.
type Sheet struct {
	Width, Height uint16
	Pixels        []render.Color
	Rectangles    []Rectangle

	runs []run
	rows []uint32 // Height+1 bornes : la ligne y occupe [rows[y], rows[y+1])
}

// NewSheet rend une planche transparente aux dimensions demandées.
//
// C'est par elle que les scènes de référence se construisent : un damier, un
// aplat, un dégradé d'alpha se calculent en quelques lignes, et la suite de
// non-régression ne dépend alors d'aucun fichier qu'il faudrait savoir refaire.
func NewSheet(width, height uint16) Sheet {
	return Sheet{
		Width:  width,
		Height: height,
		Pixels: make([]render.Color, int(width)*int(height)),
	}
}

// CutGrid remplit Rectangles d'une grille de cases, de gauche à droite puis
// de haut en bas. Cet ordre est celui qu'un fournisseur d'apparence suppose pour
// numéroter ses directions et ses images d'animation.
//
// La case doit diviser la planche : un reste ferait tomber une colonne ou une
// ligne en silence, et c'est sur une planche de directions que personne ne le
// verrait.
func (p *Sheet) CutGrid(width, height uint16) error {
	if width == 0 || height == 0 {
		return fmt.Errorf("grid cut: cell of %d×%d", width, height)
	}
	if p.Width%width != 0 || p.Height%height != 0 {
		return fmt.Errorf("grid cut: a cell of %d×%d does not divide a sheet of %d×%d",
			width, height, p.Width, p.Height)
	}

	columns, rows := p.Width/width, p.Height/height
	p.Rectangles = make([]Rectangle, 0, int(columns)*int(rows))
	for r := uint16(0); r < rows; r++ {
		for c := uint16(0); c < columns; c++ {
			p.Rectangles = append(p.Rectangles, Rectangle{c * width, r * height, width, height})
		}
	}
	return nil
}
