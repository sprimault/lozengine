// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'étape éprouvée sans fenêtre et sans horloge, ce qui est exactement sa raison
// d'être : si elle demandait l'une ou l'autre, un hôte ne pourrait pas la piloter et
// la suite de non-régression ne pourrait pas la juger.

package lozengine

import (
	"fmt"
	"testing"
	"time"

	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// scenario monte une scène minimale et son atlas d'une planche unie, de quoi faire une
// image dont on peut compter les pixels.
func scenario(t *testing.T) (*Scene, *Atlas) {
	t.Helper()

	s, err := NewScene(8, 8, Grid(16), SceneOptions{})
	if err != nil {
		t.Fatalf("montage de la scène : %v", err)
	}

	sheet := NewSheet(16, 16)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = Color{R: 255, G: 255, B: 255, A: 255}
	}

	var atlas Atlas
	if _, err := atlas.Add(sheet); err != nil {
		t.Fatalf("ajout de la planche : %v", err)
	}
	return s, &atlas
}

// counter est un jeu qui ne fait que compter ses pas, pour vérifier que l'étape
// l'appelle une fois et une seule.
type counter struct {
	steps int
	seen  Input
}

// Step compte le pas et retient les entrées qu'il a reçues.
func (c *counter) Step(in Input) {
	c.steps++
	c.seen = in
}

// TestNewEngineRefuses vérifie que tout ce qui peut échouer échoue au montage, parce
// qu'un pas ne rend pas d'erreur : il n'y a personne à qui se plaindre soixante fois
// par seconde.
func TestNewEngineRefuses(t *testing.T) {
	s, atlas := scenario(t)

	cases := []struct {
		name  string
		scene *Scene
		o     Options
	}{
		{"sans scène", nil, Options{Atlas: atlas}},
		{"sans atlas", s, Options{}},
		{"largeur négative", s, Options{Atlas: atlas, Width: -1}},
		{"hauteur au-delà de la borne", s, Options{Atlas: atlas, Height: MaxBufferSide + 1}},
		{"pas négatif", s, Options{Atlas: atlas, Tick: -time.Second}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewEngine(c.scene, c.o); err == nil {
				t.Error("montage accepté")
			}
		})
	}
}

// TestNewEngineDefaults vérifie les défauts que l'invariant des options promet : ce
// qui n'est pas renseigné en a un, et un jeu qui ne règle rien obtient une image
// utilisable.
func TestNewEngineDefaults(t *testing.T) {
	s, atlas := scenario(t)

	e, err := NewEngine(s, Options{Atlas: atlas})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	frame := e.Frame()
	if frame.Width != defaultWidth || frame.Height != defaultHeight {
		t.Errorf("résolution interne : %d×%d, attendu %d×%d",
			frame.Width, frame.Height, defaultWidth, defaultHeight)
	}
	if e.Tick() != defaultTick {
		t.Errorf("pas : %v, attendu %v", e.Tick(), defaultTick)
	}
	if e.background != (Color{A: 255}) {
		t.Errorf("fond : %v, attendu un noir opaque", e.background)
	}
}

// TestBarsDefaultApartFromTheBackground vérifie que les bandes ne prennent pas la
// couleur du fond du monde quand le jeu ne demande rien pour elles.
//
// C'est le cas qui garde la raison d'être du champ, et il est le seul à pouvoir le
// faire : si le défaut recopiait Background, un jeu au ciel clair obtiendrait des bandes
// claires sans les avoir demandées, et séparer les deux champs n'aurait servi à rien.
// fit reçoit la couleur, il ne la choisit pas, donc son propre cas ne dirait rien.
func TestBarsDefaultApartFromTheBackground(t *testing.T) {
	sky := Color{R: 120, G: 180, B: 230, A: 255}
	o := Options{Background: sky}.withDefaults()

	if o.Background != sky {
		t.Errorf("fond : %v, attendu %v — le défaut a écrasé ce que le jeu demandait",
			o.Background, sky)
	}
	if want := (Color{A: 255}); o.Bars != want {
		t.Errorf("bandes : %v, attendu %v — elles ont suivi le fond du monde", o.Bars, want)
	}
}

