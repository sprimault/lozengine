// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package raster

import (
	"fmt"

	"github.com/sprimault/lozengine/internal/rendu"
)

// Planche est une page de pixels dans laquelle les quads puisent, ligne par ligne
// et sans remplissage : le pixel (x, y) est à l'index y×Largeur + x.
//
// Ses composantes sont prémultipliées, comme celles de [rendu.Couleur] : le
// mélange d'un pixel se ramène ainsi à une multiplication et une addition, et la
// conversion depuis un format qui ne l'est pas a lieu au chargement.
//
// Rectangles porte la découpe, remplie par [Planche.DecouperGrille] ou à la main.
// Vide, la planche vaut pour elle-même — une tuile unique n'a rien à découper.
//
// Les séquences de lignes, elles, ne se remplissent pas à la main : [Atlas.Ajouter]
// les calcule, et c'est le seul passage obligé avant qu'un quad puisse désigner la
// planche. Une planche construite pixel par pixel est donc encodée comme les
// autres, sans que personne ait à y penser.
type Planche struct {
	Largeur, Hauteur uint16
	Pixels           []rendu.Couleur
	Rectangles       []Rectangle

	sequences []sequence
	lignes    []uint32 // Hauteur+1 bornes : la ligne y occupe [lignes[y], lignes[y+1])
}

// NouvellePlanche rend une planche transparente aux dimensions demandées.
//
// C'est par elle que les scènes de référence se construisent : un damier, un
// aplat, un dégradé d'alpha se calculent en quelques lignes, et la suite de
// non-régression ne dépend alors d'aucun fichier qu'il faudrait savoir refaire.
func NouvellePlanche(largeur, hauteur uint16) Planche {
	return Planche{
		Largeur: largeur,
		Hauteur: hauteur,
		Pixels:  make([]rendu.Couleur, int(largeur)*int(hauteur)),
	}
}

// DecouperGrille remplit Rectangles d'une grille de cases, de gauche à droite puis
// de haut en bas. Cet ordre est celui qu'un fournisseur d'apparence suppose pour
// numéroter ses directions et ses images d'animation.
//
// La case doit diviser la planche : un reste ferait tomber une colonne ou une
// ligne en silence, et c'est sur une planche de directions que personne ne le
// verrait.
func (p *Planche) DecouperGrille(largeur, hauteur uint16) error {
	if largeur == 0 || hauteur == 0 {
		return fmt.Errorf("découpe en grille : case de %d×%d", largeur, hauteur)
	}
	if p.Largeur%largeur != 0 || p.Hauteur%hauteur != 0 {
		return fmt.Errorf("découpe en grille : une case de %d×%d ne divise pas une planche de %d×%d",
			largeur, hauteur, p.Largeur, p.Hauteur)
	}

	colonnes, lignes := p.Largeur/largeur, p.Hauteur/hauteur
	p.Rectangles = make([]Rectangle, 0, int(colonnes)*int(lignes))
	for l := uint16(0); l < lignes; l++ {
		for c := uint16(0); c < colonnes; c++ {
			p.Rectangles = append(p.Rectangles, Rectangle{c * largeur, l * hauteur, largeur, hauteur})
		}
	}
	return nil
}
