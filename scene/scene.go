// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La scène et les tables qui la tiennent : les entités serrées sans trou pour que leur
// parcours soit dense, les rangs qui survivent au retrait d'une voisine, et une tranche
// de décor par strate, allouée seulement si le jeu s'en sert.

package scene

import (
	"fmt"

	"github.com/sprimault/lozengine/geometry"
	"github.com/sprimault/lozengine/render"
)

// MaxTiers est le nombre de choses qu'une cellule peut empiler.
//
// Une strate n'est pas une couche. La couche d'un aspect est un rang global que la clé
// de tri place au-dessus de tout — ce qu'une interface emploiera —, là où une strate ne
// départage que ce qui occupe la même cellule : une chaussée, son marquage, le véhicule
// garé dessus. Les confondre ferait passer tous les véhicules d'une carte devant toutes
// les créatures, quelle que soit leur profondeur.
//
// L'empilement est l'usage courant du genre — un sol, la voirie peinte dessus, les
// plantations, les véhicules —, donc seize laisse de la marge sans en promettre trop :
// un décor qui en empile davantage sur une seule cellule décrit autre chose qu'un
// décor.
//
// Chaque strate ouverte coûte une tranche de la taille de la carte — environ cinq
// mégaoctets sur cinq cents cellules de côté —, qu'elle porte trois objets ou la carte
// entière. Quelques véhicules sur une grande carte se décrivent donc mieux en entités.
const MaxTiers = 16

// defaultEntities est la capacité qu'une scène prend quand l'appelant n'en demande
// aucune : deux cents kilooctets, et [Options.Entities] la règle au-delà.
const defaultEntities = 4096

// Options porte ce qu'une scène admet de régler, et sa valeur zéro convient.
//
// Ce qui n'y figure pas n'a pas de défaut possible. Une grille n'a pas de taille
// raisonnable par défaut, et une projection encore moins : lui en donner une
// reviendrait à choisir une taille de tuile à la place du jeu.
type Options struct {
	// Entities est le nombre d'entités vivantes que la scène porte à la fois. Zéro
	// prend le défaut du moteur.
	//
	// La capacité est fixe et allouée une fois. Une tranche qui croîtrait au milieu
	// d'une horde ferait passer le ramassage de mémoire là où il se verrait le plus.
	Entities int
}

// Scene porte une grille de décor, des entités, et la vue par laquelle on les regarde.
//
// Elle s'obtient de [New] et s'emploie par ses méthodes. Rien n'y est alloué après
// la construction, hors la pose d'une couche de décor, et c'est la condition pour que
// le tri n'alloue pas davantage.
type Scene struct {
	width, height int
	projection    geometry.Matrix

	// Les entités vivantes occupent le début de la tranche, sans trou, ce qui rend
	// leur parcours dense. Retirer comble le trou avec la dernière : c'est ce qui
	// garde la densité, et c'est aussi ce qui déplace une entité qui n'a rien
	// demandé — d'où les poignées, qui survivent à ce déplacement.
	entities    []Entity
	ranks       []uint32 // rang de poignée → place dans entities
	slots       []uint32 // place dans entities → rang de poignée
	generations []uint32 // rang de poignée → génération courante
	free        []uint32 // les rangs qu'un retrait a rendus

	// Le décor : une tranche par strate, allouée à la première pose sur celle-ci. Un
	// jeu qui n'en empile qu'une ne paie que celle-là.
	decor [][]Aspect

	// La vue. Le point visé est gardé en cellules plutôt qu'en pixels, parce qu'une
	// rotation le change de place à l'écran sans le changer de place dans le monde.
	focusX, focusY        float64
	viewX, viewY          int32
	viewWidth, viewHeight int32
	orientation           Orientation
}

