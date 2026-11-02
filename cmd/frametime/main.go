// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Commande qui relève ce qu'une image coûte, étape par étape, sur la plateforme qui
// l'exécute. Elle tient sa propre boucle plutôt que d'emprunter celle du moteur : c'est
// la seule façon de chronométrer ce qu'un jeu ne voit pas.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/sprimault/lozengine"
	"github.com/sprimault/lozengine/internal/backend"
	"github.com/sprimault/lozengine/internal/bench"
	"github.com/sprimault/lozengine/raster"
	"github.com/sprimault/lozengine/render"
)

// stage est une étape du tour, et ce qu'elle a cumulé.
//
// **Le cumul et non la moyenne par image**, et ce n'est pas une commodité : l'horloge
// d'une page est arrondie, à cinq microsecondes dans le meilleur cas et cent dans le pire,
// là qu'une étape en coûte quelques dizaines. Mesurée image par image, la plus courte
// disparaîtrait dans l'arrondi ; cumulée sur des centaines d'images, elle ressort.
type stage struct {
	name  string
	total time.Duration
}

// add cumule une durée.
func (s *stage) add(d time.Duration) { s.total += d }

// part rend ce que l'étape prend d'un budget d'image, en pourcentage.
func (s *stage) part(frames int) float64 {
	if frames == 0 {
		return 0
	}
	return float64(s.total) * 100 / (float64(frames) * bench.FrameAt60Hz)
}

// mean rend la durée moyenne par image.
func (s *stage) mean(frames int) time.Duration {
	if frames == 0 {
		return 0
	}
	return s.total / time.Duration(frames)
}

// quiet est un jeu qui ne décrit rien : la scène est posée une fois, et ce qu'on mesure
// est ce que le moteur en fait à chaque image, pas ce qu'un jeu y ajoute.
type quiet struct{}

// Step ne fait rien.
func (quiet) Step(lozengine.Input) {}

func main() {
	frames := flag.Int("frames", 180, "images à relever, après l'échauffement")
	warmup := flag.Int("warmup", 60, "images d'échauffement, écartées du relevé")
	side := flag.Int("side", 64, "côté de la carte, en cellules")
	entities := flag.Int("entities", 200, "entités posées sur la carte")
	width := flag.Int("width", 480, "largeur de la résolution interne")
	height := flag.Int("height", 270, "hauteur de la résolution interne")
	flag.Parse()

	if err := run(*frames, *warmup, *side, *entities, *width, *height); err != nil {
		fmt.Fprintf(os.Stderr, "frametime: %v\n", err)
		os.Exit(1)
	}
}

// run monte la scène, tient la boucle et rend le relevé.
//
// **L'échauffement n'est pas du zèle** : la première série après une construction lit
// jusqu'à trois fois trop haut, et un module qui vient de démarrer paie en plus la
// compilation de son code par le moteur de la page. Les images écartées le sont avant
// toute mesure, donc elles ne pèsent sur aucun cumul.
func run(frames, warmup, side, entities, width, height int) error {
	scene, atlas, err := build(side, entities)
	if err != nil {
		return err
	}

	options := lozengine.Options{
		Title: "lozengine — relevé du temps par image",
		Atlas: atlas, Width: width, Height: height,
	}
	engine, err := lozengine.NewEngine(scene, options)
	if err != nil {
		return fmt.Errorf("mounting the engine: %w", err)
	}

	window, err := backend.Open(backend.Options{
		Title: options.Title, Width: uint16(width), Height: uint16(height),
	})
	if err != nil {
		return fmt.Errorf("opening the window: %w", err)
	}
	defer window.Close()

	step := &stage{name: "le pas, tri et dessin compris"}
	upscale := &stage{name: "l'agrandissement vers la fenêtre"}
	present := &stage{name: "la présentation à la plateforme"}

	var presented *render.Buffer
	for i := 0; i < warmup+frames; i++ {
		in := window.Input()
		if window.Closed() {
			return fmt.Errorf("the window closed after %d frames: %w", i, window.Err())
		}

		w, h := window.Size()
		engine.Fit(int(w), int(h))
		if w == 0 || h == 0 {
			// Fenêtre réduite : rien à mesurer, et l'image ne compte pas.
			i--
			continue
		}
		if presented == nil || presented.Width != w || presented.Height != h {
			presented = render.NewBuffer(w, h)
		}

		measured := i >= warmup

		at := time.Now()
		engine.Step(quiet{}, in)
		if measured {
			step.add(time.Since(at))
		}

		at = time.Now()
		raster.Upscale(engine.Frame(), presented,
			raster.ScaleFactor(engine.Frame(), presented), render.Color{A: 255})
		if measured {
			upscale.add(time.Since(at))
		}

		at = time.Now()
		window.Present(presented)
		if measured {
			present.add(time.Since(at))
		}
	}

	report(frames, presented, step, upscale, present)
	return nil
}

// build monte une carte couverte de sol et les entités demandées.
//
// La scène est volontairement banale : ce qu'on relève est le coût des étapes du moteur, et
// une scène savante déplacerait la mesure vers ce qu'elle a de particulier.
func build(side, entities int) (*lozengine.Scene, *lozengine.Atlas, error) {
	scene, err := lozengine.NewScene(side, side, lozengine.Isometric(32), lozengine.SceneOptions{
		Entities: entities + 1,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("mounting the scene: %w", err)
	}

	sheet := lozengine.NewSheet(32, 16)
	for i := range sheet.Pixels {
		sheet.Pixels[i] = lozengine.White()
	}
	atlas := &lozengine.Atlas{}
	if _, err := atlas.Add(sheet); err != nil {
		return nil, nil, fmt.Errorf("adding the sheet: %w", err)
	}

	ground := lozengine.Aspect{Width: 32, Height: 16, Tint: lozengine.White()}
	for y := range side {
		for x := range side {
			scene.PlaceCell(x, y, 0, ground)
		}
	}

	// Les entités sont réparties sur la diagonale et non groupées : un tas sur une seule
	// cellule mesurerait le départage plutôt que le tri.
	for i := range entities {
		at := float64(i%side) + 0.5
		scene.Place(lozengine.Entity{X: at, Y: float64((i*7)%side) + 0.5, Aspect: ground})
	}
	return scene, atlas, nil
}

// report écrit le relevé, une fois la boucle finie.
//
// **Rien ne s'écrit pendant la boucle**, et c'est une règle du dépôt payée cher : une
// écriture par image a déjà fabriqué un symptôme qui n'existait pas, en gelant la boucle
// le temps d'un tube. Tout est cumulé en mémoire et rendu ici.
//
// La part se lit contre le budget d'une image à soixante par seconde, et contre rien
// d'autre : la somme des étapes n'est pas un dénominateur, elle dit seulement laquelle
// domine.
func report(frames int, presented *render.Buffer, stages ...*stage) {
	fmt.Printf("relevé sur %d images, surface %d×%d\n", frames, presented.Width, presented.Height)
	fmt.Printf("  %s/%s, %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if page := agent(); page != "" {
		fmt.Printf("  moteur de page : %s\n", page)
	}

	var total time.Duration
	for _, s := range stages {
		total += s.total
		fmt.Printf("  %-34s %8.3f ms   %5.2f %%image\n", s.name,
			float64(s.mean(frames))/float64(time.Millisecond), s.part(frames))
	}

	whole := &stage{total: total}
	fmt.Printf("  %-34s %8.3f ms   %5.2f %%image\n", "les trois ensemble",
		float64(whole.mean(frames))/float64(time.Millisecond), whole.part(frames))
	fmt.Println("budget d'une image à 60 Hz : 16,667 ms")
}
