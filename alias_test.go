// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"testing"

	"github.com/sprimault/lozengine/geometrie"
	"github.com/sprimault/lozengine/rendu"
	"github.com/sprimault/lozengine/scene"
)

// TestConstantesRepubliees vérifie que chaque constante republiée vaut celle du
// noyau. Un bloc const Go répète l'expression précédente quand une ligne n'en
// porte pas : l'oubli ne se compile pas en erreur, il se compile en valeur
// fausse.
func TestConstantesRepubliees(t *testing.T) {
	cas := []struct {
		nom           string
		racine, noyau Drapeaux
	}{
		{"MiroirX", MiroirX, rendu.MiroirX},
		{"MiroirY", MiroirY, rendu.MiroirY},
		{"Aplat", Aplat, rendu.Aplat},
	}
	for _, c := range cas {
		if c.racine != c.noyau {
			t.Errorf("%s republié : %#b, attendu %#b", c.nom, c.racine, c.noyau)
		}
	}
}

// TestTamponRepublie vérifie que le tampon traverse la surface publique sans
// conversion, et que sa borne republiée est bien celle du noyau.
//
// L'annotation de type explicite est le test lui-même : elle ne compile que si Tampon
// désigne le type du noyau. Une enveloppe, ou un type distinct de même forme,
// obligerait un hôte à convertir l'image à chaque passage de frontière.
func TestTamponRepublie(t *testing.T) {
	var racine *Tampon = rendu.NouveauTampon(2, 1)

	if BorneTampon != rendu.BorneTampon {
		t.Errorf("BorneTampon republié : %d, attendu %d", BorneTampon, rendu.BorneTampon)
	}
	if got := NouveauTampon(2, 1); got.Largeur != racine.Largeur || len(got.Pixels) != len(racine.Pixels) {
		t.Errorf("NouveauTampon republié rend %d×%d et %d pixels", got.Largeur, got.Hauteur, len(got.Pixels))
	}
}

// TestConstructeursRepublies vérifie que chaque enveloppe rend la matrice du noyau.
//
// Ce qu'elle est seule à pouvoir attraper : une enveloppe qui appellerait le mauvais
// constructeur, ou qui échangerait ses deux paramètres, se compilerait sans broncher
// et ne se verrait qu'à l'image. La comparaison elle-même prouve au passage que
// Matrice est bien un alias et non un type distinct — deux types différents ne se
// compareraient pas.
func TestConstructeursRepublies(t *testing.T) {
	cas := []struct {
		nom           string
		racine, noyau Matrice
	}{
		{"Dimetrique", Dimetrique(32, 16), geometrie.Dimetrique(32, 16)},
		{"Isometrique", Isometrique(32), geometrie.Isometrique(32)},
		{"Grille", Grille(32), geometrie.Grille(32)},
		{"Brute", Brute(2, 3, 5, 7), geometrie.Brute(2, 3, 5, 7)},
	}
	for _, c := range cas {
		if c.racine != c.noyau {
			t.Errorf("%s republié ne rend pas la matrice du noyau", c.nom)
		}
	}
}

// TestOrientationsRepubliees vérifie que les quatre quarts de tour valent ceux du
// noyau, et les plafonds de couche et d'étage avec eux.
//
// Même piège que pour les drapeaux, plus un second : une ligne qui recopierait la
// mauvaise constante — UnQuart valant le demi-tour — se compile sans broncher et ne se
// voit qu'à l'image, où elle ressemble à un défaut de caméra.
func TestOrientationsRepubliees(t *testing.T) {
	cas := []struct {
		nom           string
		racine, noyau Orientation
	}{
		{"SansRotation", SansRotation, scene.SansRotation},
		{"UnQuart", UnQuart, scene.UnQuart},
		{"DemiTour", DemiTour, scene.DemiTour},
		{"TroisQuarts", TroisQuarts, scene.TroisQuarts},
	}
	for _, c := range cas {
		if c.racine != c.noyau {
			t.Errorf("%s republié : %d, attendu %d", c.nom, c.racine, c.noyau)
		}
	}

	if CoucheMax != rendu.CoucheMax {
		t.Errorf("CoucheMax republié : %d, attendu %d", CoucheMax, rendu.CoucheMax)
	}
	if EtageMax != rendu.EtageMax {
		t.Errorf("EtageMax republié : %d, attendu %d", EtageMax, rendu.EtageMax)
	}
	if CoteMax != rendu.CoteMax {
		t.Errorf("CoteMax republié : %d, attendu %d", CoteMax, rendu.CoteMax)
	}
	if StratesMax != scene.StratesMax {
		t.Errorf("StratesMax republié : %d, attendu %d", StratesMax, scene.StratesMax)
	}
}

