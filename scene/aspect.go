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
// mégaoctets par strate.
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
// **La pose en pixels fait exception, et c'est le piège de cette page** :
// [Scene.PlaceScreen] ne vise aucun point du monde, son premier niveau est le pixel
// donné, et c'est le **coin haut-gauche** de l'image qui s'y pose, non son pied.
//
// Deux conséquences pour un jeu qui dérive de là une position d'écran, pour une barre
// de vie ou une infobulle. Le dernier pixel peint est **une ligne au-dessus** du point
// projeté, qui est donc la première ligne sous l'image et n'en fait pas partie. Et le
// centrage se fait en entiers : une largeur **paire penche d'un pixel vers la gauche**
// — dix colonnes en donnent cinq à gauche du point projeté et quatre à droite —, là
// qu'une largeur impaire centre exactement.
//
// Et une troisième pour le sol : une tuile plate dont le centre doit tomber sur celui
// de sa case se pose une demi-hauteur trop haut, puisque c'est son pied qui y tombe.
// Un [Aspect.AnchorY] égal à la moitié de sa hauteur la recentre.
type Aspect struct {
	// La planche où lire, et le rectangle qu'on y prend.
	Sheet            uint16
	SourceX, SourceY uint16
	Width, Height    uint16

	// Tint multiplie la source, et [render.White] est la valeur qui la laisse
	// intacte. Attention : la valeur zéro d'une couleur est transparente, donc elle
	// ne dessine rien — une cellule qu'on veut voir porte une teinte explicite.
	//
	// **`White` étant le maximum, une teinte ne peut qu'atténuer.** Aucune valeur n'y
	// éclaircit : une multiplication par une composante d'au plus 255 ne monte jamais.
	// Éclaircir un sprite, le surligner, le marquer comme choisi se fait donc
	// autrement — par le champ `Flags` ci-dessous, et le champ `Tier` d'une [Entity]
	// pour le ranger.
	//
	// Elle est prémultipliée, comme toute couleur du moteur : un voile translucide
	// s'écrit par [render.Premultiply] ou s'atténue par [render.Color.Scale], et
	// baisser son seul alpha ne décrit aucune couleur.
	Tint render.Color

	// Flags change la façon de lire la source et de la mélanger au fond :
	// [render.MirrorX] et [render.MirrorY] la lisent à l'envers, [render.Flat] peint le
	// rectangle en teinte unie sans lire aucune source, et [render.Additive] ajoute au
	// fond au lieu de composer par-dessus.
	//
	// **C'est `Additive` qui éclaircit**, là où `Tint` ne sait qu'atténuer : il ajoute
	// composante par composante, chacune plafonnant à sa valeur pleine. Un surlignage
	// se décrit donc par un second quad additif posé au même point, et le champ `Tier`
	// d'une [Entity] porte la recette pour le ranger au-dessus du premier.
	//
	// Combiné à `Flat`, il donne un fondu lumineux — un rectangle de lumière qui
	// n'emprunte sa forme à aucune planche.
	Flags render.Flags

	// AnchorX et AnchorY décalent l'image, vers la droite et vers le bas, depuis là
	// où la voie de pose la met sans eux : le pied centré sur le pixel que le point
	// du monde atteint pour [Scene.Place] et [Scene.PlaceCell], le coin haut-gauche
	// sur le pixel donné pour [Scene.PlaceScreen]. La valeur zéro laisse l'image à
	// cette origine, quelle qu'elle soit.
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
	//
	// **Une ancre écrite pour une pose monde ne se transfère pas à une pose en
	// pixels**, et c'est le piège de ce champ : elle y corrige une origine qui a
	// changé. Une ancre qui recentre le pied d'une tuile de sol vaut la moitié de sa
	// hauteur pour [Scene.Place] ; sur [Scene.PlaceScreen], où l'origine est déjà le
	// coin, elle décale l'image d'autant. Deux remèdes, selon ce qu'on veut : l'annuler
	// pour que l'image se pose au coin donné, ou passer le pixel visé par
	// [Aspect.CornerFromFoot] pour qu'elle se pose par son pied comme dans le monde.
	AnchorX, AnchorY int16

	// Reveal est la couche où la chose se redessine quand ce qui passe devant elle la
	// cache, et zéro veut dire jamais. C'est à cela que sert la couche la plus haute
	// qu'un jeu réserve : perdre son personnage sous un empilement est ce qui peut
	// arriver de pire à la lisibilité, et cela survient précisément quand on est
	// encerclé.
	//
	// **C'est une couche, donc [render.MaxLayer] la borne** : une pose qui la dépasse
	// est refusée, comme pour [Aspect.Layer], et jamais ramenée au plafond.
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

	// Uncounted retire ce quad de la mesure de recouvrement : ce qu'il recouvre ne
	// compte dans **aucune** mesure de recouvrement de [render.Overlap], ni dans le
	// seuil de [Aspect.Reveal]. Il se dessine et cache à l'œil comme tout autre ; c'est
	// la mesure qu'il quitte, pas l'image.
	//
	// Les quatre et non deux : le relevé par boîte comme celui au pixel l'écartent, et
	// nommer les deux premiers seulement poussait un hôte venu du second à croire son
	// rappel encore compté — donc à renoncer au drapeau, et à retomber sur le défaut
	// qu'il existe pour fermer. « De recouvrement » est nécessaire dans la phrase :
	// [render.Overlap.Painted] ne compte jamais ce qu'un quad recouvre, et ce drapeau
	// ne la touche pas.
	//
	// **C'est pour ce qu'un jeu pose afin de rappeler une chose cachée** — un contour,
	// une flèche, une silhouette — et pour ce qui ne cache jamais rien à ses yeux : une
	// ombre au sol, un cadre. Un rappel posé plus haut que ce qui cache la chose
	// recouvre la chose elle-même ; sans ce champ, le relevé suivant le compte,
	// dépasse le seuil qui l'a déclenché, et le rappel ne s'éteint plus.
	//
	// **Et plus généralement, pour un quad qui cache à l'œil sans que ce masquage soit
	// une information de jeu.** Un panneau d'interface posé par-dessus le monde est
	// dans ce cas : il cache réellement ce qui passe dessous, mais le compter
	// allumerait le rappel de chaque chose qui s'en approche. Ce second usage est resté
	// hors de cette godoc à la livraison du champ, et c'est le relevé d'un intégrateur
	// qui l'a nommé.
	//
	// **Une propriété de la chose posée, jamais d'un couple.** Elle vaut pour tous les
	// relevés : marquer une créature que d'autres doivent continuer à compter la retire
	// de tous. Ce que le décor cache et ce que les entités cachent se lit à part, dans
	// [render.Overlap.ByDecor] pour l'échelle de boîte et [render.Overlap.CoveredByDecor]
	// pour celle du pixel.
	//
	// **À remettre dans chaque aspect passé à [Scene.SetAspect]**, qui remplace
	// l'aspect en entier : un jeu qui change l'apparence de son rappel à chaque image
	// perdrait la marque sans que rien ne le dise. C'est le même piège que la godoc
	// d'[Entity.Measure] nomme pour justifier sa place sur l'entité ; il est accepté
	// ici parce que ne pas compter est une propriété de l'**image** posée — comme
	// [Aspect.Reveal], qui vit déjà sur l'aspect pour cette raison — et parce que
	// c'est le seul champ que les trois voies de pose traversent, y compris
	// [Scene.PlaceScreen], par laquelle un rappel d'interface se pose.
	Uncounted bool
}