// TestNewEngineFramesTheViewOnce vérifie les deux moitiés d'une même règle : la scène
// possède l'origine de la vue, et le moteur la pose seulement si personne ne l'a fait.
func TestNewEngineFramesTheViewOnce(t *testing.T) {
	t.Run("vue vide, le moteur la pose", func(t *testing.T) {
		s, atlas := scenario(t)

		if _, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32}); err != nil {
			t.Fatalf("montage : %v", err)
		}
		if _, _, width, height := s.View(); width != 64 || height != 32 {
			t.Errorf("vue : %d×%d, attendu 64×32", width, height)
		}
	})

	t.Run("vue déjà cadrée, le moteur n'y touche pas", func(t *testing.T) {
		s, atlas := scenario(t)
		s.Focus(1, 1, 40, 20)

		if _, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32}); err != nil {
			t.Fatalf("montage : %v", err)
		}
		if _, _, width, height := s.View(); width != 40 || height != 20 {
			t.Errorf("vue : %d×%d, attendu 40×20 — le cadrage du jeu a été reprise", width, height)
		}
	})
}

// TestFitSizesTheViewOnTheOutput vérifie la règle entière du dimensionnement, qui est
// la seule chose que ce lot ajoute et dont tout le reste découle.
//
// **La règle maximise le facteur d'abord, et la vue n'absorbe que le reste.** C'est le
// fait que les cas disent ensemble, et il n'est pas celui qu'on attend : une sortie
// exactement double du minimum garde la vue au minimum et double le pixel, là qu'une
// sortie qui ne tombe pas juste élargit la vue de ce que le facteur ne peut pas
// prendre. Les deux donnent zéro bande, par deux moyens opposés.
//
// D'où ce que chaque cas est seul à montrer : le multiple exact, qui ne doit pas
// élargir ; le reste, qui doit l'être ; l'axe non limitant, qui s'étire librement et
// explique qu'une aire visible atteigne cinq fois le minimum ; le plafond, qui ne mord
// que sur ce reste ; un plafond égal au minimum, qui rend le figé de la 0.5.3 ; et une
// sortie sous le minimum, qui ne descend pas, parce qu'une interface posée dans ce
// rectangle doit y tenir.
//
// Les trois derniers cas tiennent l'ordre des deux primitifs : le retrait sort de la
// surface avant que le plafond ne borne, et il mord dès la taille d'ouverture, là qu'un
// plafond seul n'a rien à retirer tant que la fenêtre ne s'est pas étirée.
//
// **Le cas qui les porte tous les deux distingue trois résultats, et c'est ce qui le rend
// mordant** : 65 pour l'ordre retenu, 60 si le plafond bornait d'abord — il mangerait
// alors la marge que le retrait garantit —, 70 si le retrait était ignoré. Les deux
// défauts se lisent donc l'un de l'autre, là qu'une seule valeur attendue les
// confondrait.
func TestFitSizesTheViewOnTheOutput(t *testing.T) {
	for _, c := range []struct {
		name                       string
		minW, minH                 int
		maxW, maxH                 int
		inset                      Inset
		outW, outH                 int
		wantSurfaceW, wantSurfaceH int
		wantWorldW, wantWorldH     int
		wantFactor                 int
	}{
		{"sortie au minimum", 64, 32, 0, 0, Inset{}, 64, 32, 64, 32, 64, 32, 1},
		{"multiple exact, le pixel grossit et la surface tient", 64, 32, 0, 0, Inset{}, 128, 64, 64, 32, 64, 32, 2},
		{"reste au facteur un, la surface l'absorbe", 64, 32, 0, 0, Inset{}, 100, 50, 100, 50, 100, 50, 1},
		{"reste au facteur deux", 64, 32, 0, 0, Inset{}, 150, 70, 75, 35, 75, 35, 2},
		{"axe non limitant, la surface s'étire seule", 64, 32, 0, 0, Inset{}, 192, 32, 192, 32, 192, 32, 1},
		{"plafond, la surface remplit et l'aire seule est bornée", 64, 32, 70, 40, Inset{}, 150, 70, 75, 35, 70, 35, 2},
		{"plafond égal au minimum, l'aire est figée et la marge prend le reste", 64, 32, 64, 32, Inset{}, 150, 70, 75, 35, 64, 32, 2},
		{"plafond sous le plancher, une marge dès la taille d'ouverture", 64, 32, 64, 24, Inset{}, 64, 32, 64, 32, 64, 24, 1},
		{"sortie sous le minimum, la surface ne descend pas", 64, 32, 0, 0, Inset{}, 32, 16, 64, 32, 64, 32, 1},
		{"retrait, l'aire prend tout ce qui reste", 64, 32, 0, 0, Inset{Bottom: 8}, 150, 70, 75, 35, 75, 27, 2},
		{"retrait et plafond, le retrait sort d'abord", 64, 32, 70, 40, Inset{Left: 5, Right: 5}, 150, 70, 75, 35, 65, 35, 2},
		{"retrait dès l'ouverture, sans plafond ni fenêtre étirée", 64, 32, 0, 0, Inset{Top: 10}, 64, 32, 64, 32, 64, 22, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, atlas := scenario(t)
			e, err := NewEngine(s, Options{
				Atlas: atlas, Width: c.minW, Height: c.minH,
				MaxWidth: c.maxW, MaxHeight: c.maxH,
				WorldInset: c.inset,
			})
			if err != nil {
				t.Fatalf("montage : %v", err)
			}

			e.Fit(c.outW, c.outH)

			if got := e.Frame(); int(got.Width) != c.wantSurfaceW || int(got.Height) != c.wantSurfaceH {
				t.Errorf("tampon %d×%d, attendu %d×%d",
					got.Width, got.Height, c.wantSurfaceW, c.wantSurfaceH)
			}
			if w, h := s.Surface(); int(w) != c.wantSurfaceW || int(h) != c.wantSurfaceH {
				t.Errorf("surface %d×%d, attendu %d×%d", w, h, c.wantSurfaceW, c.wantSurfaceH)
			}
			if _, _, w, h := s.View(); int(w) != c.wantWorldW || int(h) != c.wantWorldH {
				t.Errorf("aire de monde %d×%d, attendu %d×%d", w, h, c.wantWorldW, c.wantWorldH)
			}

			// Le facteur que l'agrandissement appliquera : c'est lui qui dit que
			// l'invariant tient, et qu'il ne reste au plus qu'un pixel de bande par
			// axe — ce qui ne dépend plus d'un plafond depuis que la surface remplit.
			out := &Buffer{Width: uint16(c.outW), Height: uint16(c.outH)}
			if got := raster.ScaleFactor(e.Frame(), out); got != c.wantFactor {
				t.Errorf("facteur %d, attendu %d", got, c.wantFactor)
			}
		})
	}
}

