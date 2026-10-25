// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package scene

import (
	"testing"

	"github.com/sprimault/lozengine/internal/geometrie"
	"github.com/sprimault/lozengine/rendu"
)

// scenePourTest monte une petite scène, la capacité d'entités étant le seul réglage
// que les cas font varier.
func scenePourTest(t *testing.T, capacite int) *Scene {
	t.Helper()

	s, err := Nouvelle(8, 8, geometrie.Dimetrique(32, 16), Options{Entites: capacite})
	if err != nil {
		t.Fatalf("montage : %v", err)
	}
	return s
}

// plein rend un aspect visible, la valeur zéro d'un aspect ne montrant rien.
func plein() Aspect {
	return Aspect{Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc()}
}

// TestNouvelleRefuse vérifie que ce qui ne peut pas marcher se dit au montage, le seul
// endroit où une scène a quelqu'un à qui le dire.
func TestNouvelleRefuse(t *testing.T) {
	cas := []struct {
		nom              string
		largeur, hauteur int
		o                Options
	}{
		{"largeur nulle", 0, 8, Options{}},
		{"hauteur négative", 8, -1, Options{}},
		{"au-delà de ce que la clé ordonne", coteMax + 1, 8, Options{}},
		{"capacité au-delà du champ de rang", 8, 8, Options{Entites: rangMax + 1}},
		{"capacité négative", 8, 8, Options{Entites: -1}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := Nouvelle(c.largeur, c.hauteur, geometrie.Dimetrique(32, 16), c.o); err == nil {
				t.Error("scène acceptée")
			}
		})
	}
}

// TestNouvelleAccepteLaBorne vérifie que la garde s'arrête là où la clé de tri cesse
// de distinguer, et pas une cellule avant.
func TestNouvelleAccepteLaBorne(t *testing.T) {
	s, err := Nouvelle(coteMax, 1, geometrie.Dimetrique(32, 16), Options{Entites: 1})
	if err != nil {
		t.Fatalf("grille de %d de côté refusée : %v", coteMax, err)
	}

	if l, h := s.Dimensions(); l != coteMax || h != 1 {
		t.Errorf("dimensions %d×%d, attendu %d×1", l, h, coteMax)
	}
}

// TestRetraitGardeLesPoigneesDesAutres couvre le piège pour lequel les poignées
// existent.
//
// Retirer comble le trou avec la dernière entité, donc une entité que personne n'a
// touchée change de place. Sans la table qui suit ce déplacement, sa poignée
// désignerait une place morte : le moteur rendrait une image irréprochable pour une
// scène fausse, et la cause serait la disparition d'une troisième entité ailleurs.
// Aucun autre cas ne peut faire échouer celui-là.
func TestRetraitGardeLesPoigneesDesAutres(t *testing.T) {
	s := scenePourTest(t, 4)

	premiere, _ := s.Poser(Entite{X: 1, Aspect: plein()})
	milieu, _ := s.Poser(Entite{X: 2, Aspect: plein()})
	derniere, _ := s.Poser(Entite{X: 3, Aspect: plein()})

	s.Retirer(milieu)

	if !s.Deplacer(premiere, 10, 0, 0) {
		t.Fatal("la première ne se désigne plus")
	}
	if !s.Deplacer(derniere, 30, 0, 0) {
		t.Fatal("la dernière ne se désigne plus après avoir comblé le trou")
	}

	vues := map[float64]bool{}
	for _, e := range Entites(s) {
		vues[e.X] = true
	}
	if len(vues) != 2 || !vues[10] || !vues[30] {
		t.Errorf("positions %v, attendu 10 et 30", vues)
	}
}

// TestPoigneePerimee vérifie les trois façons qu'une poignée a de ne plus rien
// désigner, dont la plus traître : son rang a été rendu puis repris par une autre
// entité.
func TestPoigneePerimee(t *testing.T) {
	s := scenePourTest(t, 4)

	seule, _ := s.Poser(Entite{X: 1, Aspect: plein()})
	s.Retirer(seule)

	if s.Deplacer(seule, 5, 0, 0) {
		t.Error("une poignée retirée désigne encore")
	}

	reprise, _ := s.Poser(Entite{X: 2, Aspect: plein()})
	if reprise == seule {
		t.Error("le rang repris rend la même poignée, donc la génération n'avance pas")
	}
	if s.Deplacer(seule, 5, 0, 0) {
		t.Error("l'ancienne poignée désigne l'entité qui a repris son rang")
	}

	if s.Deplacer(0, 5, 0, 0) {
		t.Error("la poignée vide désigne quelque chose")
	}
}

