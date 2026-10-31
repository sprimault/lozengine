// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La matrice et ses quatre constructeurs. C'est le seul endroit du moteur où une
// constante de ratio a le droit d'apparaître : ailleurs, une division par deux
// écrite à la main serait une seconde vérité sur la forme d'une cellule.

package geometry

import "math"

// Matrix est une projection du monde vers l'écran, avec son inverse gardée.
//
// Elle s'obtient d'un constructeur et s'emploie par [Matrix.Project] et
// [Matrix.Unproject] ; ses coefficients ne se lisent pas. C'est ce qui lui permet de
// rester interne tout en étant republiée par alias à la racine : un type qu'on
// construit par fonction et qu'on lit par méthode n'a pas besoin de montrer ses
// champs, contrairement à ceux qu'un hôte remplit lui-même.
//
// Sa valeur zéro ne projette rien d'utile — tout point y tombe sur l'origine et
// l'inverse est dégénérée. Un constructeur est obligatoire, et c'est l'échec visible
// plutôt que l'image subtilement fausse.
//
// **Et c'est aussi ce qu'un constructeur rend quand il ne peut pas faire mieux** :
// des coefficients dont l'inverse ne tient pas dans les entiers du moteur, une
// projection qui écrase le monde sur une droite, ou une ordonnée dont la somme
// quitterait l'`int64` où elle se calcule. Le même repli dans les trois cas, et pour
// la même raison — un point ramené sur l'origine se voit, une image repliée d'un
// facteur que personne ne peut rattraper, non.
type Matrix struct {
	// ex = a·x + b·y, et ey = c·x + d·y − floor·z. Coefficients et coordonnées sont
	// en virgule fixe, donc le produit de deux d'entre eux porte deux fois les bits
	// de fraction et se ramène au pixel par un seul décalage.
	a, b, c, d, floor int32

	// L'inverse, calculée à la construction parce qu'on s'en sert à chaque image pour
	// savoir quelles cellules l'écran peut montrer, et que son calcul est une
	// division. En entiers comme l'aller : c'est elle qui décide quelles cellules sont
	// relevées, donc un arrondi variant d'une architecture à l'autre y changerait
	// l'image et non seulement un chiffre.
	ia, ib, ic, id int64

	// La plus grande entrée que l'inverse accepte avant que le produit par un de ses
	// coefficients quitte un int64. Elle ne peut pas être une constante : une
	// cellule minuscule donne de grands coefficients et donc une span courte.
	span int32
}

// Dimetric rend la projection d'une cellule losange au ratio libre, celle d'un
// rendu isométrique de jeu : une tuile de 32 sur 16 avance de seize pixels à
// l'horizontale et de huit vers le bas, par cellule.
//
// **Le sens, que l'avance en pixels ne dit pas : `x` s'enfonce vers le bas-gauche de
// l'écran et `y` vers le bas-droite.** Un pas en `x` retranche donc ses seize pixels
// à la position horizontale au lieu de les ajouter, et un jeu qui attend son premier
// axe vers la droite obtient son monde en miroir. Monter à l'écran demande de reculer
// sur les deux axes à la fois.
//
// Deux façons de ne pas en dépendre : [Matrix.Unproject] rend le pas du monde qui
// répond à un déplacement d'écran, sans qu'on ait à connaître ce sens ; et [Raw]
// exprime la convention inverse, au prix de la perdre pour tout le reste — le
// départage de l'ordre de dessin et le relevé de recouvrement supposent celle-ci.
//
// C'est le constructeur du cas courant, et le seul qui laisse la hauteur libre.
func Dimetric(width, height int) Matrix {
	halfW, halfH := float64(width)/2, float64(height)/2
	return matrix(-halfW, halfW, halfH, halfH)
}