// TestPlanchesSUtilisentDepuisLaRacine suit le chemin complet d'une image de jeu : une
// planche, sa découpe en grille, son entrée dans un atlas, et l'index qu'un quad
// désignera — sans rien importer d'autre que ce paquet.
//
// Le rectangle est comparé à un littéral, ce qui ne compile que si ses champs sont
// atteignables depuis ici. C'est précisément ce qu'un alias vers un paquet interne ne
// donnait pas.
func TestPlanchesSUtilisentDepuisLaRacine(t *testing.T) {
	planche := NouvellePlanche(64, 32)
	if err := planche.DecouperGrille(32, 16); err != nil {
		t.Fatalf("découpe : %v", err)
	}

	if n := len(planche.Rectangles); n != 4 {
		t.Fatalf("%d rectangles, attendu 4", n)
	}
	if got := planche.Rectangles[3]; got != (Rectangle{X: 32, Y: 16, Largeur: 32, Hauteur: 16}) {
		t.Errorf("dernière case %+v", got)
	}

	var atlas Atlas
	index, err := atlas.Ajouter(planche)
	if err != nil {
		t.Fatalf("ajout à l'atlas : %v", err)
	}
	if index != 0 {
		t.Errorf("index %d, attendu 0", index)
	}
}

// TestSceneSUtiliseDepuisLaRacine vérifie qu'un jeu décrit une scène entière sans rien
// importer d'autre que ce paquet : grille, entités, décor, caméra.
//
// C'est la porte que ce lot ouvre, et le seul cas qui la démontre de bout en bout — un
// alias manquant, ou un constructeur qui échangerait ses paramètres, ne se verrait
// nulle part ailleurs.
func TestSceneSUtiliseDepuisLaRacine(t *testing.T) {
	s, err := NouvelleScene(16, 16, Dimetrique(32, 16), OptionsScene{Entites: 8})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}

	p, ok := s.Poser(Entite{X: 2, Y: 3, Aspect: Aspect{Largeur: 32, Hauteur: 16, Teinte: Blanc()}})
	if !ok {
		t.Fatal("entité refusée")
	}
	if !s.Deplacer(p, 4, 5, 1) {
		t.Fatal("déplacement refusé")
	}

	// Aux deux bornes, pour que le cas démontre aussi qu'elles sont employables.
	haut := Aspect{Largeur: 32, Hauteur: 16, Teinte: Blanc(), Couche: CoucheMax, Etage: EtageMax}
	if !s.PoserCellule(1, 1, 0, haut) {
		t.Fatal("cellule refusée aux bornes")
	}
	if got := s.Cellule(1, 1, 0); got.Couche != CoucheMax || got.Etage != EtageMax {
		t.Errorf("cellule relue en couche %d étage %d", got.Couche, got.Etage)
	}

	s.Orienter(DemiTour)
	if got := s.Orientation(); got != DemiTour {
		t.Errorf("orientation %d, attendu %d", got, DemiTour)
	}

	s.Cadrer(4, 5, 480, 270)
	if _, _, largeur, hauteur := s.Vue(); largeur != 480 || hauteur != 270 {
		t.Errorf("vue de %d×%d, attendu 480×270", largeur, hauteur)
	}
}

// TestMatriceSUtiliseDepuisLaRacine vérifie qu'une matrice obtenue de la racine
// s'emploie sans rien importer d'autre : c'est tout ce qu'un consommateur peut en
// faire au sortir de ce jalon, et le seul cas qui le démontre.
func TestMatriceSUtiliseDepuisLaRacine(t *testing.T) {
	m := Dimetrique(32, 16).AvecEtage(8)

	ex, ey, profondeur, hauteur := m.Projeter(1, 1, 1)
	if ex != 0 || ey != 8 || profondeur != 8 || hauteur != 4 {
		t.Errorf("projeté (%d, %d), profondeur %d, hauteur %d — attendu (0, 8), 8 et 4", ex, ey, profondeur, hauteur)
	}

	if x, y := m.Inverser(ex, ey); x != 0.5 || y != 0.5 {
		t.Errorf("inversé (%g, %g), attendu (0.5, 0.5)", x, y)
	}
	if n := m.CellulesMax(480, 270); n == 0 {
		t.Error("majorant de cellules nul")
	}
}
