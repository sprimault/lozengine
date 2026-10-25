// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package entree

import "testing"

// Deux touches arbitraires, dans deux mots différents du jeu de bits : la seconde est
// au-delà du soixante-quatrième bit, donc un décalage calculé sur le mauvais mot se
// verrait.
const (
	toucheA Touche = 17
	toucheB Touche = 130
)

// TestFrontMontantNeVautQuUneImage est le cas qui porte tout l'intérêt des deux
// régimes, et celui qui casserait en silence.
//
// Une touche tenue sans interruption doit rendre Pressee vrai à la première image et
// faux ensuite, Enfoncee restant vrai tout du long. Sans ce cas, un front montant qui
// se répéterait passerait inaperçu de la suite — l'image serait juste, et le jeu
// tirerait soixante fois par seconde.
func TestFrontMontantNeVautQuUneImage(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheA)

	premiere := etat.Image()
	if !premiere.Enfoncee(toucheA) || !premiere.Pressee(toucheA) {
		t.Fatalf("première image : tenue %v, montée %v, attendu les deux",
			premiere.Enfoncee(toucheA), premiere.Pressee(toucheA))
	}

	seconde := etat.Image()
	if !seconde.Enfoncee(toucheA) {
		t.Error("seconde image : touche relâchée alors que le doigt est resté")
	}
	if seconde.Pressee(toucheA) {
		t.Error("seconde image : la montée s'est répétée")
	}
}

// TestRepetitionAutomatiqueNeRearmePas vérifie qu'un second Enfoncer sans relâchement
// ne redonne pas de front montant.
//
// C'est le cas réel et non théorique : le système renvoie l'événement d'appui tant que
// le doigt reste posé, à la cadence de répétition du clavier. Un front par répétition
// ferait tirer en rafale sans que le joueur l'ait demandé.
func TestRepetitionAutomatiqueNeRearmePas(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheA)
	etat.Image()

	etat.Enfoncer(toucheA)
	if etat.Image().Pressee(toucheA) {
		t.Error("la répétition du système a rendu un nouveau front montant")
	}
}

// TestRelachementRearmeLeFront vérifie qu'un appui suivant compte, ce qui est l'autre
// moitié du régime : sans cela, une touche ne servirait qu'une fois par partie.
func TestRelachementRearmeLeFront(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheA)
	etat.Image()

	etat.Relacher(toucheA)
	relachee := etat.Image()
	if relachee.Enfoncee(toucheA) || relachee.Pressee(toucheA) {
		t.Fatal("touche encore tenue après relâchement")
	}

	etat.Enfoncer(toucheA)
	if !etat.Image().Pressee(toucheA) {
		t.Error("le second appui n'a pas rendu de front montant")
	}
}

// TestOublierRelacheTout couvre la perte de focus, où le système cesse d'envoyer les
// relâchements : une touche tenue à cet instant le resterait indéfiniment, et le
// personnage marcherait tout seul au retour dans la fenêtre.
func TestOublierRelacheTout(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheA)
	etat.Enfoncer(toucheB)
	etat.Image()

	etat.Oublier()
	apres := etat.Image()
	if apres.Enfoncee(toucheA) || apres.Enfoncee(toucheB) {
		t.Error("une touche est restée tenue après l'oubli")
	}

	// Et le front se réarme : revenir dans la fenêtre et appuyer doit compter.
	etat.Enfoncer(toucheA)
	if !etat.Image().Pressee(toucheA) {
		t.Error("l'appui qui suit un oubli n'a pas rendu de front montant")
	}
}

// TestTouchesIndependantes vérifie que deux touches de mots différents ne se marchent
// pas dessus : un décalage appliqué au mauvais mot se verrait ici, et nulle part
// ailleurs.
func TestTouchesIndependantes(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheB)

	image := etat.Image()
	if !image.Enfoncee(toucheB) {
		t.Error("la touche du second mot n'est pas tenue")
	}
	if image.Enfoncee(toucheA) {
		t.Error("la touche du premier mot est tenue alors que rien ne l'a enfoncée")
	}
}

// TestImageNAllouePas vérifie que la lecture des entrées reste hors du chemin
// d'allocation, comme l'effacement et le dessin qui la suivent dans un pas.
func TestImageNAllouePas(t *testing.T) {
	var etat Etat
	etat.Enfoncer(toucheA)

	if n := testing.AllocsPerRun(10, func() { etat.Image() }); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
