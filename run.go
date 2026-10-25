// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La boucle, pour qui écrit un jeu en Go : une fenêtre, un pas fixe, et une seule
// méthode à écrire.
//
// Elle n'ajoute aucune capacité à l'étape — tout ce qu'elle permet se fait par
// [Engine] —, et elle n'abrite aucune décision de rendu. Ce qu'elle apporte est la
// fenêtre, l'horloge et l'agrandissement, soit ce qu'un jeu hôte possède déjà.

package lozengine

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sprimault/lozengine/internal/backend"
	"github.com/sprimault/lozengine/raster"
)

// Run ouvre une fenêtre et avance le jeu jusqu'à sa fermeture.
//
// Le pas est fixe : le jeu avance de [Options.Tick] à chaque image, quoi qu'ait duré
// la précédente. Les images, elles, sortent aussi régulièrement que la machine le
// permet, et l'agrandissement vers la fenêtre est entier avec des bandes autour quand
// la taille ne tombe pas juste.
//
// Elle ne rend la main qu'à la fermeture, ce qui convient à un jeu écrit en Go et à
// lui seul : un hôte qui possède sa boucle emploie [Engine] et appelle un pas par
// image.
func Run(s *Scene, g Game, o Options) error {
	// La file de messages d'une fenêtre appartient au thread qui l'a créée, et
	// Windows l'impose. Sans cela, un changement de goroutine ferait pomper depuis
	// un thread qui ne reçoit rien, et la fenêtre paraîtrait figée.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	engine, err := NewEngine(s, o)
	if err != nil {
		return err
	}
	o = o.withDefaults()

	window, err := backend.Open(backend.Options{
		Title:  o.Title,
		Width:  uint16(o.Width),
		Height: uint16(o.Height),
	})
	if err != nil {
		return fmt.Errorf("opening the window: %w", err)
	}
	defer window.Close()

	// Le tampon de la fenêtre est monté une fois et refait seulement quand la
	// fenêtre change de taille : l'agrandissement écrit dedans à chaque image, et
	// l'allouer par image mettrait le ramasse-miettes dans le chemin de rendu.
	var presented *Buffer

	// L'instant du prochain pas, tenu en absolu. C'est ce qui empêche l'erreur de
	// s'accumuler : dormir d'une durée fixe à chaque tour ajoute à chaque fois ce
	// que l'horloge du système arrondit — de quinze à trente et une millisecondes
	// sous Windows selon ce que font les autres processus —, et la boucle dérive.
	// En visant une date, un tour trop long est absorbé par le suivant.
	next := time.Now().Add(o.Tick)

	for {
		in := window.Input()
		if window.Closed() {
			return nil
		}

		engine.Step(g, in)

		width, height := window.Size()
		if presented == nil || presented.Width != width || presented.Height != height {
			if width == 0 || height == 0 {
				// Fenêtre réduite : rien à présenter, et la taille nulle ne
				// permet même pas de monter un tampon.
				presented = nil
			} else {
				presented = NewBuffer(width, height)
			}
		}
		if presented != nil {
			fit(engine.Frame(), presented, o.Bars)
			window.Present(presented)
		}

		wait, after := pace(time.Now(), next, o.Tick)
		next = after
		if wait > 0 {
			time.Sleep(wait)
		}
	}
}

// fit écrit l'image interne dans le tampon de la fenêtre, agrandie du plus grand facteur
// entier qui l'y fait tenir, et peint de bars ce qui reste autour.
//
// Séparée de la boucle pour la raison qui a séparé pace : une décision mêlée à une
// fenêtre ne s'éprouve pas. Celle-ci porte le fait que les bandes ne suivent pas le fond
// du monde, et rien d'autre ne pourrait le vérifier puisqu'aucun test n'ouvre de fenêtre.
func fit(inner, out *Buffer, bars Color) {
	raster.Upscale(inner, out, raster.ScaleFactor(inner, out), bars)
}

// pace dit combien attendre avant le pas suivant, et quand il tombera.
//
// Elle est séparée de la boucle parce qu'elle est la seule décision de temps du
// moteur, et qu'une décision de temps mêlée à une fenêtre ne s'éprouve pas. Ses deux
// règles :
//
//   - **la date du pas suivant se calcule en absolu**, en ajoutant le pas à la
//     précédente. Dormir d'une durée fixe à chaque tour ajoute à chaque fois ce que
//     l'horloge du système arrondit — de quinze à trente et une millisecondes sous
//     Windows, selon ce que font les autres processus — et la boucle dérive sans que
//     rien ne le signale ;
//   - **un retard de plus de deux pas ne se rattrape pas.** Après une pause du
//     système, rejouer vingt images d'un coup ferait sauter le jeu et personne n'y
//     gagnerait. On repart de maintenant.
func pace(now, next time.Time, tick time.Duration) (wait time.Duration, after time.Time) {
	if now.Sub(next) > 2*tick {
		return 0, now.Add(tick)
	}
	return next.Sub(now), next.Add(tick)
}
