// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package geometrie

import "math"

// Le format de la virgule fixe du moteur : seize bits de fraction sur un int32.
//
// Puissance de deux, donc une remise à l'échelle est un décalage et non une
// division, et la plage d'un int32 laisse ±32 768 cellules à la partie entière —
// exactement la limite de cases de côté que la clé de tri sait distinguer.
//
// Le format n'est pas qu'un choix de plage : c'est ce qui rend l'entrée flottante
// gratuite pour un consommateur qui tient déjà ses positions en virgule fixe au
// même format. L'aller-retour par un float64, qui porte 53 bits de mantisse contre
// 32, est alors exact. Vingt bits de fraction, ou un int64, et ce même consommateur
// perdrait un cran à chaque appel.
const (
	fractionBits = 16
	unite        = 1 << fractionBits
)

// quartBits mène du fixe aux quarts de cellule, l'unité dans laquelle la clé de tri
// compte la profondeur : deux bits de fraction au lieu de seize.
//
// C'est ce décalage qui garantit l'unicité de l'arrondi. La profondeur se dérive du
// fixe déjà calculé, et non d'une seconde conversion depuis le flottant : deux
// arrondis indépendants sur la même entrée peuvent diverger, et la divergence
// réordonnerait des quads sans que rien ne le signale.
const quartBits = fractionBits - 2

// inverseBits est le nombre de bits de fraction des coefficients de l'inverse, qui
// vivent sur un int64 et non sur un int32 comme ceux de l'aller.
//
// Seize n'y suffiraient pas, et c'est la tolérance annoncée qui l'impose : l'inverse
// d'une hauteur dérivée — celle de l'isométrie vraie, qui ne tombe pas sur un
// entier — s'y quantifierait à sept cent-millièmes près, soit près de deux cellules
// d'écart sur une ordonnée écran de cinq cent mille pixels. L'inexactitude de la
// matrice dominerait alors l'arrondi au pixel, qui vaut trois centièmes de cellule,
// alors que c'est lui qui doit rester le plancher.
//
// Vingt-quatre ramènent l'écart à quelques millièmes de cellule, bien sous ce
// plancher. Les prendre plus larges le ramènerait plus bas encore, mais chaque bit
// gagné raccourcit d'autant la plage de pixels que l'inverse peut accepter sans que
// le produit quitte un int64 — voir le champ plage d'une [Matrice], qui s'en déduit.
const inverseBits = 24

// quantifier convertit une coordonnée de monde en virgule fixe, en saturant.
//
// La saturation n'est pas une précaution : hors plage, la conversion d'un flottant
// vers un int32 est laissée à l'implémentation par la spécification du langage, et
// amd64 rend le plus petit entier là où arm64 sature au plus grand. Sans ces
// comparaisons, une entité posée hors plage rendrait des pixels différents selon la
// machine et toute la virgule fixe serait vaine.
//
// Le NaN se traite à part parce qu'aucune comparaison de borne ne l'attrape : il
// n'est ni supérieur ni inférieur à quoi que ce soit, et serait donc passé à une
// conversion dont le résultat est lui aussi laissé à l'implémentation.
//
// C'est le seul calcul flottant que le chemin de rendu traverse, et la
// multiplication y est seule sur sa ligne : elle doit le rester. Une addition au
// même endroit en ferait le motif « x·y + z » que la spécification autorise à
// fusionner en une instruction unique, pratiquée par arm64 et riscv64 mais pas par
// amd64 — et l'arrondi redeviendrait dépendant de la machine, ce que toute cette
// virgule fixe sert justement à empêcher.
func quantifier(v float64) int32 {
	f := math.Round(v * unite)

	if f != f {
		return 0
	}
	if f >= math.MaxInt32 {
		return math.MaxInt32
	}
	if f <= math.MinInt32 {
		return math.MinInt32
	}
	return int32(f)
}

// pixel ramène un produit en fixe à un pixel entier, arrondi au plus proche et
// saturé.
//
// Le biais précède le décalage parce qu'un décalage arithmétique arrondit vers
// moins l'infini : sans lui, un point à mi-pixel tomberait d'un côté au-dessus de
// l'origine et de l'autre en dessous, et une scène symétrique ne le serait plus.
//
// La saturation couvre le produit d'une coordonnée extrême par une grande cellule,
// qui atteint la plage d'un int32 bien avant d'être absurde : 32 768 cellules de
// 65 535 pixels y arrivent.
func pixel(fixe64 int64) int32 {
	arrondi := (fixe64 + 1<<(2*fractionBits-1)) >> (2 * fractionBits)

	if arrondi > math.MaxInt32 {
		return math.MaxInt32
	}
	if arrondi < math.MinInt32 {
		return math.MinInt32
	}
	return int32(arrondi)
}