// TestWorldInsetHoldsAcrossWindowSizes garde la raison d'être du retrait, et c'est la
// seule chose qu'un plafond ne sait pas faire.
//
// **Le défaut n'était pas la taille de la marge, c'était qu'elle varie.** Mesuré sur le
// jeu avec un plafond : 77 pixels de marge à l'ouverture, 221 une fois la fenêtre
// maximisée. Un bandeau dimensionné pour l'un flotte dans l'autre, donc le jeu doit
// porter une mise en page conditionnelle — avec un saut pendant qu'on étire — qui est
// exactement ce que le plafond existait pour lui retirer.
//
// Le cas éprouve les deux régimes côte à côte sur les mêmes deux fenêtres, parce que
// c'est le **contraste** qui est la propriété : la marge du retrait est la même aux deux
// tailles, celle du plafond non. Un test qui ne tiendrait que le retrait passerait encore
// si le retrait s'était mis à se comporter comme un plafond sur une seule fenêtre.
func TestWorldInsetHoldsAcrossWindowSizes(t *testing.T) {
	const (
		small, large = 128, 300
		margin       = 8
	)

	s, atlas := scenario(t)
	inset, err := NewEngine(s, Options{
		Atlas: atlas, Width: 64, Height: 32, WorldInset: Inset{Bottom: margin},
	})
	if err != nil {
		t.Fatalf("montage du retrait : %v", err)
	}

	ceiled, atlas := scenario(t)
	ceiling, err := NewEngine(ceiled, Options{
		Atlas: atlas, Width: 64, Height: 32, MaxHeight: 32 - margin,
	})
	if err != nil {
		t.Fatalf("montage du plafond : %v", err)
	}

	// La marge que chaque régime laisse, aux deux tailles de fenêtre : ce que la surface
	// a de plus que l'aire sur l'axe vertical.
	marginOf := func(e *Engine, sc *Scene, out int) int32 {
		e.Fit(out, out)
		_, surface := sc.Surface()
		_, _, _, area := sc.View()
		return surface - area
	}

	insetSmall, insetLarge := marginOf(inset, s, small), marginOf(inset, s, large)
	if insetSmall != margin || insetLarge != margin {
		t.Errorf("marge du retrait de %d puis %d pixels, attendue %d aux deux tailles — "+
			"une marge qui varie oblige le jeu à la mise en page conditionnelle",
			insetSmall, insetLarge, margin)
	}

	ceilingSmall, ceilingLarge := marginOf(ceiling, ceiled, small), marginOf(ceiling, ceiled, large)
	if ceilingSmall == ceilingLarge {
		t.Errorf("marge du plafond de %d pixels aux deux tailles — le cas ne conclut pas, "+
			"les deux régimes devant se distinguer pour que le contraste soit mesuré",
			ceilingSmall)
	}
}

