// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'étape qu'un hôte pilote lui-même : une image, sans fenêtre ni horloge.
//
// C'est la capacité, et la boucle n'est que du sucre au-dessus. Un jeu hôte possède
// déjà sa boucle et son thread principal, et ne les cédera pas — c'est pourquoi le
// moteur n'expose pas de boucle qui ne rend la main qu'à l'arrêt, et aussi ce qui rend
// le navigateur atteignable : sous WebAssembly, une boucle Go qui ne rend pas la main
// empêche `requestAnimationFrame` de se déclencher.

package lozengine

import (
	"errors"
	"fmt"
	"time"

	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
)

// Game est ce qu'un jeu écrit, et c'est tout ce qu'il écrit.
//
// Une seule méthode, parce que l'ordre de dessin vient de la clé de tri et l'échelle du
// facteur entier : il ne reste au jeu qu'à décrire sa scène. C'est ce que rapporte une
// scène décrite plutôt que dessinée, et c'est la raison d'être du moteur.
type Game interface {
	// Step avance le jeu d'un pas, les entrées de ce pas à la main.
	//
	// Rien n'y est dessiné : le jeu pose, déplace et retire dans la scène, et le
	// moteur décide de l'ordre. Un pas dure ce que [Options.Tick] dit, quelle que
	// soit la durée réelle de l'image.
	Step(in Input)
}

// Quitter est ce qu'un jeu implémente **en plus** de [Game] quand il veut pouvoir
// partir de lui-même : une entrée « quitter » dans un menu, une touche d'échappement,
// une fin de partie.
//
// Sans elle, seule la fermeture de la fenêtre arrête [Run], et un jeu qui voulait
// partir n'avait que la sortie du processus — ce qui saute tout ce que la boucle fait
// en se terminant, et ne veut rien dire dans un navigateur.
//
// **C'est la boucle qui demande, et le jeu qui répond.** Une valeur rendue par
// [Game.Step] aurait touché la signature par laquelle un jeu écrit sa méthode unique,
// et le jalon 5 restructurera cette boucle pour rendre la main au navigateur entre
// deux images : une question posée traverse cette reprise, une valeur rendue par le
// flot de contrôle serait reprise avec elle.
//
// Elle n'ajoute aucune capacité à [Engine], et c'est voulu : un hôte qui tient sa
// propre boucle décide seul quand s'arrêter, et n'a personne à qui le demander. C'est
// donc du confort de la première porte d'entrée, comme l'invariant des deux portes
// l'exige.
//
// **Une interface facultative porte ce qu'[Options] ne peut pas porter**, et il n'y en a
// que deux : celle-ci, parce que la fin de la boucle est une décision que la boucle ne
// peut apprendre de nulle part ailleurs, et [Overlapped], parce qu'un relevé est un
// résultat et non un réglage. Tout ce qui *se règle* se règle par [Options], avant.
type Quitter interface {
	// Quitting dit que le jeu a fini. La boucle rend la main après l'image en cours,
	// donc l'appel suivant à [Game.Step] n'a pas lieu.
	//
	// Appelée une fois par image, après le pas : un jeu qui décide de partir pendant
	// son pas voit sa décision prise dans la même image, et l'image part quand même
	// — c'est ce qui permet d'afficher un dernier écran.
	Quitting() bool
}

// Overlapped est ce qu'un jeu implémente **en plus** de [Game] quand il veut savoir ce
// qui recouvre les entités qu'il suit, pour en dessiner son propre rappel.
//
// Elle existe parce que [Game.Step] ne reçoit que les entrées, et qu'un jeu qui passe
// par [Run] n'a donc aucune poignée vers [Engine.Overlaps] — qui est la même capacité
// pour un hôte tenant sa propre boucle. La première porte n'ajoute ici que le confort de
// recevoir, comme l'invariant des deux portes l'exige.
//
// Ce qu'un jeu en fait est à lui : un contour, une flèche, une silhouette tirée d'un
// autre dessin, ou rien du tout si le seuil qu'il se donne n'est pas atteint.
// [Aspect.Reveal] redessine la chose elle-même et ne demande pas cette interface ; tout
// autre rappel passe par ici, et [Aspect.Uncounted] évite qu'il se mesure lui-même.
type Overlapped interface {
	// Overlaps reçoit le relevé du pas précédent, et la tranche n'est valable que
	// pendant l'appel : un jeu qui veut la garder en fait une copie.
	//
	// Appelée une fois par image, **avant** le pas, et c'est ce qui rend le décalage
	// lisible : le recouvrement ne se connaît qu'après un tri, donc le relevé le plus
	// récent est celui de l'image précédente. La recevoir avant le pas la met entre les
	// mains du jeu au moment où il décide.
	Overlaps(list OverlapList)
}

