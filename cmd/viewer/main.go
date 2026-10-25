// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La commande viewer rejoue un scénario scripté et en écrit les images.
//
//	viewer -scenario target -facteur 4 -sortie .tmp/images
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

	"github.com/sprimault/lozengine/internal/viewer"
)

// main lit les drapeaux de la ligne de commande et rejoue le scénario demandé.
//
// Les drapeaux restent en français : c'est ce qu'une personne tape, et leur aide
// l'est aussi. Tout le travail est dans run ; ce qui reste ici est la traduction des
// drapeaux et le code de sortie, pour qu'un échec se voie depuis un script au lieu de
// se lire dans la sortie.
func main() {
	name := flag.String("scenario", "target", "scénario à rejouer, parmi "+strings.Join(viewer.Names(), ", "))
	dir := flag.String("sortie", filepath.Join(".tmp", "images"), "répertoire des images écrites")
	factor := flag.Int("facteur", 1, "facteur d'agrandissement entier des images écrites")
	frames := flag.Int("images", 0, "nombre d'images, 0 pour celui du scénario")
	path := flag.String("entrees", "", "fichier d'entrées simulées, une ligne par image")
	flag.Parse()

	if err := run(*name, *dir, *path, *factor, *frames); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run monte le scénario demandé et écrit ses images.
func run(name, dir, path string, factor, frames int) error {
	scenario, ok := viewer.ByName(name)
	if !ok {
		return fmt.Errorf("scénario %q inconnu, connus : %s", name, strings.Join(viewer.Names(), ", "))
	}
	if frames > 0 {
		scenario.Frames = frames
	}

	lines, err := readInputLines(path)
	if err != nil {
		return err
	}

	replay, err := viewer.NewReplay(scenario)
	if err != nil {
		return err
	}
	if err := replay.Run(dir, lines, factor); err != nil {
		return err
	}

	fmt.Printf("%d images dans %s\n", scenario.Frames, dir)
	return nil
}

// readInputLines lit le fichier d'entrées, ou rend une liste vide s'il n'y en a pas.
func readInputLines(path string) ([]viewer.InputLine, error) {
	if path == "" {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ouverture des entrées : %w", err)
	}
	defer file.Close()

	return viewer.ReadInputLines(file)
}
