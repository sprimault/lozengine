// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// L'ouverture sous Linux : joindre le serveur, créer la fenêtre, la nommer, et la
// montrer.
//
// L'ordre compte, mais pas pour la même raison que sous Windows : tout part dans un
// seul envoi, et ce qui doit précéder l'apparition de la fenêtre est ce qui se verrait
// changer après — le titre et les bornes de taille.

package backend

// Open monte une fenêtre et rend de quoi la piloter.
//
// Le dialogue tient en deux allers-retours ici : un pour demander les atomes dont les
// propriétés ont besoin, un pour s'assurer que le serveur a tout accepté. Entre les
// deux, la fenêtre, son contexte graphique, le titre, les bornes et l'apparition
// partent ensemble. La connexion en a demandé deux autres avant, pour le handshake et
// pour lever le plafond de taille d'une requête.
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
// convient. Aucune icône laisse celle du système.
func Open(o Options) (Window, error) {
	c, err := connect()
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
		maskKeyPress|maskKeyRelease|maskKeymapState|maskStructureNotify|maskFocusChange)

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

	c.mapWindow(id)

	// La synchronisation est ce qui fait rendre ici les erreurs des cinq requêtes
	// ci-dessus, aucune n'ayant de réponse à elle.
	if err := c.sync(); err != nil {
		c.Close()
		return nil, err
	}

	return &window{
		c:            c,
		id:           id,
		gc:           gc,
		protocols:    protocols,
		deleteWindow: deleteWindow,
		width:        o.Width,
		height:       o.Height,
	}, nil
}
