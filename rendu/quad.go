// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package rendu

// Drapeaux porte les variantes de dessin d'un quad, un bit par variante.
type Drapeaux uint8

const (
	// MiroirX lit la source de droite à gauche. C'est ce qui permet à un
	// fournisseur d'apparence de ne dessiner que la moitié de ses directions :
	// l'est se tire de l'ouest sans rien ajouter à l'atlas.
	MiroirX Drapeaux = 1 << iota

	// MiroirY lit la source de bas en haut.
	MiroirY

	// Aplat peint le rectangle en Teinte sans lire aucune source ; Planche,
	// SourceX et SourceY sont alors ignorés.
	//
	// Sans lui, un panneau d'interface ou un fondu au noir demanderait d'étirer
	// un pixel de planche sur toute une surface, c'est-à-dire la mise à l'échelle
	// fractionnaire que le moteur s'interdit. Le quad reste la primitive unique,
	// et c'est son chemin le plus court.
	Aplat
)

// PlanchesMax est le nombre de planches qu'un atlas peut porter, donc la plus grande
// valeur que le champ Planche d'un [Quad] puisse désigner.
//
// Elle vit ici parce que deux paquets doivent la respecter sans pouvoir s'atteindre :
// celui qui range les planches, qui refuse la mille vingt-cinquième au chargement, et
// celui qui compose l'ordre de dessin, dont un champ ne distingue pas au-delà. Ce
// dernier est la raison du chiffre, mais c'est à l'un et à l'autre de le tenir.
//
// **Seule la limite sort, jamais la répartition.** Un quad ne porte pas sa clé de
// tri, et l'agencement de ses champs n'a pas à devenir irrévocable en traversant la
// frontière d'un autre langage. Ce qui se publie ici est une contrainte d'usage — un
// hôte qui construit un atlas doit la connaître —, pas un détail de format.
const PlanchesMax = 1 << 10

// CoucheMax et EtageMax sont les plus grandes valeurs que les champs Couche et Etage
// d'une chose à peindre puissent porter : seize couches et soixante-quatre étages.
//
// Elles sortent ici pour la raison qui fait sortir [PlanchesMax], et pour une de plus :
// **c'est le jeu qui décide combien de couches il emploie**, donc c'est lui qui doit
// savoir jusqu'où il peut aller. Les laisser dans le paquet qui compose l'ordre de
// dessin les aurait rendues invisibles de l'extérieur, et un jeu qui demande une
// vingtième couche aurait vu six de ses couches se confondre avec la quinzième sans
// qu'aucun contrôle ne le dise.
//
// Ce ne sont pas des choix de jeu codés en dur, mais ce que le format porte — une clé
// de tri sur soixante-quatre bits est par nature un jeu de plafonds, et les annoncer
// est le contraire de décider à la place de qui écrit le jeu.
const (
	CoucheMax = 1<<4 - 1
	EtageMax  = 1<<6 - 1
)

// Quad est la primitive unique du moteur : un rectangle de planche posé dans le
// tampon, déjà placé à l'écran, rangé dans l'ordre de dessin, et retenu — ceux qui
// sortaient du tampon ont été retirés en amont.
//
// Tout converge ici et rien n'en sort — les fonctions de confort produisent des
// quads, jamais un second chemin de dessin. Un quad ne porte pas sa clé de tri :
// un Rendu est ordonné par construction, et la répartition des bits de la clé n'a
// pas à traverser la frontière C, où elle deviendrait irrévocable.
//
// Teinte multiplie la source composante par composante, 255 étant neutre, et son
// alpha multiplie en plus l'opacité. À blanc opaque, le rastériseur retombe sur
// la copie directe : la modulation se décide une fois par quad et non par pixel.
//
// La disposition est celle qu'un hôte écrirait en C — champs de taille fixe,
// vingt octets, alignement de deux, dont un octet de remplissage final que la
// documentation résume en « champ non exporté » : les huit champs nommés en
// occupent dix-neuf. D'où trois limites, que le chargement signale
// quand il peut mais que le chemin de rendu ne vérifie jamais : un tampon de
// 32 767 pixels de côté, une planche de 65 535, et [PlanchesMax] planches.
//
// La valeur zéro ne dessine rien, ce qui est l'échec sûr : une structure remplie
// de zéros par un hôte reste invisible au lieu de peindre n'importe quoi.
type Quad struct {
	X, Y             int16   // destination dans le tampon, coin haut-gauche
	SourceX, SourceY uint16  // coin haut-gauche dans la planche
	Largeur, Hauteur uint16  // taille, commune à la source et à la destination
	Planche          uint16  // index dans l'atlas que le rastériseur reçoit à côté
	Teinte           Couleur // modulation, voir ci-dessus
	Drapeaux         Drapeaux
	_                uint8 // complète à vingt octets ; explicite pour l'en-tête C
}