// Isometric rend la projection de l'isométrie vraie, dont elle dérive la hauteur
// par largeur sur racine de trois — dix-huit pixels et demi pour une largeur de
// trente-deux, là où le dimétrique courant en prend seize.
//
// **Elle tourne les axes dans le même sens que [Dimetric]** : `x` vers le bas-gauche
// de l'écran, `y` vers le bas-droite. Seul le ratio les distingue.
//
// La hauteur n'est pas un paramètre parce qu'un constructeur qui n'impose rien n'en
// est pas un : avec elle, celui-ci serait [Dimetric] sous un autre nom, et aucun
// cas de test ne pourrait plus rougir d'une erreur de ratio. La racine carrée est à
// arrondi correct comme la division, et ne s'évalue qu'ici.
func Isometric(width int) Matrix {
	height := float64(width) / math.Sqrt(3)
	halfW, halfH := float64(width)/2, height/2
	return matrix(-halfW, halfW, halfH, halfH)
}

// Grid rend la projection de la vue de dessus, où une cellule est un carré et non
// un losange.
//
// Elle existe parce que [Dimetric] ne peut pas l'exprimer, quel que soit son
// ratio : une vue de dessus ne fait pas tourner les axes. La projection étant une
// matrice, c'est un constructeur de plus et non un second moteur.
//
// **Les axes ne tournant pas, ils ne vont pas où les deux losanges les mettent** :
// `x` va vers la droite de l'écran et `y` vers le bas, donc le premier à l'inverse
// de [Dimetric]. Un jeu qui passe d'une projection à l'autre voit son monde se
// retourner sur ce seul axe.
func Grid(side int) Matrix {
	c := float64(side)
	return matrix(c, 0, 0, c)
}

// Raw rend la projection de coefficients donnés, pour ce qu'aucun des trois autres
// constructeurs n'exprime.
//
// Les quatre coefficients sont ceux d'une application linéaire : ex = a·x + b·y et
// ey = c·x + d·y. Pas de translation, parce que la scène retranche l'origine de la
// vue et qu'aucun appelant n'en a l'usage.
//
// **C'est le seul constructeur qui puisse rompre le sens des axes**, et c'est ici que
// l'avertissement manquait : [Dimetric] et [Isometric] le portent, eux qui l'imposent.
// La convention est `x` vers le bas-gauche de l'écran et `y` vers le bas-droite sous
// les deux losanges, `x` vers la droite et `y` vers le bas sous [Grid].
//
// Trois mécanismes la supposent et aucun ne se règle : le départage de l'ordre de
// dessin, le relevé de recouvrement, et la table de directions de `appearance.Turn`.
// Chacun qui s'y adosse ajoute un site, donc le coût d'un rétablissement croît — la
// rompre se fait les yeux ouverts, pas en découvrant son monde retourné.
func Raw(a, b, c, d float64) Matrix {
	return matrix(a, b, c, d)
}

// WithFloorHeight change le facteur par lequel z monte à l'écran et rend la matrice
// modifiée, l'originale restant intacte.
//
// Ce facteur vit dans la matrice et non chez l'appelant : sinon chacun appliquerait
// lui-même la soustraction et le calcul se disséminerait. Il vaut par défaut la
// hauteur qu'une cellule occupe à l'écran, le cas d'un cube dont la face du dessus
// couvre exactement le sol de sa case.
//
// L'inverse n'est pas recalculée : elle ne porte que le plan des deux premiers axes,
// où z n'intervient pas.
//
// **Une hauteur que l'ordonnée ne peut pas porter rend la matrice nulle**, comme au
// montage : ce champ entre dans la somme que [Matrix.Project] calcule, donc l'écraser
// sans revérifier rouvrirait le débordement qu'un constructeur venait de refuser.
//
// **Aucune hauteur ne l'atteint aujourd'hui, et c'est mesuré** — mais il n'y a pas un
// bit de marge, ce qui est la raison de garder le contrôle. Une matrice acceptée a
// `|c|+|d| ≤ 2^31 − 1`, puisque son `floor` vaut cette même somme et que les deux
// doivent tenir sous [maxOrdinateSum] ; et [quantize] sature `floor` à `2^31 − 1`.
// Leur somme vaut donc `2^32 − 2` au pire, soit **exactement** le seuil. La propriété
// tient par la rencontre de trois valeurs qu'on peut changer séparément, et un cas
// témoin du paquet rougit si l'une bouge.
func (m Matrix) WithFloorHeight(height int) Matrix {
	m.floor = quantize(float64(height))
	if !m.ordinateFits() {
		return Matrix{}
	}
	return m
}

