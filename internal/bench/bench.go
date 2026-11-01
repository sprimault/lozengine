// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les rapporteurs que les bancs du dépôt partagent : le budget d'une image, la part
// qu'une étape en prend, et le débit qu'elle atteint.
//
// # Pourquoi un paquet plutôt qu'une copie par banc
//
// Le budget était écrit quatre fois et le rapporteur trois, et deux de ces trois
// rapportaient des unités **différentes sous le même nom** — pondérée dans les bancs du
// rastériseur, brute dans ceux de l'apparence. Trois d'entre eux portaient la même
// justification : qu'un paquet commun « coûterait une ligne de plus au tableau des
// frontières ». C'est faux, et mesuré tel le 1er novembre 2026 — le tableau se dérive des
// imports du code et non de ceux des tests, l'apparence y atteignant déjà la géométrie par
// les siens sans que sa ligne le dise. La factorisation ne coûte donc rien de ce qu'on lui
// reprochait, et la règle de trois était dépassée depuis la quatrième copie.
//
// # Pourquoi il n'y a plus de pondération
//
// Les bancs rapportaient une part multipliée par quatre, pour représenter une machine dont
// la bande passante mémoire serait quatre fois plus étroite. Elle est retirée, et la mesure
// qui l'a défaite est la comparaison du débit atteint au débit crête de la machine :
// l'effacement du tampon, seul chemin où la pondération était gardée, **dépasse** le débit
// de la mémoire principale des deux machines du parc, donc il est servi par le cache et une
// machine plus étroite ne le ralentit pas d'autant. L'agrandissement, dont elle avait été
// retirée, tient le quart du crête : c'est lui qui dépend de la mémoire.
//
// Elle était donc appliquée exactement à l'envers. Et le seul chemin qui la portait vaut
// 0,044 % d'une image en brut, soit cinquante-cinq fois sous le seuil de dix pour cent :
// elle n'a jamais changé un verdict là où elle s'appliquait, et tous ceux qu'elle a changés
// portaient sur des chemins où elle ne s'appliquait pas.
//
// Ce qui la remplace est le débit atteint, que `b.SetBytes` rend déjà en `MB/s` : comparé
// au débit de la mémoire principale de la machine qui mesure, il tranche sur **une**
// machine et sans extrapoler. Un débit qui l'approche ou le dépasse dit que le chemin est
// servi par le cache, donc qu'une machine plus étroite ne le ralentira pas d'autant ; un
// débit qui en tient une fraction dit l'inverse. C'est ce qui a renversé le classement de
// l'effacement et de l'agrandissement, et il n'y avait pas de rapporteur à écrire pour
// cela — la bibliothèque standard le porte.
//
// Il remplace le rapport entre les deux machines du parc, qui ne classait rien : ce rapport
// n'est reproductible qu'à dix pour cent près pour des bandes séparées par l'intervalle
// 1,25 à 2,5, il laisse deux chemins sur quatre sans classe, et il demande une seconde
// machine à chaque nouveau chemin.
//
// Le seuil, lui, se juge sur la part à la surface qu'on nomme : un chiffre pris à la
// résolution d'ouverture ne dit rien de ce qu'une fenêtre maximisée coûte, la surface de
// dessin suivant la fenêtre.
package bench

import "testing"

// FrameAt60Hz est le budget d'une image à soixante par seconde, en nanosecondes.
//
// **C'est le seul dénominateur du seuil de dix pour cent.** Une somme d'étapes qu'on a
// choisi de mesurer n'en est pas un : lue contre les quatre étapes du pas, l'une d'elles
// valait 66 % et dépassait tout seuil ; lue contre une image, elle en vaut 6,7. Un rapport
// entre chemins dit lequel domine, il ne dit pas si un chemin mérite qu'on y touche.
const FrameAt60Hz = 16_666_667

// Part rapporte la part d'une image que l'étape mesurée prend, en pourcentage.
//
// Le `ns/op` brut reste rapporté à côté : c'est lui que `benchstat` compare, la part
// n'étant qu'une lecture. Elle existe parce que, faute d'un chiffre comparable au seuil,
// les verdicts se recalculaient à la main — et divergeaient.
func Part(b *testing.B) {
	perOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	b.ReportMetric(perOp*100/FrameAt60Hz, "%image")
}
