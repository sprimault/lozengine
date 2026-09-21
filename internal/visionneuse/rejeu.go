// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sprimault/lozengine/internal/raster"
	"github.com/sprimault/lozengine/internal/rendu"
)

// Rejeu tient ce qu'un scénario réutilise d'une image à l'autre : son atlas, son
// tampon, sa liste de quads et le tampon de sortie s'il agrandit.
type Rejeu struct {
	scenario Scenario
	atlas    *raster.Atlas
	tampon   *raster.Tampon
	sortie   *raster.Tampon
	quads    rendu.Rendu
}

// NouveauRejeu monte l'atlas du scénario et prépare le tampon à sa résolution.
//
// C'est ici que les erreurs ont le droit d'exister — un atlas se construit une fois,
// avant la première image. Passé ce point, plus rien n'échoue jusqu'à l'écriture.
func NouveauRejeu(s Scenario) (*Rejeu, error) {
	if s.Atlas == nil || s.Quads == nil {
		return nil, fmt.Errorf("scénario %q : Atlas et Quads sont obligatoires", s.Nom)
	}

	atlas, err := s.Atlas()
	if err != nil {
		return nil, fmt.Errorf("atlas du scénario %q : %w", s.Nom, err)
	}
	return &Rejeu{
		scenario: s,
		atlas:    atlas,
		tampon:   raster.NouveauTampon(s.Largeur, s.Hauteur),
	}, nil
}

// Image rend une image et rend le tampon interne, à la résolution du scénario.
//
// Le tampon est celui du rejeu et se réécrit à chaque appel : le lire, le comparer ou
// l'écrire se fait avant l'image suivante. C'est cette image-là que la suite de
// références compare, jamais une image agrandie — l'échelle est un habillage, pas un
// résultat de rendu.
func (r *Rejeu) Image(numero int, e Entrees) *raster.Tampon {
	r.quads = r.scenario.Quads(numero, e, r.quads[:0])
	r.tampon.Effacer(r.scenario.Fond)
	r.tampon.Dessiner(r.quads, r.atlas)
	return r.tampon
}

// Ecrire écrit la dernière image rendue dans un PNG numéroté du répertoire, agrandie
// au facteur donné.
//
// Le nom porte le numéro sur quatre chiffres, pour que l'ordre des images soit celui
// des noms dans un répertoire comme dans une visionneuse d'images.
func (r *Rejeu) Ecrire(repertoire string, numero, facteur int) error {
	if err := os.MkdirAll(repertoire, 0o755); err != nil {
		return fmt.Errorf("création de %s : %w", repertoire, err)
	}

	source := r.tampon
	if facteur > 1 {
		large := uint16(int(r.tampon.Largeur) * facteur)
		haut := uint16(int(r.tampon.Hauteur) * facteur)
		if r.sortie == nil || r.sortie.Largeur != large || r.sortie.Hauteur != haut {
			r.sortie = raster.NouveauTampon(large, haut)
		}
		r.tampon.Agrandir(r.sortie, facteur, r.scenario.Fond)
		source = r.sortie
	}

	chemin := filepath.Join(repertoire, fmt.Sprintf("%s-%04d.png", r.scenario.Nom, numero))
	fichier, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("création de %s : %w", chemin, err)
	}
	if err := source.EcrirePNG(fichier); err != nil {
		fichier.Close()
		return fmt.Errorf("%s : %w", chemin, err)
	}
	if err := fichier.Close(); err != nil {
		return fmt.Errorf("fermeture de %s : %w", chemin, err)
	}
	return nil
}

// Rejouer rend toutes les images du scénario et les écrit.
//
// Les entrées se lisent dans l'ordre, une ligne par image, et un rejeu plus long que
// le fichier continue sur des entrées vides.
func (r *Rejeu) Rejouer(repertoire string, entrees []Entrees, facteur int) error {
	for numero := range r.scenario.Images {
		r.Image(numero, Entree(entrees, numero))
		if err := r.Ecrire(repertoire, numero, facteur); err != nil {
			return err
		}
	}
	return nil
}
