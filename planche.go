// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"fmt"
	"os"

	"github.com/sprimault/lozengine/raster"
)

// OuvrirPlanche lit un PNG depuis un chemin et rend une planche en alpha
// prémultiplié.
//
// C'est le confort que [ChargerPlanche] n'offre pas, et la seule différence entre les
// deux : celle-là prend un lecteur, ce qui sert aussi bien un fichier qu'une ressource
// embarquée par go:embed ou une archive. Un jeu qui embarque ses images appelle donc
// ChargerPlanche, et celui qui les pose à côté de son binaire appelle celle-ci.
//
// Le fichier est refermé dans tous les cas, y compris quand le décodage échoue.
func OuvrirPlanche(chemin string) (Planche, error) {
	fichier, err := os.Open(chemin)
	if err != nil {
		return Planche{}, fmt.Errorf("ouverture de la planche : %w", err)
	}
	defer fichier.Close()

	planche, err := raster.ChargerPlanche(fichier)
	if err != nil {
		return Planche{}, fmt.Errorf("%s : %w", chemin, err)
	}
	return planche, nil
}
