// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que coûte un pas entier à la charge de contrôle du projet, et comment il se
// répartit entre ses étapes. C'est le seul banc qui mesure la chaîne complète : les
// autres mesurent un maillon, et leur somme ne dit pas ce qu'une image coûte.

package lozengine

import (
	"testing"

	"github.com/sprimault/lozengine/internal/sorting"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// benchSide et benchEntities sont la charge de contrôle que la doctrine du dépôt fixe :
// 5 000 entités sur une carte de 500×500.
const (
	benchSide     = 500
	benchEntities = 5000
)

// benchEngine monte un moteur sur la charge de contrôle, décor et entités compris.
//
// Les positions se tirent de deux pas premiers avec le côté, ce qui répartit les entités
// sans qu'aucune ne retombe sur une autre. C'est ce que fait déjà le banc du trieur, et
// la charge doit rester la même pour que les deux mesures se comparent.
func benchEngine(b *testing.B, width, height int) (*Engine, *Scene) {
	b.Helper()

	sheet := NewSheet(32, 16)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = render.Color{R: 120, G: 130, B: 110, A: 255}
	}

	figure := NewSheet(12, 24)
	for i := range figure.Pixels {
		figure.Pixels[i] = render.Color{R: 200, G: 120, B: 120, A: 255}
	}

	var atlas Atlas
	if _, err := atlas.Add(sheet); err != nil {
		b.Fatalf("planche de sol : %v", err)
	}
	if _, err := atlas.Add(figure); err != nil {
		b.Fatalf("planche de figure : %v", err)
	}

	s, err := NewScene(benchSide, benchSide, Isometric(32), SceneOptions{Entities: benchEntities})
	if err != nil {
		b.Fatalf("montage de la scène : %v", err)
	}
	s.Focus(benchSide/2, benchSide/2, int32(width), int32(height))

	ground := Aspect{Sheet: 0, Width: 32, Height: 16, Tint: White()}
	for v := range benchSide {
		for u := range benchSide {
			s.PlaceCell(u, v, 0, ground)
		}
	}

	shape := Aspect{Sheet: 1, Width: 12, Height: 24, Tint: White()}
	for i := range benchEntities {
		x := float64((i*37)%benchSide) + 0.5
		y := float64((i*61)%benchSide) + 0.5
		if _, ok := s.Place(Entity{X: x, Y: y, Aspect: shape}); !ok {
			b.Fatalf("entité %d refusée", i)
		}
	}

	e, err := NewEngine(s, Options{Atlas: &atlas, Width: width, Height: height})
	if err != nil {
		b.Fatalf("montage du moteur : %v", err)
	}
	return e, s
}

// BenchmarkStep mesure le pas entier, à la résolution d'ouverture et au-delà.
//
// **C'est le chiffre qui manquait pour appliquer la règle du rastériseur.** Elle demande
// une charge *observée* au-delà de dix pour cent pondérés avant d'autoriser à toucher au
// dessin ; les bancs du dépôt mesurent le tri et le rastériseur séparément, donc ce
// pourcentage se déduisait de mesures prises sur des chemins isolés. Ici la chaîne est
// celle d'une image réelle : le pas du jeu, le tri, l'effacement, le dessin.
//
// Les deux tailles ne sont pas une redite. La surface de dessin suit la fenêtre depuis
// que la vue s'élargit, donc le dessin croît avec elle là que le tri croît avec ce que la
// vue montre : le rapport entre les deux lignes dit laquelle domine, et c'est lui qui est
// un fait du moteur. Les valeurs absolues ne sont que des faits de la machine.
func BenchmarkStep(b *testing.B) {
	for _, c := range []struct {
		name          string
		width, height int
	}{
		{"résolution d'ouverture", 480, 270},
		{"fenêtre de bureau", 1920, 1080},
	} {
		b.Run(c.name, func(b *testing.B) {
			e, _ := benchEngine(b, c.width, c.height)

			// Une première image monte les tampons, que les suivantes réemploient :
			// la mesurer avec les autres compterait une allocation qui n'a lieu
			// qu'une fois par partie.
			e.Step(nil, Input{})

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				e.Step(nil, Input{})
			}
		})
	}
}

