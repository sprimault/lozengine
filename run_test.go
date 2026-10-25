// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La cadence de la boucle, éprouvée seule.
//
// C'est la seule partie de [Run] qui s'éprouve : le reste est une fenêtre, et aucun
// test n'en ouvre — l'intégration continue tourne là où rien ne garantit qu'une
// fenêtre puisse s'ouvrir. La boucle elle-même se vérifie à la main, une fois, comme
// le dit la règle des plateformes.
//
// Et c'est ici que vit le défaut : une cadence qui dérive ne se voit pas en lisant le
// code, elle se voit après trente secondes de jeu.

package lozengine

import (
	"testing"
	"time"
)

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
