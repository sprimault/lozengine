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
// **La seule interface facultative du moteur, et elle le restera** : la fin de la
// boucle est la seule chose que la boucle ne peut apprendre de nulle part ailleurs.
// Tout le reste de ce qu'un jeu voudrait régler se règle par [Options], avant.
type Quitter interface {
	// Quitting dit que le jeu a fini. La boucle rend la main après l'image en cours,
	// donc l'appel suivant à [Game.Step] n'a pas lieu.
	//
	// Appelée une fois par image, après le pas : un jeu qui décide de partir pendant
	// son pas voit sa décision prise dans la même image, et l'image part quand même
	// — c'est ce qui permet d'afficher un dernier écran.
	Quitting() bool
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

	// Width et Height sont la résolution interne, celle dans laquelle le moteur
	// dessine avant l'agrandissement entier vers la fenêtre. Zéro vaut 480×270, et
	// [MaxBufferSide] est la borne.
	//
	// **La fenêtre s'ouvre à cette taille**, donc l'agrandissement vaut un tant que
	// le joueur ne l'a pas redimensionnée. Un jeu qui se veut grand à l'ouverture
	// demande une résolution interne plus grande, et non une fenêtre plus grande : le
	// facteur est entier, et c'est la résolution interne qui dit ce qu'une tuile
	// couvre.
	Width, Height int

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
		background: o.Background,
		tick:       o.Tick,
	}, nil
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