// BenchmarkStepStages donne la part de chaque étape d'un pas, ce qu'un chiffre global ne
// dit pas.
//
// `tests.md` demande de mesurer du dedans et de porter un compteur par étape quand c'est
// la part d'une étape qu'on cherche : un seul nombre ne dirait pas lequel des maillons a
// bougé, et c'est exactement le défaut qui a laissé vivre un facteur 47 dans la
// présentation Windows.
//
// **L'agrandissement n'est pas une étape du pas, et il figure ici quand même** : il
// appartient à celui qui présente, pas au moteur. Mais tout jeu le traverse à chaque
// image, donc l'exclure de la répartition donnerait une part trop belle au dessin. Son
// facteur est celui qu'une fenêtre de bureau impose à la résolution d'ouverture.
//
// **Le fait de structure que ces cas disent ensemble, mesuré le 30 octobre 2026 à la
// charge de contrôle : l'agrandissement domine l'image.** Les trois étapes du pas
// expliquent bien le pas — tri 57 %, dessin 41 %, effacement 4,5 % —, mais le pas entier
// ne pèse qu'un tiers de ce qu'une image coûte vraiment : l'agrandissement au facteur 4
// vaut plus du double du pas à lui seul, soit environ deux tiers du budget.
//
// Ce qui en découle, et c'est une surprise : à cette charge, **le chemin le plus chaud du
// moteur n'est pas le tri mais la recopie de pixels**. Le rapport se lit dans une seule
// série, donc la dérive thermique ne le touche pas ; les valeurs absolues, elles,
// n'appartiennent qu'à la machine qui mesure.
//
// **Et ce rapport dépend de la machine, ce qui dit comment le pondérer.** Mesuré sur deux
// processeurs de même génération dont la bande passante mémoire diffère d'un facteur deux
// et demi : le tri et le dessin se tiennent à 10 % près — la machine la plus large est même
// légèrement plus lente, le calcul la limitant —, là que l'effacement y va deux fois plus
// vite et l'agrandissement d'un cinquième. Ces deux-là sont donc limités par la mémoire, et
// **la part de l'agrandissement croît quand la bande passante baisse** : deux tiers de
// l'image sur la machine la plus étroite, trois cinquièmes sur l'autre.
//
// Conséquence pour qui dimensionne une scène : une machine modeste est plus défavorable à
// ce maillon, jamais moins. C'est l'inverse de ce qu'on suppose en lisant un budget mesuré
// sur du matériel récent.
func BenchmarkStepStages(b *testing.B) {
	const width, height = 480, 270

	e, s := benchEngine(b, width, height)
	e.Step(nil, Input{})

	window := render.NewBuffer(width*4, height*4)

	// Un trieur à part, pour mesurer le tri sans le dessin qui le suit. Celui du moteur
	// n'est pas atteignable, et le dupliquer coûte une image de montage, pas une mesure.
	sorter := sorting.New(s)
	for i := range e.atlas.Sheets {
		p := &e.atlas.Sheets[i]
		sorter.AddSheet(p.Width, p.Height, p.Pixels)
	}
	list := sorter.Frame()

	for _, c := range []struct {
		name string
		run  func()
	}{
		{"le pas entier", func() { e.Step(nil, Input{}) }},
		{"le tri seul", func() { sorter.Frame() }},
		{"le dessin seul", func() { raster.Draw(e.Frame(), list, e.atlas) }},
		{"l'effacement du tampon", func() { raster.Clear(e.Frame(), render.Color{A: 255}) }},
		{"l'agrandissement vers la fenêtre", func() {
			raster.Upscale(e.Frame(), window, 4, render.Color{A: 255})
		}},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				c.run()
			}
		})
	}
}

