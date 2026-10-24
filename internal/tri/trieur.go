// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"math"

	"github.com/sprimault/lozengine/internal/scene"
	"github.com/sprimault/lozengine/rendu"
)

// Trieur transforme une scène en la suite ordonnée des quads d'une image.
//
// C'est la frontière du moteur : ce qui sort d'ici ne connaît plus ni grille, ni
// entité, ni caméra — un rastériseur ou un backend n'a qu'à peindre dans l'ordre reçu.
//
// Ses tampons se montent à la première image et se réemploient ensuite. Une image ne
// touche donc pas au tas, sauf la première et celles qui suivent un agrandissement de
// la vue.
type Trieur struct {
	scene   *scene.Scene
	rangeur *Rangeur

	// Les quads dans l'ordre où ils ont été produits, et de quoi les ranger : une clé
	// par quad, et l'indice qui dit d'où chaque clé vient. Ranger déplace les
	// indices, pas les quads.
	bruts   []rendu.Quad
	cles    []Cle
	sources []uint32

	sortie rendu.Rendu
}

// Nouveau monte un trieur sur une scène.
func Nouveau(s *scene.Scene) *Trieur {
	return &Trieur{scene: s, rangeur: NouveauRangeur(0)}
}

// Image rend la suite ordonnée des quads, valable jusqu'à l'appel suivant.
//
// Elle ne rend pas d'erreur, et c'est voulu : une image n'a personne à qui se
// plaindre. Ce qui peut échouer a échoué au montage de la scène.
func (t *Trieur) Image() rendu.Rendu {
	t.bruts, t.cles, t.sources = t.bruts[:0], t.cles[:0], t.sources[:0]

	vueX, vueY, vueLargeur, vueHauteur := t.scene.Vue()
	if vueLargeur <= 0 || vueHauteur <= 0 {
		return t.sortie[:0]
	}

	t.releverDecor(vueX, vueY, vueLargeur, vueHauteur)
	t.releverEntites(vueX, vueY, vueLargeur, vueHauteur)

	t.rangeur.Ranger(t.cles, t.sources)
	t.indexerParProfondeur()

	return t.assembler()
}

// releverEntites parcourt toutes les entités, sans en écarter aucune d'avance.
//
// Toutes et non les visibles : une scène en porte quelques milliers et presque toutes
// sont à l'image, donc les trier d'abord coûterait plus que de les projeter. C'est le
// décor qui se relève par fenêtre, parce qu'une carte en compte cent fois plus que ce
// qu'un écran en montre.
func (t *Trieur) releverEntites(vueX, vueY, vueLargeur, vueHauteur int32) {
	m := t.scene.Projection()

	for _, e := range t.scene.Entites() {
		rx, ry := t.scene.Remapper(e.X, e.Y)
		ex, ey, profondeur, hauteur := m.Projeter(rx, ry, e.Z)

		t.poser(e.Aspect, ex-vueX, ey-vueY, profondeur, hauteur, vueLargeur, vueHauteur)
	}
}

// releverDecor parcourt les cellules que la vue peut montrer, couche par couche.
func (t *Trieur) releverDecor(vueX, vueY, vueLargeur, vueHauteur int32) {
	couches := t.scene.Couches()
	if couches == 0 {
		return
	}

	m := t.scene.Projection()
	u0, v0, u1, v1 := t.fenetre(vueX, vueY, vueLargeur, vueHauteur)

	for couche := range couches {
		for v := v0; v <= v1; v++ {
			for u := u0; u <= u1; u++ {
				a := t.scene.Cellule(u, v, uint8(couche))
				if a.Largeur == 0 || a.Hauteur == 0 {
					continue
				}

				// Le centre de la cellule, comme une entité se tient au centre de la
				// sienne : c'est ce qui met une créature et la case qu'elle foule sur
				// le même point de comparaison.
				rx, ry := t.scene.Remapper(float64(u)+0.5, float64(v)+0.5)
				ex, ey, profondeur, hauteur := m.Projeter(rx, ry, 0)

				t.poser(a, ex-vueX, ey-vueY, profondeur, hauteur, vueLargeur, vueHauteur)
			}
		}
	}
}