// TestStepKeepsTheWorldInset garde le retrait contre la réimposition du pas, qui est
// l'endroit exact où il se perdrait.
//
// **[Engine.Step] réimpose les deux rectangles après le pas du jeu**, et c'est pour cela
// qu'un jeu ne peut pas obtenir un retrait en appelant [Scene.ResizeView] lui-même : ce
// qu'il pose est écrasé avant que l'image soit assemblée. Le retrait doit donc se relire
// à chaque pas plutôt que se figer au montage — sans quoi une interface qui se replie en
// cours de partie ne rendrait jamais au monde ce qu'elle occupait, et rien ne le dirait.
//
// Le cas l'éprouve dans l'ordre où le défaut se produit : un pas d'abord, pour que la
// réimposition ait lieu une fois ; le retrait ensuite ; puis un second pas, qui est celui
// qui doit en tenir compte.
func TestStepKeepsTheWorldInset(t *testing.T) {
	s, atlas := scenario(t)
	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	e.Step(nil, Input{})
	if _, _, _, height := s.View(); height != 32 {
		t.Fatalf("aire de monde haute de %d avant tout retrait, attendue 32", height)
	}

	s.SetWorldInset(Inset{Top: 10})
	e.Step(nil, Input{})

	if _, _, _, height := s.View(); height != 22 {
		t.Errorf("aire de monde haute de %d après un retrait de 10, attendue 22 — "+
			"le pas a réimposé la vue sans relire le retrait", height)
	}
	if _, y := s.WorldOrigin(); y != 10 {
		t.Errorf("origine de l'aire en y à %d, attendue 10", y)
	}
}

