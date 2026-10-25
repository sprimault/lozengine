// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"math"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
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
	bruts   []render.Quad
	cles    []Cle
	sources []uint32
	ordres  []ordre

	sortie render.DrawList

	// La vue de l'image en cours, gardée le temps du relevé pour ne pas la traîner
	// en paramètre jusqu'au fond.
	vueX, vueY, vueLargeur, vueHauteur int32
}

// ordre porte ce qui départage deux choses que la clé laisse à égalité.
//
// L'abscisse écran d'abord : à profondeur égale, ce qui est à gauche se peint avant.
// Elle vaut l'axe transverse du monde à un facteur près, et elle est déjà calculée —
// la recalculer depuis les coordonnées demanderait un second arrondi, qui peut tomber
// de l'autre côté d'un quart.
//
// Puis une identité qui ne bouge jamais. Sans elle, l'ordre retomberait sur celui de
// production, lequel change dès qu'une entité est retirée ailleurs : deux sprites
// superposés se relaieraient au premier plan d'une image à l'autre.
type ordre struct {
	abscisse int32
	identite uint32

	// La strate ne sépare que ce qui occupe la même cellule, donc elle vient après
	// l'abscisse : deux strates d'une cellule la partagent, deux cellules distinctes
	// d'une même profondeur ne la partagent pas.
	strate uint8
}

// Nouveau monte un trieur sur une scène.
func Nouveau(s *scene.Scene) *Trieur {
	return &Trieur{scene: s, rangeur: NouveauRangeur(0)}
}

// Image rend la suite ordonnée des quads, valable jusqu'à l'appel suivant.
//
// Elle ne rend pas d'erreur, et c'est voulu : une image n'a personne à qui se
// plaindre. Ce qui peut échouer a échoué au montage de la scène.
func (t *Trieur) Image() render.DrawList {
	t.bruts, t.cles, t.sources, t.ordres = t.bruts[:0], t.cles[:0], t.sources[:0], t.ordres[:0]

	t.vueX, t.vueY, t.vueLargeur, t.vueHauteur = t.scene.View()
	if t.vueLargeur <= 0 || t.vueHauteur <= 0 {
		return t.sortie[:0]
	}

	t.releverDecor()
	t.releverEntites()

	t.rangeur.Ranger(t.cles, t.sources)
	t.departager()

	return t.assembler()
}

// releverEntites parcourt toutes les entités, sans en écarter aucune d'avance.
//
// Toutes et non les visibles : une scène en porte quelques milliers et presque toutes
// sont à l'image, donc les trier d'abord coûterait plus que de les projeter. C'est le
// décor qui se relève par fenêtre, parce qu'une carte en compte cent fois plus que ce
// qu'un écran en montre.
func (t *Trieur) releverEntites() {
	m := t.scene.Projection()
	identites := scene.Identities(t.scene)

	for i, e := range scene.Entities(t.scene) {
		rx, ry := t.scene.Remap(e.X, e.Y)
		ex, ey, profondeur, hauteur := m.Project(rx, ry, e.Z)

		// Une entité n'est posée sur aucune strate : elle se tient sur le décor, pas
		// dedans, et la strate zéro la met donc sous tout ce qui s'y empile.
		t.poser(e.Aspect, ex, ey, profondeur, hauteur, identites[i], 0)
	}
}

