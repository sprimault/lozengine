// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Les requêtes que le backend envoie, écrites octet par octet.
//
// Chacune déclare sa longueur en mots de quatre octets, et c'est cette longueur que le
// serveur suit pour trouver la requête suivante : une erreur d'un seul mot ne produit
// pas une erreur sur la requête fautive, elle décale tout le flux. Les longueurs sont
// donc calculées depuis ce qui a été écrit, jamais posées en constante.

package backend

import "github.com/sprimault/lozengine/render"

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

// createGC crée le contexte graphique par lequel l'image entre dans la fenêtre.
//
// Aucune valeur n'est posée, et le masque vide est voulu : les défauts du protocole
// conviennent exactement — la fonction de combinaison recopie, le masque de plans
// couvre tout. Les régler reviendrait à redire ce que le serveur fait déjà.
func (c *conn) createGC(id, drawable uint32) {
	at := len(c.out)
	c.out = append(c.out, opCreateGC, 0, 0, 0)
	order.PutUint16(c.out[at+2:], 4)

	c.out = order.AppendUint32(c.out, id)
	c.out = order.AppendUint32(c.out, drawable)
	c.out = order.AppendUint32(c.out, 0)

	c.next()
}

// putImageHeader est la taille de l'en-tête d'un transfert d'image, bourrage compris
// et avant l'éventuel mot de longueur étendue.
const putImageHeader = 24

// putImage envoie une bande de pixels dans la fenêtre, convertis au passage.
//
// Les pixels sont écrits **directement dans le tampon d'émission**, derrière l'en-tête
// qu'on vient d'y poser : un tampon intermédiaire par image coûterait une copie
// complète de plus, et le brief ne concède qu'une seule passe.
//
// **Aucun bourrage à écrire**, et c'est le format qui le donne : quatre octets par
// pixel font une longueur toujours multiple de quatre. C'est la seule requête du
// backend dont on peut l'affirmer.
//
// Le découpage se fait en hauteur, jamais en largeur — une ligne de l'image doit
// rester contiguë en mémoire pour partir sans second découpage — d'où l'abscisse nulle
// et l'ordonnée reçue.
func (c *conn) putImage(drawable, gc uint32, depth byte, y int16, width, height uint16, pixels []render.Color) {
	data := c.pixels.size * len(pixels)
	words := (putImageHeader + data) / 4

	at := len(c.out)
	c.out = append(c.out, opPutImage, zPixmap, 0, 0)
	if words > requestWordsMax {
		// Forme étendue de `BIG-REQUESTS` : les seize bits valent zéro et la
		// longueur suit sur trente-deux, le mot qui la porte étant compté dedans.
		order.PutUint16(c.out[at+2:], 0)
		c.out = order.AppendUint32(c.out, uint32(words+1))
	} else {
		order.PutUint16(c.out[at+2:], uint16(words))
	}

	c.out = order.AppendUint32(c.out, drawable)
	c.out = order.AppendUint32(c.out, gc)
	c.out = order.AppendUint16(c.out, width)
	c.out = order.AppendUint16(c.out, height)
	c.out = order.AppendUint16(c.out, 0)
	c.out = order.AppendUint16(c.out, uint16(y))

	// Le bourrage à gauche doit valoir zéro en zPixmap, et la profondeur doit égaler
	// celle de la fenêtre : l'une ou l'autre fausse rend une erreur de
	// correspondance, qui arrive plus tard et ne nomme ni l'un ni l'autre.
	c.out = append(c.out, 0, depth, 0, 0)

	c.out = c.pixels.appendPixels(c.out, pixels)

	c.next()
}

// linesPerRequest dit combien de lignes d'une image de cette largeur tiennent dans une
// requête.
//
// La marge de quatre octets couvre le mot de longueur étendue, pour que le compte
// reste bon dans les deux formes de requête plutôt que de dépendre de laquelle on
// emploiera.
func (c *conn) linesPerRequest(width uint16) int {
	perLine := c.pixels.size * int(width)
	if perLine == 0 {
		return 0
	}
	return (int(c.setup.maxRequest) - putImageHeader - 4) / perLine
}

// queryExtension demande si le serveur porte une extension, et sous quel opcode.
//
// Un serveur qui ne l'a pas répond sans erreur, avec un drapeau à faux : l'absence
// d'une extension est un régime ordinaire et non un échec, et c'est ce qui permet au
// repli de ne rien avoir à rattraper.
func (c *conn) queryExtension(name string) (present bool, major byte, err error) {
	var zeros [3]byte

	words := 2 + (len(name)+pad(len(name)))/4

	at := len(c.out)
	c.out = append(c.out, opQueryExtension, 0, 0, 0, 0, 0, 0, 0)
	order.PutUint16(c.out[at+2:], uint16(words))
	order.PutUint16(c.out[at+4:], uint16(len(name)))
	c.out = append(c.out, name...)
	c.out = append(c.out, zeros[:pad(len(name))]...)

	seq := c.next()
	if err := c.flush(); err != nil {
		return false, 0, err
	}

	m, err := c.reply(seq)
	if err != nil {
		return false, 0, err
	}
	return m[8] != 0, m[9], nil
}

// enableBigRequests lève le plafond de taille d'une requête et rend le nouveau.
//
// Sans elle, une requête ne peut pas dépasser quatre fois ce que seize bits portent,
// et une image de la taille d'un écran part en dizaines de morceaux. C'est le meilleur
// rapport entre l'effort et le résultat du jalon : une requête à l'ouverture, un champ
// de longueur conditionnel, et le découpage disparaît.
func (c *conn) enableBigRequests(major byte) (uint32, error) {
	at := len(c.out)
	c.out = append(c.out, major, bigRequestsEnable, 0, 0)
	order.PutUint16(c.out[at+2:], 1)

	seq := c.next()
	if err := c.flush(); err != nil {
		return 0, err
	}

	m, err := c.reply(seq)
	if err != nil {
		return 0, err
	}
	return order.Uint32(m[8:]), nil
}

