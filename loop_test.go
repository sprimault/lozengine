// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que la boucle doit faire des deux régimes de cadence, éprouvé sur une fenêtre de
// papier : aucune plateforme n'est en jeu, et c'est ce qui rend ces cas possibles.

package lozengine

import (
	"errors"
	"testing"
	"time"

	"github.com/sprimault/lozengine/input"
	"github.com/sprimault/lozengine/internal/backend"
	"github.com/sprimault/lozengine/render"
)

// paperWindow est une fenêtre qui ne touche à aucune plateforme : elle compte ses tours,
// ses présentations, et s'arrête d'elle-même au bout du nombre de tours demandé.
//
// **Son attente est réglable, et elle ne l'est pas pour faire joli** : la boucle compare
// des instants, et l'horloge de certains systèmes n'avance pas entre deux appels aussi
// rapprochés que les tours de cette fenêtre. Un cas qui veut voir une échéance dépassée
// doit donc laisser le temps passer pour de bon — relevé en écrivant ces cas, où vingt
// tours instantanés tenaient dans un seul quantum d'horloge et aucune échéance d'une
// nanoseconde n'était jamais franchie.
//
// L'assertion plus bas est ce qui la tient au contrat : une méthode ajoutée à [Window]
// sans qu'elle suive casse la compilation de ces cas, au lieu de les laisser éprouver une
// interface qui n'est plus celle du moteur.
type paperWindow struct {
	paced bool

	// wait est ce que chaque tour attend, comme une plateforme le ferait.
	wait time.Duration

	// turns est le nombre de tours après lequel la fenêtre se déclare fermée, et seen
	// ce qu'elle en a compté.
	turns, seen int

	// armsOnPresent modélise une plateforme qui n'autorise l'image suivante qu'en
	// réponse à la présentation précédente — Wayland par son `wl_surface.frame`,
	// DRM/KMS par l'événement de sa bascule de page. armed est le jeton qu'elle
	// accorde, et starved compte les tours servis sans jeton : une vraie famille y
	// gèlerait ou tournerait à plein régime, donc le compte est ce qu'un test peut
	// observer de ce régime.
	armsOnPresent bool
	armed         bool
	starved       int

	presents int
	err      error
}

// Input compte un tour et ferme la fenêtre quand le compte est atteint.
func (w *paperWindow) Input() input.Input {
	if w.armsOnPresent {
		if !w.armed {
			w.starved++
		}
		w.armed = false
	}
	if w.wait > 0 {
		time.Sleep(w.wait)
	}
	w.seen++
	if w.seen > w.turns {
		w.err = errClosed
	}
	var s input.State
	return s.Frame()
}

// Present compte une présentation, et réarme le jeton quand c'est elle qui le produit.
func (w *paperWindow) Present(*render.Buffer) {
	w.presents++
	w.armed = true
}

// Size rend une taille fixe, qui n'est pas le sujet de ces cas.
func (w *paperWindow) Size() (uint16, uint16) { return 64, 64 }

// Closed dit que le compte de tours est épuisé.
func (w *paperWindow) Closed() bool { return w.err != nil }

// Err rend ce qui a arrêté la fenêtre.
func (w *paperWindow) Err() error { return w.err }

// Close ne rend rien à personne : cette fenêtre n'a rien pris.
func (w *paperWindow) Close() {}

// Paced rend le régime que le cas a demandé.
func (w *paperWindow) Paced() bool { return w.paced }

// errClosed est ce que la fenêtre de papier rend quand son compte de tours est épuisé.
var errClosed = errors.New("paper window: out of turns")

// La fenêtre de papier remplit le contrat que la boucle consomme, et cette ligne est ce
// qui le garde.
var _ backend.Window = (*paperWindow)(nil)

// runLoop fait tourner la boucle sur une scène minimale et rend le temps qu'elle a pris.
func runLoop(t *testing.T, w *paperWindow, tick time.Duration) time.Duration {
	t.Helper()

	s, err := NewScene(4, 4, Isometric(32), SceneOptions{})
	if err != nil {
		t.Fatalf("montage de la scène : %v", err)
	}
	sheet := NewSheet(32, 16)
	var atlas Atlas
	if _, err := atlas.Add(sheet); err != nil {
		t.Fatalf("ajout de la planche : %v", err)
	}

	o := Options{Atlas: &atlas, Width: 64, Height: 64, Tick: tick}
	engine, err := NewEngine(s, o)
	if err != nil {
		t.Fatalf("montage du moteur : %v", err)
	}

	start := time.Now()
	if err := loop(w, engine, silentGame{}, o.withDefaults()); !errors.Is(err, errClosed) {
		t.Fatalf("la boucle rend %v, attendu la fermeture de la fenêtre", err)
	}
	return time.Since(start)
}

// silentGame est un jeu qui ne décrit rien : ces cas mesurent la boucle, pas la scène.
type silentGame struct{}

// Step ne fait rien.
func (silentGame) Step(Input) {}

// TestLoopSkipsEarlyTurnsOnAPacedWindow garde ce que la boucle doit faire des tours qu'une
// plateforme lui donne en trop : les rendre sans faire de pas.
//
// **Une page compose bien plus souvent que le jeu n'avance** — trois cents fois par
// seconde sur un écran du parc, mesuré le 2 novembre 2026, pour un pas à soixante. Sans
// ce rejet, le jeu avancerait à la cadence de la plateforme : cinq fois trop vite sur cet
// écran-là, et à une vitesse qui dépendrait du matériel du joueur.
//
// Le tick est très long devant la durée d'un tour de cette fenêtre, donc **aucun** tour
// n'atteint l'échéance : la boucle ne doit présenter rien du tout. Le cas rougit si l'on
// retire le rejet, puisqu'elle présenterait alors à chaque tour.
func TestLoopSkipsEarlyTurnsOnAPacedWindow(t *testing.T) {
	w := &paperWindow{paced: true, turns: 50}
	runLoop(t, w, time.Hour)

	if w.presents != 0 {
		t.Errorf("%d présentations sur %d tours, attendu aucune : la boucle a fait des pas "+
			"en avance sur son échéance", w.presents, w.seen)
	}
}

