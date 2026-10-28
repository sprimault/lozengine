// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La cadence de la boucle et la mise à l'échelle vers la fenêtre, éprouvées seules.
//
// Ce sont les deux seules parties de [Run] qui s'éprouvent : le reste est une fenêtre,
// et aucun test n'en ouvre — l'intégration continue tourne là où rien ne garantit
// qu'une fenêtre puisse s'ouvrir. La boucle elle-même se vérifie à la main, une fois,
// comme le dit la règle des plateformes.
//
// Et c'est ici que vit le défaut : une cadence qui dérive ne se voit pas en lisant le
// code, elle se voit après trente secondes de jeu.

package lozengine

import (
	"testing"
	"time"
)

// TestFitPaintsTheBarsWithTheGivenColour vérifie que ce qui entoure l'image porte la
// couleur des bandes, et que l'image reste centrée et nette.
//
// Le cas est construit pour que les bandes existent des deux côtés : une image interne
// de 4×2 dans une fenêtre de 10×5 donne le facteur 2, donc une image de 8×4 centrée, un
// pixel de bande à gauche, un à droite et une ligne en bas — le reste d'un centrage
// impair allant à droite et en bas, ce que la godoc d'Upscale fixe.
func TestFitPaintsTheBarsWithTheGivenColour(t *testing.T) {
	white := Color{R: 255, G: 255, B: 255, A: 255}
	bars := Color{R: 200, A: 255}

	inner := NewBuffer(4, 2)
	for i := range inner.Pixels {
		inner.Pixels[i] = white
	}
	out := NewBuffer(10, 5)

	fit(inner, out, bars)

	at := func(x, y int) Color { return out.Pixels[y*int(out.Width)+x] }
	cases := []struct {
		name string
		x, y int
		want Color
	}{
		{"la bande de gauche", 0, 0, bars},
		{"la bande de droite", 9, 0, bars},
		{"la ligne du bas", 0, 4, bars},
		{"le centre de l'image", 4, 2, white},
		{"le premier pixel de l'image", 1, 0, white},
	}
	for _, c := range cases {
		if got := at(c.x, c.y); got != c.want {
			t.Errorf("%s en (%d, %d) : %v, attendu %v", c.name, c.x, c.y, got, c.want)
		}
	}
}

// TestPaceDoesNotDrift garde la correction qui vaut tout ce lot : la date du pas
// suivant se calcule en absolu.
//
// Dormir d'une durée fixe à chaque tour ajoute à chaque fois ce que l'horloge arrondit.
// Mesuré le 25 octobre 2026 : `time.Sleep(16ms)` rend de quinze à trente et une
// millisecondes sous Windows selon ce que font les autres processus, et une boucle qui
// s'en contente tombe à cinquante-sept images par seconde au lieu de soixante. En
// visant une date, un tour trop long est absorbé par le suivant.
func TestPaceDoesNotDrift(t *testing.T) {
	const tick = 10 * time.Millisecond

	debut := time.Now()
	next := debut.Add(tick)

	// Chaque tour prend six millisecondes de plus que prévu, ce qu'une boucle naïve
	// ajouterait à son retard.
	now := debut
	for range 10 {
		now = now.Add(6 * time.Millisecond)
		wait, after := pace(now, next, tick)
		now = now.Add(wait)
		next = after
	}

	// Dix pas de dix millisecondes font cent millisecondes, et le retard ne s'est pas
	// accumulé : chaque tour a attendu moins, pas davantage.
	ecoule := now.Sub(debut)
	if ecoule < 95*time.Millisecond || ecoule > 105*time.Millisecond {
		t.Errorf("dix pas de %v ont pris %v, attendu cent millisecondes", tick, ecoule)
	}
}

// TestPaceCatchesUpWithinTwoSteps vérifie qu'un tour trop long est absorbé par le
// suivant, et non reporté.
func TestPaceCatchesUpWithinTwoSteps(t *testing.T) {
	const tick = 10 * time.Millisecond

	debut := time.Now()
	next := debut.Add(tick)

	// Un tour qui a duré quinze millisecondes au lieu de dix.
	wait, after := pace(debut.Add(15*time.Millisecond), next, tick)

	if wait > 0 {
		t.Errorf("attente de %v alors que le pas est déjà dépassé", wait)
	}
	if got := after.Sub(next); got != tick {
		t.Errorf("le pas suivant est à +%v de l'ancien, attendu +%v", got, tick)
	}
}

// TestPaceResetsWhenFarBehind vérifie qu'un retard considérable ne se rattrape pas.
//
// Après une pause du système — une mise en veille, un point d'arrêt —, rattraper
// rejouerait des centaines d'images d'un coup et le jeu sauterait. Le seuil est de deux
// pas, assez pour absorber une image longue sans ouvrir la porte au rattrapage en
// rafale.
func TestPaceResetsWhenFarBehind(t *testing.T) {
	const tick = 10 * time.Millisecond

	next := time.Now()
	tard := next.Add(5 * time.Second)

	wait, after := pace(tard, next, tick)

	if wait != 0 {
		t.Errorf("attente de %v après cinq secondes de retard", wait)
	}
	if got := after.Sub(tard); got != tick {
		t.Errorf("le pas suivant est à +%v de maintenant, attendu +%v", got, tick)
	}
	if after.Before(tard) {
		t.Error("le pas suivant est dans le passé : le retard se rattraperait en rafale")
	}
}

