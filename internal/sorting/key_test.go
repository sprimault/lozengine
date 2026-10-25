// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les plafonds que la clé et la surface publique doivent annoncer du même nombre —
// deux déclarations qui divergeraient sans que rien ne le dise —, et la propriété qui
// tient tout le reste : un champ plus fort l'emporte sur tous les suivants réunis.

package sorting

import (
	"testing"

	"github.com/sprimault/lozengine/render"
)

// TestSheetCeilingMatches vérifie que ce que la clé sait distinguer et ce qu'un
// atlas accepte sont le même nombre.
//
// Les deux plafonds vivent dans des paquets qui ne peuvent pas s'atteindre — l'un
// range les planches, l'autre compose l'ordre de dessin —, et ce cas est le seul lien
// entre eux. Élargir le champ de la clé sans toucher à l'atlas laisserait des planches
// qu'aucun quad ne désignerait ; le rétrécir ferait ranger deux planches ensemble.
func TestSheetCeilingMatches(t *testing.T) {
	if MaxSheet != render.MaxSheets-1 {
		t.Errorf("la clé distingue %d planches, un atlas en accepte %d", MaxSheet+1, render.MaxSheets)
	}
}

// TestLayerAndFloorCeilingsMatch vérifie que les bornes qu'un jeu lit sont bien
// ce que les champs de la clé distinguent.
//
// Celles-ci sont annoncées hors de ce paquet, puisque c'est le jeu qui décide combien
// de couches il emploie. Élargir un champ sans toucher à l'annonce laisserait une
// capacité que personne n'oserait employer ; le rétrécir ferait ranger ensemble deux
// couches qu'un jeu croit distinctes, et une pose hors borne serait refusée sans
// raison apparente.
func TestLayerAndFloorCeilingsMatch(t *testing.T) {
	if got := 1<<layerBits - 1; got != render.MaxLayer {
		t.Errorf("la clé distingue %d couches, le moteur en annonce %d", got+1, render.MaxLayer+1)
	}
	if got := 1<<floorBits - 1; got != render.MaxFloor {
		t.Errorf("la clé distingue %d étages, le moteur en annonce %d", got+1, render.MaxFloor+1)
	}
}

// TestSideCeilingFitsInDepth vérifie que la plus grande grille annoncée
// produit une profondeur que le champ porte encore.
//
// La profondeur est la somme des deux axes, comptée en quarts de cellule : deux coins
// opposés d'une grille au maximum en donnent la plus grande valeur possible. C'est le
// seul lien entre la limite qu'un jeu lit et la largeur du champ qui la supporte, et
// il se vérifie par le calcul plutôt que par un chiffre recopié — qui aurait péri à la
// première des deux qui bouge.
func TestSideCeilingFitsInDepth(t *testing.T) {
	maxSum := 2 * (render.MaxSide - 1)

	if quarters := maxSum * 4; quarters > MaxDepth {
		t.Errorf("une grille de %d de côté donne %d quarts de profondeur, le champ en porte %d",
			render.MaxSide, quarters, MaxDepth)
	}
}

// TestFieldOrder vérifie la propriété qui tient tout le reste : un champ plus fort
// l'emporte sur tous les suivants réunis.
//
// Chaque cas oppose une clé dont le champ étudié vaut zéro mais dont tous les champs
// plus faibles sont au maximum, à une clé dont le seul champ étudié vaut un. C'est ce
// qui attrape une inversion de deux décalages — laquelle se compile sans broncher et
// ne se voit que dans l'image, où elle ressemble à un défaut de tri.
func TestFieldOrder(t *testing.T) {
	cases := []struct {
		name          string
		before, after Key
	}{
		{"la couche l'emporte sur tout",
			Compose(0, render.MaxFloor, MaxDepth, MaxHeight, MaxSheet, MaxIndex),
			Compose(1, 0, 0, 0, 0, 0)},
		{"l'étage l'emporte sur la profondeur",
			Compose(0, 0, MaxDepth, MaxHeight, MaxSheet, MaxIndex),
			Compose(0, 1, 0, 0, 0, 0)},
		{"la profondeur l'emporte sur la hauteur",
			Compose(0, 0, 0, MaxHeight, MaxSheet, MaxIndex),
			Compose(0, 0, 1, 0, 0, 0)},
		{"la hauteur l'emporte sur la planche",
			Compose(0, 0, 0, 0, MaxSheet, MaxIndex),
			Compose(0, 0, 0, 1, 0, 0)},
		{"la planche l'emporte sur l'index",
			Compose(0, 0, 0, 0, 0, MaxIndex),
			Compose(0, 0, 0, 0, 1, 0)},
		{"l'index tranche en dernier",
			Compose(0, 0, 0, 0, 0, 0),
			Compose(0, 0, 0, 0, 0, 1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.before >= c.after {
				t.Errorf("%#016x ne précède pas %#016x", uint64(c.before), uint64(c.after))
			}
		})
	}
}

// TestComposeClampsToCeiling vérifie qu'un champ trop grand se range au bord de son
// champ au lieu de s'ajouter au voisin.
//
// Sans ce rabattement, un étage de soixante-quatre vaudrait une couche de plus : la
// chose passerait devant tout l'écran, et la cause serait un nombre dépassé de un.
func TestComposeClampsToCeiling(t *testing.T) {
	cases := []struct {
		name            string
		tooBig, clamped Key
	}{
		{"couche", Compose(render.MaxLayer+1, 0, 0, 0, 0, 0), Compose(render.MaxLayer, 0, 0, 0, 0, 0)},
		{"étage", Compose(0, render.MaxFloor+1, 0, 0, 0, 0), Compose(0, render.MaxFloor, 0, 0, 0, 0)},
		{"profondeur", Compose(0, 0, MaxDepth+1, 0, 0, 0), Compose(0, 0, MaxDepth, 0, 0, 0)},
		{"planche", Compose(0, 0, 0, 0, MaxSheet+1, 0), Compose(0, 0, 0, 0, MaxSheet, 0)},
		{"index", Compose(0, 0, 0, 0, 0, MaxIndex+1), Compose(0, 0, 0, 0, 0, MaxIndex)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.tooBig != c.clamped {
				t.Errorf("%#016x au lieu de %#016x", uint64(c.tooBig), uint64(c.clamped))
			}
		})
	}
}

// TestIndexReadsBackAndChanges vérifie le seul champ que le départage touche :
// l'ajuster ne doit déranger aucun autre.
func TestIndexReadsBackAndChanges(t *testing.T) {
	c := Compose(3, 7, 1234, 567, 89, 12)

	if got := c.Index(); got != 12 {
		t.Errorf("index %d, attendu 12", got)
	}
	if got := c.Depth(); got != 1234 {
		t.Errorf("profondeur %d, attendu 1234", got)
	}

	change := c.WithIndex(999)
	if got := change.Index(); got != 999 {
		t.Errorf("index changé %d, attendu 999", got)
	}
	if change>>sheetShift != c>>sheetShift {
		t.Error("changer l'index a dérangé un champ plus fort")
	}
}
