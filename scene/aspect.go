// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce qu'une chose montre, et ce qu'elle montre à un endroit : l'aspect sans position,
// que le décor emploie parce qu'une cellule est située par son rang, et l'entité qui
// lui ajoute un point du monde.

package scene

import "github.com/sprimault/lozengine/render"

// Aspect est ce qu'une chose montre : de quoi remplir un quad, sans dire où.
//
// Séparé de la position parce que le décor n'en a pas besoin — une cellule de grille
// est située par son rang, et lui recopier des coordonnées doublerait ce que la grille
// sait déjà. Sur une carte de cinq cents cellules de côté, le doublon pèserait six
// mégaoctets par couche.
//
// Sa valeur zéro ne montre rien, et c'est ainsi qu'une cellule vide se reconnaît : une
// largeur ou une hauteur nulle ne couvre aucun pixel, donc aucune table de présence
// n'a à exister à côté.
//
// **L'ancre a deux niveaux, et c'est le second qui se règle.** Le premier est le point
// du monde visé, celui du contact au sol : le centre de sa case pour une cellule de
// décor, la position donnée pour une entité. Il ne se surcharge pas — la grille et la
// projection le décident. Le second est l'endroit où l'image se pose autour du pixel
// que ce point atteint : par défaut le pied de l'image, centré horizontalement, soit
// un coin haut-gauche à (−Width/2, −Height). C'est celui-là que [Aspect.AnchorX] et
// [Aspect.AnchorY] décalent.
//
// Deux conséquences pour un jeu qui dérive de là une position d'écran, pour une barre
// de vie ou une infobulle. Le dernier pixel peint est **une ligne au-dessus** du point
// projeté, qui est donc la première ligne sous l'image et n'en fait pas partie. Et le
// centrage se fait en entiers : une largeur **paire penche d'un pixel vers la gauche**
// — dix colonnes en donnent cinq à gauche du point projeté et quatre à droite —, là
// qu'une largeur impaire centre exactement.
type Aspect struct {
	// La planche où lire, et le rectangle qu'on y prend.
	Sheet            uint16
	SourceX, SourceY uint16
	Width, Height    uint16

	// Tint multiplie la source, et [render.White] est la valeur qui la laisse
	// intacte. Attention : la valeur zéro d'une couleur est transparente, donc elle
	// ne dessine rien — une cellule qu'on veut voir porte une teinte explicite.
	Tint  render.Color
	Flags render.Flags

	// AnchorX et AnchorY décalent l'image autour du pixel que le point du monde
	// atteint, vers la droite et vers le bas. La valeur zéro laisse le pied centré.
	//
	// C'est la seule voie, et l'ancre ne se donne jamais par appel : elle est une
	// propriété de l'image, pas du moment où on la dessine. Deux appels qui la
	// donneraient différemment pour la même planche feraient sauter une entité d'une
	// image à la suivante sans que rien ne le signale.
	//
	// **Le décalage ne touche pas l'ordre de dessin**, qui se départage sur l'abscisse
	// du point projeté et non sur celle de l'image. Sans cela, rembourrer une planche
	// pour recaler son ancre réordonnerait ce qu'elle côtoie, et un détail de
	// découpage déciderait de ce qui passe devant.
	AnchorX, AnchorY int16

	// Reveal est la couche où la chose se redessine quand le décor la cache, et zéro
	// veut dire jamais. C'est à cela que sert la couche la plus haute qu'un jeu
	// réserve : perdre son personnage sous un empilement est ce qui peut arriver de
	// pire à la lisibilité, et cela survient précisément quand on est encerclé.
	//
	// **C'est le suivi qu'on révèle, jamais le décor qu'on efface.** Rendre un mur
	// transparent découvre ce qui se tient derrière lui autant que ce qui s'y trouve,
	// et ce n'est pas ce qu'on demandait. L'étage ne sert pas non plus de critère :
	// un sol surélevé reste du sol, et ne cache rien.
	//
	// À partir de quel recouvrement la chose se redessine, et sous quelle teinte, se
	// règlent une fois pour la scène par [Options.RevealAfter] et [Options.RevealTint]
	// — ce sont des conventions de jeu, pas des propriétés d'une image.
	Reveal uint8

	// Layer et Floor ouvrent les deux champs les plus forts de la clé de tri. La
	// couche range ce qui doit passer devant tout le reste, une interface par exemple ;
	// l'étage sépare les niveaux d'un décor. Combien un jeu en emploie ne regarde que
	// lui — la clé sait en distinguer seize et soixante-quatre.
	//
	// **L'étage n'est pas qu'un rang : c'est aussi une hauteur.** Il monte à l'écran
	// du facteur que porte la projection, réglé par
	// [geometry.Matrix.WithFloorHeight] et valant par défaut la hauteur d'une
	// cellule — le cas d'un cube dont la face du dessus couvre exactement le sol de
	// sa case. Un niveau se décrit donc par ce seul champ, et [Entity.Z] est la
	// hauteur **au-dessus** de l'étage où l'on se tient : un personnage qui grimpe
	// change son étage, un personnage qui saute change son z.
	//
	// Porter la hauteur dans les deux serait deux valeurs à tenir d'accord, et rien
	// ne le dirait au jeu. La couche, elle, ne déplace rien ; elle ne fait que ranger.
	//
	// Ni l'une ni l'autre n'est la strate de [Scene.PlaceCell], qui dit seulement
	// ce qui s'empile sur une même cellule. Une couche passe devant toute l'image, une
	// strate ne passe devant que ce qu'elle recouvre.
	Layer uint8
	Floor uint8
}

