// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La commande visionneuse rejoue un scénario scripté et en écrit les images.
//
//	visionneuse -scenario mire -facteur 4 -sortie .tmp/images
//
// Elle n'ouvre aucune fenêtre : le moteur n'a pas encore de backend, et la suite de
// non-régression se juge sur des fichiers. Les images sont numérotées, donc une
// visionneuse d'images les enchaîne dans l'ordre.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sprimault/lozengine/internal/visionneuse"
)

func main() {
	nom := flag.String("scenario", "mire", "scénario à rejouer, parmi "+strings.Join(visionneuse.Noms(), ", "))
	repertoire := flag.String("sortie", filepath.Join(".tmp", "images"), "répertoire des images écrites")
	facteur := flag.Int("facteur", 1, "facteur d'agrandissement entier des images écrites")
	images := flag.Int("images", 0, "nombre d'images, 0 pour celui du scénario")
	fichier := flag.String("entrees", "", "fichier d'entrées simulées, une ligne par image")
	flag.Parse()

	if err := rejouer(*nom, *repertoire, *fichier, *facteur, *images); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// rejouer monte le scénario demandé et écrit ses images.
func rejouer(nom, repertoire, fichier string, facteur, images int) error {
	scenario, ok := visionneuse.Par(nom)
	if !ok {
		return fmt.Errorf("scénario %q inconnu, connus : %s", nom, strings.Join(visionneuse.Noms(), ", "))
	}
	if images > 0 {
		scenario.Images = images
	}

	entrees, err := lireEntrees(fichier)
	if err != nil {
		return err
	}

	rejeu, err := visionneuse.NouveauRejeu(scenario)
	if err != nil {
		return err
	}
	if err := rejeu.Rejouer(repertoire, entrees, facteur); err != nil {
		return err
	}

	fmt.Printf("%d images dans %s\n", scenario.Images, repertoire)
	return nil
}

// lireEntrees lit le fichier d'entrées, ou rend une liste vide s'il n'y en a pas.
func lireEntrees(chemin string) ([]visionneuse.Entrees, error) {
	if chemin == "" {
		return nil, nil
	}

	fichier, err := os.Open(chemin)
	if err != nil {
		return nil, fmt.Errorf("ouverture des entrées : %w", err)
	}
	defer fichier.Close()

	return visionneuse.LireEntrees(fichier)
}
