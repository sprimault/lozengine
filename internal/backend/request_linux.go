// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les requêtes que le backend envoie, écrites octet par octet.
//
// Chacune déclare sa longueur en mots de quatre octets, et c'est cette longueur que le
// serveur suit pour trouver la requête suivante : une erreur d'un seul mot ne produit
// pas une erreur sur la requête fautive, elle décale tout le flux. Les longueurs sont
// donc calculées depuis ce qui a été écrit, jamais posées en constante.

package backend

// internAtom demande le numéro des atomes nommés, en un seul aller-retour.
//
// Un atome est le nom d'une propriété tel que le serveur le numérote : les clients
// s'échangent des nombres plutôt que des chaînes, et c'est à l'ouverture qu'on paie la
// traduction. Les demander ensemble plutôt qu'un par un fait un aller-retour au lieu
// de quatre — le serveur traite les requêtes d'un même client dans l'ordre reçu, donc
// les réponses arrivent dans celui où on les attend.
func (c *conn) internAtom(names ...string) ([]uint32, error) {
	var zeros [3]byte

	first := c.seq + 1
	for _, name := range names {
		words := 2 + (len(name)+pad(len(name)))/4

		at := len(c.out)
		c.out = append(c.out, opInternAtom, 0, 0, 0, 0, 0, 0, 0)
		order.PutUint16(c.out[at+2:], uint16(words))
		order.PutUint16(c.out[at+4:], uint16(len(name)))
		c.out = append(c.out, name...)
		c.out = append(c.out, zeros[:pad(len(name))]...)

		c.next()
	}

	if err := c.flush(); err != nil {
		return nil, err
	}

	atoms := make([]uint32, len(names))
	for i := range names {
		m, err := c.reply(first + uint16(i))
		if err != nil {
			return nil, err
		}
		atoms[i] = order.Uint32(m[8:])
	}
	return atoms, nil
}

// createWindow crée une fenêtre de la taille demandée, fille de la racine.
//
// La taille est celle de la zone utile, et il n'y a rien à ajuster : X11 ne compte ni
// bordure ni barre de titre dans ce que le client demande, celles-ci appartenant à une
// fenêtre que le gestionnaire met autour de la nôtre. C'est l'inverse de Win32, qui
// demande la fenêtre entière.
func (c *conn) createWindow(id uint32, width, height uint16, events uint32) {
	at := len(c.out)
	c.out = append(c.out, opCreateWindow, copyFromParent, 0, 0)
	order.PutUint16(c.out[at+2:], 10)

	c.out = order.AppendUint32(c.out, id)
	c.out = order.AppendUint32(c.out, c.setup.root)
	c.out = order.AppendUint16(c.out, 0) // x, laissé au gestionnaire de fenêtres
	c.out = order.AppendUint16(c.out, 0) // y
	c.out = order.AppendUint16(c.out, width)
	c.out = order.AppendUint16(c.out, height)
	c.out = order.AppendUint16(c.out, 0) // bordure, que le moteur ne dessine pas
	c.out = order.AppendUint16(c.out, classInputOutput)
	c.out = order.AppendUint32(c.out, copyFromParent)
	c.out = order.AppendUint32(c.out, cwBackPixel|cwEventMask)

	// Les deux valeurs, dans l'ordre croissant de leurs bits. Le fond noir n'est vu
	// qu'avant la première image, et il vaut mieux que ce qu'on verrait sans lui :
	// X11 ne garde pas le contenu d'une fenêtre, donc le fond est ce que le serveur
	// peint quand il n'a rien d'autre.
	c.out = order.AppendUint32(c.out, c.setup.blackPixel)
	c.out = order.AppendUint32(c.out, events)

	c.next()
}

// changeProperty pose une propriété sur la fenêtre.
//
// Le format est la taille d'un élément en bits — huit pour du texte, trente-deux pour
// une liste d'atomes ou de nombres —, et la longueur se compte en éléments et non en
// octets. Les confondre sur une propriété de trente-deux bits en enverrait quatre fois
// trop, ce que le serveur refuse par une erreur de longueur.
func (c *conn) changeProperty(window, property, kind uint32, format byte, data []byte) {
	var zeros [3]byte

	words := 6 + (len(data)+pad(len(data)))/4

	at := len(c.out)
	c.out = append(c.out, opChangeProperty, propReplace, 0, 0)
	order.PutUint16(c.out[at+2:], uint16(words))

	c.out = order.AppendUint32(c.out, window)
	c.out = order.AppendUint32(c.out, property)
	c.out = order.AppendUint32(c.out, kind)
	c.out = append(c.out, format, 0, 0, 0)
	c.out = order.AppendUint32(c.out, uint32(len(data))/uint32(format/8))
	c.out = append(c.out, data...)
	c.out = append(c.out, zeros[:pad(len(data))]...)

	c.next()
}