// CornerFromFoot rend le coin haut-gauche qui pose l'image là où le moteur la poserait
// pour une entité visant le même pixel.
//
// **Les deux familles de pose n'ont pas la même origine, et c'est tout ce qu'il y a à
// retenir** : [Scene.Place] et [Scene.PlaceCell] visent le pied de l'image, centré, là
// que [Scene.PlaceArea] et [Scene.PlaceScreen] visent son coin haut-gauche. Un aspect
// réutilisé de l'une à l'autre se décale donc de la moitié de sa largeur et de toute sa
// hauteur — sans qu'un compilateur le refuse ni qu'une image désigne la cause.
//
// Le coin rendu se donne tel quel à [Scene.PlaceArea], qui le lit dans l'aire de monde
// et y découpe l'image. Par [Scene.PlaceScreen] il vaut dans le repère de la surface, où
// c'est au jeu d'ajouter [Scene.WorldOrigin] s'il veut viser l'aire, et où rien ne se
// découpe.
//
// **L'ancre n'entre pas dans le calcul, et il ne faut pas l'y ajouter** : les deux voies
// l'appliquent d'elles-mêmes. Ce qu'une ancre écrite pour une pose monde devient sur une
// pose en pixels est dit sur le champ `AnchorX`.
func (a Aspect) CornerFromFoot(ex, ey int32) (int32, int32) {
	return ex - int32(a.Width)/2, ey - int32(a.Height)
}

