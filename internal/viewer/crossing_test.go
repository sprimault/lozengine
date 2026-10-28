// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Le critère de fin du jalon 4, rendu mécanique : un décor à deux niveaux se traverse
// sans perdre le personnage, et « sans le perdre » se compte au lieu de se juger.

package viewer

import (
	"testing"

	"github.com/sprimault/lozengine/internal/sorting"
)

// visibleFloor est le nombre de pixels du personnage qui doivent rester visibles à
// chaque pas de la traversée.
//
// Son sprite couvre dix pixels sur vingt-quatre, dont l'intérieur — la seule part qui
// porte exactement sa teinte, la bordure en portant le tiers — fait huit sur
// vingt-deux, soit cent soixante-seize. Le plancher est posé à cent vingt : il laisse
// passer qu'un bord sorte de l'image, et il refuse tout ce qui ressemble à une
// disparition.
const visibleFloor = 120

// TestTheCharacterIsNeverLost compte, sur chaque image de la traversée, les pixels qui
// portent la teinte du personnage.
//
// **C'est ce qui rend le critère de fin du jalon vérifiable.** « Se traverse sans
// perdre le personnage » est un verdict de lisibilité, et une image de référence
// verrouille des pixels, pas une impression : elle dit que le rendu n'a pas changé,
// jamais qu'il est bon. Le comptage, lui, échoue si le personnage disparaît — et il
// parcourt les huit images, là où trois seulement sont versées en référence.
//
// Ce qu'il attrape et que rien d'autre n'attraperait : une révélation qui cesserait
// de s'appliquer en cours de déplacement, un seuil réglé trop haut, un étage qui
// reprendrait le dessus. Dans ces trois régimes la suite de référence resterait verte
// tant que les images versées tombent hors du passage.
func TestTheCharacterIsNeverLost(t *testing.T) {
	replay := crossingReplay(t)

	for frame := range crossingFrames {
		buffer := replay.Frame(frame, "")

		seen := 0
		for _, p := range buffer.Pixels {
			if p == crossingFigureTint {
				seen++
			}
		}

		if seen < visibleFloor {
			t.Errorf("image %d : %d pixels du personnage visibles, au moins %d attendus",
				frame, seen, visibleFloor)
		}
	}
}

// TestTheCrossingReallyPassesUnderThePlatform garde ce sans quoi le cas précédent ne
// prouverait rien : que le personnage se fasse bien cacher quelque part sur sa course.
//
// **Un trajet qui passerait à côté de la plateforme ne devrait jamais rien à la
// révélation**, et le comptage resterait alors vert sur un moteur qui ne révèle plus
// rien du tout. C'est le défaut le plus silencieux qu'un scénario puisse avoir : il
// garde quelque chose, mais pas ce qu'il annonce.
//
// La preuve se lit sur le nombre de quads du personnage plutôt que sur des pixels :
// deux veulent dire que la révélation s'est déclenchée, donc que l'image le cachait
// assez. Il en faut au moins un pas sur la course.
func TestTheCrossingReallyPassesUnderThePlatform(t *testing.T) {
	revealed := 0

	for frame := range crossingFrames {
		quads := 0
		for _, q := range sorting.New(crossingScene(frame)).Frame() {
			if q.Sheet == crossingFigureSheet {
				quads++
			}
		}

		switch quads {
		case 1:
			// Visible sans rien devoir à la révélation.
		case 2:
			revealed++
		default:
			t.Fatalf("image %d : %d quads de personnage, attendu un ou deux", frame, quads)
		}
	}

	if revealed == 0 {
		t.Error("le personnage n'est caché à aucun pas : la traversée ne pose pas le problème")
	}
	if revealed == crossingFrames {
		t.Error("le personnage est caché à tous les pas : la traversée n'a ni entrée ni sortie")
	}
}

// crossingReplay monte le rejeu du scénario de traversée.
func crossingReplay(t *testing.T) *Replay {
	t.Helper()

	scenario, ok := ByName("crossing")
	if !ok {
		t.Fatal("scénario crossing absent du catalogue")
	}

	replay, err := NewReplay(scenario)
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	return replay
}
