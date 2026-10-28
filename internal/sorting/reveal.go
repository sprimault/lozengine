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
// chose demande de savoir ce qui passe devant elle — mais tout l'ordre est déjà connu :
// une clé plus grande se peint après, et ce qu'elle laisse à égalité se départage sur
// des pièces que la collecte a posées. Attendre l'ordre final obligerait à réinsérer les
// quads produits dans une liste déjà rangée ; produits ici, ils se rangent avec les
// autres et rien n'a à les traiter à part.
//
// Le coût est d'un parcours de l'image par chose marquée. Un jeu en marque une ou deux
// — son personnage —, et le compte tenu à la pose évite même ce parcours quand il n'y
// en a aucune, ce qui est le cas courant.
//
// **Ce qui cache est ce qui se peint après, et rien d'autre ne le qualifie** : une
// cellule de décor, une entité qui s'interpose, une pièce d'écran posée plus bas que
// la couche de révélation. La comparaison porte sur l'ordre de dessin, donc l'origine
// du quad n'entre jamais en compte — ce qui est voulu, l'encerclement par des créatures
// étant le cas que la révélation sert d'abord.
//
// Ce qui est révélé est la chose, jamais ce qui la cache. Rendre un mur transparent
// découvrirait ce qui se tient derrière lui autant que ce qui s'y trouve, et cohue a
// payé deux défauts pour avoir cru que l'élévation pouvait servir de critère : un sol
// surélevé reste du sol, et ne cache rien.
func (t *Sorter) revealHidden() {
	if t.revealing == 0 {
		return
	}

	o := scene.Revealing(t.scene)

	// La borne est figée : la boucle ajoute à la même tranche qu'elle parcourt, et une
	// chose révélée n'a pas à l'être une seconde fois.
	for i, n := 0, len(t.raw); i < n; i++ {
		layer := t.reveals[i]
		if layer == 0 {
			continue
		}

		hidden, byDecor := t.hiddenEnough(i, n, o.After)
		if !hidden {
			continue
		}

		tint := o.Tint
		if byDecor && o.TintByDecor.A != 0 {
			tint = o.TintByDecor
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
// `after` pixels, et si le décor y suffirait à lui seul.
//
// **Le second résultat emploie le même seuil que le premier, et c'est ce qui le
// justifie.** La teinte ne bascule pas parce qu'un pixel de décor s'est glissé dans le
// recouvrement : un pixel caché ne porte aucun indice de profondeur, puisqu'on ne voit
// pas qu'il y a quelque chose devant — l'argument même du seuil. Sans cela, un
// personnage encerclé qui tourne autour d'un pilier verrait sa teinte battre à chaque
// frôlement alors que l'essentiel du recouvrement vient de la mêlée, et un signal qui
// clignote sur un écran chargé est pire que pas de signal.
//
// La boucle ne peut donc plus sortir dès que le total dépasse : il reste à savoir si
// le décor seul y arrive. Elle sort en revanche dès que **lui** dépasse, puisque plus
// rien ne peut alors changer la réponse.
//
// Le cumul compte deux fois ce que deux choses se recouvrant entre elles cachent au
// même endroit, et c'est assumé : corriger demanderait de tenir la surface déjà
// couverte, pour un seuil que le jeu règle de toute façon à l'œil.
//
// **L'ordre de dessin ne se lit pas sur la clé seule.** Deux choses de même couche,
// étage, profondeur, hauteur et planche ont des clés égales, l'index stable n'étant pas
// encore écrit : c'est le départage qui range leur groupe, et il est déjà connu ici
// puisque [Sorter.before] ne lit que ce que la collecte a posé. Le quad de rang i
// s'écarte du même geste, le départage d'une chose avec elle-même rendant faux.
func (t *Sorter) hiddenEnough(i, n, after int) (hidden, byDecor bool) {
	total, decor := 0, 0
	key := t.keys[i]

	for j := range n {
		if key > t.keys[j] || (key == t.keys[j] && !t.before(uint32(i), uint32(j))) {
			continue
		}

		covered := overlap(t.raw[i], t.raw[j])
		total += covered
		if t.fromDecor[j] {
			if decor += covered; decor > after {
				return true, true
			}
		}
	}
	return total > after, false
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