// fitsInKey dit si les trois champs de l'aspect que la clé de tri porte — la couche,
// l'étage et la couche de révélation — tiennent dans ce que l'ordre de dessin sait
// distinguer, soit [render.MaxLayer] et [render.MaxFloor].
//
// Les champs sont des octets alors que la clé de tri n'en porte que seize et
// soixante-quatre valeurs, et la composition de la clé ramène une valeur trop grande
// au plafond. Sans contrôle, un jeu qui demande une vingtième couche en verrait six se
// confondre avec la quinzième : l'image serait fausse, aucun code ne l'aurait dit, et
// la cause se chercherait dans le jeu. La pose refuse donc, ce qui est le seul endroit
// où elle peut être vue — le chemin d'une image ne vérifie rien.
//
// **[Aspect.Reveal] est entré ici après les deux autres, et son dégât n'a pas la même
// forme** : saturé à quinze, un rappel ne sort pas de la couche la plus haute — il
// entre en collision avec ce qu'un jeu y a posé, son interface, où l'ordre retombe sur
// la profondeur puis la hauteur. Le rappel peut donc se peindre sous le panneau qu'il
// devait franchir, et plusieurs couches de révélation distinctes au-delà du plafond
// s'effondrent en une seule, perdant leur ordre relatif.
func (a Aspect) fitsInKey() bool {
	return a.Layer <= render.MaxLayer && a.Floor <= render.MaxFloor &&
		a.Reveal <= render.MaxLayer
}