// releverDecor parcourt les cellules que la vue peut montrer, couche par couche.
func (t *Trieur) releverDecor() {
	strates := t.scene.Tiers()
	if strates == 0 {
		return
	}

	m := t.scene.Projection()
	largeur, _ := t.scene.Dimensions()
	u0, v0, u1, v1 := t.fenetre()

	for strate := range strates {
		for v := v0; v <= v1; v++ {
			for u := u0; u <= u1; u++ {
				a := t.scene.Cell(u, v, uint8(strate))
				if a.Width == 0 || a.Height == 0 {
					continue
				}

				// Le centre de la cellule, comme une entité se tient au centre de la
				// sienne : c'est ce qui met une créature et la case qu'elle foule sur
				// le même point de comparaison.
				rx, ry := t.scene.Remap(float64(u)+0.5, float64(v)+0.5)
				ex, ey, profondeur, hauteur := m.Project(rx, ry, 0)

				// Le rang de la cellule ne bouge jamais, contrairement à la place
				// d'une entité : c'est l'identité que le départage lui demande.
				t.poser(a, ex, ey, profondeur, hauteur, uint32(v*largeur+u), uint8(strate))
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
func (t *Trieur) fenetre() (u0, v0, u1, v1 int) {
	m := t.scene.Projection()
	largeur, hauteur := t.scene.Dimensions()

	minU, minV := math.MaxFloat64, math.MaxFloat64
	maxU, maxV := -math.MaxFloat64, -math.MaxFloat64

	for _, coin := range [4][2]int32{{0, 0}, {t.vueLargeur, 0}, {0, t.vueHauteur}, {t.vueLargeur, t.vueHauteur}} {
		rx, ry := m.Unproject(t.vueX+coin[0], t.vueY+coin[1])
		x, y := t.scene.Unremap(rx, ry)

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
func (t *Trieur) poser(a scene.Aspect, absolueX, absolueY int32, profondeur uint32, hauteur uint16, identite uint32, strate uint8) {
	if a.Width == 0 || a.Height == 0 {
		return
	}

	ex, ey := absolueX-t.vueX, absolueY-t.vueY
	largeur, haut := int32(a.Width), int32(a.Height)
	x, y := ex-largeur/2, ey-haut

	if x+largeur <= 0 || x >= t.vueLargeur || y+haut <= 0 || y >= t.vueHauteur {
		return
	}

	t.sources = append(t.sources, uint32(len(t.bruts)))
	t.cles = append(t.cles, Composer(a.Layer, a.Floor, profondeur, hauteur, a.Sheet, 0))
	t.ordres = append(t.ordres, ordre{abscisse: ex, identite: identite, strate: strate})
	t.bruts = append(t.bruts, render.Quad{
		X: int16(x), Y: int16(y),
		SourceX: a.SourceX, SourceY: a.SourceY,
		Width: a.Width, Height: a.Height,
		Sheet: a.Sheet,
		Tint:  a.Tint,
		Flags: a.Flags,
	})
}

// departager range ce que la clé laisse à égalité, puis inscrit le résultat dans
// l'index stable.
//
// **Par groupe et jamais sur toute l'image.** L'index ne porte que mille vingt-quatre
// valeurs là où une image en compte couramment des milliers : numéroter globalement le
// ferait saturer dès le millier et quelques, et la clé cesserait d'exprimer l'ordre
// sans que rien ne le signale. Un groupe ne réunit que des choses partageant déjà
// couche, étage, profondeur, hauteur et planche, donc il reste court — quelques unités
// sur une scène ordinaire.
//
// Les clés d'un groupe étant identiques, seules leurs provenances se déplacent, et une
// insertion suffit : elle ne coûte que la taille du groupe, pas celle de l'image.
func (t *Trieur) departager() {
	debut := 0

	for i := 1; i <= len(t.cles); i++ {
		if i < len(t.cles) && t.cles[i] == t.cles[debut] {
			continue
		}

		t.departagerGroupe(debut, i)
		debut = i
	}
}

// departagerGroupe ordonne une suite de clés égales et leur donne leurs index.
func (t *Trieur) departagerGroupe(debut, fin int) {
	for i := debut + 1; i < fin; i++ {
		for j := i; j > debut && t.avant(t.sources[j], t.sources[j-1]); j-- {
			t.sources[j], t.sources[j-1] = t.sources[j-1], t.sources[j]
		}
	}

	for rang := debut; rang < fin; rang++ {
		t.cles[rang] = t.cles[rang].AvecIndex(uint16(min(rang-debut, IndexMax)))
	}
}

// avant dit laquelle de deux choses d'une même clé se peint en premier.
//
// L'abscisse écran range de la gauche vers la droite, ce qui est l'ordre naturel d'une
// même diagonale. L'identité tranche ensuite, et elle seule rend l'ordre reproductible
// d'une image à l'autre : la place d'une entité change quand une autre est retirée.
func (t *Trieur) avant(a, b uint32) bool {
	gauche, droite := t.ordres[a], t.ordres[b]

	if gauche.abscisse != droite.abscisse {
		return gauche.abscisse < droite.abscisse
	}
	if gauche.strate != droite.strate {
		return gauche.strate < droite.strate
	}
	return gauche.identite < droite.identite
}

// assembler recopie les quads dans l'ordre que le rangement a fixé.
func (t *Trieur) assembler() render.DrawList {
	if cap(t.sortie) < len(t.sources) {
		t.sortie = make(render.DrawList, len(t.sources))
	}
	t.sortie = t.sortie[:len(t.sources)]

	for rang, source := range t.sources {
		t.sortie[rang] = t.bruts[source]
	}
	return t.sortie
}
