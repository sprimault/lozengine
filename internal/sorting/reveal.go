// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La révélation : ce qu'une chose marquée redessine d'elle-même au-dessus de ce qui la
// cache, et la mesure de recouvrement qui décide quand.

package sorting

import (
	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// revealHidden ajoute, pour chaque chose marquée que l'image cache assez, un second
// quad d'elle-même dans la couche qu'elle a demandée.
//
// **Elle passe avant le tri, et c'est ce qui la rend simple.** Savoir ce qui cache une
// chose demande de savoir ce qui passe devant elle — mais la clé le dit déjà, une clé
// plus grande se peignant après. Attendre l'ordre final obligerait à réinsérer les
// quads produits dans une liste déjà rangée ; produits ici, ils se rangent avec les
// autres et rien n'a à les traiter à part.
//
// Le coût est d'un parcours de l'image par chose marquée. Un jeu en marque une ou deux
// — son personnage —, et le compte tenu à la pose évite même ce parcours quand il n'y
// en a aucune, ce qui est le cas courant.
//
// Ce qui est révélé est la chose, jamais le décor. Rendre un mur transparent
// découvrirait ce qui se tient derrière lui autant que ce qui s'y trouve, et cohue a
// payé deux défauts pour avoir cru que l'élévation pouvait servir de critère : un sol
// surélevé reste du sol, et ne cache rien.
func (t *Sorter) revealHidden() {
	if t.revealing == 0 {
		return
	}

	after, tint := scene.Revealing(t.scene)

	// La borne est figée : la boucle ajoute à la même tranche qu'elle parcourt, et une
	// chose révélée n'a pas à l'être une seconde fois.
	for i, n := 0, len(t.raw); i < n; i++ {
		layer := t.reveals[i]
		if layer == 0 {
			continue
		}

		if !t.hiddenEnough(i, n, after) {
			continue
		}

		q := t.raw[i]
		if tint.A != 0 {
			q.Tint = tint
		}

		t.sources = append(t.sources, uint32(len(t.raw)))
		t.keys = append(t.keys, t.keys[i].WithLayer(layer))
		t.ties = append(t.ties, t.ties[i])
		t.reveals = append(t.reveals, 0)
		t.raw = append(t.raw, q)
	}
}

// hiddenEnough dit si ce qui se peint après le quad de rang i le recouvre de plus de
// `after` pixels.
//
// Le cumul compte deux fois ce que deux choses se recouvrant entre elles cachent au
// même endroit, et c'est assumé : corriger demanderait de tenir la surface déjà
// couverte, pour un seuil que le jeu règle de toute façon à l'œil. La somme s'arrête
// dès qu'elle dépasse, donc l'approximation ne coûte même pas de temps.
func (t *Sorter) hiddenEnough(i, n, after int) bool {
	hidden := 0

	for j := range n {
		if t.keys[j] <= t.keys[i] {
			continue
		}
		if hidden += overlap(t.raw[i], t.raw[j]); hidden > after {
			return true
		}
	}
	return false
}

// overlap rend le nombre de pixels que deux quads se partagent.
//
// En entiers de machine et non en seize bits : deux quads de trente-deux mille pixels
// de côté donneraient un produit qui déborde, et le seuil se compare à une somme de
// plusieurs de ces aires.
func overlap(a, b render.Quad) int {
	left, right := max(int(a.X), int(b.X)), min(int(a.X)+int(a.Width), int(b.X)+int(b.Width))
	if right <= left {
		return 0
	}

	top, bottom := max(int(a.Y), int(b.Y)), min(int(a.Y)+int(a.Height), int(b.Y)+int(b.Height))
	if bottom <= top {
		return 0
	}

	return (right - left) * (bottom - top)
}
