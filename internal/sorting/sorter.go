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

	// La couche où chaque quad se redessine s'il se fait cacher, zéro pour aucune, et
	// le compte de ceux qui en portent une. Le compte évite de parcourir l'image pour
	// découvrir qu'aucune chose n'est à révéler, ce qui est le cas courant.
	reveals   []uint8
	revealing int

	// Ce que chaque quad vient du décor, pour que la révélation sache lequel de ses
	// deux compteurs incrémenter. Ici et non dans le quad : la disposition compatible
	// C n'a pas de place à donner à un détail de relevé, et ce bit ne sort jamais du
	// trieur.
	fromDecor []bool

	// uncounted dit, par rang de quad, que ce quad ne compte pas dans la mesure de
	// recouvrement. Une tranche de plus et non deux bits dans celle ci-dessus : les
	// deux formes ont été mesurées sur la charge du jeu, de zéro à trois cents entités
	// marquées, et aucune ne se distingue de l'autre ni de la base — l'écart entre
	// deux séries de la même forme était du même ordre. À coût égal, deux tranches
	// nommées se lisent mieux qu'un octet qu'il faut masquer.
	uncounted []bool

	// masks porte la forme de chaque planche de l'atlas, par index, pour la mesure
	// exacte. Vide tant que personne ne l'a remplie : le trieur se monte sans atlas
	// dans cinq scénarios de référence, et la mesure approchée reste alors seule.
	masks []mask

	// candidates porte, le temps d'une mesure exacte, les rangs des quads dont la
	// boîte recouvre celle de la chose mesurée. Gardée d'une mesure à l'autre pour
	// n'allouer qu'à la première : une quinzaine d'entrées là où la scène en compte
	// deux mille.
	candidates []uint32

	// Ce qu'un jeu demande à faire relever, dans l'ordre de pose donc par rang de quad
	// croissant. Une liste courte et non une tranche parallèle à raw : un jeu suit une
	// chose ou deux, et une tranche de la taille de l'image coûterait sa remise à zéro
	// pour deux entrées utiles.
	requests []request

	// Le relevé de l'image, réutilisé d'une image à l'autre : l'allouer par image
	// mettrait le ramasse-miettes dans le chemin de rendu.
	overlaps render.OverlapList

	// collectingDecor dit à place d'où vient ce qu'elle reçoit. Un état plutôt qu'un
	// paramètre de plus : place en porte déjà sept, et les trois relevés qui
	// l'appellent le font chacun dans une boucle serrée.
	collectingDecor bool

	out render.DrawList

	// La vue de l'image en cours, gardée le temps du relevé pour ne pas la traîner
	// en paramètre jusqu'au fond.
	viewX, viewY, viewWidth, viewHeight int32

	// Le rectangle contre lequel [Sorter.place] écarte et découpe, et ce qu'il ajoute
	// ensuite pour passer dans la surface. Les deux changent selon ce qu'on collecte :
	// le monde se découpe sur l'aire et se décale de sa place dans la surface, une
	// pièce d'écran se découpe sur la surface entière et ne se décale pas.
	//
	// Des champs plutôt que des paramètres, pour la raison que collectingDecor donne
	// déjà : place en porte sept, et quatre de plus la rendraient illisible.
	offsetX, offsetY            int32
	boundsWidth, boundsHeight   int32
	surfaceWidth, surfaceHeight int32
}

// request est un relevé de recouvrement demandé : la poignée à nommer, et le rang du
// quad qui porte la chose.
//
// Le rang plutôt qu'un renvoi vers l'entité, parce que la mesure se fait sur des quads
// et que l'entité peut en produire un ou aucun — hors de la vue, elle n'en produit pas,
// et il n'y a alors rien à relever.
type request struct {
	handle uint32
	rank   uint32
}

// tie porte ce qui départage deux choses que la clé laisse à égalité.
//
// L'abscisse écran d'abord : à profondeur égale, ce qui est à gauche se peint avant.
// Elle croît avec `y − x`, l'axe transverse du monde au signe près, et elle est déjà
// calculée —
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

// Sheets rend le nombre de planches dont le trieur connaît la forme, pour que son
// appelant sache lesquelles lui manquent.
func (t *Sorter) Sheets() int { return len(t.masks) }