// New monte une scène sur une grille et une projection.
//
// La projection est obligatoire et n'est pas vérifiée : sa valeur zéro projette tout
// sur l'origine, ce que sa propre documentation annonce comme l'échec visible.
//
// C'est le seul moment où une scène rend une erreur. Passé ce point, plus rien sur le
// chemin d'une image n'en rend, parce qu'une image n'a nulle part où la mettre.
func New(width, height int, projection geometry.Matrix, o Options) (*Scene, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("grid of %d×%d: both sides are required", width, height)
	}
	if width > render.MaxSide || height > render.MaxSide {
		return nil, fmt.Errorf("grid of %d×%d: %d cells per side at most, what the sort key can order", width, height, render.MaxSide)
	}

	capacity := o.Entities
	if capacity == 0 {
		capacity = defaultEntities
	}
	if capacity < 0 || capacity > maxRank {
		return nil, fmt.Errorf("capacity of %d entities: %d at most", capacity, maxRank)
	}

	return &Scene{
		width:       width,
		height:      height,
		projection:  projection,
		entities:    make([]Entity, 0, capacity),
		ranks:       make([]uint32, 0, capacity),
		slots:       make([]uint32, 0, capacity),
		generations: make([]uint32, 0, capacity),
		free:        make([]uint32, 0, capacity),
	}, nil
}

// Dimensions rend les côtés de la grille, en cellules.
func (s *Scene) Dimensions() (width, height int) { return s.width, s.height }

// Projection rend la matrice sur laquelle la scène a été montée.
func (s *Scene) Projection() geometry.Matrix { return s.projection }

// Place ajoute une entité et rend la poignée qui la désignera.
//
// Le second résultat est faux quand la capacité est atteinte, ou quand la couche ou
// l'étage de l'aspect dépasse ce que l'ordre de dessin distingue ; rien n'a alors été
// posé. C'est un refus et non une erreur : il arrive en cours de partie, là où une
// erreur n'aurait personne à qui se plaindre.
func (s *Scene) Place(e Entity) (Handle, bool) {
	if !e.fitsInKey() {
		return 0, false
	}
	if len(s.entities) == cap(s.entities) {
		return 0, false
	}

	slot := uint32(len(s.entities))
	s.entities = append(s.entities, e)

	var rank uint32
	if n := len(s.free); n > 0 {
		rank, s.free = s.free[n-1], s.free[:n-1]
	} else {
		rank = uint32(len(s.ranks))
		s.ranks = append(s.ranks, 0)
		s.generations = append(s.generations, 1)
	}

	s.ranks[rank] = slot
	s.slots = append(s.slots, rank)
	return compose(rank, s.generations[rank]), true
}

// Remove enlève l'entité qu'une poignée désigne, et ne fait rien quand elle ne
// désigne plus rien.
//
// La dernière entité vient combler le trou, donc sa place change — mais pas sa
// poignée, et c'est tout l'intérêt d'en avoir une.
func (s *Scene) Remove(p Handle) {
	slot, ok := s.slot(p)
	if !ok {
		return
	}

	rank := s.slots[slot]
	last := uint32(len(s.entities) - 1)

	if slot != last {
		moved := s.slots[last]
		s.entities[slot] = s.entities[last]
		s.slots[slot] = moved
		s.ranks[moved] = slot
	}

	s.entities = s.entities[:last]
	s.slots = s.slots[:last]

	// La génération avance au retrait et non au réemploi : entre les deux, le rang
	// existe encore et l'ancienne poignée passerait sinon pour valide.
	if s.generations[rank]++; s.generations[rank] > maxGeneration {
		s.generations[rank] = 1
	}
	s.free = append(s.free, rank)
}

// Move change la position d'une entité, et rend faux quand la poignée ne désigne
// plus rien.
func (s *Scene) Move(p Handle, x, y, z float64) bool {
	slot, ok := s.slot(p)
	if !ok {
		return false
	}

	s.entities[slot].X, s.entities[slot].Y, s.entities[slot].Z = x, y, z
	return true
}

// SetAspect change ce qu'une entité montre sans toucher à sa position, et rend faux
// quand la poignée ne désigne plus rien ou que l'aspect sort des bornes de la clé.
//
// C'est par là que passe un changement de direction ou d'image d'animation, qui
// touchent des centaines d'entités sur une même image.
func (s *Scene) SetAspect(p Handle, a Aspect) bool {
	if !a.fitsInKey() {
		return false
	}
	slot, ok := s.slot(p)
	if !ok {
		return false
	}

	s.entities[slot].Aspect = a
	return true
}

