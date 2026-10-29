// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Un décor à deux niveaux, une case qu'on désigne à la souris, et ses coordonnées
// écrites à l'écran.
//
// C'est l'exemple qui exerce ce que le jalon 4 ajoute et qu'aucun autre ne touche : la
// position du pointeur ramenée en cases du monde, la couche d'interface posée en
// pixels, et le texte depuis une planche de glyphes. Les trois pièges sont commentés
// là où ils se posent — un étage se désigne à son étage, une interface se place par
// son coin, et un texte se mesure avant d'être centré.
package main

import (
	"fmt"

	"github.com/sprimault/lozengine"
)

// Les dimensions de la carte et de la plateforme qu'elle porte.
const (
	side = 12

	platformFrom = 4
	platformTo   = 7
)

// Les planches, dans l'ordre où elles entrent dans l'atlas.
const (
	groundSheet = iota
	platformSheet
	markerSheet
	glyphSheet
)

// game porte ce que le jeu sait : sa scène, la case que le pointeur désigne, et s'il
// faut partir.
type game struct {
	scene *lozengine.Scene
	font  lozengine.Font
	mark  lozengine.Handle

	cellX, cellY int
	onMap        bool
	done         bool
}

// Step lit le pointeur, en déduit la case désignée, et réécrit l'interface.
func (g *game) Step(in lozengine.Input) {
	if in.JustPressed(lozengine.KeyEscape) {
		g.done = true
		return
	}

	px, py, inside := in.Pointer()
	g.onMap = false

	if inside {
		// **Une case se désigne à son étage.** Le même pixel montre le sol et la
		// plateforme qui le surplombe ; demander l'étage zéro rend la case du sol,
		// l'étage un celle du dessus. Un jeu qui veut l'une ou l'autre le dit, le
		// moteur ne devinant pas lequel des deux on vise.
		x, y := g.scene.At(int32(px), int32(py), 0)

		g.cellX, g.cellY = int(x), int(y)
		g.onMap = x >= 0 && y >= 0 && g.cellX < side && g.cellY < side
	}

	// Le repère suit la case désignée, et se range hors de la carte quand il n'y en a
	// pas : rien n'oblige à le retirer, et le replacer coûte moins qu'une pose.
	if g.onMap {
		g.scene.Move(g.mark, float64(g.cellX)+0.5, float64(g.cellY)+0.5, 0)
	} else {
		g.scene.Move(g.mark, -10, -10, 0)
	}

	g.writeOverlay()
}

// writeOverlay réécrit la ligne d'interface du pas.
//
// Elle se vide puis se repose en entier : c'est le geste courant pour un texte qui
// change à chaque image, et la tranche garde sa capacité, donc rien ne se réalloue.
func (g *game) writeOverlay() {
	g.scene.ClearScreen()

	line := "DESIGNE AUCUNE CASE"
	if g.onMap {
		line = fmt.Sprintf("CASE %d %d", g.cellX, g.cellY)
	}

	// **Mesurer avant de poser**, parce que la largeur rendue par la pose arrive trop
	// tard pour centrer quoi que ce soit. C'est tout l'intérêt d'avoir les deux.
	width, height := g.font.Measure(line)

	// **La taille de la vue se lit, elle ne se suppose pas** : elle suit la fenêtre, et
	// une constante écrite ici mettrait la ligne hors de l'image dès le premier
	// agrandissement — sans erreur, puisque ce qui sort du cadre est simplement écarté.
	_, _, view, tall := g.scene.View()
	g.font.Place(g.scene, (view-int32(width))/2, tall-int32(height)-4, line)
}

// Quitting dit à la boucle qu'on a demandé à partir.
func (g *game) Quitting() bool { return g.done }

func main() {
	atlas, font := build()

	s, err := lozengine.NewScene(side, side, lozengine.Dimetric(32, 16),
		lozengine.SceneOptions{Entities: 4})
	if err != nil {
		panic(err)
	}
	// Pas de cadrage ici : le moteur centre la vue sur la grille quand le jeu ne l'a
	// pas fait, et à la taille qu'il décide. Le dire deux fois ferait dépendre
	// l'exemple du défaut de la résolution interne sans que rien ne les relie.

	ground := lozengine.Aspect{
		Sheet: groundSheet, Width: 32, Height: 16, Tint: lozengine.White(),
	}
	for v := range side {
		for u := range side {
			s.PlaceCell(u, v, 0, ground)
		}
	}

	// La plateforme est à l'étage un : elle monte à l'écran d'une hauteur de cellule,
	// et passe devant tout l'étage du sol.
	platform := lozengine.Aspect{
		Sheet: platformSheet, Width: 32, Height: 32, Tint: lozengine.White(), Floor: 1,
	}
	for v := platformFrom; v <= platformTo; v++ {
		for u := platformFrom; u <= platformTo; u++ {
			s.PlaceCell(u, v, 1, platform)
		}
	}

	mark, ok := s.Place(lozengine.Entity{X: -10, Y: -10, Aspect: lozengine.Aspect{
		Sheet: markerSheet, Width: 32, Height: 16, Tint: lozengine.White(), Layer: 1,
	}})
	if !ok {
		panic("repère refusé")
	}

	g := &game{scene: s, font: font, mark: mark}

	// Le curseur du système s'efface : le repère dessiné dans la scène tient sa place,
	// et les deux ensemble donneraient deux pointeurs décalés l'un de l'autre.
	if err := lozengine.Run(s, g, lozengine.Options{
		Title:      "lozengine — désignation",
		Atlas:      atlas,
		Background: lozengine.Color{R: 18, G: 20, B: 28, A: 255},
		HideCursor: true,
	}); err != nil {
		fmt.Println(err)
	}
}
