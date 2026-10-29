// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'étape éprouvée sans fenêtre et sans horloge, ce qui est exactement sa raison
// d'être : si elle demandait l'une ou l'autre, un hôte ne pourrait pas la piloter et
// la suite de non-régression ne pourrait pas la juger.

package lozengine

import (
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

// compteur est un jeu qui ne fait que compter ses pas, pour vérifier que l'étape
// l'appelle une fois et une seule.
type compteur struct {
	pas int
	vu  Input
}

func (c *compteur) Step(in Input) {
	c.pas++
	c.vu = in
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
func TestFitSizesTheViewOnTheOutput(t *testing.T) {
	for _, c := range []struct {
		name         string
		minW, minH   int
		maxW, maxH   int
		outW, outH   int
		wantW, wantH int
		wantFactor   int
	}{
		{"sortie au minimum", 64, 32, 0, 0, 64, 32, 64, 32, 1},
		{"multiple exact, le pixel grossit et la vue tient", 64, 32, 0, 0, 128, 64, 64, 32, 2},
		{"reste au facteur un, la vue l'absorbe", 64, 32, 0, 0, 100, 50, 100, 50, 1},
		{"reste au facteur deux", 64, 32, 0, 0, 150, 70, 75, 35, 2},
		{"axe non limitant, la vue s'étire seule", 64, 32, 0, 0, 192, 32, 192, 32, 1},
		{"plafond, qui ne mord que sur le reste", 64, 32, 70, 40, 150, 70, 70, 35, 2},
		{"plafond égal au minimum, le figé de la 0.5.3", 64, 32, 64, 32, 150, 70, 64, 32, 2},
		{"sortie sous le minimum, la vue ne descend pas", 64, 32, 0, 0, 32, 16, 64, 32, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, atlas := scenario(t)
			e, err := NewEngine(s, Options{
				Atlas: atlas, Width: c.minW, Height: c.minH,
				MaxWidth: c.maxW, MaxHeight: c.maxH,
			})
			if err != nil {
				t.Fatalf("montage : %v", err)
			}

			e.Fit(c.outW, c.outH)

			if got := e.Frame(); int(got.Width) != c.wantW || int(got.Height) != c.wantH {
				t.Errorf("tampon %d×%d, attendu %d×%d", got.Width, got.Height, c.wantW, c.wantH)
			}
			if _, _, w, h := s.View(); int(w) != c.wantW || int(h) != c.wantH {
				t.Errorf("vue %d×%d, attendu %d×%d", w, h, c.wantW, c.wantH)
			}

			// Le facteur que l'agrandissement appliquera : c'est lui qui dit que
			// l'invariant tient, et qu'il ne reste au plus qu'un pixel de bande par
			// axe quand aucun plafond ne mord.
			out := &Buffer{Width: uint16(c.outW), Height: uint16(c.outH)}
			if got := raster.ScaleFactor(e.Frame(), out); got != c.wantFactor {
				t.Errorf("facteur %d, attendu %d", got, c.wantFactor)
			}
		})
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

	g := &compteur{}
	e.Step(g, in)

	if g.pas != 1 {
		t.Errorf("pas joués : %d, attendu 1", g.pas)
	}
	if !g.vu.Held(KeyW) {
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
	s.Place(Entity{X: 4, Y: 4, Aspect: Aspect{Width: 16, Height: 16, Tint: White()}})

	e.Step(nil, Input{})

	if n := testing.AllocsPerRun(10, func() { e.Step(nil, Input{}) }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
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
