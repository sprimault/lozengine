// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"io"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/rendu"
	"github.com/sprimault/lozengine/scene"
)

// Quad est la primitive unique du moteur : un rectangle de planche posé dans le
// tampon, déjà placé à l'écran et rangé dans l'ordre de dessin. Sa disposition est
// compatible C.
type Quad = rendu.Quad

// Rendu est la suite ordonnée des quads d'une image : l'ordre du tableau est
// l'ordre de dessin.
type Rendu = rendu.Rendu

// Tampon est l'image dans laquelle le moteur écrit : des pixels contigus, ligne par
// ligne, prémultipliés comme ceux d'une planche.
//
// Un hôte peut l'allouer lui-même et laisser le moteur écrire dedans, ce qui évite
// d'avoir à négocier qui possède cette mémoire.
type Tampon = rendu.Tampon

// BorneTampon est le plus grand côté qu'un tampon puisse avoir : la plus grande
// coordonnée qu'un [Quad] sait désigner.
const BorneTampon = rendu.BorneTampon

// NouveauTampon rend un tampon transparent aux dimensions demandées, et panique
// au-delà de [BorneTampon] de côté — une taille pareille signale un calcul fautif en
// amont plutôt qu'une donnée du jeu.
//
// Une fonction et non un alias, pour la raison qui vaut pour [Blanc].
func NouveauTampon(largeur, hauteur uint16) *Tampon { return rendu.NouveauTampon(largeur, hauteur) }

// Couleur est une couleur RVBA sur quatre octets, à alpha prémultiplié.
type Couleur = rendu.Couleur

// Drapeaux porte les variantes de dessin d'un quad, un bit par variante.
type Drapeaux = rendu.Drapeaux

const (
	// MiroirX lit la source de droite à gauche.
	MiroirX = rendu.MiroirX

	// MiroirY lit la source de bas en haut.
	MiroirY = rendu.MiroirY

	// Aplat peint le rectangle en Teinte sans lire aucune source.
	Aplat = rendu.Aplat
)

// CoucheMax et EtageMax sont les plus grandes valeurs qu'une couche et un étage
// puissent porter : seize couches et soixante-quatre étages.
//
// Combien un jeu en emploie ne regarde que lui ; ce sont les bornes au-delà desquelles
// l'ordre de dessin ne distinguerait plus deux choses, et une pose qui les dépasse est
// refusée au lieu d'être ramenée au plafond en silence.
const (
	CoucheMax = rendu.CoucheMax
	EtageMax  = rendu.EtageMax
)

// CoteMax est le plus grand côté qu'une grille de scène puisse avoir, en cellules, et
// StratesMax le nombre de choses qu'une cellule peut empiler.
//
// [NouvelleScene] et [Scene.PoserCellule] refusent au-delà. Les lire évite d'avoir à
// essayer pour le savoir.
const (
	CoteMax    = rendu.CoteMax
	StratesMax = scene.StratesMax
)

// Blanc rend la teinte neutre, celle qui laisse la source intacte.
//
// Une fonction et non un alias : seuls les types se republient sans enveloppe, et
// celle-ci ne coûte rien — le compilateur la remplace par son corps.
func Blanc() Couleur { return rendu.Blanc() }

// Matrice convertit entre les cellules du monde et les pixels de l'écran, dans les
// deux sens, et garde l'inverse qu'elle a calculée une fois.
//
// Le calcul est entier de bout en bout, ce qui fait qu'une même scène rend les mêmes
// pixels d'une machine à l'autre : les nombres à virgule flottante ne le
// garantiraient pas, la spécification du langage autorisant à fusionner une
// multiplication suivie d'une addition en une instruction unique que les
// architectures ne pratiquent pas toutes.
//
// Une matrice s'obtient d'un des quatre constructeurs qui suivent, republiés en
// fonctions pour la raison qui vaut pour [Blanc], et s'emploie par ses méthodes.
// L'ordre de dessin n'en vient pas : c'est la clé de tri qui le porte.
type Matrice = geometry.Matrix

// Dimetrique rend la projection d'une cellule en losange au ratio libre, celle d'un
// rendu isométrique de jeu : une cellule de 32 sur 16 avance de seize pixels vers le
// côté et de huit vers le bas.
//
// C'est le constructeur du cas courant, et le seul qui laisse la hauteur libre.
func Dimetrique(largeur, hauteur int) Matrice { return geometry.Dimetric(largeur, hauteur) }

// Isometrique rend la projection de l'isométrie vraie, dont la hauteur ne se choisit
// pas : elle vaut la largeur divisée par racine de trois, soit dix-huit pixels et
// demi pour une largeur de trente-deux là où le dimétrique courant en prend seize.
func Isometrique(largeur int) Matrice { return geometry.Isometric(largeur) }

// Grille rend la projection de la vue de dessus, où une cellule est un carré et non
// un losange. Les axes n'y tournent pas, ce qu'aucun ratio de [Dimetrique] ne donne.
func Grille(cote int) Matrice { return geometry.Grid(cote) }

// Brute rend la projection de quatre coefficients donnés, pour ce qu'aucun des trois
// autres constructeurs n'exprime : l'abscisse écran vaut a·x + b·y et l'ordonnée
// c·x + d·y. Sans translation, la vue étant cadrée ailleurs.
func Brute(a, b, c, d float64) Matrice { return geometry.Raw(a, b, c, d) }