// AddSheet donne au trieur la forme d'une planche, pour la mesure exacte du
// recouvrement. L'ordre des appels est celui des index que les quads portent.
//
// **Les pixels et non une planche d'atlas** : `sorting` ne connaît pas `raster`, et la
// frontière des paquets est ce qui permet au noyau de n'avoir qu'un sens de dépendance.
// L'appelant tient l'atlas et passe ce qu'il en lit.
//
// **Elle dérive la forme et ne garde pas les pixels** : un parcours par planche, au
// chargement, jamais à l'image. Dériver les 5,5 millions de pixels du premier jeu coûte
// 7,5 ms une fois, et les deux plans de bits qui en sortent pèsent 6,4 % de ce que les
// couleurs occupent.
//
// Sans aucune planche donnée, le relevé ne porte que la mesure approchée, celle des
// rectangles. C'est le régime des scénarios de référence, qui montent le trieur sans
// atlas, et celui d'un hôte qui n'en veut pas.
func (t *Sorter) AddSheet(width, height uint16, pixels []render.Color) {
	t.masks = append(t.masks, newMask(width, height, pixels))
}

// Frame rend la suite ordonnée des quads, valable jusqu'à l'appel suivant.
//
// Elle ne rend pas d'erreur, et c'est voulu : une image n'a personne à qui se
// plaindre. Ce qui peut échouer a échoué au montage de la scène.
func (t *Sorter) Frame() render.DrawList {
	t.raw, t.keys, t.sources, t.ties = t.raw[:0], t.keys[:0], t.sources[:0], t.ties[:0]
	t.reveals, t.revealing, t.fromDecor = t.reveals[:0], 0, t.fromDecor[:0]
	t.uncounted = t.uncounted[:0]
	t.requests, t.overlaps = t.requests[:0], t.overlaps[:0]

	t.viewX, t.viewY, t.viewWidth, t.viewHeight = t.scene.View()
	if t.viewWidth <= 0 || t.viewHeight <= 0 {
		return t.out[:0]
	}
	t.surfaceWidth, t.surfaceHeight = t.scene.Surface()
	t.offsetX, t.offsetY = t.scene.WorldOrigin()
	t.boundsWidth, t.boundsHeight = t.viewWidth, t.viewHeight

	t.collectDecor()
	t.collectEntities()

	// Une interface n'est pas du monde : elle se pose sur toute la surface, marge
	// comprise, et c'est ce que la marge sert à porter. Le monde, lui, s'est arrêté à
	// l'aire.
	t.offsetX, t.offsetY = 0, 0
	t.boundsWidth, t.boundsHeight = t.surfaceWidth, t.surfaceHeight

	t.collectScreen()

	t.offsetX, t.offsetY = t.scene.WorldOrigin()
	t.boundsWidth, t.boundsHeight = t.viewWidth, t.viewHeight

	t.revealHidden()

	t.radix.Sort(t.keys, t.sources)
	t.breakTies()

	return t.assemble()
}

// Overlaps rend le relevé de recouvrement de la dernière image, valable jusqu'à l'appel
// suivant à [Sorter.Frame].
//
// Il ne porte que les entités qu'un jeu a marquées, dans l'ordre de leur pose, et il est
// vide avant la première image : la mesure est un produit du tri, donc elle n'existe pas
// avant qu'il ait eu lieu.
func (t *Sorter) Overlaps() render.OverlapList { return t.overlaps }

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

		// L'étage monte à l'écran pour une entité comme il monte pour le décor, et z
		// est la hauteur au-dessus de cet étage. Un personnage qui change de niveau
		// change donc son étage et rien d'autre ; porter la hauteur deux fois serait
		// deux valeurs à tenir d'accord, et rien ne le dirait au jeu.
		//
		// La somme se fait ici parce que la projection ne connaît pas l'apparence, et
		// elle ne traverse qu'une quantification : deux conversions séparées
		// arrondiraient chacune de son côté.
		ex, ey, depth, height := m.Project(rx, ry, e.Z+float64(e.Floor))

		// La strate d'une entité départage ce que la clé laisse à égalité, et sa valeur
		// zéro la met sous tout ce qui s'empile au même point. C'est le même champ que
		// pour le décor, et c'est voulu : les deux disent ce qui se superpose à un
		// endroit sans s'y effacer.
		rank := t.place(e.Aspect, ex, ey, depth, height, identities[i], e.Tier)

		// La demande s'enregistre après la pose, et seulement si elle a produit un
		// quad : une entité hors de la vue n'a rien à faire relever.
		if rank >= 0 && e.Measure {
			t.requests = append(t.requests, request{
				handle: uint32(scene.HandleAt(t.scene, i)),
				rank:   uint32(rank),
			})
		}
	}
}