// Options porte ce que l'étape et la boucle admettent de régler.
//
// Sa valeur zéro ne convient pas, contrairement au reste du moteur, et pour une seule
// raison : un atlas n'a pas de défaut raisonnable. Tout le reste en a un — la
// résolution interne, le pas, le titre, le fond —, et les laisser vides est le cas
// courant.
type Options struct {
	// Title est le titre de la fenêtre. Vide, il vaut le nom du moteur. La boucle
	// seule s'en sert.
	Title string

	// Width et Height sont la résolution interne **minimale**, celle dans laquelle le
	// moteur dessine avant l'agrandissement entier vers la fenêtre. Zéro vaut 480×270,
	// et [MaxBufferSide] est la borne.
	//
	// **La fenêtre s'ouvre à cette taille**, donc l'agrandissement vaut un tant que
	// le joueur ne l'a pas redimensionnée. Un jeu qui se veut grand à l'ouverture
	// demande une résolution interne plus grande, et non une fenêtre plus grande : le
	// facteur est entier, et c'est la résolution interne qui dit ce qu'une tuile
	// couvre.
	//
	// **C'est un plancher et non une taille fixe** : au-delà, la vue s'élargit avec la
	// fenêtre jusqu'à [Options.MaxWidth] et [Options.MaxHeight], et c'est ce qui évite
	// les bandes. Ce qui reste garanti est que la vue n'est jamais plus petite, donc
	// qu'une interface posée dans ce rectangle y tient toujours.
	Width, Height int

	// MaxWidth et MaxHeight bornent l'élargissement de la vue. Zéro ne borne pas :
	// la vue remplit la fenêtre, au facteur entier près.
	//
	// **Le défaut remplit, et c'est un changement de comportement par rapport à la
	// 0.5.3**, où la vue restait figée et où le reste de la fenêtre était peint en
	// bandes — 19 % de la fenêtre en 1600×900, 30 % sur un téléphone en paysage,
	// 82,5 % en portrait. Un jeu qui veut l'ancien comportement écrit ces deux champs
	// égaux à [Options.Width] et [Options.Height].
	//
	// **Ce que cela demande à un jeu qui pose une interface** : lire la taille par
	// [Scene.View] au lieu de la supposer. Une position calculée depuis une constante
	// ne tombe plus au bon endroit dès que la fenêtre change de forme, et rien ne le
	// signale — c'est l'interface qui se décale, pas le monde.
	//
	// **Et ce qu'un plafond achète, qui n'est pas qu'un confort** : l'aire de monde
	// visible va jusqu'à 5,4 fois le minimum entre une fenêtre 4:3 et une 21:9, et
	// 17 fois dans un format extrême. Combien le joueur voit est une donnée de jeu,
	// pas une conséquence de la fenêtre qu'il a étirée.
	MaxWidth, MaxHeight int

	// Tick est la durée d'un pas. Zéro vaut un soixantième de seconde.
	//
	// C'est un pas fixe : le jeu avance toujours de cette durée, et la boucle rend
	// les images aussi régulièrement qu'elle peut. Un pas variable ferait dépendre
	// la physique de la charge de la machine.
	Tick time.Duration

	// Background est la couleur dont chaque image part. Sa valeur zéro est
	// transparente, ce qui ne convient pas à une image qu'on présente : vide, elle
	// vaut donc le noir opaque.
	Background Color

	// Bars est la couleur des bandes qui entourent l'image dans la fenêtre, quand sa
	// taille ne tombe pas sur un multiple exact de la résolution interne. Vide, elle
	// vaut le noir opaque, comme Background.
	//
	// Séparée de Background, parce que les deux ne désignent pas la même chose :
	// Background est la couleur du vide **du monde** — le ciel, le néant sous la
	// carte —, Bars celle de ce qui entoure l'image **à l'écran** et n'appartient pas
	// au monde. Les confondre donnerait des bandes bleu clair à un jeu dont le ciel
	// l'est, et le joueur ne verrait plus où son jeu s'arrête.
	//
	// La boucle seule s'en sert : un hôte qui emploie [Engine] présente lui-même et
	// décide donc lui-même de ce qu'il met autour.
	Bars Color

	// Atlas porte les planches dans lesquelles les quads puisent. Il est requis :
	// un quad qui désigne une planche absente arrête le programme, ce que le
	// rastériseur assume pour que la cause se lise à l'endroit de l'erreur.
	Atlas *Atlas

	// Icons porte l'icône de la fenêtre, une planche par taille que le jeu dessine.
	// Vide, la fenêtre porte l'icône du système.
	//
	// Plusieurs tailles plutôt qu'une, parce que les systèmes en demandent
	// plusieurs — Windows une grande et une petite — et que le moteur ne
	// redimensionne pas : l'échelle entière obligatoire ne sait pas passer de 24 à
	// 16 pixels, et une réduction douce brouillerait un dessin fait pixel par pixel.
	// La taille la plus proche est retenue, et c'est au système de réduire s'il y
	// tient.
	//
	// La boucle seule s'en sert : un hôte qui emploie [Engine] possède sa fenêtre.
	Icons []Sheet

	// HideCursor retire le curseur du système de la zone de jeu.
	//
	// C'est ce qu'un jeu demande quand il dessine le sien dans son image, et il n'y a
	// pas de milieu : laisser les deux fait suivre au joueur deux pointeurs, celui du
	// jeu agrandi au facteur entier et celui du système net et décalé. Les bordures
	// et la barre de titre gardent les leurs.
	//
	// Le moteur ne dessine aucun curseur à la place : ce qu'on montre sous le
	// pointeur — une flèche, une case surlignée, rien — est une décision de jeu, et
	// [Input.Pointer] donne où le poser.
	//
	// La boucle seule s'en sert, comme pour les icônes.
	HideCursor bool
}

