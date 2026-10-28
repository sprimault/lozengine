// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'implémentation de référence du fournisseur : des planches dessinées, découpées en
// grille, une par couple de chose et d'état.

package appearance

import (
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Directions est le nombre de directions que portent des planches dessinées : les
// quatre cardinales de l'écran et les quatre diagonales.
//
// **C'est la seule occurrence de ce nombre dans le moteur**, et un contrôle le
// vérifie. Un fournisseur qui en porte un autre — seize pour une fabrique volumique —
// déclare le sien, et rien ailleurs ne suppose l'un ou l'autre.
//
// Huit parce que c'est ce qu'un jeu dessine à la main sans y passer sa vie, et parce
// que [render.MirrorX] permet d'en tirer la moitié : l'est se déduit de l'ouest sans
// rien ajouter à l'atlas.
const Directions = 8

// Entry décrit ce qu'une planche porte pour un couple de chose et d'état.
//
// La planche se lit en grille : une colonne par direction, une ligne par image
// d'animation, dans l'ordre que [raster.Sheet.CutGrid] produit. Une entrée dont la
// largeur ou la hauteur est nulle ne dessine rien, comme la valeur zéro d'un aspect.
type Entry struct {
	// Sheet est la planche dans l'atlas, Width et Height la taille d'une case.
	Sheet         uint16
	Width, Height uint16

	// AnchorX et AnchorY décalent l'image autour du point qu'elle vise, pour une
	// planche dont le pied n'est pas au milieu de sa case.
	AnchorX, AnchorY int16

	// Tint multiplie la source. La valeur zéro d'une couleur étant transparente, une
	// entrée qu'on veut voir porte [render.White].
	Tint  render.Color
	Flags render.Flags

	// Clip est la cadence de l'animation. Sa valeur zéro tient sur la première ligne
	// de la planche, ce qui convient à ce qui ne s'anime pas.
	Clip Clip
}

// Sheets résout une apparence depuis des planches dessinées.
//
// Il garde une entrée par couple de chose et d'état, et compose l'aspect à la demande.
// La résolution passe par une table de hachage, ce qu'elle peut se permettre : le jeu
// la consulte quand une apparence change, jamais pendant le rendu.
//
// **Ce qu'il ne pose pas** : la couche et l'étage. Ceux-là disent où une chose se range
// dans le monde, ce qui dépend d'où elle se trouve et non de ce qu'elle montre — un
// personnage qui grimpe change d'étage sans changer d'apparence. Le jeu les ajoute à
// l'aspect reçu avant de le poser.
//
// Sa valeur zéro n'est pas utilisable : [NewSheets] monte la table.
type Sheets struct {
	entries map[uint32]Entry
}

// NewSheets monte un fournisseur sans aucune entrée.
func NewSheets() *Sheets {
	return &Sheets{entries: make(map[uint32]Entry)}
}

// Set déclare ce qu'une chose montre dans un état donné, et remplace ce qui s'y
// trouvait.
//
// Les deux premiers nombres n'ont de sens que pour le jeu : il nomme ses types et ses
// états comme il l'entend, et c'est ce qui permet au moteur de n'en coder aucun.
func (s *Sheets) Set(kind, cycle uint16, e Entry) {
	s.entries[entryKey(kind, cycle)] = e
}

// Directions rend [Directions].
func (s *Sheets) Directions() int { return Directions }

// Aspect rend de quoi remplir un quad.
//
// Une demande sans entrée rend la valeur zéro, qui ne dessine rien plutôt que de
// montrer la planche d'autre chose. C'est le repli le plus lisible : un trou se voit,
// un mauvais sprite se discute.
//
// La direction et l'image sont ramenées dans ce que l'entrée porte, de sorte qu'un
// numéro venu d'un compteur qui a débordé reste sur la planche.
func (s *Sheets) Aspect(kind, cycle uint16, facing Facing, frame int) scene.Aspect {
	e, ok := s.entries[entryKey(kind, cycle)]
	if !ok {
		return scene.Aspect{}
	}

	column := int(facing) % Directions
	row := 0
	if n := e.Clip.Frames; n > 1 {
		row = (frame%n + n) % n
	}

	return scene.Aspect{
		Sheet:   e.Sheet,
		SourceX: uint16(column) * e.Width,
		SourceY: uint16(row) * e.Height,
		Width:   e.Width,
		Height:  e.Height,
		AnchorX: e.AnchorX,
		AnchorY: e.AnchorY,
		Tint:    e.Tint,
		Flags:   e.Flags,
	}
}

// entryKey assemble la clé d'une entrée. Deux entiers de seize bits tiennent dans un
// de trente-deux, donc la table n'a pas de structure à hacher.
func entryKey(kind, cycle uint16) uint32 {
	return uint32(kind)<<16 | uint32(cycle)
}
