// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Où le serveur écoute, d'après la variable qui le nomme.
//
// C'est le seul endroit du backend qui lit l'environnement, et il n'en tire que des
// chemins : ce qu'on trouve au bout est l'affaire de la connexion.

package backend

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// socketDir est le répertoire où un serveur X pose sa socket locale, par convention
// aussi ancienne que le protocole.
const socketDir = "/tmp/.X11-unix/X"

// displaySockets rend les chemins où joindre le serveur que display nomme, dans
// l'ordre où les essayer, avec le numéro de session et celui de l'écran.
//
// **Deux chemins et non un**, et c'est une leçon de serveur sans écran : la même
// session peut n'exister que sous l'une des deux formes. La socket de système de
// fichiers est la forme documentée ; la socket abstraite, propre à Linux, n'a pas
// d'entrée dans l'arborescence — un serveur qui reçoit sa socket déjà ouverte du
// compositeur, ce que fait Xwayland, peut n'offrir que celle-là.
//
// Seules les sessions locales sont admises. Un `DISPLAY` qui nomme une machine
// demanderait une connexion par le réseau, que les serveurs refusent par défaut depuis
// des années et que le moteur n'a aucune raison d'ouvrir : le dire est plus utile que
// d'échouer plus loin sur un refus de connexion.
func displaySockets(display string) (paths []string, number, screen int, err error) {
	if display == "" {
		return nil, 0, 0, errors.New("DISPLAY is not set: there is no display server to connect to")
	}

	// Le dernier deux-points et non le premier : une adresse IPv6 en porte
	// plusieurs, et c'est le seul moyen de couper au bon endroit avant de refuser
	// l'adresse elle-même.
	colon := strings.LastIndexByte(display, ':')
	if colon < 0 {
		return nil, 0, 0, fmt.Errorf("DISPLAY=%q carries no screen number, which the form host:number[.screen] requires", display)
	}

	if host := display[:colon]; host != "" && host != "unix" {
		return nil, 0, 0, fmt.Errorf("DISPLAY=%q names the host %q: only local displays are supported, written :number or unix:number", display, host)
	}

	// Le numéro d'écran est facultatif, et il choisit parmi les écrans d'une même
	// session — ce que le handshake énumère, non une seconde socket.
	rest := display[colon+1:]
	screen = 0
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		if screen, err = strconv.Atoi(rest[dot+1:]); err != nil || screen < 0 {
			return nil, 0, 0, fmt.Errorf("DISPLAY=%q carries an unreadable screen number", display)
		}
		rest = rest[:dot]
	}

	if number, err = strconv.Atoi(rest); err != nil || number < 0 {
		return nil, 0, 0, fmt.Errorf("DISPLAY=%q carries an unreadable display number", display)
	}

	path := socketDir + strconv.Itoa(number)
	return []string{path, "@" + path}, number, screen, nil
}
