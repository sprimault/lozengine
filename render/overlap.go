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

	// Hidden est le nombre de pixels de la chose que recouvre ce qui se peint après
	// elle, décor et entités confondus.
	//
	// Le cumul compte deux fois ce que deux choses se recouvrant entre elles cachent
	// au même endroit. C'est la mesure qui sert aussi au seuil de révélation, et la
	// corriger demanderait de tenir la surface déjà couverte.
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
