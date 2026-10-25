// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package scene

import (
	"fmt"

	"github.com/sprimault/lozengine/internal/geometrie"
)

// coteMax est le plus grand côté qu'une grille puisse avoir.
//
// Le chiffre n'est pas rond par hasard : la clé de tri compte la profondeur — la somme
// des deux axes — sur dix-huit bits en quarts de cellule, soit 65 536 cellules de
// somme, donc la moitié par côté. Au-delà, deux cellules éloignées se replieraient sur
// la même valeur et s'ordonneraient par leur rang d'insertion : le rendu resterait
// prévisible, et faux.
const coteMax = 32768

// stratesMax est le nombre de choses qu'une cellule peut empiler.
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
// Chaque strate ouverte coûte une tranche de la taille de la carte, soit environ cinq
// mégaoctets sur cinq cents cellules de côté. Une strate creuse — quelques véhicules
// sur une grande carte — la paie donc pleine, ce qui est à reconsidérer le jour où une
// mesure le demande, pas avant.
const stratesMax = 16

// entitesParDefaut est la capacité qu'une scène prend quand l'appelant n'en demande
// aucune — au-dessus de ce qu'un jeu de horde emploie, pour deux cents kilooctets.
const entitesParDefaut = 4096

// Options porte ce qu'une scène admet de régler, et sa valeur zéro convient.
//
// Ce qui n'y figure pas n'a pas de défaut possible. Une grille n'a pas de taille
// raisonnable par défaut, et une projection encore moins : lui en donner une
// reviendrait à choisir une taille de tuile à la place du jeu.
type Options struct {
	// Entites est le nombre d'entités vivantes que la scène porte à la fois. Zéro
	// prend le défaut du moteur.
	//
	// La capacité est fixe et allouée une fois. Une tranche qui croîtrait au milieu
	// d'une horde ferait passer le ramassage de mémoire là où il se verrait le plus.
	Entites int
}

// Scene porte une grille de décor, des entités, et la vue par laquelle on les regarde.
//
// Elle s'obtient de [Nouvelle] et s'emploie par ses méthodes. Rien n'y est alloué après
// la construction, hors la pose d'une couche de décor, et c'est la condition pour que
// le tri n'alloue pas davantage.
type Scene struct {
	largeur, hauteur int
	projection       geometrie.Matrice

	// Les entités vivantes occupent le début de la tranche, sans trou, ce qui rend
	// leur parcours dense. Retirer comble le trou avec la dernière : c'est ce qui
	// garde la densité, et c'est aussi ce qui déplace une entité qui n'a rien
	// demandé — d'où les poignées, qui survivent à ce déplacement.
	entites []Entite
	rangs   []uint32 // rang de poignée → place dans entites
	places  []uint32 // place dans entites → rang de poignée
	ages    []uint32 // rang de poignée → génération courante
	libres  []uint32 // les rangs qu'un retrait a rendus

	// Le décor : une tranche par strate, allouée à la première pose sur celle-ci. Un
	// jeu qui n'en empile qu'une ne paie que celle-là.
	decor [][]Aspect

	// La vue. Le point visé est gardé en cellules plutôt qu'en pixels, parce qu'une
	// rotation le change de place à l'écran sans le changer de place dans le monde.
	cibleX, cibleY         float64
	vueX, vueY             int32
	vueLargeur, vueHauteur int32
	orientation            Orientation
}

// Nouvelle monte une scène sur une grille et une projection.
//
// La projection est obligatoire et n'est pas vérifiée : sa valeur zéro projette tout
// sur l'origine, ce que sa propre documentation annonce comme l'échec visible.
//
// C'est le seul moment où une scène rend une erreur. Passé ce point, plus rien sur le
// chemin d'une image n'en rend, parce qu'une image n'a nulle part où la mettre.
func Nouvelle(largeur, hauteur int, projection geometrie.Matrice, o Options) (*Scene, error) {
	if largeur <= 0 || hauteur <= 0 {
		return nil, fmt.Errorf("grille de %d×%d : les deux côtés sont obligatoires", largeur, hauteur)
	}
	if largeur > coteMax || hauteur > coteMax {
		return nil, fmt.Errorf("grille de %d×%d : %d cellules de côté au plus, ce que la clé de tri sait ordonner", largeur, hauteur, coteMax)
	}

	capacite := o.Entites
	if capacite == 0 {
		capacite = entitesParDefaut
	}
	if capacite < 0 || capacite > rangMax {
		return nil, fmt.Errorf("capacité de %d entités : %d au plus", capacite, rangMax)
	}

	return &Scene{
		largeur:    largeur,
		hauteur:    hauteur,
		projection: projection,
		entites:    make([]Entite, 0, capacite),
		rangs:      make([]uint32, 0, capacite),
		places:     make([]uint32, 0, capacite),
		ages:       make([]uint32, 0, capacite),
		libres:     make([]uint32, 0, capacite),
	}, nil
}

// Dimensions rend les côtés de la grille, en cellules.
func (s *Scene) Dimensions() (largeur, hauteur int) { return s.largeur, s.hauteur }

// Projection rend la matrice sur laquelle la scène a été montée.
func (s *Scene) Projection() geometrie.Matrice { return s.projection }

// Poser ajoute une entité et rend la poignée qui la désignera.
//
// Le second résultat est faux quand la capacité est atteinte, ou quand la couche ou
// l'étage de l'aspect dépasse ce que l'ordre de dessin distingue ; rien n'a alors été
// posé. C'est un refus et non une erreur : il arrive en cours de partie, là où une
// erreur n'aurait personne à qui se plaindre.
func (s *Scene) Poser(e Entite) (Poignee, bool) {
	if !e.tientDansLaCle() {
		return 0, false
	}
	if len(s.entites) == cap(s.entites) {
		return 0, false
	}

	place := uint32(len(s.entites))
	s.entites = append(s.entites, e)

	var rang uint32
	if n := len(s.libres); n > 0 {
		rang, s.libres = s.libres[n-1], s.libres[:n-1]
	} else {
		rang = uint32(len(s.rangs))
		s.rangs = append(s.rangs, 0)
		s.ages = append(s.ages, 1)
	}

	s.rangs[rang] = place
	s.places = append(s.places, rang)
	return composer(rang, s.ages[rang]), true
}

