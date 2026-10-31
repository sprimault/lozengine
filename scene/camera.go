// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La vue : les quatre quarts de tour, le cadrage sur un point du monde, et le
// remappage qui fait tourner la grille sous la caméra. La matrice n'y est pour rien et
// reste la même pour les quatre orientations.

package scene

import "github.com/sprimault/lozengine/render"

// Orientation est l'un des quatre quarts de tour sous lesquels une scène se regarde :
// [NoRotation], [QuarterTurn], [HalfTurn] et [ThreeQuarters].
//
// Quatre et non un angle libre : une apparence est dessinée pour un nombre fini de
// directions, et un angle quelconque demanderait de l'interpoler — ce qu'un rendu en
// pixels ne sait pas faire sans se brouiller.
type Orientation uint8

// Les quatre quarts de tour, dans le sens où la grille pivote sous la vue.
const (
	NoRotation Orientation = iota
	QuarterTurn
	HalfTurn
	ThreeQuarters
)

// Orient fait pivoter la vue d'un quart de tour ou d'un multiple, et recadre sur le
// point visé.
//
// La matrice n'y est pour rien et ne change pas : ce sont les coordonnées du monde qui
// sont remappées avant d'y entrer. Une rotation par changement de signe donnerait des
// coordonnées négatives, donc une profondeur signée, un biais à ajouter et la moitié
// de la plage perdue sur le champ que la clé de tri lui réserve. La scène connaissant
// l'étendue de sa grille, elle peut renverser un axe au lieu de l'inverser, et tout
// reste positif.
func (s *Scene) Orient(o Orientation) {
	s.orientation = o & 3
	s.reframe()
}

// Orientation rend le quart de tour courant.
func (s *Scene) Orientation() Orientation { return s.orientation }

// Focus centre la vue sur un point du monde, pour un tampon de la taille donnée.
//
// La taille vient de l'appelant et n'est jamais supposée ici : la résolution interne
// est le choix du jeu, et la mise à l'échelle vers une fenêtre appartient au backend.
//
// **Qui passe par la boucle du paquet racine ne la décide pas** : celle-ci la réimpose
// à chaque pas, d'après la fenêtre et les bornes des options, parce que la vue y suit
// la fenêtre. La taille donnée ici est alors celle du premier cadrage, et [Scene.View]
// dit ce qu'elle est devenue. Un hôte qui tient sa propre boucle garde les deux.
//
// # Cadrer sur un pixel plutôt que sur un point du monde
//
// Le point visé reste en cellules, et ce n'est pas un oubli : une rotation le déplace à
// l'écran sans le déplacer dans le monde, donc un focus en pixels ne survivrait pas à
// [Scene.Orient]. Viser une origine de vue en pixels se fait en trois lignes, et le
// résultat est **exact au pixel** — [geometry.Matrix.Unproject] garantit ce sens :
//
//	_, _, w, h := s.View()
//	rx, ry := s.Projection().Unproject(originX+w/2, originY+h/2)
//	fx, fy := s.Unremap(rx, ry)
//	s.Focus(fx, fy, w, h)
//
// La taille vient de [Scene.View] et non d'une constante, pour que la division entière
// par deux soit la même ici et dans le recadrage : les deux troncatures portent alors
// la même largeur et se compensent. Une vue de largeur impaire décalerait l'origine
// d'un pixel autrement.
//
// La réserve, et elle ne vaut que pour la boucle du paquet racine : celle-ci réimpose
// la taille **après** le pas du jeu, donc un jeu qui change son retrait de monde dans
// le même pas voit l'origine glisser de la moitié de la variation d'aire.
//
// **Et le piège qui coûte vraiment des pixels, qui n'est pas celui qu'on attend** :
// une borne d'écran ne se traduit pas en bornes par axe du monde. L'abscisse écran est
// une combinaison des deux axes, donc borner `x` et `y` séparément contre des valeurs
// dérivées d'un pixel laisse l'origine dépasser la borne visée. Un cadrage se borne sur
// un **point** en pixels, avant la conversion ci-dessus, jamais axe par axe après.
//
// Où la caméra s'arrête reste une décision de jeu, et le moteur n'applique donc aucune
// borne aux bords de la grille : certains veulent buter sur le bord, d'autres montrer le
// vide autour.
func (s *Scene) Focus(x, y float64, width, height int32) {
	s.focusX, s.focusY = x, y
	s.viewWidth, s.viewHeight = sides(width, height)
	s.reframe()
}

