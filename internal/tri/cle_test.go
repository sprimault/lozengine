// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import (
	"testing"

	"github.com/sprimault/lozengine/rendu"
)

// TestPlafondDePlancheConcorde vérifie que ce que la clé sait distinguer et ce qu'un
// atlas accepte sont le même nombre.
//
// Les deux plafonds vivent dans des paquets qui ne peuvent pas s'atteindre — l'un
// range les planches, l'autre compose l'ordre de dessin —, et ce cas est le seul lien
// entre eux. Élargir le champ de la clé sans toucher à l'atlas laisserait des planches
// qu'aucun quad ne désignerait ; le rétrécir ferait ranger deux planches ensemble.
func TestPlafondDePlancheConcorde(t *testing.T) {
	if PlancheMax != rendu.PlanchesMax-1 {
		t.Errorf("la clé distingue %d planches, un atlas en accepte %d", PlancheMax+1, rendu.PlanchesMax)
	}
}

// TestPlafondsDeCoucheEtEtageConcordent vérifie que les bornes qu'un jeu lit sont bien
// ce que les champs de la clé distinguent.
//
// Celles-ci sont annoncées hors de ce paquet, puisque c'est le jeu qui décide combien
// de couches il emploie. Élargir un champ sans toucher à l'annonce laisserait une
// capacité que personne n'oserait employer ; le rétrécir ferait ranger ensemble deux
// couches qu'un jeu croit distinctes, et une pose hors borne serait refusée sans
// raison apparente.
func TestPlafondsDeCoucheEtEtageConcordent(t *testing.T) {
	if got := 1<<bitsCouche - 1; got != rendu.CoucheMax {
		t.Errorf("la clé distingue %d couches, le moteur en annonce %d", got+1, rendu.CoucheMax+1)
	}
	if got := 1<<bitsEtage - 1; got != rendu.EtageMax {
		t.Errorf("la clé distingue %d étages, le moteur en annonce %d", got+1, rendu.EtageMax+1)
	}
}

// TestOrdreDesChamps vérifie la propriété qui tient tout le reste : un champ plus fort
// l'emporte sur tous les suivants réunis.
//
// Chaque cas oppose une clé dont le champ étudié vaut zéro mais dont tous les champs
// plus faibles sont au maximum, à une clé dont le seul champ étudié vaut un. C'est ce
// qui attrape une inversion de deux décalages — laquelle se compile sans broncher et
// ne se voit que dans l'image, où elle ressemble à un défaut de tri.
func TestOrdreDesChamps(t *testing.T) {
	cas := []struct {
		nom          string
		avant, apres Cle
	}{
		{"la couche l'emporte sur tout",
			Composer(0, rendu.EtageMax, ProfondeurMax, HauteurMax, PlancheMax, IndexMax),
			Composer(1, 0, 0, 0, 0, 0)},
		{"l'étage l'emporte sur la profondeur",
			Composer(0, 0, ProfondeurMax, HauteurMax, PlancheMax, IndexMax),
			Composer(0, 1, 0, 0, 0, 0)},
		{"la profondeur l'emporte sur la hauteur",
			Composer(0, 0, 0, HauteurMax, PlancheMax, IndexMax),
			Composer(0, 0, 1, 0, 0, 0)},
		{"la hauteur l'emporte sur la planche",
			Composer(0, 0, 0, 0, PlancheMax, IndexMax),
			Composer(0, 0, 0, 1, 0, 0)},
		{"la planche l'emporte sur l'index",
			Composer(0, 0, 0, 0, 0, IndexMax),
			Composer(0, 0, 0, 0, 1, 0)},
		{"l'index tranche en dernier",
			Composer(0, 0, 0, 0, 0, 0),
			Composer(0, 0, 0, 0, 0, 1)},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if c.avant >= c.apres {
				t.Errorf("%#016x ne précède pas %#016x", uint64(c.avant), uint64(c.apres))
			}
		})
	}
}

// TestComposerRameneAuPlafond vérifie qu'un champ trop grand se range au bord de son
// champ au lieu de s'ajouter au voisin.
//
// Sans ce rabattement, un étage de soixante-quatre vaudrait une couche de plus : la
// chose passerait devant tout l'écran, et la cause serait un nombre dépassé de un.
func TestComposerRameneAuPlafond(t *testing.T) {
	cas := []struct {
		nom            string
		trop, plafonne Cle
	}{
		{"couche", Composer(rendu.CoucheMax+1, 0, 0, 0, 0, 0), Composer(rendu.CoucheMax, 0, 0, 0, 0, 0)},
		{"étage", Composer(0, rendu.EtageMax+1, 0, 0, 0, 0), Composer(0, rendu.EtageMax, 0, 0, 0, 0)},
		{"profondeur", Composer(0, 0, ProfondeurMax+1, 0, 0, 0), Composer(0, 0, ProfondeurMax, 0, 0, 0)},
		{"planche", Composer(0, 0, 0, 0, PlancheMax+1, 0), Composer(0, 0, 0, 0, PlancheMax, 0)},
		{"index", Composer(0, 0, 0, 0, 0, IndexMax+1), Composer(0, 0, 0, 0, 0, IndexMax)},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if c.trop != c.plafonne {
				t.Errorf("%#016x au lieu de %#016x", uint64(c.trop), uint64(c.plafonne))
			}
		})
	}
}

// TestIndexSeRelitEtSeChange vérifie le seul champ que le départage touche :
// l'ajuster ne doit déranger aucun autre.
func TestIndexSeRelitEtSeChange(t *testing.T) {
	c := Composer(3, 7, 1234, 567, 89, 12)

	if got := c.Index(); got != 12 {
		t.Errorf("index %d, attendu 12", got)
	}
	if got := c.Profondeur(); got != 1234 {
		t.Errorf("profondeur %d, attendu 1234", got)
	}

	change := c.AvecIndex(999)
	if got := change.Index(); got != 999 {
		t.Errorf("index changé %d, attendu 999", got)
	}
	if change>>decalagePlanche != c>>decalagePlanche {
		t.Error("changer l'index a dérangé un champ plus fort")
	}
}