// halfCellQuarters est ce qu'un demi-cellule pèse dans la profondeur, sur les deux
// axes : celle-ci se compte en quarts de cellule, donc deux fois deux quarts.
//
// Elle sert à obtenir la profondeur du coin d'une cellule depuis celle de son centre
// sans reprojeter. La soustraction est **exacte** et non une approximation : la
// profondeur est une somme décalée de quatorze bits, un demi en virgule fixe vaut
// 32 768, et deux demis font 65 536, soit exactement quatre fois le pas du décalage.
// Reprojeter le coin donnerait le même nombre et coûtait huit pour cent du décor,
// mesuré.
//
// Le chiffre dépend de la quantification de `geometry`, qui ne l'exporte pas :
// [TestCornerDepthMatchesProjection] garde l'égalité, et rougirait si elle changeait.
const halfCellQuarters = 4

// collectDecor parcourt les cellules que la vue peut montrer, couche par couche.
func (t *Sorter) collectDecor() {
	tiers := t.scene.Tiers()
	if tiers == 0 {
		return
	}

	m := t.scene.Projection()
	width, _ := t.scene.Dimensions()
	u0, v0, u1, v1 := t.window()

	t.collectingDecor = true
	defer func() { t.collectingDecor = false }()

	for tier := range tiers {
		for v := v0; v <= v1; v++ {
			for u := u0; u <= u1; u++ {
				a := t.scene.Cell(u, v, uint8(tier))
				if a.Width == 0 || a.Height == 0 {
					continue
				}

				// L'écran prend le centre de la cellule, la profondeur son coin le
				// plus lointain, et les deux ne peuvent pas être le même point.
				//
				// Une entité porte la profondeur de sa position brute. Si le décor
				// portait celle de son centre, une entité dans la moitié nord de sa
				// case aurait une profondeur plus petite que la case : elle serait
				// peinte avant, et le sol lui passerait sur les pieds. Le coin met le
				// décor derrière tout ce qui se tient dedans par construction, les
				// coordonnées d'une entité de la cellule (u, v) étant dans
				// [u, u+1[ × [v, v+1[.
				//
				// Le remappage est une rotation d'un quart de tour et des
				// renversements d'axes, donc l'image d'une cellule reste un carré
				// unité : son coin le plus lointain est le centre remappé moins un
				// demi sur chaque axe, quelle que soit l'orientation.
				//
				// L'étage est une hauteur et pas seulement un rang dans la clé. Une
				// cellule de décor n'a aucun autre champ où la porter — PlaceCell ne
				// prend que deux coordonnées et une strate —, et le facteur qui la
				// convertit en pixels vit dans la matrice. Sans cela un second niveau
				// se rangerait devant le premier tout en se dessinant au ras du sol.
				rx, ry := t.scene.Remap(float64(u)+0.5, float64(v)+0.5)
				ex, ey, depth, height := m.Project(rx, ry, float64(a.Floor))
				depth = max(depth, halfCellQuarters) - halfCellQuarters

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
// L'ancre est le pied de l'image, centré horizontalement, et l'aspect la décale de ce
// défaut — jamais l'appelant, pour les raisons que [scene.Aspect] donne.
//
// Le décalage entre dans le découpage mais pas dans le départage : celui-ci lit `ex`,
// l'abscisse du point projeté, de sorte qu'une planche rembourrée pour recaler son
// ancre ne réordonne pas ce qu'elle côtoie.
//
// **Elle rend le rang du quad posé, et -1 quand elle n'en a posé aucun.** Un retour
// plutôt qu'un huitième paramètre : elle en porte déjà sept, ce qui est la raison pour
// laquelle collectingDecor est un état, et seul le relevé de recouvrement a besoin de
// savoir où le quad a atterri.
func (t *Sorter) place(a scene.Aspect, absoluteX, absoluteY int32, depth uint32, height uint16, identity uint32, tier uint8) int {
	if a.Width == 0 || a.Height == 0 {
		return -1
	}

	ex, ey := absoluteX-t.viewX, absoluteY-t.viewY
	w, h := int32(a.Width), int32(a.Height)
	x, y := ex-w/2+int32(a.AnchorX), ey-h+int32(a.AnchorY)

	if x+w <= 0 || x >= t.boundsWidth || y+h <= 0 || y >= t.boundsHeight {
		return -1
	}

	// **Découper ici, avant que le quad existe, est un invariant et pas un détail de
	// place.** Les masques de recouvrement se bâtissent sur `t.raw`, donc découper en
	// amont est ce qui fait que la mesure ne voit jamais de pixels qui ne seront pas
	// peints. Dans l'autre ordre, une entité au bord de l'aire se lirait cachée en
	// permanence : son peint resterait entier là que son recouvert aurait déjà compté
	// la coupe, et son rappel s'allumerait sans que rien ne cloche à l'écran.
	//
	// Le miroir se réfléchit autour du rectangle du quad et non de la planche : une
	// colonne retirée à gauche de ce qui se peint vient donc de la **droite** de la
	// source. Confondre les deux est invisible sur une planche d'une seule case, et ça
	// l'a été une fois.
	left, top := max(0, -x), max(0, -y)
	right, bottom := max(0, x+w-t.boundsWidth), max(0, y+h-t.boundsHeight)

	sourceX, sourceY := a.SourceX, a.SourceY
	if a.Flags&render.MirrorX != 0 {
		sourceX += uint16(right)
	} else {
		sourceX += uint16(left)
	}
	if a.Flags&render.MirrorY != 0 {
		sourceY += uint16(bottom)
	} else {
		sourceY += uint16(top)
	}

	x, y = x+left, y+top
	w, h = w-left-right, h-top-bottom

	if a.Reveal != 0 {
		t.revealing++
	}

	// Le rang d'empilement n'est une strate que pour le décor : une entité, une pièce
	// d'écran ou un quad révélé porte [entityTier], au-dessus de toutes les strates.
	//
	// **Ce rang reproduit l'ordre que l'index de planche donnait par accident**, et
	// c'est ce qui le rend juste : un jeu charge ses tuiles avant ses personnages, donc
	// le sol sortait d'abord et l'entité par-dessus. Mesuré en essayant l'inverse —
	// laisser les entités à zéro —, sept images de référence changent et le sol se
	// peint sur les pieds du personnage. Le dire ici rend vrai pour tout jeu ce qui
	// dépendait du nom de ses fichiers.
	//
	// [scene.Entity.Tier] n'est pas ce rang et vit dans le départage, où il fonctionne
	// désormais : la planche ne figurant plus dans la clé, deux entités superposées
	// tombent dans le même groupe.
	rank := uint8(entityTier)
	if t.collectingDecor {
		rank = tier
	}

	t.sources = append(t.sources, uint32(len(t.raw)))
	t.keys = append(t.keys, Compose(a.Layer, a.Floor, depth, height, rank, 0))
	t.ties = append(t.ties, tie{x: ex, identity: identity, tier: tier})
	t.reveals = append(t.reveals, a.Reveal)
	t.fromDecor = append(t.fromDecor, t.collectingDecor)
	t.uncounted = append(t.uncounted, a.Uncounted)
	t.raw = append(t.raw, render.Quad{
		X: int16(x + t.offsetX), Y: int16(y + t.offsetY),
		SourceX: sourceX, SourceY: sourceY,
		Width: uint16(w), Height: uint16(h),
		Sheet: a.Sheet,
		Tint:  a.Tint,
		Flags: a.Flags,
	})
	return len(t.raw) - 1
}

// breakTies range ce que la clé laisse à égalité, puis inscrit le résultat dans
// l'index stable.
//
// **Par groupe et jamais sur toute l'image.** L'index ne porte que trente-deux mille
// sept cent soixante-sept valeurs là où une image en compte couramment des milliers :
// numéroter globalement le ferait saturer sur une scène dense, et la clé cesserait
// d'exprimer l'ordre sans que rien ne le signale. Un groupe ne réunit que des choses
// partageant déjà couche, étage, profondeur, hauteur et rang d'empilement.
//
// **Un groupe n'est pas court pour autant, et le croire a coûté une godoc fausse** :
// mesuré sur le banc du jeu — six strates pleines, qui est son pire cas annoncé —,
// 3 397 quads sur 3 441 appartiennent à un groupe de deux ou plus, et le plus grand en
// compte quatre-vingt-dix-neuf. Le coût reste borné parce qu'une insertion ne paie que
// la taille de son groupe, pas parce que les groupes seraient petits.
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