// TestFitKeepsTheAim vérifie que redimensionner ne déplace pas ce qu'on regarde.
//
// C'est la propriété qui sépare cette voie de [Scene.Focus] : le point visé est une
// donnée du jeu, et une fenêtre étirée n'a aucune raison de le déplacer. Le cas
// l'éprouve sur un point décentré, le centre de la grille ne bougeant pas même quand
// on le recalcule faux.
func TestFitKeepsTheAim(t *testing.T) {
	s, atlas := scenario(t)
	s.Focus(2, 6, 64, 32)

	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	beforeX, beforeY, _, _ := s.View()

	// Une sortie qui ne tombe pas juste, donc que la vue absorbe : un multiple exact
	// du minimum grossirait le pixel sans changer la vue, et ne prouverait rien ici.
	e.Fit(100, 50)
	afterX, afterY, w, h := s.View()

	if w != 100 || h != 50 {
		t.Fatalf("vue %d×%d, attendu 100×50", w, h)
	}

	// L'origine se déplace de la moitié de ce que la vue a gagné, puisque le point
	// visé reste au centre : c'est la seule valeur qui garde le même point au milieu.
	if wantX, wantY := beforeX-18, beforeY-9; afterX != wantX || afterY != wantY {
		t.Errorf("origine (%d, %d), attendu (%d, %d) — le point visé a bougé",
			afterX, afterY, wantX, wantY)
	}
}

// TestStepReimposesTheViewSize garde ce qui corrige un cas silencieusement faux.
//
// Un jeu qui recentre sa caméra appelle [Scene.Focus], qui prend une taille : s'il
// passe la sienne plutôt que celle du moteur, la scène se croit plus grande ou plus
// petite que le tampon, et l'image sort coupée ou décentrée sans qu'aucune erreur ne
// le dise. Le pas reprend donc la taille, et la taille seule.
func TestStepReimposesTheViewSize(t *testing.T) {
	s, atlas := scenario(t)

	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	// Ce qu'un jeu écrit de travers, et que rien ne signalait.
	s.Focus(4, 4, 999, 999)
	e.Step(nil, Input{})

	if _, _, w, h := s.View(); w != 64 || h != 32 {
		t.Errorf("vue %d×%d après le pas, attendu 64×32", w, h)
	}
}