// matrix quantifie des coefficients et calcule l'inverse qui va avec.
//
// Le facteur d'étage s'y déduit au lieu d'être passé : la hauteur qu'une cellule
// occupe à l'écran est celle de l'empreinte de son côté, soit la somme des valeurs
// absolues des deux coefficients d'ordonnée. La formule rend seize pour une tuile de
// 32 sur 16, le côté pour une grille, et la hauteur dérivée pour l'isométrie —
// chacun étant bien le défaut que son constructeur annonce.
func matrix(a, b, c, d float64) Matrix {
	m := Matrix{
		a: quantize(a), b: quantize(b),
		c: quantize(c), d: quantize(d),
		floor: quantize(math.Abs(c) + math.Abs(d)),
	}

	// **La matrice nulle et non `m`**, à la différence du déterminant nul juste en
	// dessous : ici ce sont les coefficients directs qui débordent, donc les garder
	// laisserait [Matrix.Project] replier à chaque point.
	if !m.ordinateFits() {
		return Matrix{}
	}

	// Le déterminant se calcule sur les coefficients déjà entiers, et non sur les
	// flottants reçus : « a·d − b·c » est exactement le motif qu'une architecture
	// peut fusionner et une autre pas. En entiers, le produit est exact et la
	// question ne se pose plus.
	det := int64(m.a)*int64(m.d) - int64(m.b)*int64(m.c)
	if det == 0 {
		return m
	}

	ia, okA := inverseCoefficient(m.d, det)
	ib, okB := inverseCoefficient(m.b, det)
	ic, okC := inverseCoefficient(m.c, det)
	id, okD := inverseCoefficient(m.a, det)

	// Même repli que le déterminant nul, et pour la même raison : un inverse saturé
	// serait faux d'un facteur que personne ne peut rattraper, là qu'une inverse nulle
	// ramène tout point sur l'origine, ce qui se voit.
	if !okA || !okB || !okC || !okD {
		return m
	}

	m.ia, m.ib = ia, -ib
	m.ic, m.id = -ic, id

	// Trois parts et non deux : l'inverse additionne deux produits et le biais
	// d'arrondi, et il reste ainsi de quoi les porter tous les trois. La division
	// précède la multiplication, sans quoi le produit déborderait l'int64 qu'il sert à
	// mesurer et la plage en sortirait négative — donc inversant le signe de chaque
	// pixel au lieu de le borner.
	if worst := maxAbs(m.ia, m.ib, m.ic, m.id); worst > 0 {
		m.span = int32(min(math.MaxInt64/worst/3, math.MaxInt32))
	}
	return m
}

// maxOrdinateSum est la plus grande somme de coefficients d'ordonnée que
// [Matrix.Project] peut porter : `2^32 − 2`.
//
// Les deux retraits ne sont pas une marge de confort, ils viennent du pire cas. La
// coordonnée extrême est `math.MinInt32`, dont la valeur absolue vaut `2^31` et non
// `2^31 − 1` ; et [pixel] ajoute encore son biais d'arrondi de `2^31` avant de
// décaler. À `2^32 − 1`, la somme atteint exactement `2^63` et se replie.
//
// L'échelle à retenir : `Dimetric(32, 16)` somme deux millions, soit un deux-millième
// du seuil, qui ne se croise qu'à partir d'une tuile de 32 768 pixels de haut — là où
// `floor` sature de son côté.
const maxOrdinateSum = int64(1)<<32 - 2

