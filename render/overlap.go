// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le relevé de recouvrement : combien de pixels d'une chose suivie se font cacher, et
// par quoi. C'est la mesure que le tri fait déjà pour décider d'une révélation, rendue
// à qui veut dessiner son propre rappel.

package render

// Overlap dit, pour une chose suivie, combien de ses pixels se font recouvrir par ce
// qui se peint après elle.
//
// **La mesure existait et ne sortait pas.** Le tri la fait pour décider si une chose
// marquée se redessine, en pixels et par origine ; un hôte dont le rappel est d'une
// autre nature — un contour, une flèche, une silhouette tirée d'un autre dessin — devait
// la refaire en entier. Il la reçoit maintenant, et choisit son seuil comme la forme de
// son rappel.
//
// Disposition compatible C, comme [Quad] : douze octets, champs de taille fixe, aucun
// pointeur. C'est ce qui permet à un hôte de lire le tableau sans conversion.
type Overlap struct {
	// Identity est la poignée de l'entité relevée, celle que sa pose a rendue.
	//
	// Un entier et non le type de poignée du paquet de scène, que ce paquet-ci ne
	// connaît pas : la poignée en est un, et la conversion est donc gratuite.
	Identity uint32

	// Hidden est une mesure de recouvrement, pas une aire : un hôte la compare à un
	// seuil qu'il règle sur son jeu, décor et entités confondus.
	//
	// Elle additionne, quad après quad, le rectangle que ce qui se peint après la
	// chose partage avec le sien, transparent compris : deux choses superposées
	// versent chacune leur part, et un quad à moitié hors de l'image verse tout de
	// même le rectangle entier qu'il partage. Ce ne sont donc pas des pixels de la
	// chose, mais des pixels de boîte — une case de personnage de soixante-quatre sur
	// soixante-quatre n'en peint en moyenne que quatre cent quarante.
	//
	// **C'est ce qu'elle fait dans cette version, et il ne faut pas en dépendre.**
	// Deux choses seulement tiennent d'une version à l'autre : ajouter un cachant ne
	// la diminue jamais, et elle est nulle quand rien ne se peint après la chose. En
	// particulier, ne pas retrancher de ce nombre ce qu'un rappel y a versé —
	// [scene.Aspect.Uncounted] l'exclut à la source, et c'est la voie prévue pour
	// cela. La soustraction marche aujourd'hui parce que la loi est additive ; une
	// mesure exacte ne compterait chaque pixel qu'une fois et la rendrait fausse.
	Hidden uint32

	// ByDecor est la part de [Overlap.Hidden] que le décor seul recouvre.
	//
	// Les deux origines ne portent pas la même information : ce qu'un mur cache, il le
	// cache parce que la chose est derrière lui, et un rappel y retire cet indice ; une
	// créature qui s'interpose ne dit rien de tel. Un hôte qui veut les traiter
	// différemment a ici de quoi.
	ByDecor uint32
}

// OverlapList est le relevé d'une image, dans l'ordre où les choses ont été posées.
//
// Un tableau nommé et non une structure qui en porterait un, pour la raison qui vaut
// pour [DrawList] : un hôte en reçoit le pointeur du premier élément avec leur nombre,
// sans conversion ni propriété mémoire à négocier.
type OverlapList []Overlap
