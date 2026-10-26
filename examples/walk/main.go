// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Un damier plus grand que l'écran, un personnage qu'on déplace, et la carte qui défile
// sous lui.
//
// Les trois pièges d'une vue isométrique sont ici, chacun commenté à l'endroit où il se
// pose : les touches disent une direction d'écran et non du monde, un mouvement se
// refuse sur les deux axes ensemble, et la borne haute d'une grille est le côté exclu.
// Aucun des trois ne se devine, et les trois ont été trouvés en éprouvant la boucle.
package main

import (
	"fmt"

	"github.com/sprimault/lozengine"
)

// side est le côté de la carte, en cellules.
//
// Elle est bien plus grande que la vue, ce qui est le cas courant d'un jeu : on ne
// montre qu'une portion de la carte à un moment donné. Seize cents cellules ici, dont
// la vue n'en affiche qu'une poignée — et une grande carte ne coûte pas plus qu'une
// petite, le décor ne se relisant que sur la portion visible.
const side = 40

// game porte l'état du jeu : la scène qu'il décrit, où se trouve son personnage, et
// s'il a demandé à partir.
type game struct {
	scene *lozengine.Scene
	pawn  lozengine.Handle
	x, y  float64
	done  bool
}

// Step déplace le personnage selon les touches tenues, puis recadre la vue sur lui.
func (g *game) Step(in lozengine.Input) {
	// Échap demande à partir, et c'est tout ce que le jeu a à faire : la boucle pose
	// la question après l'image, donc celle-ci sort quand même.
	if in.JustPressed(lozengine.KeyEscape) {
		g.done = true
		return
	}

	// La vitesse se répartit sur les deux axes, pour qu'une diagonale d'écran n'aille
	// pas plus vite qu'un déplacement de côté.
	const speed = 0.06

	// Les touches disent une direction d'écran, la scène attend des cellules du
	// monde, et en isométrie les deux ne coïncident pas : un pas sur x va vers le
	// bas-gauche de l'écran, un pas sur y vers le bas-droite. Monter à l'écran demande
	// donc de reculer sur les deux à la fois.
	//
	// Le moteur ne tranche pas ce choix, et il ne peut pas : un jeu de stratégie se
	// déplace souvent selon les axes du monde, un jeu d'action selon ceux de l'écran.
	// Ces quatre conditions appartiennent au jeu.
	var dx, dy float64
	if in.Held(lozengine.KeyUp) || in.Held(lozengine.KeyW) {
		dx, dy = dx-speed, dy-speed
	}
	if in.Held(lozengine.KeyDown) || in.Held(lozengine.KeyS) {
		dx, dy = dx+speed, dy+speed
	}
	if in.Held(lozengine.KeyLeft) || in.Held(lozengine.KeyA) {
		dx, dy = dx+speed, dy-speed
	}
	if in.Held(lozengine.KeyRight) || in.Held(lozengine.KeyD) {
		dx, dy = dx-speed, dy+speed
	}

	// Le mouvement se refuse en entier ou pas du tout. Borner chaque axe séparément
	// fait dévier le personnage en diagonale dès qu'il touche un bord : l'un des deux
	// axes passerait, l'autre non, et une touche « gauche » le ferait monter.
	x, y := g.x+dx, g.y+dy
	if !inside(x) || !inside(y) {
		return
	}
	g.x, g.y = x, y
	g.scene.Move(g.pawn, g.x, g.y, 0)

	// La vue suit le personnage, donc c'est la carte qui défile sous lui. Sa taille se
	// relit plutôt que de se supposer : la résolution interne est un réglage du jeu, et
	// la recopier ici ferait deux endroits à tenir d'accord.
	_, _, width, height := g.scene.View()
	g.scene.Focus(g.x, g.y, width, height)
}

// Quitting dit au moteur que le joueur a demandé à partir.
//
// Elle fait de ce jeu un [lozengine.Quitter] en plus d'un [lozengine.Game], et c'est
// tout ce qu'il y a à écrire : sans elle, seule la fermeture de la fenêtre arrêterait
// la boucle.
func (g *game) Quitting() bool { return g.done }

// inside dit si une coordonnée reste sur la carte.
//
// La borne haute est le côté exclu, et non le côté moins un : la cellule k couvre
// l'intervalle de k à k+1, donc la dernière case va de 39 à 40. Borner à 39 arrête le
// personnage une case avant le bord, et seulement de deux côtés — ceux du maximum, la
// borne basse n'étant jamais faussée.
func inside(v float64) bool {
	return v >= 0 && v < side
}

