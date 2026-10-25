// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La clé de tri et la répartition de ses soixante-quatre bits, qui est la seule
// vérité de l'ordre de dessin. Les largeurs qu'on lit ici fixent ce que le moteur
// annonce savoir distinguer, et un champ dépassé sature au lieu de déborder sur son
// voisin.

package sorting

import "github.com/sprimault/lozengine/render"

// La répartition des bits d'une clé, du plus fort au plus faible.
//
// L'ordre des champs est l'ordre de priorité, et c'est tout le procédé : ranger les
// clés par valeur croissante range les choses à peindre dans l'ordre où on les peint.
//
// Les largeurs ne sont pas négociables une fois qu'une image a été rendue avec : elles
// fixent ce que le moteur annonce savoir distinguer — seize couches, soixante-quatre
// étages, 32 768 cellules de côté.
const (
	indexBits  = 10
	sheetBits  = 10
	heightBits = 16
	depthBits  = 18
	floorBits  = 6
	layerBits  = 4

	sheetShift  = indexBits
	heightShift = sheetShift + sheetBits
	depthShift  = heightShift + heightBits
	floorShift  = depthShift + depthBits
	layerShift  = floorShift + floorBits
)

// Ce que chaque champ porte au plus. Dépasser ne déborde pas sur le voisin — la
// composition masque — mais range la chose ailleurs qu'où elle devrait aller.
const (
	MaxIndex  = 1<<indexBits - 1
	MaxSheet  = 1<<sheetBits - 1
	MaxHeight = 1<<heightBits - 1
	MaxDepth  = 1<<depthBits - 1
)

// Les plafonds de la couche et de l'étage ne sont pas déclarés ici : ce sont
// [render.MaxLayer] et [render.MaxFloor], parce qu'un jeu décide combien de couches il
// emploie et doit donc pouvoir les lire. La répartition des bits reste ici, et un test
// vérifie que les deux disent le même nombre.

// Key porte l'ordre de dessin d'une chose à peindre, du champ le plus fort au plus
// faible : couche, étage, profondeur, hauteur, planche, index stable.
//
// La profondeur se compte en quarts de cellule et la hauteur aussi, toutes deux telles
// que la projection les a rendues — les recalculer ici ferait un second arrondi, qui
// peut tomber de l'autre côté d'un quart et réordonner deux choses sans que rien ne le
// signale.
//
// La planche est avant-dernière pour que les quads d'une même profondeur sortent
// groupés par planche. Le rastériseur peut n'en changer que par séries ; il ne doit pas
// en dépendre.
type Key uint64

// Compose assemble une clé. Chaque champ est ramené à ce qu'il porte, donc un
// dépassement range la chose au bord plutôt que dans le champ voisin.
func Compose(layer, floor uint8, depth uint32, height uint16, sheet, index uint16) Key {
	return Key(min(uint64(layer), render.MaxLayer))<<layerShift |
		Key(min(uint64(floor), render.MaxFloor))<<floorShift |
		Key(min(uint64(depth), MaxDepth))<<depthShift |
		Key(uint64(height))<<heightShift |
		Key(min(uint64(sheet), MaxSheet))<<sheetShift |
		Key(min(uint64(index), MaxIndex))
}

// Index rend l'index stable, le seul champ que le départage ajuste.
func (c Key) Index() uint16 { return uint16(c) & MaxIndex }

// WithIndex rend la même clé portant un autre index stable.
//
// C'est par là que passe tout ce que le départage a décidé : la clé reste la seule
// vérité de l'ordre de dessin, et un rang qui vivrait ailleurs se perdrait au moment
// du rangement.
func (c Key) WithIndex(index uint16) Key {
	return c&^MaxIndex | Key(min(uint64(index), MaxIndex))
}

// Depth rend la somme des deux axes du monde, en quarts de cellule.
//
// Elle sert au départage, qui n'ordonne que des choses d'une même profondeur :
// au-delà, la clé a déjà tranché et il n'a rien à dire.
func (c Key) Depth() uint32 {
	return uint32(c>>depthShift) & MaxDepth
}