// TestLoopStarvesAPacedWindowThatArmsOnPresent garde la condition qu'une réponse vraie à
// [backend.Window.Paced] exige, en montrant ce qui arrive quand elle n'est pas tenue.
//
// **C'est un témoin et non une correction** : il affirme le piège au lieu de son absence,
// parce que le piège est la conséquence assumée du rejet des tours en avance. Une fenêtre
// dont le jeton d'image naît de Present ne le voit jamais revenir — c'est justement la
// présentation que la boucle saute —, donc tous ses tours après le premier sont servis à
// vide. Une vraie famille y gèlerait, si son Input attend un jeton que personne n'a
// demandé, ou tournerait à plein régime, le sommeil étant interdit sous ce régime.
//
// C'est ce qui a fait répondre faux aux deux familles du jalon 6, et c'est ce que ce cas
// garde : **il rougit si la boucle cesse de sauter la présentation d'un tour en avance**,
// qui est exactement le changement qui rendrait une réponse vraie tenable. La godoc de
// Paced et les backends concernés se reprendraient alors dans le même lot. Vérifié par
// mutation en inversant la comparaison de `pace`.
//
// **Des deux assertions, une seule est à lui**, et il faut le dire : le compte de
// présentations est la précondition du cas — sans un tick qui domine, il ne dirait rien —
// et le cas voisin la garde déjà. Le compte de tours à vide est le sien, aucun autre cas
// ne modélisant le jeton d'une plateforme.
func TestLoopStarvesAPacedWindowThatArmsOnPresent(t *testing.T) {
	w := &paperWindow{paced: true, armsOnPresent: true, armed: true, turns: 50}
	runLoop(t, w, time.Hour)

	if w.presents != 0 {
		t.Errorf("%d présentations, attendu aucune : le tick dépasse toute la durée du cas",
			w.presents)
	}
	if w.starved != w.seen-1 {
		t.Errorf("%d tours à vide sur %d, attendu %d : un jeton s'est réarmé alors que seule "+
			"Present le produit, donc la condition de Paced a changé",
			w.starved, w.seen, w.seen-1)
	}
}

// TestLoopStepsEveryTurnWhenTheDeadlineIsPast garde l'autre bord du même rejet : une
// fenêtre cadencée dont les tours arrivent après l'échéance fait un pas à chaque tour.
//
// Sans ce cas, le précédent passerait aussi avec une boucle qui ne ferait **jamais** de
// pas sur une fenêtre cadencée, ce qui est le défaut le plus facile à écrire ici.
func TestLoopStepsEveryTurnWhenTheDeadlineIsPast(t *testing.T) {
	w := &paperWindow{paced: true, turns: 20, wait: time.Millisecond}
	runLoop(t, w, time.Nanosecond)

	if w.presents != w.seen-1 {
		t.Errorf("%d présentations sur %d tours, attendu un pas par tour : l'échéance "+
			"était dépassée à chaque fois", w.presents, w.seen)
	}
}

// TestLoopStepsEveryTurnOnAnUnpacedWindow garde que le rejet ne s'applique pas aux
// familles qui ne cadencent pas : leur Input rend aussitôt, donc un tour vaut un pas et
// c'est le sommeil qui tient la cadence.
//
// **C'est le cas qui protège Windows et X11 de ce lot.** Il rougit si le rejet s'y
// applique : la boucle n'y présenterait plus rien, la fenêtre de papier rendant ses tours
// sans attendre.
func TestLoopStepsEveryTurnOnAnUnpacedWindow(t *testing.T) {
	w := &paperWindow{paced: false, turns: 20, wait: time.Millisecond}
	runLoop(t, w, time.Nanosecond)

	if w.presents != w.seen-1 {
		t.Errorf("%d présentations sur %d tours, attendu un pas par tour", w.presents, w.seen)
	}
}

// TestLoopSleepsForAnUnpacedWindow garde le sommeil lui-même, et c'est le seul cas à le
// pouvoir.
//
// **Sans lui, rien ne distinguait une boucle qui dort d'une boucle qui tourne à plein
// régime** : les cas qui comptent les pas restent vrais dans les deux régimes, puisque
// leur fenêtre attend d'elle-même. Mesuré en retirant le sommeil — les trois autres cas
// passaient toujours, et une famille native aurait brûlé un cœur sans que rien ne le dise.
//
// La fenêtre ne fait ici attendre personne : tout le temps que la boucle prend vient donc
// du sommeil. La borne est large devant la précision d'une horloge de système, et c'est
// une borne **basse** — une machine chargée dort plus longtemps, jamais moins.
func TestLoopSleepsForAnUnpacedWindow(t *testing.T) {
	const tick = 10 * time.Millisecond
	const turns = 5

	w := &paperWindow{paced: false, turns: turns}
	took := runLoop(t, w, tick)

	if floor := (turns - 1) * tick; took < floor {
		t.Errorf("la boucle a pris %v pour %d pas à %v, attendu au moins %v : elle n'a pas "+
			"dormi", took, w.presents, tick, floor)
	}
}
