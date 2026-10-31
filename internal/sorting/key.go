// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La clé de tri et la répartition de ses soixante-quatre bits, qui ordonne le dessin
// partout où elle ne laisse pas deux choses à égalité. Les largeurs qu'on lit ici fixent
// ce que le moteur annonce savoir distinguer, et un champ dépassé sature au lieu de
// déborder sur son voisin.

package sorting

import (
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// La répartition des bits d'une clé, du plus fort au plus faible.
//
// L'ordre des champs est l'ordre de priorité, et c'est tout le procédé : ranger les
// clés par valeur croissante range les choses à peindre dans l'ordre où on les peint.
//
// Les largeurs ne sont pas négociables une fois qu'une image a été rendue avec : elles
// fixent ce que le moteur annonce savoir distinguer — seize couches, soixante-quatre
// étages, 32 768 cellules de côté.
const (
	indexBits  = 15
	tierBits   = 5
	heightBits = 16
	depthBits  = 18
	floorBits  = 6
	layerBits  = 4

	tierShift   = indexBits
	heightShift = tierShift + tierBits
	depthShift  = heightShift + heightBits
	floorShift  = depthShift + depthBits
	layerShift  = floorShift + floorBits
)

// Ce que chaque champ porte au plus. Dépasser ne déborde pas sur le voisin — la
// composition masque — mais range la chose ailleurs qu'où elle devrait aller.
const (
	MaxIndex  = 1<<indexBits - 1
	MaxTier   = 1<<tierBits - 1
	MaxHeight = 1<<heightBits - 1
	MaxDepth  = 1<<depthBits - 1
)

// entityTier est le rang que porte tout ce qui n'est pas une strate de décor : une
// entité, une pièce d'écran, un quad révélé.
//
// **Au-dessus de toutes les strates, et c'est ce qui rend l'ordre déterministe** : à
// profondeur et hauteur égales, le décor d'une cellule se peint avant ce qui s'y
// tient. Cet ordre existait déjà, mais il venait de l'index de planche, donc de
// l'ordre dans lequel le jeu avait chargé ses images — une entité dont la planche
// portait un index bas passait sous le sol. Le dire ici le rend vrai pour tout jeu.
//
// `scene.MaxTiers` valant seize, les strates occupent zéro à quinze et cette valeur
// est la première libre ; [MaxTier] en porte trente-et-une, donc le champ ne sature
// pas.
const entityTier = scene.MaxTiers

// Les plafonds de la couche et de l'étage ne sont pas déclarés ici : ce sont
// [render.MaxLayer] et [render.MaxFloor], parce qu'un jeu décide combien de couches il
// emploie et doit donc pouvoir les lire. La répartition des bits reste ici, et un test
// vérifie que les deux disent le même nombre.

// Key porte l'ordre de dessin d'une chose à peindre, du champ le plus fort au plus
// faible : couche, étage, profondeur, hauteur, rang d'empilement, index stable.
//
// La profondeur se compte en quarts de cellule et la hauteur aussi, toutes deux telles
// que la projection les a rendues — les recalculer ici ferait un second arrondi, qui
// peut tomber de l'autre côté d'un quart et réordonner deux choses sans que rien ne le
// signale.
//
// **Le rang d'empilement est la strate d'une cellule de décor, et [entityTier] pour
// tout le reste.** C'était l'index de planche jusqu'à la 0.5.3, ce qui faisait décider
// l'ordre par celui dans lequel un jeu avait chargé ses images : relevé sur un niveau
// réel, un décal de sol posé au-dessus de son comblement disparaissait sur cent
// quarante-quatre cellules, et renommer le fichier de l'image suffisait à corriger
// l'affichage. La planche ne décide donc plus rien, et le rastériseur n'en souffre pas
// — il relit sa planche à chaque quad, sans garder d'état de l'un à l'autre.
type Key uint64

// Compose assemble une clé. Chaque champ est ramené à ce qu'il porte, donc un
// dépassement range la chose au bord plutôt que dans le champ voisin.
func Compose(layer, floor uint8, depth uint32, height uint16, tier uint8, index uint16) Key {
	return Key(min(uint64(layer), render.MaxLayer))<<layerShift |
		Key(min(uint64(floor), render.MaxFloor))<<floorShift |
		Key(min(uint64(depth), MaxDepth))<<depthShift |
		Key(uint64(height))<<heightShift |
		Key(min(uint64(tier), MaxTier))<<tierShift |
		Key(min(uint64(index), MaxIndex))
}

// WithLayer rend la même clé dans une autre couche, les cinq autres champs intacts.
//
// C'est ce qui permet à une chose révélée de repartir au-dessus de ce qui la cachait
// sans être reprojetée : sa profondeur, sa hauteur et son rang d'empilement sont ceux
// de la chose elle-même, et seule sa place dans l'empilement des couches change.
func (c Key) WithLayer(layer uint8) Key {
	const mask = Key(render.MaxLayer) << layerShift
	return c&^mask | Key(min(uint64(layer), render.MaxLayer))<<layerShift
}

// Index rend l'index stable, le seul champ que le départage ajuste.
func (c Key) Index() uint16 { return uint16(c) & MaxIndex }

// WithIndex rend la même clé portant un autre index stable.
//
// **Ce que le départage décide vit dans la liste des provenances qu'il arrange, et cette
// écriture n'en est qu'un enregistrement.** Rien ne la relit : le rangement suit cette
// liste, et [Key.Index] n'a aucun appelant de production. Une clé lue après coup dit donc
// le rang qu'elle a reçu, elle ne le porte pas.
//
// **Et cet enregistrement se garde, ce qui a été tranché le 31 octobre 2026.** Puisque
// personne ne le relit, le retirer gagnerait un passage sur les rangs de chaque groupe —
// sous le demi pour cent du tri, mesuré. Ce qui l'emporte est ce qu'il rend possible :
// c'est le seul endroit où le départage laisse une trace lisible, trois cas l'éprouvent
// contre le rang qu'ils attendent, et c'est par lui que le brief du jalon 4 a pu établir
// que la vérité de l'ordre était la liste des provenances. Une clé qui ne dirait rien de
// son groupe se déboguerait par `print`.
func (c Key) WithIndex(index uint16) Key {
	return c&^MaxIndex | Key(min(uint64(index), MaxIndex))
}

// Depth rend la somme des deux axes du monde, en quarts de cellule.
//
// C'est le champ par lequel la clé ordonne la profondeur, et il se relit pour
// comprendre une clé plutôt que pour trier : le départage n'en a pas besoin, n'ordonnant
// que des choses d'une même profondeur, et il se décide sur l'abscisse, le rang
// d'empilement puis l'identité.
func (c Key) Depth() uint32 {
	return uint32(c>>depthShift) & MaxDepth
}
