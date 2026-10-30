// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le texte depuis une planche de glyphes : ce qu'une ligne mesure, et comment elle se
// pose. La mesure vient avant la pose, et c'est elle qui fait tout le travail.

package appearance

import (
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Font décrit une planche de glyphes à cellule fixe.
//
// **Une planche, pas une police.** Le moteur ne rastérise aucun fichier de police : un
// atlas de glyphes est une image comme une autre, produite par la chaîne qui produit
// les sprites, et le moteur n'a donc besoin d'aucune bibliothèque pour afficher du
// texte. C'est ce qui garde son `go.mod` vide.
//
// Les glyphes sont rangés en grille, de gauche à droite puis de haut en bas, dans
// l'ordre que [raster.Sheet.CutGrid] produit. Le caractère de rang zéro est [Font.First]
// et les suivants se déduisent de leur écart avec lui, ce qui convient à un jeu de
// caractères contigu — l'ASCII imprimable, par exemple, de l'espace au tilde.
//
// Sa valeur zéro ne mesure ni ne pose rien, la cellule étant de taille nulle.
type Font struct {
	// Sheet est la planche dans l'atlas, Width et Height la taille d'une cellule.
	Sheet         uint16
	Width, Height uint16

	// Columns est le nombre de colonnes de la planche, et Count le nombre de glyphes
	// qu'elle porte. Au-delà de Count, un caractère est absent.
	Columns, Count int

	// First est le caractère du premier glyphe.
	First rune

	// Advance est de combien une ligne avance par caractère. Zéro prend la largeur
	// d'une cellule, ce qui convient à une planche à chasse fixe ; une valeur plus
	// petite resserre un dessin qui laisse de la marge dans sa cellule.
	Advance uint16

	// Tint multiplie les glyphes, et Layer dit dans quelle couche ils se posent. La
	// valeur zéro d'une couleur étant transparente, une police qu'on veut voir porte
	// [render.White].
	Tint  render.Color
	Flags render.Flags
	Layer uint8
}

// Measure rend la place qu'une ligne occupera, sans rien poser.
//
// **C'est la fonction qui fait le travail, et c'est pourquoi elle existe seule.**
// Centrer un texte, le cadrer à droite, dimensionner le panneau qui le porte, décider
// qu'il ne tient pas : tout cela demande de savoir ce qu'il mesure **avant** de le
// poser. Un moteur qui ne sait que dessiner oblige à recalculer cette largeur à côté,
// et les deux calculs finissent par diverger d'un pixel.
//
// La largeur compte tous les caractères, les absents comme les autres : un glyphe
// manquant occupe sa place au lieu de décaler la fin de la ligne. C'est le seul choix
// qui laisse deux lignes s'aligner quand l'une porte un caractère que la planche n'a
// pas.
func (f Font) Measure(text string) (width, height int) {
	if f.Width == 0 || f.Height == 0 {
		return 0, 0
	}

	count := 0
	for range text {
		count++
	}
	return count * int(f.advance()), int(f.Height)
}

// Glyph rend l'aspect d'un caractère, et dit s'il se trouve dans la planche.
//
// Un caractère absent rend la valeur zéro, qui ne dessine rien. Le composer malgré
// tout, en prenant le glyphe du bord de la planche, afficherait un signe qui n'a pas
// été demandé — et qu'on chercherait dans le texte plutôt que dans la planche.
func (f Font) Glyph(r rune) (scene.Aspect, bool) {
	if f.Width == 0 || f.Height == 0 || f.Columns <= 0 {
		return scene.Aspect{}, false
	}

	index := int(r - f.First)
	if index < 0 || index >= f.Count {
		return scene.Aspect{}, false
	}

	return scene.Aspect{
		Sheet:   f.Sheet,
		SourceX: uint16(index%f.Columns) * f.Width,
		SourceY: uint16(index/f.Columns) * f.Height,
		Width:   f.Width,
		Height:  f.Height,
		Tint:    f.Tint,
		Flags:   f.Flags,
		Layer:   f.Layer,
	}, true
}

// Place pose une ligne sur la scène, en pixels de l'image, et rend sa largeur.
//
// C'est du sucre au-dessus de [Font.Glyph] et de [scene.Scene.PlaceScreen], et non un
// second chemin : chaque glyphe devient un quad ordinaire, rangé par la couche de la
// police comme tout le reste. Un jeu qui veut autre chose — une courbe, un espacement
// variable, une ombre portée — pose ses glyphes lui-même.
//
// Le pixel est celui de la surface, marge comprise. [Font.PlaceArea] vise l'aire de
// monde, pour un texte accroché à une chose du monde.
//
// Le point donné est le coin haut-gauche de la ligne. Pour centrer, mesurer d'abord :
// la largeur rendue ici arrive trop tard pour cela, et c'est à quoi sert [Font.Measure].
//
// Un caractère absent avance sans rien poser, pour que la suite de la ligne reste à sa
// place.
func (f Font) Place(s *scene.Scene, ex, ey int32, text string) int {
	return f.place(s, ex, ey, text, false)
}

// PlaceArea pose une ligne dans l'aire de monde plutôt que sur la surface, et rend sa
// largeur.
//
// **C'est ce qu'il faut pour un texte accroché à une chose du monde** — un chiffre de
// dégâts, un nom au-dessus d'une créature : le pixel donné est relatif à l'aire, la
// ligne s'y découpe, et un objet situé juste hors de l'aire ne vient donc pas écrire son
// chiffre dans la marge, à côté du jeu. [scene.Scene.PlaceArea] porte le repère et la
// raison.
//
// Tout le reste est celui de [Font.Place] : le point donné est le coin haut-gauche de la
// ligne, [Font.Measure] sert à centrer, et un caractère absent avance sans rien poser.
func (f Font) PlaceArea(s *scene.Scene, ex, ey int32, text string) int {
	return f.place(s, ex, ey, text, true)
}

// place avance la ligne glyphe par glyphe, dans l'un ou l'autre repère.
//
// Un drapeau et non la voie de pose en paramètre : passer `s.PlaceScreen` serait une
// valeur de méthode liée, donc une fermeture, et une ligne se pose à chaque image.
// Qu'elle tienne sur la pile dépend de l'analyse d'échappement, ce qui est exactement le
// genre d'hypothèse qu'on ne veut pas dans une fonction appelée par image sans banc qui
// la garde.
func (f Font) place(s *scene.Scene, ex, ey int32, text string, inArea bool) int {
	advance := int32(f.advance())

	at := ex
	for _, r := range text {
		if a, ok := f.Glyph(r); ok {
			if inArea {
				s.PlaceArea(at, ey, a)
			} else {
				s.PlaceScreen(at, ey, a)
			}
		}
		at += advance
	}
	return int(at - ex)
}

// advance rend le pas d'un caractère, la largeur d'une cellule tenant lieu de défaut.
func (f Font) advance() uint16 {
	if f.Advance == 0 {
		return f.Width
	}
	return f.Advance
}