// Entities rend les entités vivantes, dans un ordre qui n'est pas celui du dessin.
//
// La tranche vaut jusqu'à la prochaine pose ou au prochain retrait. Le tri la lit sans
// la copier, et c'est ce qui lui évite d'allouer à chaque image.
//
// **Une fonction et non une méthode, délibérément.** C'est une porte de service pour le
// tri, pas une opération qu'un jeu appelle, et le paquet racine republie [Scene] par un
// alias — qui emporte ses méthodes et aucune de ses fonctions. Un jeu qui importe la
// racine seule ne la voit donc pas.
//
// **Pour modifier une entité, c'est [Scene.Move] et [Scene.SetAspect]**, jamais la
// tranche rendue ici. Écrire dedans contourne le contrôle de bornes de SetAspect : une
// couche trop grande repasserait sous la saturation de la clé, où deux couches
// distinctes se confondent. Rien d'autre ne casse — la longueur de la tranche n'est pas
// modifiable de l'extérieur, et changer X, Y ou Z ne fait que ce que Move fait.
func Entities(s *Scene) []Entity { return s.entities }

// Identities rend, dans le même ordre qu'[Entities], un numéro par entité qui ne change
// jamais tant qu'elle vit.
//
// La place d'une entité, elle, change : un retrait comble le trou avec la dernière.
// Un ordre de dessin qui se départagerait sur la place changerait donc parce qu'une
// **troisième** entité est morte ailleurs, et deux images de suite ne montreraient pas
// la même chose. C'est ce que le tri emploie pour départager ce que la profondeur ne
// sépare pas.
//
// Fonction et non méthode, pour la raison qui vaut pour [Entities].
func Identities(s *Scene) []uint32 { return s.slots }

// PlaceCell pose un aspect sur une cellule de la grille, à la strate donnée, et
// rend faux si la cellule ou la strate sort des bornes, ou si la couche ou l'étage de
// l'aspect dépasse ce que l'ordre de dessin distingue.
//
// La strate dit ce qui s'empile sur une même cellule sans l'effacer — une chaussée, le
// passage piéton peint dessus, le véhicule qui s'y gare. Sans elle il faudrait choisir
// lequel des trois la cellule porte, et un objet posé effacerait le sol sous lui. C'est
// la façon dont un jeu isométrique pose son décor, et un moteur qui ne l'offre pas
// oblige chacun à la réinventer.
//
// Elle ne range qu'entre choses d'une même cellule : à l'image, une strate haute ne
// passe pas devant ce qui est plus proche, elle passe devant ce qu'elle recouvre. Le
// rang global, celui qu'une interface emploiera, est la couche de l'aspect.
//
// Une strate s'alloue à sa première pose. Un aspect de largeur ou de hauteur nulle
// vide la cellule.
func (s *Scene) PlaceCell(x, y int, tier uint8, a Aspect) bool {
	if x < 0 || y < 0 || x >= s.width || y >= s.height || int(tier) >= MaxTiers {
		return false
	}
	if !a.fitsInKey() {
		return false
	}

	t := int(tier)
	for len(s.decor) <= t {
		s.decor = append(s.decor, nil)
	}
	if s.decor[t] == nil {
		s.decor[t] = make([]Aspect, s.width*s.height)
	}

	s.decor[t][y*s.width+x] = a
	return true
}

// Cell rend l'aspect posé sur une cellule, ou sa valeur zéro quand il n'y a rien —
// ce qui couvre aussi une cellule ou une strate hors bornes.
//
// Le tri l'appelle une fois par cellule visible et par strate, soit quelques milliers
// de fois par image : pas de tranche rendue, pas de copie, rien à libérer.
func (s *Scene) Cell(x, y int, tier uint8) Aspect {
	if x < 0 || y < 0 || x >= s.width || y >= s.height || int(tier) >= len(s.decor) {
		return Aspect{}
	}
	if s.decor[tier] == nil {
		return Aspect{}
	}

	return s.decor[tier][y*s.width+x]
}

// Tiers rend le nombre de strates qu'une pose a ouvertes, et donc la borne du
// parcours que le tri a à faire.
func (s *Scene) Tiers() int { return len(s.decor) }

// slot rend la position d'une entité dans la tranche, et dit si la poignée vaut
// encore.
func (s *Scene) slot(p Handle) (uint32, bool) {
	rank, generation := p.rank(), p.generation()

	if generation == 0 || rank >= uint32(len(s.ranks)) || s.generations[rank] != generation {
		return 0, false
	}
	return s.ranks[rank], true
}
