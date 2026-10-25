// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'assemblage d'une image, du relevé de la scène à la suite de quads : ce qui sort
// d'ici ne connaît plus ni grille, ni entité, ni caméra. C'est aussi là que vit le
// départage des clés égales, qui est la seule part de l'ordre qu'un calcul ne donne
// pas.

package sorting

import (
	"math"

	"github.com/sprimault/lozengine/render"
	"github.com/sprimault/lozengine/scene"
)

// Sorter transforme une scène en la suite ordonnée des quads d'une image.
//
// C'est la frontière du moteur : ce qui sort d'ici ne connaît plus ni grille, ni
// entité, ni caméra — un rastériseur ou un backend n'a qu'à peindre dans l'ordre reçu.
//
// Ses tampons se montent à la première image et se réemploient ensuite. Une image ne
// touche donc pas au tas, sauf la première et celles qui suivent un agrandissement de
// la vue.
type Sorter struct {
	scene *scene.Scene
	radix *Radix

	// Les quads dans l'ordre où ils ont été produits, et de quoi les ranger : une clé
	// par quad, et l'indice qui dit d'où chaque clé vient. Le rangement déplace les
	// indices, pas les quads.
	raw     []render.Quad
	keys    []Key
	sources []uint32
	ties    []tie

	out render.DrawList

	// La vue de l'image en cours, gardée le temps du relevé pour ne pas la traîner
	// en paramètre jusqu'au fond.
	viewX, viewY, viewWidth, viewHeight int32
}

// tie porte ce qui départage deux choses que la clé laisse à égalité.
//
// L'abscisse écran d'abord : à profondeur égale, ce qui est à gauche se peint avant.
// Elle vaut l'axe transverse du monde à un facteur près, et elle est déjà calculée —
// la recalculer depuis les coordonnées demanderait un second arrondi, qui peut tomber
// de l'autre côté d'un quart.
//
// Puis une identité qui ne bouge jamais. Sans elle, l'ordre retomberait sur celui de
// production, lequel change dès qu'une entité est retirée ailleurs : deux sprites
// superposés se relaieraient au premier plan d'une image à l'autre.
type tie struct {
	x        int32
	identity uint32

	// La strate ne sépare que ce qui occupe la même cellule, donc elle vient après
	// l'abscisse : deux strates d'une cellule la partagent, deux cellules distinctes
	// d'une même profondeur ne la partagent pas.
	tier uint8
}

// New monte un trieur sur une scène.
func New(s *scene.Scene) *Sorter {
	return &Sorter{scene: s, radix: NewRadix(0)}
}

// Frame rend la suite ordonnée des quads, valable jusqu'à l'appel suivant.
//
// Elle ne rend pas d'erreur, et c'est voulu : une image n'a personne à qui se
// plaindre. Ce qui peut échouer a échoué au montage de la scène.
func (t *Sorter) Frame() render.DrawList {
	t.raw, t.keys, t.sources, t.ties = t.raw[:0], t.keys[:0], t.sources[:0], t.ties[:0]

	t.viewX, t.viewY, t.viewWidth, t.viewHeight = t.scene.View()
	if t.viewWidth <= 0 || t.viewHeight <= 0 {
		return t.out[:0]
	}

	t.collectDecor()
	t.collectEntities()

	t.radix.Sort(t.keys, t.sources)
	t.breakTies()

	return t.assemble()
}

// collectEntities parcourt toutes les entités, sans en écarter aucune d'avance.
//
// Toutes et non les visibles : une scène en porte quelques milliers et presque toutes
// sont à l'image, donc les trier d'abord coûterait plus que de les projeter. C'est le
// décor qui se relève par fenêtre, parce qu'une carte en compte cent fois plus que ce
// qu'un écran en montre.
func (t *Sorter) collectEntities() {
	m := t.scene.Projection()
	identities := scene.Identities(t.scene)

	for i, e := range scene.Entities(t.scene) {
		rx, ry := t.scene.Remap(e.X, e.Y)
		ex, ey, depth, height := m.Project(rx, ry, e.Z)

		// Une entité n'est posée sur aucune strate : elle se tient sur le décor, pas
		// dedans, et la strate zéro la met donc sous tout ce qui s'y empile.
		t.place(e.Aspect, ex, ey, depth, height, identities[i], 0)
	}
}

// collectDecor parcourt les cellules que la vue peut montrer, couche par couche.
func (t *Sorter) collectDecor() {
	tiers := t.scene.Tiers()
	if tiers == 0 {
		return
	}

	m := t.scene.Projection()
	width, _ := t.scene.Dimensions()
	u0, v0, u1, v1 := t.window()

	for tier := range tiers {
		for v := v0; v <= v1; v++ {
			for u := u0; u <= u1; u++ {
				a := t.scene.Cell(u, v, uint8(tier))
				if a.Width == 0 || a.Height == 0 {
					continue
				}

				// Le centre de la cellule, comme une entité se tient au centre de la
				// sienne : c'est ce qui met une créature et la case qu'elle foule sur
				// le même point de comparaison.
				rx, ry := t.scene.Remap(float64(u)+0.5, float64(v)+0.5)
				ex, ey, depth, height := m.Project(rx, ry, 0)

				// Le rang de la cellule ne bouge jamais, contrairement à la place
				// d'une entité : c'est l'identité que le départage lui demande.
				t.place(a, ex, ey, depth, height, uint32(v*width+u), uint8(tier))
			}
		}
	}
}

