// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'interface qu'un fournisseur d'apparence remplit, la direction qu'elle emploie, et
// la rotation qui accorde celle-ci avec l'orientation de la caméra.

package appearance

import "github.com/sprimault/lozengine/scene"

// Facing est une direction, comptée en pas dans le sens horaire à l'écran depuis le
// sud.
//
// Le sud sert d'origine parce que c'est la direction d'une chose au repos, qui fait
// face à celui qui regarde ; le sens horaire suit l'écran, où l'ordonnée croît vers le
// bas. Un pas vaut un tour complet divisé par le nombre de directions que le
// fournisseur déclare, donc **le même nombre ne désigne pas le même angle chez deux
// fournisseurs** : il se lit toujours avec le [Provider] qui l'a produit.
type Facing uint8

// Provider résout ce qu'une chose montre.
//
// Les quatre axes sont ceux qu'un jeu fait varier séparément : ce qu'une chose est,
// l'état où elle se trouve — marcher, frapper, attendre —, la direction vers laquelle
// elle regarde, et le moment de son animation. Les fondre obligerait à multiplier les
// types pour exprimer les états, ce qu'un jeu finit toujours par défaire.
//
// Les deux premiers axes sont des entiers sans signification pour le moteur : un jeu
// nomme ses types et ses états comme il l'entend, et le fournisseur seul sait à quoi
// ils renvoient. C'est ce qui permet au moteur de ne rien coder en dur de ce qui
// relève du jeu.
//
// Une implémentation n'a pas à être rapide : elle est consultée quand une apparence
// change, jamais pendant le rendu.
type Provider interface {
	// Directions rend le nombre de directions distinctes que ses planches portent.
	//
	// Multiple de quatre, parce qu'un quart de tour de caméra décale d'un quart de ce
	// nombre : un total qui ne tombe pas juste rendrait la rotation fausse sans que
	// rien ne le signale. Huit pour une implémentation en planches dessinées, seize
	// pour une fabrique volumique.
	Directions() int

	// Aspect rend de quoi remplir un quad, pour une chose, un état, une direction et
	// un moment.
	//
	// La direction est comptée dans le nombre de pas que [Provider.Directions]
	// annonce. Le moment est un numéro d'image, que l'appelant tire du numéro de pas
	// plutôt que d'un état gardé.
	//
	// Ce qu'une demande sans réponse rend est au fournisseur : un aspect de repli, ou
	// la valeur zéro, qui ne dessine rien. Pas d'erreur — elles se produisent au
	// chargement, pas quand une chose change de direction.
	Aspect(kind, cycle uint16, facing Facing, frame int) scene.Aspect
}

// Turn rend la direction sous laquelle l'orientation courante de la caméra présente
// une direction du monde.
//
// Un jeu garde la direction de ses entités dans son repère à lui, une fois pour
// toutes ; c'est la vue qui tourne. Cette fonction fait pour une direction ce que
// [scene.Scene.Remap] fait pour un point, et son résultat est ce qu'on passe à
// [Provider.Aspect].
//
// **Elle se retranche et ne s'ajoute pas** : tourner la caméra d'un quart dans un sens
// fait paraître les choses tourner d'un quart dans l'autre. Une chose qui allait vers
// le bas-droite de l'écran va vers le haut-droite après un quart de tour.
//
// **Elle suppose le sens d'axes de [geometry.Dimetric] et [geometry.Isometric]** — `x`
// vers le bas-gauche de l'écran, `y` vers le bas-droite —, parce que la table des
// directions est écrite dans ce repère. Un jeu qui bâtit sa projection par
// [geometry.Raw] en retournant le premier axe garde des directions justes dans son
// monde et les voit fausses à l'écran sur six des huit pas, les demi-tours exceptés.
//
// Un jeu dont la caméra ne tourne pas ne la croise jamais, l'orientation valant alors
// [scene.NoRotation] et le résultat la direction reçue.
func Turn(f Facing, o scene.Orientation, directions int) Facing {
	if directions <= 0 {
		return f
	}

	step := directions / 4 * int(o&3)
	return Facing(((int(f)-step)%directions + directions) % directions)
}
