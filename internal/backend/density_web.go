// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La taille qu'un canvas prend dans la boîte que la page lui laisse, aux deux unités
// qu'une page distingue. Le fichier n'importe pas `syscall/js`, pour que ses cas
// tournent sur l'hôte.

package backend

import (
	"math"

	"github.com/sprimault/lozengine/render"
)

// canvasBox est la taille d'un canvas sous ses deux unités, qui cessent d'être la même
// chose dès que l'écran a une densité.
//
// **Les pixels d'image sont ce que le moteur dessine** : c'est la taille que les
// attributs `width` et `height` de l'élément reçoivent, et celle que la méthode `Size`
// d'une fenêtre annonce. **Les pixels CSS sont ce que la page met en page** : c'est la
// taille que `style.width` et `style.height` reçoivent, et l'unité dans laquelle un
// événement de pointeur arrive.
//
// Les deux décrivent un seul rectangle, et c'est pourquoi elles voyagent ensemble :
// posées séparément, elles se désaccordent à la première densité non entière, et la
// page met alors le canvas à l'échelle d'une fraction.
type canvasBox struct {
	// width et height sont la taille en pixels d'image, celle du tampon que le moteur
	// remplit, bornée par [render.MaxBufferSide].
	width, height uint16

	// cssWidth et cssHeight sont la taille à poser en pixels CSS. Elles se dérivent
	// des deux précédentes et jamais l'inverse, ce que la godoc de [canvasFor]
	// explique.
	cssWidth, cssHeight float64
}

// integerRatio est l'écart sous lequel une densité se tient pour entière.
//
// Une page compose sa densité d'un produit — l'échelle du système par le zoom de la
// page —, donc une valeur qui vaut trois peut arriver à quelques quanta de trois. La
// tenir alors pour fractionnaire ferait changer de régime, donc changer la taille
// apparente du jeu, pour un écart que rien ne permet de voir.
//
// Le seuil est à sept ordres de grandeur de ce qu'il pourrait capturer par erreur : la
// densité fractionnaire la plus proche d'un entier que les écrans servent est 2,625,
// soit 0,375 d'un entier, et un cran de zoom en vaut cinq centièmes.
const integerRatio = 1e-9

// canvasFor décide la taille d'un canvas depuis la boîte que la page lui laisse, en
// pixels CSS, et la densité de l'écran.
//
// # Deux régimes, et c'est l'échelle entière qui les impose
//
// La taille apparente d'un pixel de jeu vaut le facteur d'agrandissement divisé par la
// densité. La garder constante demanderait donc un facteur égal à la densité multipliée
// par ce que ce facteur vaut à densité un — un nombre qui n'est pas entier dès que la
// densité ne l'est pas. Aucune forme ne fond donc les deux cas :
//
//   - **densité entière**, celle de tous les téléphones et des écrans dits fins : le
//     canvas prend la taille de la boîte en pixels CSS, et c'est la page qui l'étire,
//     d'un facteur exactement égal à la densité. Le jeu voit alors ce qu'il verrait à
//     densité un, un événement de pointeur arrive déjà dans la bonne unité, et le
//     tampon à présenter est densité fois densité plus petit ;
//   - **densité fractionnaire**, le 1,25 et le 1,5 des écrans de bureau : le canvas
//     prend la taille de la boîte en pixels d'appareil, et la taille CSS s'en déduit
//     pour que la page ne l'étire pas du tout. La taille apparente varie alors avec la
//     densité, et c'est le régime d'une fenêtre de système, qui ne fait pas autrement.
//
// Tout le choix tient dans le nombre de pixels d'image que vaut un pixel CSS : un dans
// le premier régime, la densité dans le second. Le reste de l'arithmétique est commun.
//
// # La taille CSS se dérive de la taille en pixels d'image, jamais l'inverse
//
// C'est ce qui rend le rapport des deux unités exact, et c'est le seul degré de liberté
// qu'une page laisse. La boîte est fractionnaire, la densité aussi : poser la taille CSS
// d'abord et en déduire des pixels d'appareil rendrait un rapport comme 1,0005, c'est-à-
// dire la mise à l'échelle fractionnaire que le moteur s'interdit — et que les
// navigateurs ne rendent même pas de la même façon.
//
// Ce que la troncature laisse est une marge de moins d'un pixel CSS par axe. Elle est
// aux pixels CSS ce que les bandes sont aux pixels : le prix de l'échelle entière.
//
// # Ce qu'elle ne fait pas
//
// Elle ne connaît pas la résolution interne du jeu, donc ni le minimum ni l'aire de
// monde. La surface se dimensionne plus haut, sur la taille qu'une fenêtre annonce, et
// un canvas plus petit que ce minimum fait rogner l'image comme le fait une fenêtre
// trop petite.
//
// Une boîte ou une densité qu'une page ne sait pas décrire ne fait pas échouer l'appel.
// Une densité nulle, négative, infinie ou indéfinie vaut un ; une boîte de cette sorte
// rend un côté nul, qui est ce qu'une page rend d'un élément masqué et ce que la boucle
// traite déjà comme une fenêtre réduite.
func canvasFor(boxWidth, boxHeight, ratio float64) canvasBox {
	if !(ratio > 0) || math.IsInf(ratio, 0) {
		ratio = 1
	}

	perCSS := ratio
	if nearest := math.Round(ratio); nearest >= 1 && math.Abs(ratio-nearest) < integerRatio {
		perCSS = 1
	}

	width, cssWidth := sideFor(boxWidth, perCSS)
	height, cssHeight := sideFor(boxHeight, perCSS)
	return canvasBox{width: width, height: height, cssWidth: cssWidth, cssHeight: cssHeight}
}

// sideFor porte un côté de la boîte aux deux unités, perCSS disant combien de pixels
// d'image vaut un pixel CSS.
//
// Une fonction plutôt que deux fois la même expression : la largeur et la hauteur
// doivent suivre la même convention de troncature, faute de quoi le canvas se déformerait
// d'un pixel sur un seul axe — ce qu'on verrait sans savoir d'où ça vient.
//
// La borne s'applique avant la division, pour que la taille CSS reste celle que le canvas
// porte vraiment. Bornée après, elle décrirait un rectangle que l'élément n'a pas.
func sideFor(box, perCSS float64) (uint16, float64) {
	if !(box > 0) {
		return 0, 0
	}

	side := math.Floor(box * perCSS)
	if side > render.MaxBufferSide {
		side = render.MaxBufferSide
	}
	return uint16(side), side / perCSS
}
