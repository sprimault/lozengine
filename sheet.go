// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

package lozengine

import (
	"fmt"
	"os"

	"github.com/sprimault/lozengine/raster"
)

// OpenSheet lit un PNG depuis un chemin et rend une planche en alpha prémultiplié.
//
// C'est le confort que [LoadSheet] n'offre pas, et la seule différence entre les
// deux : celle-là prend un lecteur, ce qui sert aussi bien un fichier qu'une ressource
// embarquée par go:embed ou une archive. Un jeu qui embarque ses images appelle donc
// LoadSheet, et celui qui les pose à côté de son binaire appelle celle-ci.
//
// Le fichier est refermé dans tous les cas, y compris quand le décodage échoue.
func OpenSheet(path string) (Sheet, error) {
	file, err := os.Open(path)
	if err != nil {
		return Sheet{}, fmt.Errorf("ouverture de la planche : %w", err)
	}
	defer file.Close()

	sheet, err := raster.LoadSheet(file)
	if err != nil {
		return Sheet{}, fmt.Errorf("%s : %w", path, err)
	}
	return sheet, nil
}
