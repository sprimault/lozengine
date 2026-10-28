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

	// Reveal est la couche où la chose se redessine quand ce qui passe devant elle la
	// cache, et zéro veut dire jamais. C'est à cela que sert la couche la plus haute
	// qu'un jeu réserve : perdre son personnage sous un empilement est ce qui peut
	// arriver de pire à la lisibilité, et cela survient précisément quand on est
	// encerclé.
	//
	// **Ce qui cache est ce qui se peint après, décor ou entité.** Un mur, un toit,
	// mais aussi la créature qui vient de s'interposer : être caché est être caché, et
	// l'encerclement que la phrase ci-dessus nomme est justement un recouvrement par
	// des entités. Le déclenchement les compte donc ensemble ; ce qu'on fait ensuite
	// de leur origine se règle ailleurs, et le dernier paragraphe dit où.
	//
	// **C'est le suivi qu'on révèle, jamais ce qui le cache qu'on efface.** Rendre un
	// mur transparent découvre ce qui se tient derrière lui autant que ce qui s'y
	// trouve, et ce n'est pas ce qu'on demandait. L'étage ne sert pas non plus de
	// critère : un sol surélevé reste du sol, et ne cache rien.
	//
	// À partir de quel recouvrement la chose se redessine, sous quelle teinte, et si le
	// décor et les entités se traitent différemment, sont des conventions de scène et
	// non des propriétés d'une image. [RevealOptions] les porte et les documente,
	// [Options.Revealing] les pose, [Scene.SetRevealing] les change.
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
	// Ni l'une ni l'autre n'est la strate — celle de [Scene.PlaceCell] pour une
	// cellule, [Entity.Tier] pour une entité —, qui dit seulement ce qui se superpose
	// au même endroit. Une couche passe devant toute l'image, une strate ne passe
	// devant que ce qu'elle recouvre.
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
//
// **Et elle ordonne vers le haut : à profondeur égale, la plus grande hauteur se peint
// par-dessus.** Un projectile en cloche passe donc devant la marque qu'il survole, et
// une créature volante devant celle qui marche au même point. Le sens paraît évident
// une fois écrit, et c'est pourquoi il l'est : s'y adosser sans le lire quelque part,
// c'est en dépendre sans contrat.
type Entity struct {
	X, Y, Z float64

	Aspect

	// Measure demande le relevé de ce qui recouvre cette entité, en pixels et par
	// origine, sans rien redessiner. Le moteur le rend avec l'image, à qui veut
	// dessiner son propre rappel plutôt que celui d'[Aspect.Reveal].
	//
	// **Sur l'entité et non sur son aspect, parce que [Scene.SetAspect] remplace
	// l'aspect en entier.** Un jeu change l'apparence de son personnage à chaque
	// changement de direction ; porté par l'aspect, le marqueur disparaîtrait alors
	// sans que rien ne le dise. Ce qu'on suit est la chose, pas l'image qu'elle montre.
	//
	// Le relevé coûte un parcours de l'image par entité marquée, donc n'en marquer que
	// ce qu'on suit — un personnage, pas une foule. [Aspect.Reveal] fait déjà mesurer
	// de son côté, et une entité qui porte les deux ne paie qu'une fois.
	//
	// Une cellule de décor ne se relève pas : le relevé se désigne par une poignée, et
	// le décor n'en a pas. C'est pourquoi ce champ est ici plutôt que sur l'aspect, que
	// les deux partagent.
	Measure bool

	// Tier départage deux entités que l'ordre de dessin laisse à égalité : le plus
	// grand se peint par-dessus, et sa valeur zéro les laisse dans un ordre que le
	// moteur choisit.
	//
	// **C'est ce qui rend une superposition exprimable**, et sans lui elle ne l'était
	// pas : deux entités au même point, même hauteur et même couche se départageaient
	// sur un rang de poignée, que [Scene.Place] réemploie après un retrait. Poser A,
	// poser B, retirer A puis poser C faisait reprendre à C le rang de A, donc C se
	// peignait avant B bien qu'il ait été posé après. L'ordre dépendait de l'historique
	// des retraits, et aucune valeur n'était fausse pour autant.
	//
	// Le besoin est ordinaire : une teinte additive par-dessus un sprite, qu'une teinte
	// multiplicative ne peut pas éclaircir ; un liseré par-dessous pour détacher une
	// chose d'un fond de même gamme. Les voies de contournement sont fermées ailleurs —
	// composer dans la planche bute sur [render.MaxSheets], décaler la hauteur déplace
	// l'image, monter d'une couche passe devant le décor aussi.
	//
	// **Les deux cent cinquante-six valeurs sont libres**, là que la strate du décor
	// s'arrête à [MaxTiers] : cette borne vient de ce qu'une strate pleine ajoute un
	// quad par cellule visible, donc de la grille. Une entité n'alloue rien, et rien ne
	// lui impose la borne d'autre chose.
	//
	// **Une conséquence à connaître, parce que le départage est partagé avec le
	// décor** : une entité posée exactement sur le coin d'une cellule — en (3, 4) et non
	// en (3,5 ; 4,5) — porte la profondeur de cette cellule, donc à hauteur et planche
	// égales leurs clés le sont aussi, et ce champ les départage l'une contre l'autre.
	// Un Tier élevé range alors l'entité après la cellule.
	Tier uint8
}
