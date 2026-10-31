// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le relevé de recouvrement : ce qui se fait cacher d'une chose suivie, et par quoi.
// C'est la mesure que le tri fait déjà pour décider d'une révélation, rendue à qui veut
// dessiner son propre rappel.
//
// Deux échelles, et le type dit laquelle chaque champ porte : une somme de rectangles
// de boîte, et un compte de pixels réellement peints. Ne jamais écrire l'une sans
// l'autre — « combien de ses pixels » désigne la seconde et a longtemps servi à
// annoncer la première.

package render

// Overlap dit, pour une chose suivie, ce que recouvre d'elle ce qui se peint après
// elle — dans deux échelles que ses champs distinguent : une somme de rectangles de
// boîte, et un compte de pixels réellement peints.
//
// **La mesure existait et ne sortait pas.** Le tri la fait pour décider si une chose
// marquée se redessine, en pixels et par origine ; un hôte dont le rappel est d'une
// autre nature — un contour, une flèche, une silhouette tirée d'un autre dessin — devait
// la refaire en entier. Il la reçoit maintenant, et choisit son seuil comme la forme de
// son rappel.
//
// Disposition compatible C, comme [Quad] : vingt-quatre octets, champs de taille fixe,
// aucun pointeur. C'est ce qui permet à un hôte de lire le tableau sans conversion, et
// un cas de ce paquet fige la taille comme les six décalages.
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
	// **« Transparent compris » porte sur les pixels d'un rectangle, pas sur un quad
	// qui ne peint rien**, et la nuance décide d'un seuil. Un sprite dont la planche a
	// des bords transparents verse bien tout son rectangle ; un quad dont la teinte a
	// un alpha nul n'écrit aucun pixel et ne verse rien du tout. Un rappel qu'un jeu
	// éteint par sa teinte ne franchit donc aucun seuil, là qu'il le faisait jusqu'à
	// cette version. Un quad additif, lui, compte — il peut cacher en saturant au
	// blanc, et le champ `Uncounted` d'un aspect est la voie pour l'exclure.
	//
	// **C'est ce qu'elle fait dans cette version, et il ne faut pas en dépendre.**
	// Deux choses seulement tiennent d'une version à l'autre : ajouter un cachant ne
	// la diminue jamais, et elle est nulle quand rien ne se peint après la chose. En
	// particulier, ne pas retrancher de ce nombre ce qu'un rappel y a versé —
	// le champ `Uncounted` d'un aspect l'exclut à la source, et c'est la voie prévue
	// pour cela. La soustraction marche aujourd'hui parce que la loi est additive ; une
	// mesure exacte ne compterait chaque pixel qu'une fois et la rendrait fausse.
	Hidden uint32

	// ByDecor est la part de [Overlap.Hidden] que le décor seul recouvre.
	//
	// Les deux origines ne portent pas la même information : ce qu'un mur cache, il le
	// cache parce que la chose est derrière lui, et un rappel y retire cet indice ; une
	// créature qui s'interpose ne dit rien de tel. Un hôte qui veut les traiter
	// différemment a ici de quoi.
	ByDecor uint32

	// Painted est le nombre de pixels que la chose peint réellement, ceux dont l'alpha
	// de la planche n'est pas nul. Zéro quand l'hôte n'a pas donné la forme de ses
	// planches au moteur, ou quand la teinte du quad est transparente.
	//
	// **C'est l'échelle des champs qui comptent au pixel, et elle n'est pas celle de
	// [Overlap.Hidden]** : une case de personnage de soixante-quatre sur soixante-
	// quatre en peint quatre cent quarante en moyenne, soit un dixième de sa boîte. Un
	// seuil réglé sur l'une ne vaut rien pour l'autre, et c'est la raison pour laquelle
	// ces champs s'ajoutent au lieu de remplacer.
	Painted uint32

	// Covered est le nombre de ces pixels peints qu'au moins un quad opaque recouvre.
	//
	// **Chaque pixel ne compte qu'une fois**, là où [Overlap.Hidden] additionne les
	// rectangles de chaque couple : c'est une aire, et elle ne dépasse jamais
	// [Overlap.Painted]. Sur une même scène, la mesure par boîte rend 5 534 là où
	// celle-ci rend 63.
	//
	// Ne cachent que les quads dont la teinte est pleinement opaque et qui ne
	// s'ajoutent pas à ce qu'ils couvrent : un quad additif éclaircit sans masquer, une
	// teinte partielle laisse voir au travers. Un quad plat vaut son rectangle.
	Covered uint32

	// CoveredByDecor est la part de [Overlap.Covered] que le décor seul recouvre, dans
	// la même unité et avec la même déduplication.
	CoveredByDecor uint32
}

// OverlapList est le relevé d'une image, dans l'ordre où les choses ont été posées.
//
// Un tableau nommé et non une structure qui en porterait un, pour la raison qui vaut
// pour [DrawList] : un hôte en reçoit le pointeur du premier élément avec leur nombre,
// sans conversion ni propriété mémoire à négocier.
type OverlapList []Overlap