// Les défauts de la résolution interne, choisis au jalon 0 et tenus depuis : une tuile
// de sol de 32×16 et un personnage d'environ 24 pixels y tiennent, et le facteur entier
// trouve presque toujours un cran utile sur un écran courant.
const (
	defaultWidth  = 480
	defaultHeight = 270
	defaultTick   = time.Second / 60
)

// withDefaults rend les options complétées, sans toucher à celles qu'on a reçues.
func (o Options) withDefaults() Options {
	if o.Width == 0 {
		o.Width = defaultWidth
	}
	if o.Height == 0 {
		o.Height = defaultHeight
	}

	// Le plafond est ramené au plancher quand il est dessous : un jeu qui donne un
	// maximum plus petit que son minimum décrit une vue impossible, et refuser le
	// montage pour cela coûterait une erreur de plus sur un chemin que personne ne
	// prend volontairement. Zéro ouvre au plus grand tampon possible, donc à la
	// fenêtre entière, le facteur entier bornant le reste.
	if o.MaxWidth <= 0 {
		o.MaxWidth = MaxBufferSide
	}
	if o.MaxHeight <= 0 {
		o.MaxHeight = MaxBufferSide
	}
	o.MaxWidth = max(o.MaxWidth, o.Width)
	o.MaxHeight = max(o.MaxHeight, o.Height)

	if o.Tick == 0 {
		o.Tick = defaultTick
	}
	if o.Background == (Color{}) {
		o.Background = Color{A: 255}
	}
	if o.Bars == (Color{}) {
		o.Bars = Color{A: 255}
	}
	if o.Title == "" {
		o.Title = "lozengine"
	}
	return o
}

// Engine tient ce qu'une image réutilise de la précédente, et avance d'un pas à la
// demande.
//
// Il n'a ni fenêtre ni horloge : c'est l'appelant qui lui donne les entrées et qui
// décide quand appeler. [Run] le fait pour qui écrit un jeu en Go ; un hôte dans un
// autre langage le fera depuis sa propre boucle.
type Engine struct {
	scene  *Scene
	sorter *sorting.Sorter
	frame  *Buffer
	atlas  *Atlas

	// Les bornes de la vue, en pixels de la résolution interne. Le minimum est aussi
	// la taille d'ouverture de la fenêtre et le plancher que [Engine.Fit] ne descend
	// jamais ; le maximum vaut [MaxBufferSide] quand le jeu n'en pose pas.
	minWidth, minHeight int
	maxWidth, maxHeight int

	background Color
	tick       time.Duration
}