// Entity est un aspect posé à un point du monde.
//
// Les coordonnées sont des cellules, et elles sont flottantes : une entité se tient en
// x = 3,5 sans que rien ne l'arrondisse ici. La conversion vers les entiers du rendu se
// fait à l'entrée de la projection, une fois.
//
// L'image se pose par son **pied** sur le pixel que ce point atteint : [Aspect] dit où
// il tombe exactement, et comment le déplacer.
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
// refus, et retirée de l'image si elle n'y tombe pas. C'est au jeu de borner, parce que
// lui seul sait si son personnage a le droit de sortir.
//
// **Une coordonnée négative fait exception, et ce n'est pas un retrait** : la projection
// sature au lieu de déborder, donc une profondeur négative et un `z` négatif sont
// ramenés à zéro. La chose est alors peinte au pixel du sol et triée comme si elle y
// était — elle se voit à l'image au lieu d'en disparaître, ce qui est le symptôme à
// reconnaître quand un jeu croit avoir perdu une entité qu'il a posée en négatif. La
// hauteur que [geometry.Matrix.Project] rend en quatrième est un `uint16`, qui ne sait
// pas porter un `z` négatif.
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
	// **Le relevé coûte environ six microsecondes par entité marquée visible** sur une
	// scène de trois mille quads, et le coût croît avec le nombre de quads de l'image,
	// jamais avec sa surface : la mesure compare des rectangles et ne lit aucun pixel.
	// Trois cents entités marquées coûtent donc moins de deux millisecondes. Cette
	// godoc annonçait « un parcours de l'image par entité marquée », ce qui était faux
	// de deux ordres de grandeur et n'avait jamais été mesuré. [Aspect.Reveal] fait
	// déjà mesurer de son côté, et une entité qui porte les deux ne paie qu'une fois.
	//
	// **Une entité marquée hors de l'image ne produit aucune entrée**, et le relevé est
	// donc plus court que le nombre de marques : un hôte le parcourt et lit
	// [render.Overlap.Identity], il ne l'indexe pas par l'ordre de ses poses. Rien à
	// relever n'est pas une erreur — ce qui ne se voit pas ne se fait pas recouvrir.
	//
	// **Le rappel qu'un jeu pose pour cette entité porte [Aspect.Uncounted]** : posé
	// plus haut que ce qui la cache, il la recouvre lui-même, et le relevé ne
	// retomberait plus sous le seuil qui l'a déclenché.
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
	// chose d'un fond de même gamme. Ranger les deux images dans une même planche
	// marche aussi et n'en ajoute aucune ; les deux autres voies restent fermées —
	// décaler la hauteur déplace l'image, monter d'une couche passe devant le décor.
	//
	// # Éclaircir un sprite, ou le marquer comme choisi
	//
	// Deux voies, et **la moins chère d'abord** : une variante déjà éclaircie de
	// l'image, désignée par un second `Look` d'apparence. Elle ne demande ni strate, ni
	// coïncidence de coordonnées, ni rien de ce qui suit — le jeu change d'apparence et
	// c'est tout. La table d'un fournisseur étant creuse, elle coûte une entrée par
	// apparence réellement déclarée.
	//
	// L'autre voie est **un second quad additif par-dessus**, et c'est celle d'un
	// surlignage qui varie dans le temps — une pulsation, un dosage que le jeu calcule.
	// Elle demande cinq conditions ensemble, et il faut les cinq :
	//
	//   - même `Layer` — la couche ouvre le champ le plus fort de l'ordre de dessin,
	//     donc deux couches différentes n'atteignent jamais ce champ ;
	//   - même `Floor`, pour la même raison, le champ suivant ;
	//   - mêmes `X`, `Y` et `Z`, parce que le départage lit l'abscisse projetée avant
	//     la strate : la seule profondeur égale ne suffit pas, deux points d'une même
	//     diagonale se départageant sur leur abscisse ;
	//   - un `Tier` plus grand sur le quad additif, qui est ce que ce champ règle.
	//
	// **Et deux réglages de l'aspect additif qui ne se devinent pas** : `Uncounted` à
	// vrai et `Reveal` à zéro. Sans eux, le surlignage partage la boîte de ce qu'il
	// surligne, franchit donc le même seuil de recouvrement, reçoit son propre quad
	// révélé, et la révélation remplace sa teinte de dosage par celle des options en
	// gardant l'additif — l'image est alors fausse, et `Uncounted` seul n'y suffit pas,
	// la révélation ne consultant pas ce drapeau pour le sujet qu'elle redessine.
	//
	// **Un sprite sombre ne s'éclaircit pas par lui-même**, et c'est de l'arithmétique :
	// un additif de la planche sur elle-même ajoute ses propres composantes, donc
	// presque rien là où elles sont basses. La voie est alors une silhouette claire dans
	// une seconde planche, ou la variante de la première voie.
	//
	// **Il départage désormais deux entités de planches différentes**, ce qui n'était
	// pas le cas jusqu'à la 0.5.3 : l'index de planche figurait dans la clé de tri au-
	// dessus de ce champ, donc deux images chargées dans des planches distinctes
	// n'arrivaient jamais à égalité et ce champ restait sans effet. Un liseré dérivé
	// dans sa propre planche passait alors par-dessus le sprite au lieu de dessous,
	// selon l'ordre de chargement des images.
	//
	// **Les deux cent cinquante-six valeurs sont libres**, là que la strate du décor
	// s'arrête à [MaxTiers] : cette borne vient de ce qu'une strate pleine ajoute un
	// quad par cellule visible, donc de la grille. Une entité n'alloue rien, et rien ne
	// lui impose la borne d'autre chose.
	//
	// **Il ne départage jamais contre une cellule de décor**, et c'est voulu : le rang
	// d'empilement de la clé met tout le décor d'une cellule avant ce qui s'y tient, à
	// profondeur et hauteur égales. Un sol passe donc toujours sous une entité posée
	// dessus, quelle que soit la valeur de ce champ et quel que soit le côté où
	// l'entité se trouve — cet ordre dépendait auparavant de l'index de planche, puis
	// de l'abscisse, donc il était asymétrique.
	Tier uint8
}
