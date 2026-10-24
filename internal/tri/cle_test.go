// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package tri

import "testing"

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
			Composer(0, EtageMax, ProfondeurMax, HauteurMax, PlancheMax, IndexMax),
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
		{"couche", Composer(CoucheMax+1, 0, 0, 0, 0, 0), Composer(CoucheMax, 0, 0, 0, 0, 0)},
		{"étage", Composer(0, EtageMax+1, 0, 0, 0, 0), Composer(0, EtageMax, 0, 0, 0, 0)},
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

// TestIndexSeRelitEtSeChange vérifie le seul champ que le tri topologique touche :
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