// sides ramène un couple de côtés dans ce que le trieur sait convertir, soit
// [render.MaxBufferSide].
//
// **Le trieur découpe contre ces côtés puis convertit en seize bits** : au-delà de la
// borne, la conversion se replie et un quad part à l'autre bout de l'image au lieu
// d'être écarté. `render.NewBuffer` **panique** pour exactement ce motif, et `Inset.sane`
// nomme le même dégât — ce paquet était le seul des trois à laisser passer.
//
// **Ramené et non refusé**, comme `Inset.sane` : ces trois voies ne rendent rien, et le
// chemin d'une image ne rapporte pas d'erreur. Un côté nul ou négatif reste traité par
// chaque appelant, qui en fait un non-événement.
//
// **Rien ne le montre aujourd'hui**, et l'échéance est nommée : le seul appelant de
// `Sorter.Frame` est `Engine.Step`, qui réimpose les deux rectangles à chaque pas depuis
// des côtés que `NewBuffer` a déjà bornés. Le jalon 11 expose « le noyau, jamais la
// fenêtre ni la boucle » : un hôte posera ses rectangles lui-même, et cette réimposition
// ne sera plus sur le chemin. Trois lignes aujourd'hui, la frontière C ensuite.
func sides(width, height int32) (int32, int32) {
	return min(width, render.MaxBufferSide), min(height, render.MaxBufferSide)
}

// ResizeView change la taille de la vue en gardant le point visé, et recadre dessus.
//
// C'est [Scene.Focus] sans redire où l'on regarde, pour qui redimensionne : le point
// visé est une donnée du jeu, qu'une fenêtre étirée ne doit pas déplacer. Le passer à
// nouveau obligerait l'appelant à le garder de son côté, et deux copies d'un même
// point finissent par diverger.
//
// Une taille nulle ou négative ne change rien : c'est ce que rend une fenêtre réduite,
// et la vue d'alors n'a aucune raison de disparaître.
func (s *Scene) ResizeView(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}
	width, height = sides(width, height)
	if width == s.viewWidth && height == s.viewHeight {
		return
	}

	s.viewWidth, s.viewHeight = width, height
	s.reframe()
}

// View rend l'origine de la vue en pixels absolus, et la taille du tampon.
//
// C'est ce que le tri retranche de chaque point projeté, au moment où il écarte ce qui
// tombe hors de l'image. Les deux gestes sont le même calcul.
//
// Les côtés sont bornés à [render.MaxBufferSide], ramenés et non refusés : au-delà, le
// découpage du tri convertirait en seize bits une valeur qui s'y replie, et un quad
// partirait à l'autre bout de l'image au lieu d'être écarté.
func (s *Scene) View() (x, y, width, height int32) {
	return s.viewX, s.viewY, s.viewWidth, s.viewHeight
}

// Surface rend la taille de la surface de dessin : le tampon que le moteur remplit, et
// le repère dans lequel [Scene.PlaceScreen] pose.
//
// Elle vaut la vue tant que rien ne l'élargit, donc les deux coïncident pour un jeu qui
// ne borne pas son aire de monde. Quand un plafond mord, la surface continue de remplir
// la fenêtre et c'est la vue qui rétrécit : ce que la surface a de plus est la marge, et
// elle appartient au jeu — c'est là que vont un bandeau, un score, une minicarte.
//
// Les côtés sont bornés à [render.MaxBufferSide] comme ceux de [Scene.View], et pour la
// même raison.
func (s *Scene) Surface() (width, height int32) {
	return max(s.surfaceWidth, s.viewWidth), max(s.surfaceHeight, s.viewHeight)
}

// WorldOrigin rend la place de l'aire de monde dans la surface, en pixels de celle-ci.
//
// C'est ce qui permet d'accrocher une interface au bord de l'**aire** plutôt qu'au bord
// du tampon, les deux références étant alors disponibles. Elle vaut (0, 0) quand la vue
// remplit la surface, donc un jeu sans marge n'a pas à la consulter.
//
// **Les deux marges s'y composent, dans cet ordre** : le retrait de
// [Scene.SetWorldInset] est acquis à l'interface, et l'alignement de
// [Scene.SetAlignment] ne répartit que ce qu'un plafond a retiré **en plus** de lui.
// Une aire qui prend tout ce que les retraits laissent part donc exactement du retrait
// de gauche et du retrait du haut, quel que soit l'alignement — c'est ce qui tient la
// marge fixe pendant que la fenêtre change.
//
// Où la marge se répartit vient du jeu, jamais du moteur : la place de l'aire est une
// propriété de la mise en page.
func (s *Scene) WorldOrigin() (x, y int32) {
	width, height := s.Surface()
	availableWidth, availableHeight := s.inset.Available(width, height)
	insetX, insetY := s.inset.before(width, height)

	return insetX + s.alignX.offset(availableWidth-s.viewWidth),
		insetY + s.alignY.offset(availableHeight-s.viewHeight)
}

// ResizeSurface change la taille de la surface de dessin, sans toucher à la vue.
//
// La boucle du paquet racine l'appelle d'après la fenêtre ; un hôte qui tient la sienne
// la pose lui-même, et peut ne jamais l'appeler — la surface vaut alors la vue.
//
// Une taille nulle ou négative ne change rien, pour la raison que [Scene.ResizeView]
// donne : c'est ce que rend une fenêtre réduite.
func (s *Scene) ResizeSurface(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}

	s.surfaceWidth, s.surfaceHeight = sides(width, height)
}

// SetAlignment pose où l'aire de monde se place dans la surface, un axe après l'autre.
//
// Les valeurs et le piège des axes sont dans la godoc d'[Alignment] : ce sont les axes
// de la surface, et une rotation de la scène ne les fait pas tourner.
func (s *Scene) SetAlignment(x, y Alignment) {
	s.alignX, s.alignY = x, y
}

