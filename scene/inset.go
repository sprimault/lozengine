// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La part de la surface de dessin que le monde n'occupe pas, déclarée côté par côté :
// le complément du plafond, pour un jeu qui veut une marge fixe plutôt qu'une aire
// fixe, et ce que la surface laisse au monde une fois ces côtés retirés.

package scene

// Inset est la marge que le monde ne traverse pas, en pixels de la surface de dessin et
// côté par côté.
//
// **C'est le complément du plafond de l'aire de monde, et les deux ne disent pas la
// même chose.** Le plafond fixe l'aire et laisse la marge varier avec la fenêtre ; un
// retrait fixe la marge et laisse l'aire prendre le reste. Un jeu qui loge son interface
// dans la marge veut le second : un bandeau de soixante-dix-sept pixels doit rester à
// soixante-dix-sept pixels du jeu quelle que soit la fenêtre, là qu'une marge qui vaut
// soixante-dix-sept à l'ouverture et deux cent vingt-et-un une fois la fenêtre agrandie
// l'oblige à flotter loin du jeu ou à changer de place selon la taille. Un jeu qui veut
// que le joueur voie toujours autant de monde veut le premier, et c'est une donnée de
// jeu que la fenêtre n'a pas à décider.
//
// **Les deux se composent, dans cet ordre** : le retrait sort de la surface ce qui
// n'est pas au monde, le plafond borne ensuite le monde dans ce qui reste, et
// [Alignment] place l'aire dans ce que le plafond a laissé en trop. Un jeu qui
// n'emploie qu'un des deux n'a pas à connaître l'autre : des retraits seuls donnent une
// marge constante, un plafond seul une aire constante.
//
// **Les axes sont ceux de la surface**, comme pour [Alignment] : `Top` est le haut de
// l'image, et [Scene.Orient] ne le fait pas tourner.
//
// C'est aussi par là que s'expriment deux marges **inégales** sur le même axe — les
// deux barres latérales d'un jeu de rôle classique —, que la godoc d'[Alignment] a
// portées comme un manque assumé jusqu'au 30 octobre 2026. L'alignement ne les remplace
// pas pour autant, et l'inverse non plus : un retrait retire du chrome, un alignement
// place une aire dont la taille est déjà décidée.
type Inset struct {
	Left, Right, Top, Bottom int32
}

// Available rend ce que la surface laisse au monde sur chaque axe, les retraits déduits.
//
// **C'est la seule arithmétique de la composition, et elle est publique pour cela** :
// la scène s'en sert pour placer l'aire, la boucle du paquet racine pour la
// dimensionner, et un hôte qui dispose lui-même ses deux rectangles y lit la même
// chose. Deux calculs qui doivent s'accorder finissent par diverger.
//
// Un pixel au minimum, jamais zéro : des retraits plus larges que la surface décrivent
// une mise en page impossible, mais ils arrivent par une fenêtre devenue trop petite et
// non par une option mal posée, donc le cas n'a personne à qui se plaindre. Un monde
// d'un pixel se voit, là qu'un monde nul rendrait une image vide.
func (i Inset) Available(width, height int32) (int32, int32) {
	i = i.sane()
	return max(width-i.Left-i.Right, 1), max(height-i.Top-i.Bottom, 1)
}

// sane rend le retrait avec ses côtés négatifs ramenés à zéro.
//
// Un côté négatif décrirait une aire **plus grande** que la surface, donc des quads
// placés hors du tampon et une conversion qui finit par déborder ses seize bits. Rien
// ne le signalerait : le rendu serait simplement faux sur les bords. Ramener plutôt que
// refuser garde le chemin sans erreur, et zéro est la seule valeur qui ne change rien.
func (i Inset) sane() Inset {
	return Inset{
		Left:   max(i.Left, 0),
		Right:  max(i.Right, 0),
		Top:    max(i.Top, 0),
		Bottom: max(i.Bottom, 0),
	}
}

// before rend les retraits qui précèdent l'aire sur chaque axe — gauche et haut —,
// bornés à ce que la surface laisse à l'aire.
//
// La borne sert le cas que [Inset.Available] a déjà ramené à un pixel : avec des
// retraits plus larges que la surface, le retrait de gauche seul poserait l'aire au-delà
// du bord droit du tampon. La borner la garde dedans, et l'image reste celle d'une mise
// en page trop serrée plutôt que celle d'un monde disparu.
func (i Inset) before(width, height int32) (x, y int32) {
	availableWidth, availableHeight := i.Available(width, height)

	i = i.sane()
	return min(i.Left, width-availableWidth), min(i.Top, height-availableHeight)
}
