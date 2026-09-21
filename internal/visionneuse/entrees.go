// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package visionneuse

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Entrees est la ligne d'entrées simulées d'une image, telle que le fichier la porte.
//
// Une chaîne, et non une structure de touches : le moteur n'aura de système d'entrées
// qu'au jalon 2, et en inventer un ici le figerait sans raison. Chaque scénario
// interprète sa propre ligne, ce qui garde toute sémantique de jeu hors du moteur —
// et un fichier d'entrées se relit dans un diff.
type Entrees string

// LireEntrees lit un fichier d'entrées, une ligne par image.
//
// Les lignes vides et celles qui commencent par # sont ignorées : on annote un
// fichier d'entrées sans décaler les images qui suivent.
func LireEntrees(r io.Reader) ([]Entrees, error) {
	var lignes []Entrees

	lecteur := bufio.NewScanner(r)
	for lecteur.Scan() {
		ligne := strings.TrimSpace(lecteur.Text())
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		lignes = append(lignes, Entrees(ligne))
	}
	if err := lecteur.Err(); err != nil {
		return nil, fmt.Errorf("lecture des entrées : %w", err)
	}
	return lignes, nil
}

// Entree rend l'entrée de l'image demandée, ou la chaîne vide au-delà du fichier.
//
// Un rejeu plus long que le fichier d'entrées n'est pas une erreur : on enregistre
// les quelques images qui font le cas et on laisse la suite se dérouler.
func Entree(entrees []Entrees, numero int) Entrees {
	if numero < 0 || numero >= len(entrees) {
		return ""
	}
	return entrees[numero]
}