// ordinateFits dit si la somme de l'ordonnée tient dans l'`int64` où
// [Matrix.Project] la calcule.
//
// Trois produits s'y ajoutent — `c·x`, `d·y` et `floor·z` —, chacun d'un coefficient
// par une coordonnée que [quantize] a pu saturer. Au-delà du seuil, l'addition se
// replie et rend un pixel du **signe opposé** : la chose part à l'autre bout d'un
// écran qu'elle aurait dû quitter par le bord d'en face. Rien ne la trahit, puisque le
// quad est écarté dans les deux cas et que l'image reste identique.
//
// **Une méthode plutôt qu'une expression recopiée**, et la règle de trois ne s'y
// oppose pas pour deux appelants : le défaut corrigé ici est précisément qu'un second
// site écrivait `floor` sans revérifier. Un troisième viendra, et il trouvera le
// contrôle sous la forme d'une question à poser à la matrice.
func (m Matrix) ordinateFits() bool {
	return abs(int64(m.c))+abs(int64(m.d))+abs(int64(m.floor)) <= maxOrdinateSum
}

// abs rend la valeur absolue d'un entier large.
//
// Sur `int64` et non sur `int32`, parce que le seul cas qui compte est celui où la
// valeur absolue ne tient pas dans son type d'origine : [quantize] peut rendre
// `math.MinInt32`, dont l'opposé n'est pas un `int32` et se replierait sur lui-même.
// Convertir d'abord, prendre l'absolu ensuite, est donc la seule forme juste — et
// `math.MinInt64` garde le même défaut d'un cran plus haut, hors d'atteinte de tout
// appelant du paquet, qui ne lui passent que des `int32` élargis.
func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// maxAbs rend la plus grande valeur absolue d'un jeu de coefficients.
func maxAbs(values ...int64) int64 {
	var worst int64
	for _, v := range values {
		worst = max(worst, abs(v))
	}
	return worst
}

// inverseCoefficient met un cofacteur à l'échelle de l'inverse et le divise par le
// déterminant, et dit si le résultat tient dans un int64.
//
// L'échelle porte trois jeux de bits de fraction, et pas deux : ceux du cofacteur,
// ceux que le déterminant a reçus en multipliant deux coefficients, et ceux de
// l'inverse elle-même. Le produit d'un coefficient par un pixel rend alors
// directement du fixe, une fois décalé de inverseBits.
//
// Le numérateur reste exact en flottant — un cofacteur ne porte qu'une vingtaine de
// bits significatifs contre les cinquante-trois d'un float64 — et la division qui
// suit est à arrondi correct, donc identique sur toutes les architectures.
//
// **Il y a une troisième conversion et elle manquait ici : `float64(det)`.** Le
// déterminant est un `int64`, donc il peut porter jusqu'à soixante-trois bits
// significatifs là où un `float64` en garantit cinquante-trois. Ce qui tient alors le
// déterminisme n'est pas l'exactitude mais le **mode d'arrondi** : la conversion d'un
// entier vers un flottant arrondit au plus proche sur toutes les architectures que
// `CIBLES` compile, `js/wasm` et `386` comprises — aucune n'a de précision étendue
// depuis que `GO386` vaut `sse2`. La conclusion de ce paragraphe tenait donc, mais pas
// par l'argument qu'il donnait.
//
// **Et aucune matrice du moteur n'atteint ce régime**, mesuré le 31 octobre 2026 : les
// coefficients sortent de [quantize], donc ils sont des multiples de 2^15 ou 2^16, et
// leurs produits restent exactement représentables très au-delà de 2^53. Au plafond,
// `Dimetric(32766, 32766)` rend un déterminant de l'ordre de 2^61 et la conversion est
// exacte. Ce qui compte est donc l'arrondi, pas un seuil de magnitude.
//
// **Le second retour se lit avant la conversion, et c'est lui qui tient le
// déterminisme** : hors plage, convertir un flottant vers un entier rend un résultat
// que la spécification du langage laisse à l'implémentation, et amd64 n'y choisit pas
// la même valeur qu'arm64. Le seuil est un rapport de cent vingt-huit entre le
// cofacteur et le déterminant, soit une matrice presque singulière à grands
// coefficients.
func inverseCoefficient(cofactor int32, det int64) (int64, bool) {
	const scale = 1 << (2*fractionBits + inverseBits)

	v := math.Round(float64(cofactor) * scale / float64(det))

	if v >= math.MaxInt64 || v <= math.MinInt64 {
		return 0, false
	}
	return int64(v), true
}
