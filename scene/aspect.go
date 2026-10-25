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

	// Layer et Floor ouvrent les deux champs les plus forts de la clé de tri. La
	// couche range ce qui doit passer devant tout le reste, une interface par exemple ;
	// l'étage sépare les niveaux d'un décor. Combien un jeu en emploie ne regarde que
	// lui — la clé sait en distinguer seize et soixante-quatre.
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
// z est la hauteur au-dessus du sol et reste un champ à part. L'écran la retranche de
// l'ordonnée, la clé de tri la range dans son propre champ : les fondre perdrait
// l'information dès l'entrée et rendrait l'ordre de dessin irréparable.
type Entity struct {
	X, Y, Z float64

	Aspect
}