// NewEngine monte l'étape sur une scène.
//
// Tout ce qui peut échouer échoue ici : au-delà, un pas ne rend plus d'erreur, parce
// qu'il n'y a personne à qui se plaindre soixante fois par seconde.
//
// La vue de la scène est posée à la résolution interne si elle ne l'a pas encore été,
// pour qu'un jeu qui ne s'en occupe pas voie quelque chose. Celui qui l'a déjà cadrée
// garde son cadrage : la scène possède l'origine de la vue, et le moteur ne la lui
// reprend pas.
func NewEngine(s *Scene, o Options) (*Engine, error) {
	if s == nil {
		return nil, errors.New("mounting the engine: a scene is required")
	}
	if o.Atlas == nil {
		return nil, errors.New("mounting the engine: an atlas is required, even an empty one")
	}

	o = o.withDefaults()
	if o.Width <= 0 || o.Height <= 0 || o.Width > MaxBufferSide || o.Height > MaxBufferSide {
		return nil, fmt.Errorf("internal resolution of %d×%d: each side must be between 1 and %d",
			o.Width, o.Height, MaxBufferSide)
	}
	if o.Tick <= 0 {
		return nil, fmt.Errorf("step of %v: must be positive", o.Tick)
	}

	if _, _, width, _ := s.View(); width == 0 {
		cells, rows := s.Dimensions()
		s.Focus(float64(cells)/2, float64(rows)/2, int32(o.Width), int32(o.Height))
	}

	return &Engine{
		scene:      s,
		sorter:     sorting.New(s),
		frame:      NewBuffer(uint16(o.Width), uint16(o.Height)),
		atlas:      o.Atlas,
		minWidth:   o.Width,
		minHeight:  o.Height,
		maxWidth:   o.MaxWidth,
		maxHeight:  o.MaxHeight,
		background: o.Background,
		tick:       o.Tick,
	}, nil
}

// Fit dimensionne la vue pour une sortie de la taille donnée, en pixels de fenêtre.
//
// **C'est la voie par laquelle un hôte annonce que sa fenêtre a changé de taille**, et
// la boucle de [Run] l'appelle pour un jeu qui passe par elle. Le facteur
// d'agrandissement reste le plus grand entier qui garde le minimum dans la sortie, et
// la vue devient la sortie divisée par ce facteur, bornée par les options : il reste
// donc au plus un facteur moins un pixel de bande par axe, là où une vue figée en
// laissait jusqu'à 82 % de la fenêtre sur un téléphone en portrait.
//
// Elle ne touche pas au point visé, qui est une donnée du jeu : seule la taille change,
// et la scène recadre autour du même point.
//
// Rien ne se passe si la vue ne change pas, ce qui est le cas courant — une fenêtre
// garde sa taille des milliers d'images de suite. Une taille nulle non plus : c'est ce
// que rend une fenêtre réduite, et la vue d'alors n'a aucune raison de disparaître.
func (e *Engine) Fit(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	w, h := e.viewSize(width, height)
	if w == int(e.frame.Width) && h == int(e.frame.Height) {
		return
	}

	e.frame = NewBuffer(uint16(w), uint16(h))
	e.scene.ResizeView(int32(w), int32(h))
}

// viewSize rend la taille de vue qu'une sortie donnée mérite.
//
// Le facteur vient de [raster.ScaleFactor] et de nulle part ailleurs : c'est lui que
// l'agrandissement appliquera ensuite, et deux arithmétiques qui doivent s'accorder
// finissent par diverger. Les tampons qu'elle lui passe n'ont pas de pixels — la
// fonction ne lit que leurs côtés —, donc ils ne coûtent rien.
func (e *Engine) viewSize(outWidth, outHeight int) (width, height int) {
	minimum := &Buffer{Width: uint16(e.minWidth), Height: uint16(e.minHeight)}
	out := &Buffer{Width: uint16(min(outWidth, MaxBufferSide)), Height: uint16(min(outHeight, MaxBufferSide))}
	factor := raster.ScaleFactor(minimum, out)

	width = min(max(int(out.Width)/factor, e.minWidth), e.maxWidth)
	height = min(max(int(out.Height)/factor, e.minHeight), e.maxHeight)
	return width, height
}

