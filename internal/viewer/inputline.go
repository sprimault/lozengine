// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les entrées simulées, une ligne de texte par image. Le format reste une chaîne que
// chaque scénario interprète, plutôt que les places de clavier du paquet input.

package viewer

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// InputLine est la ligne d'entrées simulées d'une image, telle que le fichier la porte.
//
// Une chaîne, et non les places de clavier que le paquet input nomme : un scénario
// décrit ce qu'il veut voir arriver, pas la touche par laquelle un joueur le
// demanderait. Chaque scénario interprète donc sa propre ligne, ce qui garde toute
// sémantique de jeu hors du moteur — et un fichier d'entrées se relit dans un diff.
type InputLine string

// ReadInputLines lit un fichier d'entrées, une ligne par image.
//
// Les lignes vides et celles qui commencent par # sont ignorées : on annote un
// fichier d'entrées sans décaler les images qui suivent.
func ReadInputLines(r io.Reader) ([]InputLine, error) {
	var lines []InputLine

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, InputLine(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading inputs: %w", err)
	}
	return lines, nil
}

// InputLineAt rend l'entrée de l'image demandée, ou la chaîne vide au-delà du fichier.
//
// Un rejeu plus long que le fichier d'entrées n'est pas une erreur : on enregistre
// les quelques images qui font le cas et on laisse la suite se dérouler.
func InputLineAt(lines []InputLine, frame int) InputLine {
	if frame < 0 || frame >= len(lines) {
		return ""
	}
	return lines[frame]
}
