// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Package tri décide l'ordre de dessin : il écarte ce qui sort de l'image, compose
// une clé par chose à peindre, et range les quads sur ces clés.
//
// L'ordre tient tout entier dans un nombre. Comparer deux clés revient à comparer
// leurs couches, puis leurs étages à couche égale, et ainsi de suite jusqu'au dernier
// champ — sans une seule condition à écrire, et sans fonction de comparaison appelée
// des milliers de fois par image.
package tri

// La répartition des bits d'une clé, du plus fort au plus faible.
//
// L'ordre des champs est l'ordre de priorité, et c'est tout le procédé : ranger les
// clés par valeur croissante range les choses à peindre dans l'ordre où on les peint.
//
// Les largeurs ne sont pas négociables une fois qu'une image a été rendue avec : elles
// fixent ce que le moteur annonce savoir distinguer — seize couches, soixante-quatre
// étages, 32 768 cellules de côté.
const (
	bitsIndex      = 10
	bitsPlanche    = 10
	bitsHauteur    = 16
	bitsProfondeur = 18
	bitsEtage      = 6
	bitsCouche     = 4

	decalagePlanche    = bitsIndex
	decalageHauteur    = decalagePlanche + bitsPlanche
	decalageProfondeur = decalageHauteur + bitsHauteur
	decalageEtage      = decalageProfondeur + bitsProfondeur
	decalageCouche     = decalageEtage + bitsEtage
)

// Ce que chaque champ porte au plus. Dépasser ne déborde pas sur le voisin — la
// composition masque — mais range la chose ailleurs qu'où elle devrait aller.
const (
	IndexMax      = 1<<bitsIndex - 1
	PlancheMax    = 1<<bitsPlanche - 1
	HauteurMax    = 1<<bitsHauteur - 1
	ProfondeurMax = 1<<bitsProfondeur - 1
	EtageMax      = 1<<bitsEtage - 1
	CoucheMax     = 1<<bitsCouche - 1
)

// Cle porte l'ordre de dessin d'une chose à peindre, du champ le plus fort au plus
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
type Cle uint64

// Composer assemble une clé. Chaque champ est ramené à ce qu'il porte, donc un
// dépassement range la chose au bord plutôt que dans le champ voisin.
func Composer(couche, etage uint8, profondeur uint32, hauteur uint16, planche, index uint16) Cle {
	return Cle(min(uint64(couche), CoucheMax))<<decalageCouche |
		Cle(min(uint64(etage), EtageMax))<<decalageEtage |
		Cle(min(uint64(profondeur), ProfondeurMax))<<decalageProfondeur |
		Cle(uint64(hauteur))<<decalageHauteur |
		Cle(min(uint64(planche), PlancheMax))<<decalagePlanche |
		Cle(min(uint64(index), IndexMax))
}

// Index rend l'index stable, le seul champ que le départage ajuste.
func (c Cle) Index() uint16 { return uint16(c) & IndexMax }

// AvecIndex rend la même clé portant un autre index stable.
//
// C'est par là que passe tout ce que le départage a décidé : la clé reste la seule
// vérité de l'ordre de dessin, et un rang qui vivrait ailleurs se perdrait au moment
// du rangement.
func (c Cle) AvecIndex(index uint16) Cle {
	return c&^IndexMax | Cle(min(uint64(index), IndexMax))
}

// Profondeur rend la somme des deux axes du monde, en quarts de cellule.
//
// Elle sert au départage, qui n'ordonne que des choses d'une même profondeur :
// au-delà, la clé a déjà tranché et il n'a rien à dire.
func (c Cle) Profondeur() uint32 {
	return uint32(c>>decalageProfondeur) & ProfondeurMax
}