// Step avance le jeu d'un pas et rend l'image qui en résulte prête à présenter.
//
// L'ordre compte et il est le même partout : le jeu décrit, le tri ordonne, le
// rastériseur peint. Rien n'alloue après la première image, et c'est ce qui permet au
// pas de tenir son budget.
//
// Trois choses qu'elle ne fait pas, et qu'un hôte fait donc lui-même. Elle **n'a pas
// d'horloge** : un appel vaut un pas, quel qu'ait été le temps réellement écoulé, et
// [Engine.Tick] dit la durée que le jeu suppose sans que le moteur la tienne — appeler
// deux fois plus souvent fait avancer le jeu deux fois plus vite. Elle **ne présente
// rien** : [Engine.Frame] rend un tampon à la résolution interne, et l'agrandissement
// entier vers une fenêtre appartient à qui présente. Elle **ne demande rien** :
// [Quitter] n'est pas consultée ici, un hôte qui tient sa boucle décidant seul quand
// s'arrêter.
//
// Un jeu nul est admis, et la scène est alors dessinée telle qu'elle est. C'est ce qui
// rend une image d'une scène qu'on vient de construire, sans avoir à écrire une
// méthode vide pour y arriver.
func (e *Engine) Step(g Game, in Input) {
	if g != nil {
		g.Step(in)
	}

	// La forme des planches part au trieur, pour la mesure exacte du recouvrement.
	// Après le pas du jeu, parce qu'il a pu en ajouter une ; et par comparaison des
	// comptes, donc sans rien faire dans le cas courant où l'atlas ne bouge plus.
	for i := e.sorter.Sheets(); i < len(e.atlas.Sheets); i++ {
		s := &e.atlas.Sheets[i]
		e.sorter.AddSheet(s.Width, s.Height, s.Pixels)
	}

	// La taille de la vue se réimpose après le pas du jeu, et c'est le moteur qui la
	// possède depuis que la vue suit la fenêtre. Un [Scene.Focus] appelé avec une
	// autre taille — ce que fait tout jeu qui recentre sa caméra — donnerait sinon une
	// image coupée ou décentrée, sans erreur ni trace : mesuré, un cadrage à 960×540
	// sur un tampon de 480×270 ne peint que la moitié de l'entité visée. Le point
	// visé, lui, reste au jeu, et c'est pourquoi c'est la taille seule qu'on reprend.
	e.scene.ResizeView(int32(e.frame.Width), int32(e.frame.Height))

	list := e.sorter.Frame()
	raster.Clear(e.frame, e.background)
	raster.Draw(e.frame, list, e.atlas)
}

// Frame rend l'image du dernier pas, à la résolution interne.
//
// Le tampon est celui du moteur et il est réécrit au pas suivant : un appelant qui
// veut le garder en fait une copie. L'agrandissement vers une fenêtre appartient à
// celui qui présente, et il est entier — jamais fractionnaire.
func (e *Engine) Frame() *Buffer {
	return e.frame
}

// Tick rend la durée d'un pas, telle que les options l'ont fixée.
//
// Un hôte qui tient sa propre boucle en a besoin : c'est le contrat de temps que le
// jeu suppose, et le lui cacher l'obligerait à le deviner.
func (e *Engine) Tick() time.Duration {
	return e.tick
}

// Overlaps rend, pour chaque entité qu'un jeu a marquée par le champ `Measure` d'une
// [Entity], combien de ses pixels le dernier pas a recouverts et quelle part vient du
// décor.
//
// **C'est la mesure que le tri fait de toute façon pour décider d'une révélation.** Le
// champ `Reveal` d'un [Aspect] redessine la chose, et un rappel d'une autre nature — un
// contour, une flèche, une silhouette tirée d'un autre dessin — demandait de refaire ce
// calcul en entier. Le relevé le rend : l'hôte y prend son propre seuil et dessine ce
// qu'il veut, en marquant son dessin d'[Aspect.Uncounted] pour qu'il ne se compte pas.
//
// Ce que ce nombre mesure exactement, et ce sur quoi on ne doit pas s'appuyer, est dit
// par [render.Overlap.Hidden].
//
// **Il porte le pas précédent**, et c'est inhérent : le recouvrement ne se connaît
// qu'après le tri, donc après le pas. Sans conséquence pour un rappel de lisibilité à
// soixante images par seconde, mais un jeu qui compare ce relevé à une position l'y
// comparera décalé d'un pas.
//
// La tranche est celle du moteur et elle est réécrite au pas suivant. Elle est vide avant
// le premier pas, et un jeu qui ne marque rien n'y trouve jamais rien.
func (e *Engine) Overlaps() OverlapList {
	return e.sorter.Overlaps()
}