// fitsInKey dit si la couche et l'étage sont dans ce que l'ordre de dessin sait
// distinguer, soit [render.MaxLayer] et [render.MaxFloor].
//
// Les deux champs sont des octets alors que la clé de tri n'en porte que seize et
// soixante-quatre valeurs, et la composition de la clé ramène une valeur trop grande
// au plafond. Sans contrôle, un jeu qui demande une vingtième couche en verrait six se
// confondre avec la quinzième : l'image serait fausse, aucun code ne l'aurait dit, et
// la cause se chercherait dans le jeu. La pose refuse donc, ce qui est le seul endroit
// où elle peut être vue — le chemin d'une image ne vérifie rien.
func (a Aspect) fitsInKey() bool {
	return a.Layer <= render.MaxLayer && a.Floor <= render.MaxFloor
}

// Entity est un aspect posé à un point du monde.
//
// Les coordonnées sont des cellules, et elles sont flottantes : une entité se tient en
// x = 3,5 sans que rien ne l'arrondisse ici. La conversion vers les entiers du rendu se
// fait à l'entrée de la projection, une fois.
//
// **La cellule k couvre l'intervalle de k à k+1, et son centre est à k+0,5.** Le décor
// suit la même convention : une case posée en (3, 4) se dessine comme une entité qui
// se tiendrait en (3,5 ; 4,5), ce qui met une créature et le sol qu'elle foule sur le
// même point de comparaison. Deux conséquences qu'on ne devine pas : un jeu qui pose
// une entité « sur » une case ajoute un demi à ses deux coordonnées, et un jeu qui
// borne ses déplacements à la grille borne à [0, côté) — borner à côté moins un
// l'arrêterait une case avant le bord, et seulement d'un côté.
//
// Les coordonnées ne sont pas vérifiées : une entité hors de la grille est posée sans
// refus, et simplement retirée de l'image si elle n'y tombe pas. C'est au jeu de
// borner, parce que lui seul sait si son personnage a le droit de sortir.
//
// z est la hauteur au-dessus du sol et reste un champ à part. L'écran la retranche de
// l'ordonnée, la clé de tri la range dans son propre champ : les fondre perdrait
// l'information dès l'entrée et rendrait l'ordre de dessin irréparable.
type Entity struct {
	X, Y, Z float64

	Aspect
}