// benchEdgeScene monte une scène propre aux chemins de bord, **distincte de
// [benchEngine] et c'est obligatoire** : la godoc de celui-ci pose que sa charge doit
// rester identique pour que les chiffres de [BenchmarkStep] et [BenchmarkStepStages] se
// comparent d'une version à l'autre. Y ajouter des pièces d'écran ou des entités suivies
// les périmerait tous les deux.
//
// Elle reste petite — ce qu'on mesure ici n'est pas le tri mais ce qui l'entoure — et
// porte les deux choses qu'aucun autre montage n'a : des poses en pixels dans les deux
// repères, et des entités dont le recouvrement se relève.
func benchEdgeScene(b *testing.B) (*Engine, *Scene) {
	b.Helper()

	sheet := NewSheet(16, 16)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = render.Color{R: 160, G: 150, B: 140, A: 255}
	}

	var atlas Atlas
	if _, err := atlas.Add(sheet); err != nil {
		b.Fatalf("planche : %v", err)
	}

	s, err := NewScene(64, 64, Isometric(32), SceneOptions{Entities: 256})
	if err != nil {
		b.Fatalf("montage de la scène : %v", err)
	}
	s.Focus(32, 32, 480, 270)

	// Des entités suivies, pour que le relevé de recouvrement ait quelque chose à
	// rendre : sans `Measure`, la liste est vide et la mesure ne traverse rien.
	shape := Aspect{Sheet: 0, Width: 16, Height: 16, Tint: White()}
	for i := range 64 {
		x := float64((i*7)%64) + 0.5
		y := float64((i*11)%64) + 0.5
		if _, ok := s.Place(Entity{X: x, Y: y, Aspect: shape, Measure: true}); !ok {
			b.Fatalf("entité %d refusée", i)
		}
	}

	e, err := NewEngine(s, Options{Atlas: &atlas, Width: 480, Height: 270})
	if err != nil {
		b.Fatalf("montage du moteur : %v", err)
	}
	return e, s
}

// BenchmarkFrameEdges mesure les trois chemins par image que `make bancs` donnait pour
// muets, et c'est leur seul chiffre.
//
// **Aucun n'est dans [Engine.Step]**, ce qui est la raison pour laquelle aucune extension
// de [BenchmarkStepStages] ne les atteindrait : la collecte des pièces d'écran est dans
// le trieur mais ne se déclenche qu'avec des poses en pixels, le relevé de recouvrement
// se demande après le pas, et la conversion du pointeur vit dans la boucle du paquet
// racine.
//
// **Ce qu'on en attend est un ordre de grandeur, pas une surveillance.** Les trois
// coûtent quelques opérations par image et non par pixel, donc le chiffre sert à
// l'établir une fois — et à voir le jour où l'un d'eux cesserait d'être négligeable.
// `aim` entraîne avec lui `floorDiv` et `raster.ScaleFactor`, qui n'avaient pas de banc
// non plus.
func BenchmarkFrameEdges(b *testing.B) {
	e, s := benchEdgeScene(b)

	// Les pièces d'écran survivent d'une image à l'autre — le moteur n'appelle jamais
	// ClearScreen —, donc les poser une fois suffit et rien n'alloue dans la mesure.
	piece := Aspect{Sheet: 0, Width: 16, Height: 16, Tint: White(), Layer: 1}
	for i := range 32 {
		if !s.PlaceScreen(int32(i*13%440), int32(i*7%240), piece) {
			b.Fatalf("pièce d'écran %d refusée", i)
		}
		if !s.PlaceArea(int32(i*11%440), int32(i*5%240), piece) {
			b.Fatalf("pièce d'aire %d refusée", i)
		}
	}

	e.Step(nil, Input{})

	window := NewBuffer(1920, 1080)

	// Une entrée qui porte un pointeur dans la fenêtre : c'est ce que le backend donne
	// à la boucle, en pixels de fenêtre et non d'image.
	var state State
	state.Move(960, 540)
	pointer := state.Frame()

	for _, c := range []struct {
		name string
		run  func()
	}{
		{"le pas avec des pièces d'écran", func() { e.Step(nil, Input{}) }},
		{"le relevé de recouvrement", func() { e.Overlaps() }},
		{"la conversion du pointeur", func() { aim(Input(pointer), e.Frame(), window) }},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				c.run()
			}
		})
	}
}
