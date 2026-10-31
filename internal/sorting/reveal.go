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
	if t.revealing == 0 && len(t.requests) == 0 {
		return
	}

	o := scene.Revealing(t.scene)

	// Les demandes sont dans l'ordre de pose, donc par rang croissant : un index qui
	// avance suffit à savoir si le rang courant est demandé, sans table ni recherche.
	next := 0

	// La borne est figée : la boucle ajoute à la même tranche qu'elle parcourt, et une
	// chose révélée n'a pas à l'être une seconde fois.
	for i, n := 0, len(t.raw); i < n; i++ {
		layer := t.reveals[i]
		asked := next < len(t.requests) && int(t.requests[next].rank) == i
		if layer == 0 && !asked {
			continue
		}

		// Une seule mesure pour les deux usages : une entité qui demande le relevé et
		// porte une couche de révélation ne la paie donc qu'une fois.
		total, decor := t.overlap(i, n)

		// La mesure exacte ne se paie que pour ce qu'un hôte a demandé à relever, et
		// jamais pour une révélation : le seuil d'[scene.Aspect.Reveal] est réglé sur
		// la mesure par rectangles, et le faire changer d'échelle casserait en silence
		// tout jeu qui l'a calibré — le peint maximal d'une case du premier jeu est de
		// 954 là où son seuil vaut 2 048.
		var painted, covered, coveredByDecor int
		if asked {
			painted, covered, coveredByDecor = t.coverage(i, n)
		}

		if asked {
			t.overlaps = append(t.overlaps, render.Overlap{
				Identity:       t.requests[next].handle,
				Painted:        uint32(painted),
				Covered:        uint32(covered),
				CoveredByDecor: uint32(coveredByDecor),
				Hidden:         uint32(total),
				ByDecor:        uint32(decor),
			})
			next++
		}

		if layer == 0 || total <= o.After {
			continue
		}

		tint := o.Tint
		if decor > o.After && o.TintByDecor.A != 0 {
			tint = o.TintByDecor
		}

		q := t.raw[i]
		if tint.A != 0 {
			q.Tint = tint
		}

		// **Les sept tranches parallèles se remplissent, pas cinq.** Deux manquaient
		// ici, et le désalignement ne se voyait pas : la borne n est figée avant cette
		// boucle, donc aucun quad révélé n'est jamais lu par la mesure. Il suffisait
		// qu'une lecture passe cette borne pour que le trieur lise le drapeau d'un
		// autre quad, ou panique.
		//
		// Le quad révélé hérite de l'origine de la chose, dont il est une copie, et il
		// ne compte jamais : c'est un rappel que le moteur dessine, exactement ce que
		// [scene.Aspect.Uncounted] exprime pour un rappel que l'hôte dessine.
		t.sources = append(t.sources, uint32(len(t.raw)))
		t.keys = append(t.keys, t.keys[i].WithLayer(layer))
		t.ties = append(t.ties, t.ties[i])
		t.reveals = append(t.reveals, 0)
		t.fromDecor = append(t.fromDecor, t.fromDecor[i])
		t.uncounted = append(t.uncounted, true)
		t.raw = append(t.raw, q)
	}
}

// overlap rend le nombre de pixels du quad de rang i que recouvre ce qui se peint après
// lui, et la part de ce total qui vient du décor.
//
// **La mesure est exacte et ne sort pas en route.** Une version antérieure s'arrêtait
// dès que le décor seul dépassait le seuil, ce qui suffisait à décider d'une révélation
// mais rend un total faux ; le relevé qu'un hôte reçoit demande le compte entier, et
// une mesure qui dépend de l'usage qu'on en fait serait deux mesures à tenir d'accord.
// Le gain de cette sortie n'avait pas été mesuré.
//
// **Les deux origines ne portent pas la même information**, et c'est pourquoi la part du
// décor est comptée à part : ce qu'un mur cache, il le cache parce que la chose est
// derrière lui ; une créature qui s'interpose ne dit rien de tel.
//
// Le cumul compte deux fois ce que deux choses se recouvrant entre elles cachent au
// même endroit, et c'est assumé : corriger demanderait de tenir la surface déjà
// couverte, pour un seuil que le jeu règle de toute façon à l'œil.
//
// **L'ordre de dessin ne se lit pas sur la clé seule.** Deux choses de même couche,
// étage, profondeur, hauteur et rang ont des clés égales, l'index stable n'étant pas
// encore écrit : c'est le départage qui range leur groupe, et il est déjà connu ici
// puisque [Sorter.before] ne lit que ce que la collecte a posé. Le quad de rang i
// s'écarte du même geste, le départage d'une chose avec elle-même rendant faux.
func (t *Sorter) overlap(i, n int) (total, decor int) {
	key := t.keys[i]

	for j := range n {
		if t.uncounted[j] {
			continue
		}

		// **Un quad qui ne peint rien ne cache rien.** Une teinte d'alpha nul fait sortir
		// `put` comme `fillRect` d'emblée, donc le quad n'écrit pas un pixel ; verser sa
		// boîte entière faisait franchir le seuil de révélation à un rappel éteint.
		//
		// **C'est un changement de la loi et non une mise en conformité**, et il faut le
		// dire ainsi : la godoc de [render.Overlap.Hidden] portait les deux lectures —
		// « ce qui **se peint** après la chose » d'un côté, « transparent compris » de
		// l'autre —, et c'est la seconde que le dépôt avait gardée, témoin compris. Le
		// statut *décrit* de ce nombre est ce qui autorise le changement, et le témoin
		// a rougi comme il devait.
		//
		// `paints` et non `hides` : ce dernier exige aussi un alpha plein, donc il
		// retirerait du compte tout le décor translucide et poserait une falaise à
		// l'alpha 254. Et un quad additif reste compté — il peut parfaitement cacher,
		// en saturant au blanc —, [scene.Aspect.Uncounted] étant la voie pour l'exclure.
		if !paints(t.raw[j]) {
			continue
		}
		if key > t.keys[j] || (key == t.keys[j] && !t.before(uint32(i), uint32(j))) {
			continue
		}

		covered := shared(t.raw[i], t.raw[j])
		total += covered
		if t.fromDecor[j] {
			decor += covered
		}
	}
	return total, decor
}

// shared rend le nombre de pixels que deux quads se partagent.
//
// En entiers de machine et non en seize bits : deux quads de trente-deux mille pixels
// de côté donneraient un produit qui déborde, et le seuil se compare à une somme de
// plusieurs de ces aires.
func shared(a, b render.Quad) int {
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