func main() {
	scene, err := lozengine.NewScene(side, side, lozengine.Isometric(32), lozengine.SceneOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	atlas, err := sheets()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Le damier se pose par cellule, et une borne tous les sept pas donne un repère :
	// sur un damier uniforme, rien ne dirait que la carte défile.
	for y := range side {
		for x := range side {
			tile := uint16(0)
			if (x+y)%2 == 1 {
				tile = 1
			}
			scene.PlaceCell(x, y, 0, lozengine.Aspect{
				Sheet: tile, Width: 32, Height: 16, Tint: lozengine.White(),
			})
			if x%7 == 0 && y%7 == 0 {
				scene.PlaceCell(x, y, 1, lozengine.Aspect{
					Sheet: 2, Width: 16, Height: 24,
					Tint: lozengine.Color{R: 200, G: 120, B: 90, A: 255},
				})
			}
		}
	}

	g := &game{scene: scene, x: side / 2, y: side / 2}
	pawn, ok := scene.Place(lozengine.Entity{
		X: g.x, Y: g.y,
		Aspect: lozengine.Aspect{Sheet: 2, Width: 16, Height: 24, Tint: lozengine.White()},
	})
	if !ok {
		fmt.Println("the pawn was refused")
		return
	}
	g.pawn = pawn

	fmt.Println("les flèches ou les quatre touches autour de S déplacent le personnage.")

	if err := lozengine.Run(scene, g, lozengine.Options{
		Title:      "lozengine — un damier et un personnage",
		Atlas:      atlas,
		Background: lozengine.Color{R: 18, G: 20, B: 28, A: 255},
		Icons:      []lozengine.Sheet{icon(16), icon(32)},
	}); err != nil {
		fmt.Println(err)
	}
}

// sheets dessine les trois planches de l'exemple : deux losanges de sol et un
// personnage.
//
// Dessinées et non chargées, pour qu'un exemple se lise sans ouvrir d'autre fichier et
// n'entraîne aucune image dans le dépôt.
func sheets() (*lozengine.Atlas, error) {
	var atlas lozengine.Atlas

	for _, tint := range [2]lozengine.Color{
		{R: 90, G: 130, B: 90, A: 255},
		{R: 70, G: 100, B: 70, A: 255},
	} {
		if _, err := atlas.Add(diamond(tint)); err != nil {
			return nil, err
		}
	}
	if _, err := atlas.Add(pawn()); err != nil {
		return nil, err
	}
	return &atlas, nil
}

// diamond dessine une tuile de sol de 32×16, pointes en haut et en bas.
func diamond(c lozengine.Color) lozengine.Sheet {
	p := lozengine.NewSheet(32, 16)
	for y := range 16 {
		rank := y
		if y >= 8 {
			rank = 15 - y
		}
		half := (rank + 1) * 2

		for x := 16 - half; x < 16+half; x++ {
			p.Pixels[y*32+x] = c
		}
	}
	return p
}

// icon dessine l'icône de la fenêtre : un losange, la forme dont le moteur tire son
// nom, au bord à demi transparent.
//
// Deux tailles sont posées plutôt qu'une, pour que le choix de taille du moteur ait de
// quoi choisir — le système en demande une grande et une petite, et rien n'est
// redimensionné.
//
// **Le bord à alpha partiel est l'intérêt de l'exemple**, et il est écrit par
// [lozengine.Premultiply] comme tout alpha partiel du moteur. C'est la seule chose
// qu'aucun test ne peut juger : le prémultiplié se défait au dessin, chez le shell du
// système, donc un bord mal composé ne se voit qu'à l'œil dans la barre des tâches.
func icon(side int) lozengine.Sheet {
	p := lozengine.NewSheet(uint16(side), uint16(side))
	half := side / 2

	body := lozengine.Premultiply(120, 170, 220, 255)
	fringe := lozengine.Premultiply(120, 170, 220, 128)

	for y := range side {
		for x := range side {
			// Un losange est l'ensemble des points dont la distance au centre,
			// comptée en pas horizontaux plus pas verticaux, ne dépasse pas le
			// demi-côté.
			switch d := abs(x-half) + abs(y-half); {
			case d < half-1:
				p.Pixels[y*side+x] = body
			case d <= half:
				p.Pixels[y*side+x] = fringe
			}
		}
	}
	return p
}

// abs rend la valeur absolue, que la bibliothèque standard ne donne que pour les
// flottants.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// pawn dessine une silhouette debout de 16×24, au contour plus sombre.
func pawn() lozengine.Sheet {
	p := lozengine.NewSheet(16, 24)
	body := lozengine.Color{R: 220, G: 200, B: 160, A: 255}
	edge := lozengine.Color{R: 90, G: 70, B: 50, A: 255}

	for y := range 24 {
		for x := 3; x <= 12; x++ {
			c := body
			if x == 3 || x == 12 || y == 0 || y == 23 {
				c = edge
			}
			p.Pixels[y*16+x] = c
		}
	}
	return p
}