// window rend l'intervalle de cellules que la vue peut montrer, bornes comprises.
//
// La vue est un rectangle à l'écran, donc un losange dans le monde : ce qu'on balaie
// est son englobant, soit environ le double de ce qui sera peint. Le coût ne dépend
// alors que de la taille de la vue, jamais de celle de la carte.
//
// Une cellule de marge de chaque côté, parce qu'une image est posée par son pied et
// s'étend vers le haut : celle dont le pied vient de sortir y a encore son corps.
func (t *Sorter) window() (u0, v0, u1, v1 int) {
	m := t.scene.Projection()
	width, height := t.scene.Dimensions()

	minU, minV := math.MaxFloat64, math.MaxFloat64
	maxU, maxV := -math.MaxFloat64, -math.MaxFloat64

	for _, corner := range [4][2]int32{{0, 0}, {t.viewWidth, 0}, {0, t.viewHeight}, {t.viewWidth, t.viewHeight}} {
		rx, ry := m.Unproject(t.viewX+corner[0], t.viewY+corner[1])
		x, y := t.scene.Unremap(rx, ry)

		minU, maxU = math.Min(minU, x), math.Max(maxU, x)
		minV, maxV = math.Min(minV, y), math.Max(maxV, y)
	}

	u0, u1 = max(0, int(minU)-1), min(width-1, int(maxU)+1)
	v0, v1 = max(0, int(minV)-1), min(height-1, int(maxV)+1)
	return u0, v0, u1, v1
}

// place écarte ce qui ne touche pas l'image, et compose le quad et la clé du reste.
//
// Les coordonnées reçues sont déjà relatives à la vue, et le test se fait en trente-
// deux bits — avant la réduction à seize que le quad impose. C'est ce qui rend le
// débordement impossible : ce qui est trop loin est parti, et ce qui dépasse d'un bord
// commence au plus à sa largeur en arrière. Un aspect plus large que 32 767 fait
// exception et n'a pas de place où dépasser à gauche.
//
// L'ancre est le pied de l'image, centré horizontalement. Elle deviendra surchargeable
// par le fournisseur d'apparence, jamais par appel.
func (t *Sorter) place(a scene.Aspect, absoluteX, absoluteY int32, depth uint32, height uint16, identity uint32, tier uint8) {
	if a.Width == 0 || a.Height == 0 {
		return
	}

	ex, ey := absoluteX-t.viewX, absoluteY-t.viewY
	w, h := int32(a.Width), int32(a.Height)
	x, y := ex-w/2, ey-h

	if x+w <= 0 || x >= t.viewWidth || y+h <= 0 || y >= t.viewHeight {
		return
	}

	t.sources = append(t.sources, uint32(len(t.raw)))
	t.keys = append(t.keys, Compose(a.Layer, a.Floor, depth, height, a.Sheet, 0))
	t.ties = append(t.ties, tie{x: ex, identity: identity, tier: tier})
	t.raw = append(t.raw, render.Quad{
		X: int16(x), Y: int16(y),
		SourceX: a.SourceX, SourceY: a.SourceY,
		Width: a.Width, Height: a.Height,
		Sheet: a.Sheet,
		Tint:  a.Tint,
		Flags: a.Flags,
	})
}

// breakTies range ce que la clé laisse à égalité, puis inscrit le résultat dans
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
func (t *Sorter) breakTies() {
	start := 0

	for i := 1; i <= len(t.keys); i++ {
		if i < len(t.keys) && t.keys[i] == t.keys[start] {
			continue
		}

		t.breakTiesIn(start, i)
		start = i
	}
}

// breakTiesIn ordonne une suite de clés égales et leur donne leurs index.
func (t *Sorter) breakTiesIn(start, end int) {
	for i := start + 1; i < end; i++ {
		for j := i; j > start && t.before(t.sources[j], t.sources[j-1]); j-- {
			t.sources[j], t.sources[j-1] = t.sources[j-1], t.sources[j]
		}
	}

	for rank := start; rank < end; rank++ {
		t.keys[rank] = t.keys[rank].WithIndex(uint16(min(rank-start, MaxIndex)))
	}
}

// before dit laquelle de deux choses d'une même clé se peint en premier.
//
// L'abscisse écran range de la gauche vers la droite, ce qui est l'ordre naturel d'une
// même diagonale. L'identité tranche ensuite, et elle seule rend l'ordre reproductible
// d'une image à l'autre : la place d'une entité change quand une autre est retirée.
func (t *Sorter) before(a, b uint32) bool {
	left, right := t.ties[a], t.ties[b]

	if left.x != right.x {
		return left.x < right.x
	}
	if left.tier != right.tier {
		return left.tier < right.tier
	}
	return left.identity < right.identity
}

// assemble recopie les quads dans l'ordre que le rangement a fixé.
func (t *Sorter) assemble() render.DrawList {
	if cap(t.out) < len(t.sources) {
		t.out = make(render.DrawList, len(t.sources))
	}
	t.out = t.out[:len(t.sources)]

	for rank, source := range t.sources {
		t.out[rank] = t.raw[source]
	}
	return t.out
}