// mapWindow demande que la fenêtre apparaisse.
//
// Rien n'est affiché avant, et la fenêtre existe pourtant : la création et l'apparition
// sont deux requêtes, ce qui laisse poser le titre et les bornes avant que quoi que ce
// soit se voie.
func (c *conn) mapWindow(id uint32) {
	at := len(c.out)
	c.out = append(c.out, opMapWindow, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 2)
	c.out = order.AppendUint32(c.out, id)

	c.next()
}

// destroyWindow détruit la fenêtre et ce qui en dépend.
//
// La fermeture de la socket y suffirait, le serveur libérant tout ce qu'un client a
// créé quand sa connexion tombe. La requête est envoyée quand même, parce qu'elle
// distingue une fermeture voulue d'un client qui a disparu — ce qu'un gestionnaire de
// fenêtres et un serveur consignent différemment.
func (c *conn) destroyWindow(id uint32) {
	at := len(c.out)
	c.out = append(c.out, opDestroyWindow, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 2)
	c.out = order.AppendUint32(c.out, id)

	c.next()
}

// sync attend que le serveur ait traité tout ce qui a été envoyé, et rend la première
// erreur qu'il aurait rapportée.
//
// **C'est ce qui rend les erreurs d'ouverture à l'ouverture.** La plupart des requêtes
// n'ont pas de réponse, donc une requête fausse ne se signale que plus tard, au milieu
// d'une image, sur un objet sans rapport. Une requête à réponse posée derrière elles
// force le rendez-vous : sa réponse ne peut pas précéder leurs erreurs.
//
// Les événements arrivés entre-temps — l'apparition de la fenêtre, sa première taille
// réelle — sont mis de côté par [conn.reply] et non perdus.
func (c *conn) sync() error {
	at := len(c.out)
	c.out = append(c.out, opGetInputFocus, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 1)

	seq := c.next()
	if err := c.flush(); err != nil {
		return err
	}
	_, err := c.reply(seq)
	return err
}

// setTitle nomme la fenêtre, dans les deux propriétés qui la nomment.
//
// Deux et non une, et ce n'est pas un doublon : `WM_NAME` est la propriété d'origine et
// son type ne porte que du latin-1, `_NET_WM_NAME` est celle que les gestionnaires
// récents lisent d'abord et elle porte de l'UTF-8. Ne poser que la seconde laisse un
// gestionnaire ancien sans titre ; ne poser que la première ampute les titres accentués
// chez tous les autres.
func (c *conn) setTitle(window uint32, title string, netName, utf8 uint32) {
	c.changeProperty(window, atomWMName, atomString, 8, []byte(title))
	c.changeProperty(window, netName, utf8, 8, []byte(title))
}

// setProtocols annonce les messages du gestionnaire de fenêtres que le client sait
// recevoir.
//
// **Un client qui n'y liste pas la demande de fermeture est tué par le serveur quand
// l'utilisateur clique la croix**, connexion coupée et processus mort. Ce n'est pas
// une dégradation de confort : c'est la différence entre une fermeture que le jeu voit
// venir et un arrêt brutal au milieu d'une image.
func (c *conn) setProtocols(window uint32, protocols, deleteWindow uint32) {
	var data [4]byte
	order.PutUint32(data[:], deleteWindow)
	c.changeProperty(window, protocols, atomAtom, 32, data[:])
}

// setFloor dit au gestionnaire de fenêtres sous quelle taille la fenêtre ne doit pas
// descendre.
//
// C'est le pendant de ce que Win32 obtient en répondant à une demande de bornes. En
// dessous de l'image, l'agrandissement entier ne tiendrait plus et les bords
// sortiraient de la vue sans que rien ne le dise.
//
// **Une indication et non une contrainte** : rien n'oblige un gestionnaire de fenêtres
// à la respecter, et sans gestionnaire — le régime d'un serveur sans écran — elle
// n'est lue par personne. La boucle ne s'y fie donc pas, et c'est elle qui tient la
// taille réellement reçue.
func (c *conn) setFloor(window uint32, width, height uint16) {
	var hints [4 * sizeHintsWords]byte
	order.PutUint32(hints[0:], sizeHintsMinSize)
	order.PutUint32(hints[4*sizeHintsMinIndex:], uint32(width))
	order.PutUint32(hints[4*(sizeHintsMinIndex+1):], uint32(height))

	c.changeProperty(window, atomWMNormalHints, atomWMSizeHints, 32, hints[:])
}