// Input est ce que le clavier dit à une image : les touches tenues, et celles qui
// viennent d'être enfoncées.
//
// Les deux régimes n'ont pas le même usage. Held sert au déplacement, tant que la
// touche est tenue ; JustPressed sert à ce qui se consomme — tirer, ramasser,
// valider —, et ne vaut qu'une fois par appui.
type Input = input.Input

// Key désigne une place sur le clavier, et non le caractère qu'elle produit : la
// touche nommée W est celle marquée Z sur un clavier français.
type Key = input.Key

// Planche est une page de pixels dans laquelle les quads puisent, ligne par ligne et
// sans remplissage : le pixel (x, y) est à l'index y×Largeur + x.
//
// Ses composantes sont prémultipliées, comme celles de [Couleur]. Son champ Rectangles
// porte la découpe, que [Planche.DecouperGrille] remplit ou qu'on écrit à la main ;
// vide, la planche vaut pour elle-même.
type Planche = raster.Planche

// Atlas est la suite des planches qu'un quad désigne par index.
//
// [Atlas.Ajouter] est le seul passage obligé avant qu'un quad puisse désigner une
// planche : c'est là qu'elle est préparée pour le dessin, donc l'oubli n'est pas
// possible.
type Atlas = raster.Atlas

// Rectangle est une découpe dans une planche, en pixels de celle-ci.
//
// Il ne voyage pas jusqu'au dessin — un quad porte déjà sa source. Il sert à qui
// construit des quads.
type Rectangle = raster.Rectangle

// NouvellePlanche rend une planche transparente aux dimensions demandées, à remplir
// pixel par pixel.
//
// C'est la seule façon d'obtenir une planche sans fichier : la retenir obligerait un
// jeu à encoder un PNG en mémoire pour dessiner un aplat.
func NouvellePlanche(largeur, hauteur uint16) Planche {
	return raster.NouvellePlanche(largeur, hauteur)
}

// ChargerPlanche lit un PNG depuis un lecteur et rend une planche en alpha
// prémultiplié.
//
// Un lecteur et non un chemin, ce qui sert aussi bien un fichier qu'une ressource
// embarquée par go:embed ou une archive. Pour un chemin, [OuvrirPlanche].
func ChargerPlanche(r io.Reader) (Planche, error) { return raster.ChargerPlanche(r) }

// Scene porte une grille de décor, des entités qui s'y déplacent, et la vue par
// laquelle on les regarde.
//
// Une scène se décrit, elle ne se dessine pas. Rien ici ne place quoi que ce soit à
// l'écran : c'est ce qui permet au moteur de décider de l'ordre de dessin au lieu de le
// demander au jeu, et c'est tout ce qu'une scène décrite rapporte. Elle s'obtient de
// [NouvelleScene] et s'emploie par ses méthodes.
//
// Deux chemins, parce que deux coûts. Le décor se pose par cellule et ne se relit que
// sur la fenêtre visible, donc une carte de cinq cents cellules de côté coûte autant
// qu'une de cinquante. Les entités se parcourent en entier, étant presque toutes à
// l'image.
type Scene = scene.Scene

// Entite est un aspect posé à un point du monde.
//
// Les coordonnées sont des cellules et elles sont flottantes : une entité se tient en
// x = 3,5 sans que rien ne l'arrondisse. z est la hauteur au-dessus du sol et reste un
// champ à part, que l'écran retranche de l'ordonnée.
type Entite = scene.Entite

// Aspect est ce qu'une chose montre : de quoi remplir un quad, sans dire où.
//
// Sa valeur zéro ne montre rien, et c'est ainsi qu'une cellule vide se reconnaît. Sa
// teinte, en revanche, se renseigne : la valeur zéro d'une couleur est transparente,
// donc une chose qu'on veut voir porte [Blanc] ou une teinte explicite.
type Aspect = scene.Aspect

// Poignee désigne une entité posée, et lui reste fidèle quand le retrait d'une autre la
// déplace dans la scène. Sa valeur zéro ne désigne rien.
type Poignee = scene.Poignee

// Orientation est l'un des quatre quarts de tour sous lesquels la vue présente la
// grille. Quatre et non un angle libre : une apparence est dessinée pour un nombre
// fini de directions.
type Orientation = scene.Orientation

const (
	// SansRotation présente la grille telle que le jeu l'a posée.
	SansRotation = scene.SansRotation

	// UnQuart, DemiTour et TroisQuarts sont les trois autres quarts de tour, dans le
	// sens où la grille pivote sous la vue.
	UnQuart     = scene.UnQuart
	DemiTour    = scene.DemiTour
	TroisQuarts = scene.TroisQuarts
)

// OptionsScene porte ce qu'une scène admet de régler, et sa valeur zéro convient.
//
// Le nom est qualifié parce que la boucle a les siennes : deux structures d'options
// dont une seule serait nommée sans suffixe laisseraient deviner laquelle.
type OptionsScene = scene.Options

// NouvelleScene monte une scène sur une grille et une projection.
//
// La projection est obligatoire et n'est pas vérifiée : sa valeur zéro projette tout
// sur l'origine, ce que sa propre documentation annonce comme l'échec visible.
//
// C'est le seul moment où une scène rend une erreur. Passé ce point, plus rien sur le
// chemin d'une image n'en rend, parce qu'une image n'a nulle part où la mettre.
func NouvelleScene(largeur, hauteur int, projection Matrice, o OptionsScene) (*Scene, error) {
	return scene.Nouvelle(largeur, hauteur, projection, o)
}