// TestPoserRefuseAuPlein vérifie qu'une capacité atteinte se refuse au lieu de faire
// croître la tranche, ce qui ferait allouer en pleine partie.
func TestPoserRefuseAuPlein(t *testing.T) {
	s := scenePourTest(t, 2)

	for i := 0; i < 2; i++ {
		if _, ok := s.Poser(Entite{Aspect: plein()}); !ok {
			t.Fatalf("entité %d refusée sous la capacité", i)
		}
	}

	if _, ok := s.Poser(Entite{Aspect: plein()}); ok {
		t.Error("entité acceptée au-delà de la capacité")
	}
	if n := len(Entites(s)); n != 2 {
		t.Errorf("%d entités, attendu 2", n)
	}
}

// TestPoseRefuseHorsDesBornesDeLaCle couvre le seul défaut de cette surface qui ne se
// verrait jamais : la composition de la clé ramène une couche ou un étage trop grand au
// plafond, donc un jeu qui en demande vingt en verrait six se confondre avec la
// quinzième, sur une image par ailleurs irréprochable.
//
// Les trois portes sont éprouvées parce qu'elles ont trois contrôles distincts, et
// qu'un aspect entre par n'importe laquelle. Chacune rend déjà un booléen : refuser ne
// coûte donc aucune signature.
func TestPoseRefuseHorsDesBornesDeLaCle(t *testing.T) {
	horsBornes := []struct {
		nom    string
		aspect Aspect
	}{
		{"couche", Aspect{Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc(), Couche: rendu.CoucheMax + 1}},
		{"étage", Aspect{Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc(), Etage: rendu.EtageMax + 1}},
	}
	for _, c := range horsBornes {
		t.Run(c.nom, func(t *testing.T) {
			s := scenePourTest(t, 4)

			if _, ok := s.Poser(Entite{Aspect: c.aspect}); ok {
				t.Error("Poser a accepté")
			}
			if n := len(Entites(s)); n != 0 {
				t.Errorf("%d entités posées, attendu aucune", n)
			}

			p, ok := s.Poser(Entite{Aspect: plein()})
			if !ok {
				t.Fatal("entité valide refusée")
			}
			if s.Habiller(p, c.aspect) {
				t.Error("Habiller a accepté")
			}
			if s.PoserCellule(1, 1, 0, c.aspect) {
				t.Error("PoserCellule a accepté")
			}
		})
	}
}

// TestPoseAccepteLesBornes vérifie que le refus s'arrête à la borne au lieu de la
// manger. Un hors-par-un ici retirerait la seizième couche et le soixante-quatrième
// étage, que le moteur annonce pourtant, et aucun autre cas ne le verrait.
func TestPoseAccepteLesBornes(t *testing.T) {
	s := scenePourTest(t, 2)
	extremes := Aspect{Largeur: 32, Hauteur: 16, Teinte: rendu.Blanc(), Couche: rendu.CoucheMax, Etage: rendu.EtageMax}

	if _, ok := s.Poser(Entite{Aspect: extremes}); !ok {
		t.Error("Poser a refusé la borne")
	}
	if !s.PoserCellule(1, 1, 0, extremes) {
		t.Error("PoserCellule a refusé la borne")
	}
}