// SetWorldInset pose la marge que le monde ne traverse pas, côté par côté.
//
// **Elle se change en cours de partie**, et c'est pourquoi c'est un réglage de scène
// plutôt qu'une option figée au montage : une interface qui se replie ou un panneau
// qu'on ouvre rend au monde ce qu'il occupait. La boucle du paquet racine relit le
// retrait à chaque pas, donc l'aire suit sans qu'on ait à redimensionner quoi que ce
// soit.
//
// Ce que le retrait signifie et comment il se compose avec un plafond sont dans la
// godoc d'[Inset].
func (s *Scene) SetWorldInset(i Inset) {
	s.inset = i
}

// WorldInset rend la marge que le monde ne traverse pas, telle qu'elle a été posée.
func (s *Scene) WorldInset() Inset {
	return s.inset
}

// Remap rend les coordonnées sous lesquelles l'orientation courante présente un
// point du monde.
//
// Un jeu pose ses entités et son décor dans son repère à lui, une fois pour toutes ;
// c'est la vue qui tourne. Le remappage renverse un axe autour de l'étendue de la
// grille plutôt que d'en changer le signe, ce qui garde tout positif.
func (s *Scene) Remap(x, y float64) (float64, float64) {
	width, height := float64(s.width), float64(s.height)

	switch s.orientation {
	case QuarterTurn:
		return height - y, x
	case HalfTurn:
		return width - x, height - y
	case ThreeQuarters:
		return y, width - x
	}
	return x, y
}

// Unremap fait le chemin inverse de [Scene.Remap].
//
// Le tri en a besoin pour relever le décor : il remonte des coins de l'image vers des
// coordonnées remappées, et doit retrouver la cellule que le jeu a posée.
func (s *Scene) Unremap(x, y float64) (float64, float64) {
	width, height := float64(s.width), float64(s.height)

	switch s.orientation {
	case QuarterTurn:
		return y, height - x
	case HalfTurn:
		return width - x, height - y
	case ThreeQuarters:
		return width - y, x
	}
	return x, y
}

// At rend le point du monde qu'un pixel de l'image désigne, à l'étage donné.
//
// C'est le chemin complet de la désignation au pointeur, et il a cinq étages qu'un
// jeu n'a pas à connaître : le pixel reçu est relatif à la surface, donc la place de
// l'aire de monde dans celle-ci s'en retranche ; l'origine de la vue s'y ajoute ; un
// étage se dessine plus haut que le sol, donc le pixel redescend d'autant avant qu'on
// inverse ; l'inverse rend des coordonnées remappées ; et le remappage se défait pour
// revenir au repère où le jeu a posé sa grille.
//
// **Le pixel est celui de la surface, pas celui de la fenêtre ni celui de l'aire de
// monde.** C'est exactement ce que rend la méthode `Pointer` d'une entrée, la boucle
// ayant déjà
// défait l'agrandissement et les bandes. Un pixel pris dans la marge rend donc un point
// hors de l'aire, et c'est au jeu de le refuser s'il y a posé son interface — ce que
// [Scene.Surface] et [Scene.WorldOrigin] lui permettent de décider.
//
// Le résultat est en cellules flottantes, dont la partie entière est la case : un
// point rendu à (3,7 ; 2,2) désigne la case (3, 2). L'aller-retour n'est pas exact —
// l'inverse d'une matrice en virgule fixe ne l'est pas — et la godoc de
// [geometry.Matrix.Unproject] dit ce que vaut ce quantum ; il est très inférieur à la
// cellule, donc il ne change pas la case désignée sauf exactement sur une bordure.
//
// Rien n'est borné à la grille : un pixel hors de la carte rend un point hors de la
// carte, négatif au besoin. C'est au jeu de décider si cela désigne quelque chose,
// puisque lui seul sait ce qu'il y a au-delà.
func (s *Scene) At(ex, ey int32, floor uint8) (x, y float64) {
	ox, oy := s.WorldOrigin()
	rx, ry := s.projection.Unproject(
		s.viewX+ex-ox,
		s.viewY+ey-oy+s.projection.Rise(float64(floor)),
	)
	return s.Unremap(rx, ry)
}

// Extent rend les côtés de la grille tels que l'orientation courante les présente :
// un quart de tour les échange.
func (s *Scene) Extent() (width, height int) {
	if s.orientation == QuarterTurn || s.orientation == ThreeQuarters {
		return s.height, s.width
	}
	return s.width, s.height
}

// reframe replace l'origine de la vue après un déplacement ou une rotation.
//
// Le point visé se retrouve au centre du tampon. Aucune borne n'est appliquée aux
// bords de la grille : où la caméra s'arrête est une décision de jeu, pas de moteur —
// certains veulent buter sur le bord, d'autres montrer le vide autour.
func (s *Scene) reframe() {
	rx, ry := s.Remap(s.focusX, s.focusY)
	ex, ey, _, _ := s.projection.Project(rx, ry, 0)

	s.viewX = ex - s.viewWidth/2
	s.viewY = ey - s.viewHeight/2
}