// TestFitDoesNothingWhenTheViewHoldds vérifie que le cas courant ne coûte rien : une
// fenêtre garde sa taille des milliers d'images de suite, et refaire le tampon à
// chacune allouerait dans le chemin de rendu.
func TestFitDoesNothingWhenTheViewHolds(t *testing.T) {
	s, atlas := scenario(t)

	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	// Le premier appel change la vue et refait donc le tampon ; ce sont les suivants,
	// à taille stable, qui ne doivent rien coûter.
	e.Fit(100, 50)
	settled := e.Frame()

	if n := testing.AllocsPerRun(10, func() { e.Fit(100, 50) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune à taille stable", n)
	}
	if e.Frame() != settled {
		t.Error("le tampon a été refait alors que la vue ne changeait pas")
	}
}

// TestStepCallsTheGameOnce vérifie que l'étape appelle le jeu une fois par pas, et lui
// passe les entrées qu'elle a reçues.
//
// Une seule fois, parce que le front montant des touches s'arme à la lecture : deux
// appels par pas rendraient la seconde lecture vide et le jeu ne verrait plus rien
// venir.
func TestStepCallsTheGameOnce(t *testing.T) {
	s, atlas := scenario(t)
	e, err := NewEngine(s, Options{Atlas: atlas, Width: 32, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	// Les entrées se fabriquent sans clavier, par le même type qu'un jeu emploie
	// pour éprouver sa propre méthode.
	var state State
	state.Press(KeyW)
	in := state.Frame()

	g := &counter{}
	e.Step(g, in)

	if g.steps != 1 {
		t.Errorf("pas joués : %d, attendu 1", g.steps)
	}
	if !g.seen.Held(KeyW) {
		t.Error("les entrées ne sont pas arrivées au jeu")
	}
}

// TestStepWithoutGame vérifie qu'un hôte peut faire une image sans jeu : il met la
// scène à jour lui-même, et l'étape ne doit pas l'exiger.
func TestStepWithoutGame(t *testing.T) {
	s, atlas := scenario(t)
	e, err := NewEngine(s, Options{Atlas: atlas, Width: 32, Height: 32})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	e.Step(nil, Input{})
}

// TestStepDrawsWhatTheSceneDescribes est le cas qui relie les trois étages : le jeu
// décrit, le tri ordonne, le rastériseur peint.
//
// Ce qu'il est seul à pouvoir attraper : un étage débranché. Chacun a ses propres
// tests, et ils passeraient tous si l'étape oubliait d'appeler le tri ou de peindre.
func TestStepDrawsWhatTheSceneDescribes(t *testing.T) {
	s, atlas := scenario(t)
	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 64})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	// Rien posé : l'image est du fond, et rien d'autre.
	e.Step(nil, Input{})
	if n := peints(e.Frame(), Color{A: 255}); n != 0 {
		t.Fatalf("%d pixels peints sur une scène vide", n)
	}

	// Une entité au centre de la vue, avec la planche unie de l'atlas.
	if _, ok := s.Place(Entity{
		X: 4, Y: 4,
		Aspect: Aspect{Width: 16, Height: 16, Tint: White()},
	}); !ok {
		t.Fatal("la pose a été refusée")
	}

	e.Step(nil, Input{})
	if n := peints(e.Frame(), Color{A: 255}); n == 0 {
		t.Error("l'entité posée n'a rien peint")
	}
}

// TestStepDoesNotAllocateAfterTheFirst vérifie la propriété dont le budget d'une image
// dépend : passé la première, un pas ne touche plus au tas.
func TestStepDoesNotAllocateAfterTheFirst(t *testing.T) {
	s, atlas := scenario(t)
	e, err := NewEngine(s, Options{Atlas: atlas, Width: 64, Height: 64})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	// Les chemins ajoutés depuis le jalon 4 doivent être exercés, sans quoi la garde
	// ne couvre que le monde : sans pièce d'écran, la collecte sort à son test de liste
	// vide et la bascule entre les deux repères ne tourne jamais ; sans entité marquée,
	// le relevé de recouvrement non plus.
	s.Place(Entity{X: 4, Y: 4, Aspect: Aspect{Width: 16, Height: 16, Tint: White()},
		Measure: true, Tier: 2})
	s.PlaceScreen(2, 2, Aspect{Width: 8, Height: 8, Tint: White(), Layer: 3})
	s.PlaceArea(4, 4, Aspect{Width: 8, Height: 8, Tint: White(), Layer: 3})

	e.Step(nil, Input{})

	if n := testing.AllocsPerRun(10, func() { e.Step(nil, Input{}) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
	if len(e.Overlaps()) == 0 {
		t.Error("aucun relevé de recouvrement : le chemin marqué n'a pas été exercé")
	}
}

// peints compte les pixels qui diffèrent du fond.
func peints(b *render.Buffer, background Color) int {
	n := 0
	for _, p := range b.Pixels {
		if p != background {
			n++
		}
	}
	return n
}

// TestPaintedMatchesWhatTheRasteriserDraws confronte la mesure exacte au rastériseur,
// qui est la seule vérité de ce qu'une image porte.
//
// **C'est le cas qui ferme la limite du test croisé de `sorting`** : là-bas, la passe
// par mots et sa référence pixel par pixel partagent la convention de miroir, donc leur
// accord ne la valide pas. Ici, le compte vient du tampon peint, et un miroir inversé
// autour de la planche plutôt qu'autour du rectangle du quad rougirait — c'est le
// défaut que la première écriture de ce lot portait, et qu'aucun autre cas n'aurait vu
// sur une planche d'une seule case.
func TestPaintedMatchesWhatTheRasteriserDraws(t *testing.T) {
	for _, flags := range []Flags{0, MirrorX, MirrorY, MirrorX | MirrorY} {
		t.Run(fmt.Sprintf("drapeaux_%d", flags), func(t *testing.T) {
			s, err := NewScene(8, 8, Grid(16), SceneOptions{Entities: 2})
			if err != nil {
				t.Fatalf("montage de la scène : %v", err)
			}

			// Une planche de deux cases côte à côte, dont une seule est prise : c'est
			// ce qui distingue un miroir du rectangle d'un miroir de la planche. La
			// case retenue porte un motif asymétrique.
			sheet := NewSheet(32, 8)
			for y := range 8 {
				for x := range 32 {
					if x >= 16 && x < 16+4+y%3 {
						sheet.Pixels[y*32+x] = Color{R: 255, G: 255, B: 255, A: 255}
					}
				}
			}
			var atlas Atlas
			if _, err := atlas.Add(sheet); err != nil {
				t.Fatalf("ajout de la planche : %v", err)
			}

			if _, ok := s.Place(Entity{X: 4, Y: 4, Measure: true, Aspect: Aspect{
				SourceX: 16, Width: 16, Height: 8, Tint: White(), Flags: flags,
			}}); !ok {
				t.Fatal("entité refusée")
			}

			e, err := NewEngine(s, Options{Atlas: &atlas, Width: 64, Height: 64,
				MaxWidth: 64, MaxHeight: 64, Background: Color{A: 255}})
			if err != nil {
				t.Fatalf("montage : %v", err)
			}
			e.Step(nil, Input{})

			list := e.Overlaps()
			if len(list) != 1 {
				t.Fatalf("%d relevés, attendu 1", len(list))
			}

			// Ce que le rastériseur a réellement posé : tout pixel qui n'est plus le
			// fond. La planche est opaque, donc le compte est exact.
			drawn := uint32(0)
			for _, c := range e.Frame().Pixels {
				if c != (Color{A: 255}) {
					drawn++
				}
			}

			if list[0].Painted != drawn {
				t.Errorf("Painted vaut %d, le rastériseur en a posé %d", list[0].Painted, drawn)
			}
		})
	}
}

// TestClippingMatchesAnUnclippedRender garde le découpage du monde sur le bord de son
// aire, contre une référence que ce découpage ne traverse pas.
//
// **Recouper le relevé contre le rastériseur ne vaut rien ici, et c'est mesuré** : le
// peint et les pixels viennent du même quad déjà découpé, donc ils s'accordent même
// quand le découpage est faux — une mutation du sens du miroir laisse le dépôt vert.
// Le recoupement de [Overlap.Painted] marche parce que le défaut qu'il vise est une
// divergence entre deux implémentations du miroir ; le découpage, lui, est en un seul
// endroit qui alimente les deux.
//
// La référence est donc le **même quad rendu sans plafond**, où rien ne le découpe : les
// pixels de l'aire doivent y être identiques, et la marge rester vide. Les deux rendus
// partagent leur origine de vue parce qu'une cellule de [Grid] vaut seize pixels et que
// l'aire double : décaler le point visé d'une cellule par axe compense exactement, donc
// l'entité tombe au même pixel de la surface des deux côtés.
//
// Trois défauts tombent avec ce cas, et aucun n'est vu ailleurs : le sens du miroir —
// une colonne retirée à gauche de ce qui se peint vient de la **droite** de la source,
// ce qui est invisible sur une planche d'une seule case —, un débordement dans la marge,
// et l'ordre du découpage et de la mesure, puisque [Overlap.Painted] est comparé à ce
// qui est réellement posé dans l'aire.
func TestClippingMatchesAnUnclippedRender(t *testing.T) {
	const side, bounded = 64, 32

	frame := func(t *testing.T, flags Flags, focus float64, maxSide int) ([]Color, uint32) {
		t.Helper()

		s, err := NewScene(16, 16, Grid(16), SceneOptions{Entities: 2})
		if err != nil {
			t.Fatalf("montage de la scène : %v", err)
		}

		// Une planche de deux cases côte à côte, dont une seule est prise : c'est ce
		// qui distingue un miroir du rectangle d'un miroir de la planche.
		//
		// Le motif est une diagonale et non une bande, parce qu'une bande d'un bord
		// tombe entièrement dans ce qu'un miroir fait retirer — mesuré, le témoin du
		// cas a refusé de conclure. Dense et penché, il laisse des pixels peints dans
		// n'importe quelle fenêtre de découpage, et il reste asymétrique sur les deux
		// axes, donc une réflexion prise à l'envers se voit.
		sheet := NewSheet(32, 8)
		for row := range 8 {
			for col := 16; col < 32; col++ {
				if (col-16+row)%4 != 0 {
					sheet.Pixels[row*32+col] = Color{R: 255, G: 255, B: 255, A: 255}
				}
			}
		}
		var atlas Atlas
		if _, err := atlas.Add(sheet); err != nil {
			t.Fatalf("ajout de la planche : %v", err)
		}

		// L'entité est au même endroit du monde des deux côtés, et à cheval sur les
		// deux bords de l'aire bornée : c'est le point visé qui bouge, pour que
		// l'origine de la vue ne bouge pas.
		if _, ok := s.Place(Entity{X: 8.9, Y: 9.2, Measure: true, Aspect: Aspect{
			SourceX: 16, Width: 16, Height: 8, Tint: White(), Flags: flags,
		}}); !ok {
			t.Fatal("entité refusée")
		}
		s.Focus(focus, focus, side, side)

		e, err := NewEngine(s, Options{Atlas: &atlas, Width: side, Height: side,
			MaxWidth: maxSide, MaxHeight: maxSide, Background: Color{A: 255},
			WorldAlignX: AlignStart, WorldAlignY: AlignStart})
		if err != nil {
			t.Fatalf("montage : %v", err)
		}
		e.Step(nil, Input{})

		list := e.Overlaps()
		if len(list) != 1 {
			t.Fatalf("%d relevés, attendu 1", len(list))
		}
		return e.Frame().Pixels, list[0].Painted
	}

	for _, flags := range []Flags{0, MirrorX, MirrorY, MirrorX | MirrorY} {
		t.Run(fmt.Sprintf("drapeaux_%d", flags), func(t *testing.T) {
			whole, _ := frame(t, flags, 9, side)
			cut, painted := frame(t, flags, 8, bounded)

			var differing, spilled, inside uint32
			for y := range side {
				for x := range side {
					i := y*side + x

					if x >= bounded || y >= bounded {
						if cut[i] != (Color{A: 255}) {
							spilled++
						}
						continue
					}
					if cut[i] != whole[i] {
						differing++
					}
					if cut[i] != (Color{A: 255}) {
						inside++
					}
				}
			}

			if differing != 0 {
				t.Errorf("%d pixels de l'aire diffèrent du rendu sans plafond", differing)
			}
			if spilled != 0 {
				t.Errorf("%d pixels ont débordé dans la marge", spilled)
			}
			if painted != inside {
				t.Errorf("Painted vaut %d, l'aire en porte %d — la mesure n'a pas suivi "+
					"le découpage", painted, inside)
			}

			// Deux témoins, sans quoi un rendu vide passerait au vert : il faut des
			// pixels dans l'aire, et il faut que le plafond en ait réellement coupé.
			if inside == 0 {
				t.Fatal("rien n'est peint dans l'aire : le cas n'éprouve rien")
			}
			var full uint32
			for _, c := range whole {
				if c != (Color{A: 255}) {
					full++
				}
			}
			if full <= inside {
				t.Fatalf("le rendu sans plafond peint %d pixels et l'aire %d : "+
					"rien n'a été découpé, le cas n'éprouve rien", full, inside)
			}
		})
	}
}
