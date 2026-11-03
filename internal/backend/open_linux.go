// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'ouverture sous Linux : quelle famille d'affichage essayer, puis le montage d'une
// fenêtre X11.
//
// Les deux vivent ensemble parce qu'un lecteur qui cherche `Open` ouvre ce fichier, comme
// pour les trois autres plateformes. Le montage Wayland est dans `wl_window_linux.go`, où
// il tient avec la négociation de taille qui le distingue.

package backend

import (
	"errors"
	"fmt"
)

// Open monte une fenêtre sur la première famille d'affichage joignable.
//
// **Wayland passe avant X11, et l'ordre inverse serait le piège** : un bureau Wayland pose
// aussi `DISPLAY` pour Xwayland, donc essayer X11 d'abord ferait passer tout le monde par
// la couche de traduction que le moteur existe pour retirer. Un poste X11 pur ne nomme
// aucun compositeur, donc rien n'y est tenté en vain.
//
// **Aucune variable d'environnement ne choisit la famille**, et c'est une décision plutôt
// qu'un oubli : elle serait la première du dépôt à changer un comportement au lieu
// d'interdire un saut de test, et un nom de variable est aussi définitif qu'une signature.
// Choisir sa famille depuis le code attend le gel du jalon 9.
//
// **Ce qui épingle une session sur X11 est l'absence de `WAYLAND_DISPLAY`**, et c'est
// pourquoi la famille n'est essayée que si l'environnement la nomme. Le défaut de la
// découverte de socket est `wayland-0`, donc l'essayer d'office ferait basculer sur
// Wayland une session qu'un compositeur voisin suffit à servir — y compris le régime
// Xwayland, qui est précisément celui qu'on veut éprouver sous X11.
//
// Les deux causes sont rendues ensemble quand les deux familles échouent : l'une seule ne
// dirait pas si le compositeur manquait ou si c'est le serveur X.
func Open(o Options) (Window, error) {
	var wlErr error
	if wlNamed() {
		w, err := wlOpen(o)
		if err == nil {
			return w, nil
		}
		wlErr = err
	}

	w, err := x11Open(o)
	if err == nil {
		return w, nil
	}
	if wlErr == nil {
		return nil, err
	}
	return nil, fmt.Errorf("no display server could be reached: %w", errors.Join(wlErr, err))
}

// x11Open monte une fenêtre sur un serveur X et rend de quoi la piloter.
//
// Le dialogue tient en deux allers-retours ici : un pour demander les atomes dont les
// propriétés ont besoin, un pour s'assurer que le serveur a tout accepté. Entre les
// deux, la fenêtre, son contexte graphique, le titre, les bornes et l'apparition
// partent ensemble. La connexion en a demandé deux autres avant, pour le handshake et
// pour lever le plafond de taille d'une requête.
//
// L'ordre des requêtes compte, mais pas pour la même raison que sous Windows : tout part
// dans un seul envoi, et ce qui doit précéder l'apparition de la fenêtre est ce qui se
// verrait changer après — le titre et les bornes de taille.
//
// **La fenêtre n'est pas encore à l'écran quand Open rend.** L'apparition est demandée
// mais le serveur, et le gestionnaire de fenêtres derrière lui, l'accordent quand ils
// veulent : attendre ici bloquerait indéfiniment là où aucun gestionnaire ne répond,
// ce qui est le régime d'un serveur sans écran. La boucle n'a rien à attendre non
// plus — ses premières images partent dans une fenêtre qui n'est pas visible, et le
// protocole les laisse tomber sans erreur.
//
// Les icônes de [Options] partent toutes, et non la mieux dimensionnée : la propriété
// d'X11 porte une suite d'images et le gestionnaire de fenêtres prend celle qui lui
// convient. Aucune icône laisse celle du système — et c'est le seul réglage que les deux
// familles Linux ne servent pas de la même façon, xdg-shell ne transportant pas d'image.
func x11Open(o Options) (*x11Window, error) {
	c, err := x11Connect()
	if err != nil {
		return nil, err
	}

	// Les quatre noms dont les propriétés de la fenêtre ont besoin. Ils se demandent
	// avant tout le reste parce que leurs numéros entrent dans les requêtes qui
	// suivent.
	atoms, err := c.internAtom("WM_PROTOCOLS", "WM_DELETE_WINDOW", "_NET_WM_NAME", "UTF8_STRING", "_NET_WM_ICON")
	if err != nil {
		c.Close()
		return nil, err
	}
	protocols, deleteWindow, netName, utf8, icon := atoms[0], atoms[1], atoms[2], atoms[3], atoms[4]

	id := c.id()
	c.createWindow(id, o.Width, o.Height,
		maskKeyPress|maskKeyRelease|maskKeymapState|maskStructureNotify|maskFocusChange|
			maskButtonPress|maskButtonRelease|maskPointerMotion|
			maskEnterWindow|maskLeaveWindow)

	// Le contexte graphique vient juste après la fenêtre, dont il a besoin comme
	// modèle, et une seule fois : il ne porte que des défauts, donc un
	// redimensionnement ne le périme pas.
	gc := c.id()
	c.createGC(gc, id)

	c.setTitle(id, o.Title, netName, utf8)
	c.setProtocols(id, protocols, deleteWindow)
	c.setFloor(id, o.Width, o.Height)

	// Les icônes se posent avant l'apparition, comme le titre : ce qui se verrait
	// changer après n'a pas à être posé après.
	c.setIcons(id, icon, o.Icons)

	if o.HideCursor {
		c.hideCursor(id)
	}

	c.mapWindow(id)

	// La synchronisation est ce qui fait rendre ici les erreurs des cinq requêtes
	// ci-dessus, aucune n'ayant de réponse à elle.
	if err := c.sync(); err != nil {
		c.Close()
		return nil, err
	}

	return &x11Window{
		c:            c,
		id:           id,
		gc:           gc,
		protocols:    protocols,
		deleteWindow: deleteWindow,
		width:        o.Width,
		height:       o.Height,
	}, nil
}