// TestHabillerNeTouchePasLaPosition vérifie que changer ce qu'une entité montre la
// laisse où elle est : c'est le chemin d'un changement de direction, qui touche des
// centaines d'entités sur une même image.
func TestHabillerNeTouchePasLaPosition(t *testing.T) {
	s := scenePourTest(t, 2)

	p, _ := s.Poser(Entite{X: 3, Y: 5, Z: 1, Aspect: plein()})
	if !s.Habiller(p, Aspect{Largeur: 8, Hauteur: 8, Planche: 7, Teinte: rendu.Blanc()}) {
		t.Fatal("habillage refusé")
	}

	e := Entites(s)[0]
	if e.X != 3 || e.Y != 5 || e.Z != 1 {
		t.Errorf("position (%g, %g, %g), attendu (3, 5, 1)", e.X, e.Y, e.Z)
	}
	if e.Planche != 7 || e.Largeur != 8 {
		t.Errorf("aspect planche %d largeur %d, attendu 7 et 8", e.Planche, e.Largeur)
	}
}

// TestDecorParStrate vérifie qu'une cellule empile sans effacer, et qu'une strate
// n'existe qu'une fois qu'on y a posé quelque chose.
//
// C'est le besoin qui a donné les strates : garer un véhicule sur une chaussée sans
// perdre la chaussée, ni le passage piéton peint dessus. Un moteur qui n'offre qu'une
// chose par cellule oblige le jeu à choisir laquelle des trois il garde.
func TestDecorParStrate(t *testing.T) {
	s := scenePourTest(t, 1)

	if n := s.Strates(); n != 0 {
		t.Errorf("%d strates sur une scène neuve, attendu aucune", n)
	}

	sol := plein()
	sol.Planche = 1
	dessus := plein()
	dessus.Planche = 2

	if !s.PoserCellule(2, 3, 0, sol) || !s.PoserCellule(2, 3, 3, dessus) {
		t.Fatal("pose refusée dans les bornes")
	}

	if got := s.Cellule(2, 3, 0).Planche; got != 1 {
		t.Errorf("strate 0 : planche %d, attendu 1 — la pose du dessus l'a effacée", got)
	}
	if got := s.Cellule(2, 3, 3).Planche; got != 2 {
		t.Errorf("strate 3 : planche %d, attendu 2", got)
	}
	if got := s.Cellule(2, 3, 1); got != (Aspect{}) {
		t.Errorf("strate intermédiaire non posée : %v, attendu rien", got)
	}
	if n := s.Strates(); n != 4 {
		t.Errorf("%d strates, attendu 4 — les rangs jusqu'à la plus haute posée", n)
	}
}

// TestCelluleHorsBornes vérifie qu'un relevé hors grille rend le vide au lieu de
// paniquer : le tri relève une fenêtre qui dépasse les bords, et c'est voulu — une
// forme dont le sommet vient de sortir a encore son corps à l'image.
func TestCelluleHorsBornes(t *testing.T) {
	s := scenePourTest(t, 1)
	s.PoserCellule(0, 0, 0, plein())

	for _, c := range [][2]int{{-1, 0}, {0, -1}, {8, 0}, {0, 8}, {-5, 99}} {
		if got := s.Cellule(c[0], c[1], 0); got != (Aspect{}) {
			t.Errorf("cellule (%d, %d) rend %v, attendu rien", c[0], c[1], got)
		}
	}
	if s.PoserCellule(8, 0, 0, plein()) {
		t.Error("pose acceptée hors grille")
	}
	if s.PoserCellule(0, 0, stratesMax, plein()) {
		t.Error("pose acceptée au-delà des strates qu'une cellule empile")
	}
}

// TestLeCheminDUneImageNAllouePas vérifie la propriété dont tout le reste dépend : ce
// qu'un jeu fait à chaque image — bouger ses entités, les rhabiller, les relire — ne
// touche pas au tas.
func TestLeCheminDUneImageNAllouePas(t *testing.T) {
	s := scenePourTest(t, 64)

	poignees := make([]Poignee, 0, 64)
	for i := 0; i < 64; i++ {
		p, _ := s.Poser(Entite{X: float64(i), Aspect: plein()})
		poignees = append(poignees, p)
	}

	image := func() {
		for i, p := range poignees {
			s.Deplacer(p, float64(i)+0.5, 0, 0)
			s.Habiller(p, plein())
		}
		for _, e := range Entites(s) {
			_ = e.X
		}
	}

	if n := testing.AllocsPerRun(10, image); n != 0 {
		t.Errorf("allocations : %v, attendu aucune", n)
	}
}