// Retirer enlève l'entité qu'une poignée désigne, et ne fait rien quand elle ne
// désigne plus rien.
//
// La dernière entité vient combler le trou, donc sa place change — mais pas sa
// poignée, et c'est tout l'intérêt d'en avoir une.
func (s *Scene) Retirer(p Poignee) {
	place, ok := s.place(p)
	if !ok {
		return
	}

	rang := s.places[place]
	derniere := uint32(len(s.entites) - 1)

	if place != derniere {
		deplacee := s.places[derniere]
		s.entites[place] = s.entites[derniere]
		s.places[place] = deplacee
		s.rangs[deplacee] = place
	}

	s.entites = s.entites[:derniere]
	s.places = s.places[:derniere]

	// La génération avance au retrait et non au réemploi : entre les deux, le rang
	// existe encore et l'ancienne poignée passerait sinon pour valide.
	if s.ages[rang]++; s.ages[rang] > generationMax {
		s.ages[rang] = 1
	}
	s.libres = append(s.libres, rang)
}

// Deplacer change la position d'une entité, et rend faux quand la poignée ne désigne
// plus rien.
func (s *Scene) Deplacer(p Poignee, x, y, z float64) bool {
	place, ok := s.place(p)
	if !ok {
		return false
	}

	s.entites[place].X, s.entites[place].Y, s.entites[place].Z = x, y, z
	return true
}

// Habiller change ce qu'une entité montre sans toucher à sa position, et rend faux
// quand la poignée ne désigne plus rien ou que l'aspect sort des bornes de la clé.
//
// C'est par là que passe un changement de direction ou d'image d'animation, qui
// touchent des centaines d'entités sur une même image.
func (s *Scene) Habiller(p Poignee, a Aspect) bool {
	if !a.tientDansLaCle() {
		return false
	}
	place, ok := s.place(p)
	if !ok {
		return false
	}

	s.entites[place].Aspect = a
	return true
}

// Entites rend les entités vivantes, dans un ordre qui n'est pas celui du dessin.
//
// La tranche vaut jusqu'à la prochaine pose ou au prochain retrait. Le tri la lit sans
// la copier, et c'est ce qui lui évite d'allouer à chaque image.
//
// **Une fonction et non une méthode, délibérément.** C'est une porte de service pour le
// tri, pas une opération qu'un jeu appelle : la tranche qu'elle rend est l'état interne
// de la scène, qu'une écriture depuis l'extérieur désaccorderait des poignées. Le paquet
// racine republie [Scene] par un alias, qui emporte toutes ses méthodes et aucune de ses
// fonctions — c'est donc ce qui garde la porte fermée, là où un commentaire ne le
// ferait pas.
func Entites(s *Scene) []Entite { return s.entites }

// Identites rend, dans le même ordre qu'[Entites], un numéro par entité qui ne change
// jamais tant qu'elle vit.
//
// La place d'une entité, elle, change : un retrait comble le trou avec la dernière.
// Un ordre de dessin qui se départagerait sur la place changerait donc parce qu'une
// **troisième** entité est morte ailleurs, et deux images de suite ne montreraient pas
// la même chose. C'est ce que le tri emploie pour départager ce que la profondeur ne
// sépare pas.
//
// Fonction et non méthode, pour la raison qui vaut pour [Entites].
func Identites(s *Scene) []uint32 { return s.places }

// PoserCellule pose un aspect sur une cellule de la grille, à la strate donnée, et
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
func (s *Scene) PoserCellule(x, y int, strate uint8, a Aspect) bool {
	if x < 0 || y < 0 || x >= s.largeur || y >= s.hauteur || int(strate) >= stratesMax {
		return false
	}
	if !a.tientDansLaCle() {
		return false
	}

	rang := int(strate)
	for len(s.decor) <= rang {
		s.decor = append(s.decor, nil)
	}
	if s.decor[rang] == nil {
		s.decor[rang] = make([]Aspect, s.largeur*s.hauteur)
	}

	s.decor[rang][y*s.largeur+x] = a
	return true
}

// Cellule rend l'aspect posé sur une cellule, ou sa valeur zéro quand il n'y a rien —
// ce qui couvre aussi une cellule ou une strate hors bornes.
//
// Le tri l'appelle une fois par cellule visible et par strate, soit quelques milliers
// de fois par image : pas de tranche rendue, pas de copie, rien à libérer.
func (s *Scene) Cellule(x, y int, strate uint8) Aspect {
	if x < 0 || y < 0 || x >= s.largeur || y >= s.hauteur || int(strate) >= len(s.decor) {
		return Aspect{}
	}
	if s.decor[strate] == nil {
		return Aspect{}
	}

	return s.decor[strate][y*s.largeur+x]
}

// Strates rend le nombre de strates qu'une pose a ouvertes, et donc la borne du
// parcours que le tri a à faire.
func (s *Scene) Strates() int { return len(s.decor) }

// place rend la position d'une entité dans la tranche, et dit si la poignée vaut
// encore.
func (s *Scene) place(p Poignee) (uint32, bool) {
	rang, generation := p.rang(), p.generation()

	if generation == 0 || rang >= uint32(len(s.rangs)) || s.ages[rang] != generation {
		return 0, false
	}
	return s.rangs[rang], true
}
