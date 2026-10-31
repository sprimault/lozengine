// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La virgule fixe : seize bits de fraction sur un entier de trente-deux, et les deux
// conversions qui entrent et sortent du format. C'est ce qui rend une même scène
// identique au pixel près d'une architecture à l'autre, là où un flottant dépendrait
// de la façon dont la machine fusionne un produit et une somme.

package geometry

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
	unit         = 1 << fractionBits
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
// le produit quitte un int64 — voir le champ plage d'une [Matrix], qui s'en déduit.
const inverseBits = 24

// quantize convertit une coordonnée de monde en virgule fixe, en saturant.
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
// C'est le seul calcul flottant que le chemin de rendu traverse, et ce qui le rend sûr
// est qu'il n'y a **aucune addition à fusionner** avec la multiplication — non le fait
// qu'elle soit seule sur sa ligne. La spécification autorise la fusion « possibly
// across statements », donc isoler une ligne ne protège de rien, et seule une
// conversion explicite vers float64 force l'arrondi. Une addition introduite ici, où
// qu'elle s'écrive, en ferait le motif « x·y + z » qu'arm64 et riscv64 fusionnent en
// une instruction unique et qu'amd64 ne fusionne pas — et l'arrondi redeviendrait
// dépendant de la machine, ce que toute cette virgule fixe sert à empêcher.
func quantize(v float64) int32 {
	f := math.Round(v * unit)

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

// saturate ramène un entier large dans la plage d'un type plus étroit, en bornant
// plutôt qu'en repliant.
//
// Trois conversions du paquet ont cette forme au bit près — [pixel], depthOf et
// heightOf —, et c'est ce compte qui autorise à les réunir plutôt qu'à le devancer.
// Ce qu'elles gardent toutes les trois : une conversion qui déborde au lieu de saturer
// rend un résultat que la spécification du langage laisse à l'implémentation, donc
// dépendant de la machine, et c'est ce que toute la virgule fixe sert à empêcher.
//
// Les bornes sont des paramètres parce qu'un type-paramètre ne donne pas les siennes :
// `math.MaxInt32` ne s'atteint pas depuis T. Elles s'écrivent donc au site d'appel, où
// elles se relisent à côté du type qu'elles bornent.
//
// **Quatre autres conversions du paquet restent dehors, chacune pour un motif propre**,
// et les réunir demanderait de les tordre : quantize traite le NaN, qu'aucune
// comparaison de borne n'attrape ; [Matrix.clamp] borne contre un champ de la matrice
// et non contre la plage d'un type ; la plage de l'inverse est déjà une expression
// unique par le `min` natif ; et inverseCoefficient **refuse** au lieu de saturer, un
// inverse hors plage n'ayant aucune valeur raisonnable à rendre.
func saturate[T int32 | uint32 | uint16](v, lo, hi int64) T {
	if v < lo {
		return T(lo)
	}
	if v > hi {
		return T(hi)
	}
	return T(v)
}

// pixel ramène un produit en fixe à un pixel entier, arrondi au plus proche et
// saturé.
//
// Le biais précède le décalage parce qu'un décalage arithmétique arrondit vers
// moins l'infini : sans lui, un point à mi-pixel tomberait d'un côté au-dessus de
// l'origine et de l'autre en dessous, et une scène symétrique ne le serait plus.
//
// **La saturation est le repli sûr d'une conversion, pas un contrôle de plage**, et
// l'annoncer comme tel a trompé : ce qui garde vraiment ce calcul est que la somme
// reçue tienne dans son `int64`, ce qu'une matrice vérifie à son montage par
// `Matrix.ordinateFits`. Sans cette garde, l'addition se repliait **en amont** et la
// saturation ne voyait plus qu'une valeur déjà fausse, du signe opposé. Ses deux
// bornes sont par ailleurs inatteignables, et un cas du paquet le démontre.
func pixel(fixed64 int64) int32 {
	rounded := (fixed64 + 1<<(2*fractionBits-1)) >> (2 * fractionBits)

	return saturate[int32](rounded, math.MinInt32, math.MaxInt32)
}