// detectableAutoRepeat demande au serveur de ne plus fabriquer de relâchements entre
// les répétitions, et dit s'il a accepté.
//
// **Le résultat se lit, il ne se suppose pas** : la spécification prévient que ce
// drapeau n'est pas garanti, et un serveur qui le refuse continue d'envoyer appui,
// relâchement, appui, relâchement pour un doigt posé. C'est pourquoi la fenêtre sait
// reconnaître ces paires de toute façon — ce dialogue épargne le trafic, il ne porte
// pas la correction.
//
// Deux requêtes et non une : l'extension refuse tout avant que les versions se soient
// accordées.
func (c *conn) detectableAutoRepeat(major byte) (bool, error) {
	at := len(c.out)
	c.out = append(c.out, major, xkbUseExtension, 0, 0)
	order.PutUint16(c.out[at+2:], 2)
	c.out = order.AppendUint16(c.out, xkbMajorVersion)
	c.out = order.AppendUint16(c.out, xkbMinorVersion)
	version := c.next()

	at = len(c.out)
	c.out = append(c.out, major, xkbPerClientFlags, 0, 0)
	order.PutUint16(c.out[at+2:], 7)
	c.out = order.AppendUint16(c.out, xkbCoreKeyboard)
	c.out = order.AppendUint16(c.out, 0)
	c.out = order.AppendUint32(c.out, xkbDetectableAutoRepeat) // ce qu'on change
	c.out = order.AppendUint32(c.out, xkbDetectableAutoRepeat) // la valeur voulue
	c.out = order.AppendUint32(c.out, 0)                       // aucun contrôle booléen
	c.out = order.AppendUint32(c.out, 0)
	c.out = order.AppendUint32(c.out, 0)
	flags := c.next()

	if err := c.flush(); err != nil {
		return false, err
	}

	// La réponse au dialogue de version porte son accord au deuxième octet. Un
	// serveur qui n'accorde pas sa version ne traitera pas la requête suivante, mais
	// il y répondra : on lit les deux réponses dans l'ordre.
	accord, err := c.reply(version)
	if err != nil {
		return false, err
	}

	supported, err := c.reply(flags)
	if err != nil {
		return false, err
	}
	if accord[1] == 0 {
		return false, nil
	}
	return order.Uint32(supported[8:])&xkbDetectableAutoRepeat != 0, nil
}

// changePropertyHeader est la taille de l'en-tête d'une pose de propriété, avant ses
// données.
//
// Elle vaut autant que celle d'un transfert d'image, et c'est une coïncidence : les
// deux requêtes n'ont ni les mêmes champs ni le même nombre. Les nommer séparément
// évite qu'une relecture croie à un en-tête commun.
const changePropertyHeader = 24

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

// setIcons pose toutes les tailles d'icône que le jeu fournit, dans la propriété que
// les gestionnaires de fenêtres lisent.
//
// **Toutes les tailles et non une**, et c'est la différence avec Windows : là-bas il
// faut choisir par emplacement — une grande pour le changement de fenêtre, une petite
// pour la barre de titre —, ici la propriété porte une suite d'images et le
// gestionnaire prend celle qui lui convient. C'est pourquoi le choix de taille
// partagé entre backends ne sert pas à ce lot.
//
// Chaque image s'annonce par sa largeur et sa hauteur, puis ses pixels, un mot par
// pixel. Les mots sont de l'ARGB à alpha **non** prémultiplié, ce que la conversion
// partagée rend déjà tel quel — le moteur prémultiplie pour que le mélange d'un pixel
// soit une multiplication, et une icône se décrit à l'envers.
//
// **Une icône dont les pixels ne remplissent pas ses dimensions est écartée.** Ce n'est
// pas de la défensive en trop : la propriété se lit en suivant les tailles annoncées,
// donc une image trop courte décalerait toutes les suivantes et le gestionnaire
// afficherait du bruit. Un jeu qui remplit une planche à la main peut produire ce cas.
//
// **La propriété entière doit tenir dans une requête**, et rien ne la découpe ici :
// deux icônes de 48 pixels font moins de vingt kilooctets, mais une de 256 en fait
// deux cent cinquante. Au-delà du plafond, le serveur rend une erreur de longueur que
// la synchronisation de l'ouverture rapporte — donc à l'ouverture, et non plus tard.
func (c *conn) setIcons(window, atom uint32, icons []*render.Buffer) {
	var data []byte

	for _, icon := range icons {
		if icon == nil || icon.Width == 0 || icon.Height == 0 {
			continue
		}
		if len(icon.Pixels) != int(icon.Width)*int(icon.Height) {
			continue
		}

		data = order.AppendUint32(data, uint32(icon.Width))
		data = order.AppendUint32(data, uint32(icon.Height))
		for _, word := range straightARGB(icon) {
			data = order.AppendUint32(data, word)
		}
	}

	// Rien à poser laisse l'icône du système, ce qui est le bon défaut : une
	// propriété vide dirait au gestionnaire que le jeu n'a pas d'icône plutôt que
	// qu'il n'en fournit pas.
	if len(data) == 0 {
		return
	}
	c.changeProperty(window, atom, atomCardinal, 32, data)
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