// TestPaceWaitsWhenAhead vérifie le cas ordinaire, celui d'une machine qui a du temps
// devant elle : la boucle attend ce qu'il faut et pas plus.
func TestPaceWaitsWhenAhead(t *testing.T) {
	const tick = 10 * time.Millisecond

	now := time.Now()
	next := now.Add(4 * time.Millisecond)

	wait, after := pace(now, next, tick)

	if wait != 4*time.Millisecond {
		t.Errorf("attente de %v, attendu quatre millisecondes", wait)
	}
	if got := after.Sub(next); got != tick {
		t.Errorf("le pas suivant est à +%v, attendu +%v", got, tick)
	}
}

// leaving est un jeu qui demande à partir dès son premier pas.
type leaving struct{ steps int }

func (g *leaving) Step(Input)     { g.steps++ }
func (g *leaving) Quitting() bool { return g.steps > 0 }

// staying est un jeu qui n'écrit que la méthode unique, donc sans [Quitter].
type staying struct{ steps int }

func (g *staying) Step(Input) { g.steps++ }

// watching est un jeu qui reçoit le relevé de recouvrement, donc avec [Overlapped].
type watching struct {
	steps  int
	served int
}

func (g *watching) Step(Input)                { g.steps++ }
func (g *watching) Overlaps(list OverlapList) { g.served++; _ = list }

// TestQuitterIsOptional vérifie que l'assertion de la boucle ne réclame rien : un jeu
// qui n'écrit que sa méthode unique reste un jeu valide.
//
// C'est la propriété qui rend l'ajout purement additif. Si l'assertion devenait une
// exigence, tous les jeux écrits jusqu'ici cesseraient de compiler — et les deux
// exemples du dépôt sont dans ce cas pour l'un d'eux.
func TestQuitterIsOptional(t *testing.T) {
	var plain Game = &staying{}
	if _, ok := plain.(Quitter); ok {
		t.Error("un jeu sans Quitting a été pris pour un Quitter")
	}

	var asking Game = &leaving{}
	if _, ok := asking.(Quitter); !ok {
		t.Error("un jeu avec Quitting n'a pas été reconnu")
	}
}

// TestOverlappedIsOptional garde la même propriété pour la seconde interface
// facultative : un jeu qui ne veut pas du relevé n'écrit rien de plus.
//
// Les deux sens sont vérifiés, et c'est ce qui compte : une assertion devenue exigence
// casserait tous les jeux écrits jusqu'ici, et une assertion qui ne reconnaîtrait pas
// un jeu qui la porte lui retirerait sans un mot ce qu'il a écrit pour recevoir.
func TestOverlappedIsOptional(t *testing.T) {
	var plain Game = &staying{}
	if _, ok := plain.(Overlapped); ok {
		t.Error("un jeu sans Overlaps a été pris pour un Overlapped")
	}

	var served Game = &watching{}
	if _, ok := served.(Overlapped); !ok {
		t.Error("un jeu avec Overlaps n'a pas été reconnu")
	}
}

// TestEngineServesTheOverlapsItMeasured garde que le relevé traverse bien la seconde
// porte d'entrée, celle qu'un hôte emploie sans boucle du moteur.
//
// Sans ce cas, le relevé pourrait n'être rendu que par la boucle de [Run] — que la suite
// ne peut pas exercer sans fenêtre —, et un hôte le trouverait vide sans rien pour le
// lui dire. Il garde aussi le décalage d'une image que la godoc annonce : rien avant le
// premier pas, le relevé du pas après.
func TestEngineServesTheOverlapsItMeasured(t *testing.T) {
	s, atlas := scenario(t)

	figure := Aspect{Width: 16, Height: 16, Tint: White()}

	h, ok := s.Place(Entity{X: 3.5, Y: 3.5, Aspect: figure, Measure: true})
	if !ok {
		t.Fatal("entité suivie refusée")
	}
	if _, ok := s.Place(Entity{X: 3.5, Y: 4.5, Aspect: figure}); !ok {
		t.Fatal("entité qui cache refusée")
	}

	e, err := NewEngine(s, Options{Atlas: atlas})
	if err != nil {
		t.Fatalf("montage du moteur : %v", err)
	}

	if n := len(e.Overlaps()); n != 0 {
		t.Errorf("%d relevés avant le premier pas, attendu aucun", n)
	}

	e.Step(&staying{}, Input{})

	list := e.Overlaps()
	if len(list) != 1 {
		t.Fatalf("%d relevés après un pas, attendu 1", len(list))
	}
	if list[0].Identity != uint32(h) {
		t.Errorf("poignée %d, attendu %d", list[0].Identity, uint32(h))
	}
}

// TestEngineDoesNotAskToQuit garde la frontière entre les deux portes d'entrée :
// l'arrêt est une affaire de boucle, et l'étape appelable n'en sait rien.
//
// Un hôte qui tient sa propre boucle décide seul quand s'arrêter et n'a personne à qui
// le demander ; si [Engine.Step] consultait [Quitter], la question traverserait l'ABI
// du jalon 11, où elle n'a pas de sens. Le cas le vérifie par le seul moyen possible
// sans fenêtre : un jeu qui demande à partir dès son premier pas est avancé une seconde
// fois, parce que rien dans l'étape ne l'en empêche.
func TestEngineDoesNotAskToQuit(t *testing.T) {
	s, atlas := scenario(t)

	e, err := NewEngine(s, Options{Atlas: atlas})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	g := &leaving{}
	e.Step(g, Input{})
	e.Step(g, Input{})

	if g.steps != 2 {
		t.Errorf("pas avancés : %d, attendu 2 — l'étape a tenu compte de Quitting", g.steps)
	}
}