// fenetre rend l'intervalle de cellules que la vue peut montrer, bornes comprises.
//
// La vue est un rectangle à l'écran, donc un losange dans le monde : ce qu'on balaie
// est son englobant, soit environ le double de ce qui sera peint. Le coût ne dépend
// alors que de la taille de la vue, jamais de celle de la carte.
//
// Une cellule de marge de chaque côté, parce qu'une image est posée par son pied et
// s'étend vers le haut : celle dont le pied vient de sortir y a encore son corps.
func (t *Trieur) fenetre(vueX, vueY, vueLargeur, vueHauteur int32) (u0, v0, u1, v1 int) {
	m := t.scene.Projection()
	largeur, hauteur := t.scene.Dimensions()

	minU, minV := math.MaxFloat64, math.MaxFloat64
	maxU, maxV := -math.MaxFloat64, -math.MaxFloat64

	for _, coin := range [4][2]int32{{0, 0}, {vueLargeur, 0}, {0, vueHauteur}, {vueLargeur, vueHauteur}} {
		rx, ry := m.Inverser(vueX+coin[0], vueY+coin[1])
		x, y := t.scene.Deremapper(rx, ry)

		minU, maxU = math.Min(minU, x), math.Max(maxU, x)
		minV, maxV = math.Min(minV, y), math.Max(maxV, y)
	}

	u0, u1 = max(0, int(minU)-1), min(largeur-1, int(maxU)+1)
	v0, v1 = max(0, int(minV)-1), min(hauteur-1, int(maxV)+1)
	return u0, v0, u1, v1
}

// poser écarte ce qui ne touche pas l'image, et compose le quad et la clé du reste.
//
// Les coordonnées reçues sont déjà relatives à la vue, et le test se fait en trente-
// deux bits — avant la réduction à seize que le quad impose. C'est ce qui rend le
// débordement impossible : ce qui est trop loin est parti, et ce qui dépasse d'un bord
// commence au plus à sa largeur en arrière. Un aspect plus large que 32 767 fait
// exception et n'a pas de place où dépasser à gauche.
//
// L'ancre est le pied de l'image, centré horizontalement. Elle deviendra surchargeable
// par le fournisseur d'apparence, jamais par appel.
func (t *Trieur) poser(a scene.Aspect, ex, ey int32, profondeur uint32, hauteur uint16, vueLargeur, vueHauteur int32) {
	if a.Largeur == 0 || a.Hauteur == 0 {
		return
	}

	largeur, haut := int32(a.Largeur), int32(a.Hauteur)
	x, y := ex-largeur/2, ey-haut

	if x+largeur <= 0 || x >= vueLargeur || y+haut <= 0 || y >= vueHauteur {
		return
	}

	t.sources = append(t.sources, uint32(len(t.bruts)))
	t.cles = append(t.cles, Composer(a.Couche, a.Etage, profondeur, hauteur, a.Planche, 0))
	t.bruts = append(t.bruts, rendu.Quad{
		X: int16(x), Y: int16(y),
		SourceX: a.SourceX, SourceY: a.SourceY,
		Largeur: a.Largeur, Hauteur: a.Hauteur,
		Planche:  a.Planche,
		Teinte:   a.Teinte,
		Drapeaux: a.Drapeaux,
	})
}

// indexerParProfondeur numérote les clés à l'intérieur de chaque profondeur.
//
// **Par profondeur et jamais sur toute l'image.** Le champ ne porte que mille
// vingt-quatre valeurs là où une image en compte couramment des milliers : numéroter
// globalement le ferait saturer dès le millier et quelques, et la clé cesserait
// d'exprimer l'ordre sans que rien ne le signale. À l'intérieur d'une profondeur ne se
// départagent que les choses qui partagent déjà couche, étage, hauteur et planche.
//
// L'ordre ne s'en trouve pas changé : le rangement vient de le fixer, et un index
// croissant dans ce même sens ne fait que l'inscrire dans la clé — là où l'ordre
// partiel des volumes pourra le reprendre.
func (t *Trieur) indexerParProfondeur() {
	var index uint16
	var precedente uint32

	for i, c := range t.cles {
		if p := c.Profondeur(); i == 0 || p != precedente {
			precedente, index = p, 0
		}

		t.cles[i] = c.AvecIndex(index)
		if index < IndexMax {
			index++
		}
	}
}

// assembler recopie les quads dans l'ordre que le rangement a fixé.
func (t *Trieur) assembler() rendu.Rendu {
	if cap(t.sortie) < len(t.sources) {
		t.sortie = make(rendu.Rendu, len(t.sources))
	}
	t.sortie = t.sortie[:len(t.sources)]

	for rang, source := range t.sources {
		t.sortie[rang] = t.bruts[source]
	}
	return t.sortie
}
