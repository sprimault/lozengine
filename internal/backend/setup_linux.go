// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// Ce que le serveur dit de lui-même à l'ouverture, et comment on le lit.
//
// Le handshake est le seul endroit du protocole où tout arrive d'un bloc, de longueur
// variable, et où la place d'un champ dépend de la longueur des précédents. Le
// découpage est donc écrit à la main, champ par champ, plutôt que posé sur une
// structure : un tampon reçu sur une socket n'a pas de disposition mémoire, il a des
// offsets.

package backend

import (
	"errors"
	"fmt"
	"io"
)

// setup est ce que le serveur a répondu au handshake, réduit à ce que le backend
// emploie.
//
// Il n'y a pas de seconde occasion de le lire : le serveur l'envoie une fois, et tout
// ce qu'une requête ultérieure présuppose — un identifiant libre, la racine dans
// laquelle créer, la taille maximale d'une requête — vient de là.
type setup struct {
	// root est la fenêtre racine de l'écran demandé, et le parent de la nôtre.
	root uint32

	// visual est le visuel de cette racine, dont on hérite par [copyFromParent].
	// Ses trois masques disent où le serveur attend le rouge, le vert et le bleu.
	visual                       uint32
	redMask, greenMask, blueMask uint32

	// depth est la profondeur de la racine, en bits significatifs par pixel, et
	// bitsPerPixel la place que le serveur leur donne sur la connexion.
	//
	// **Les deux diffèrent, et c'est le piège le plus cher du jalon** : une
	// profondeur de 24 se transmet presque toujours sur 32 bits. Les confondre ne
	// lève aucune erreur — le serveur lit ce qu'il attend, il en manque un quart, et
	// l'image sort en escalier diagonal, qui est exactement le symptôme d'un défaut
	// de projection ou d'ancre. La seule source autorisée de bitsPerPixel est la
	// liste des formats, à l'entrée de cette profondeur.
	depth        uint8
	bitsPerPixel uint8

	// imageOrder est l'ordre dans lequel le serveur veut les octets d'une image :
	// zéro pour le petit-boutien, un pour le gros-boutien.
	//
	// Il n'a rien à voir avec l'ordre de la connexion, que le client choisit : les
	// images « sont toujours transmises et reçues dans les formats, ordre des octets
	// compris, que le serveur spécifie ». C'est de ce champ et des masques du visuel
	// que la conversion se dérive, jamais de l'architecture de la machine.
	imageOrder uint8

	// blackPixel est la valeur de pixel du noir sur cet écran, et le fond de la
	// fenêtre avant la première image. En profondeur 24 elle vaut zéro, mais la
	// demander épargne de le supposer.
	blackPixel uint32

	// idBase et idMask composent les identifiants que le client a le droit de créer.
	// Le serveur lui concède une plage, et toute fenêtre ou ressource s'y nomme.
	idBase uint32
	idMask uint32

	// maxRequest est la plus grande requête que le serveur accepte, en octets.
	//
	// Elle vaut au moins 256 Kio sur un serveur courant, ce qui découpe déjà une
	// image 1920×1080 en plusieurs transferts. L'extension `BIG-REQUESTS` la lève,
	// et c'est le lot qui transfère l'image qui s'en occupe.
	maxRequest uint32

	// minKeycode et maxKeycode bornent les codes que le clavier produit. Le lot qui
	// lit le clavier s'en sert pour ignorer ce qui sort de la plage.
	minKeycode uint8
	maxKeycode uint8

	// vendor nomme l'implémentation du serveur. Elle n'oriente aucune décision — ce
	// serait écrire pour une implémentation et non pour le protocole — et ne sert
	// qu'à ce qu'un message d'erreur dise à qui on parlait.
	vendor string
}

// readSetup lit la réponse du serveur au handshake et rend ses données.
//
// Elle sépare le transport du décodage : ce qu'elle rend est le bloc de longueur
// variable qui suit l'en-tête, et [parseSetup] en tire les champs. Le refus et
// l'exigence d'authentification sont traités ici, parce que tous deux portent une
// raison en texte que le serveur mesure dans ce même en-tête.
func readSetup(r io.Reader) ([]byte, error) {
	var head [8]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return nil, fmt.Errorf("reading the connection setup: %w", err)
	}

	// La longueur est en mots de quatre octets, et elle couvre tout ce qui suit
	// l'en-tête — y compris la raison d'un refus.
	rest := make([]byte, 4*int(order.Uint16(head[6:])))
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("reading the connection setup: %w", err)
	}

	switch head[0] {
	case setupSuccess:
		return rest, nil

	case setupFailed:
		// La longueur de la raison est au deuxième octet, et non dans le bloc :
		// c'est le seul champ de l'en-tête qui ne sert qu'à ce cas.
		return nil, fmt.Errorf("the display server refused the connection: %s", reason(rest, int(head[1])))

	case setupAuthenticate:
		// Le serveur a lu notre autorisation et ne s'en contente pas. Le dire
		// séparément du refus a une raison pratique : c'est le régime d'un cookie
		// absent ou périmé, qui se corrige en trouvant le bon fichier plutôt qu'en
		// cherchant ce qui clocherait dans le client.
		return nil, fmt.Errorf("the display server requires another authorisation: %s", reason(rest, len(rest)))

	default:
		return nil, fmt.Errorf("the display server answered %d to the setup, which the protocol does not define", head[0])
	}
}

// reason rend la raison d'un refus, telle que le serveur l'a écrite.
//
// La longueur annoncée est crue mais bornée : elle vient d'un serveur qui, dans ce
// régime précis, est déjà en train de mal se comporter.
func reason(b []byte, length int) string {
	if length > len(b) {
		length = len(b)
	}
	if length == 0 {
		return "no reason given"
	}
	return string(b[:length])
}

// parseSetup tire du handshake ce que le backend emploie, pour l'écran demandé.
//
// L'écran est celui que `DISPLAY` nomme après le point — `:0.1` est le second de la
// même session. Un numéro hors de ce que le serveur annonce est une erreur et non un
// repli sur le premier : l'appelant a demandé un écran qui n'existe pas, et présenter
// ailleurs serait pire que de le lui dire.
func parseSetup(b []byte, screen int) (setup, error) {
	var s setup

	// Tout ce qui précède les listes est à offset fixe. Le dernier champ lu est à
	// l'offset 27, et les quatre octets suivants sont du bourrage.
	const fixed = 32
	if len(b) < fixed {
		return s, errors.New("the connection setup is too short to be read")
	}

	s.idBase = order.Uint32(b[4:])
	s.idMask = order.Uint32(b[8:])
	vendorLen := int(order.Uint16(b[16:]))
	s.maxRequest = 4 * uint32(order.Uint16(b[18:]))
	screens := int(b[20])
	formats := int(b[21])
	s.imageOrder = b[22]
	s.minKeycode = b[26]
	s.maxKeycode = b[27]

	if screen >= screens {
		return s, fmt.Errorf("display screen %d: the server has %d", screen, screens)
	}

	// Le nom du serveur est bourré à un multiple de quatre, et le bourrage n'est pas
	// compté dans la longueur annoncée.
	at := fixed + vendorLen + pad(vendorLen)
	if at > len(b) {
		return s, errors.New("the connection setup ends inside the server name")
	}
	s.vendor = string(b[fixed : fixed+vendorLen])

	// Les formats sont la table des profondeurs que le serveur sait transmettre, et
	// la seule source de bits-per-pixel. Ils se lisent tous, la profondeur de notre
	// écran n'étant connue qu'après.
	const formatSize = 8
	if at+formats*formatSize > len(b) {
		return s, errors.New("the connection setup ends inside the pixel formats")
	}
	depths := b[at : at+formats*formatSize]
	at += formats * formatSize

	// Les écrans se parcourent tous jusqu'à celui qu'on veut : chacun porte une
	// liste de profondeurs de longueur variable, donc le suivant n'est atteignable
	// qu'en traversant le précédent.
	for i := 0; i <= screen; i++ {
		info, next, err := parseScreen(b, at)
		if err != nil {
			return s, err
		}
		if i == screen {
			s.root, s.visual, s.depth, s.blackPixel = info.root, info.visual, info.depth, info.black
			s.redMask, s.greenMask, s.blueMask = info.redMask, info.greenMask, info.blueMask
		}
		at = next
	}

	if err := s.pixelFormat(depths); err != nil {
		return s, err
	}
	return s, nil
}

// screenInfo est ce qu'un écran du handshake dit de lui-même, réduit à ce que le
// backend emploie.
//
// Les champs qu'il laisse de côté — taille en pixels et en millimètres, tables de
// couleurs installables, conservation du contenu masqué — décideraient d'un plein
// écran ou d'une densité, et aucun des deux n'est du ressort d'un backend.
type screenInfo struct {
	root   uint32
	visual uint32
	black  uint32
	depth  uint8

	redMask, greenMask, blueMask uint32
}

// parseScreen lit un écran du handshake et dit où le suivant commence.
//
// Les masques du visuel racine se relèvent dans la même passe : son identifiant est
// annoncé à l'offset 32, donc avant la liste de profondeurs qui le décrit, et c'est ce
// qui permet de le reconnaître en traversant.
func parseScreen(b []byte, at int) (screenInfo, int, error) {
	var info screenInfo

	const screenFixed = 40
	if at+screenFixed > len(b) {
		return info, 0, errors.New("the connection setup ends inside a screen")
	}

	info.root = order.Uint32(b[at:])
	info.black = order.Uint32(b[at+12:])
	info.visual = order.Uint32(b[at+32:])
	info.depth = b[at+38]
	allowed := int(b[at+39])

	// Chaque profondeur autorisée porte le nombre de ses visuels, et chaque visuel
	// fait vingt-quatre octets.
	const (
		depthFixed = 8
		visualSize = 24
	)
	found := false
	at += screenFixed
	for i := 0; i < allowed; i++ {
		if at+depthFixed > len(b) {
			return info, 0, errors.New("the connection setup ends inside a depth")
		}
		count := int(order.Uint16(b[at+2:]))
		at += depthFixed

		if at+count*visualSize > len(b) {
			return info, 0, errors.New("the connection setup ends inside the visuals of a depth")
		}
		for j := 0; j < count; j++ {
			if !found && order.Uint32(b[at:]) == info.visual {
				info.redMask = order.Uint32(b[at+8:])
				info.greenMask = order.Uint32(b[at+12:])
				info.blueMask = order.Uint32(b[at+16:])
				found = true
			}
			at += visualSize
		}
	}
	if !found {
		return info, 0, fmt.Errorf("the root visual %#x is not among the ones its screen describes", info.visual)
	}
	return info, at, nil
}

// pixelFormat retrouve dans la table des formats celui de la profondeur de l'écran, et
// en garde la place qu'un pixel occupe sur la connexion.
//
// L'absence est une erreur et non un défaut raisonnable : un serveur est tenu
// d'annoncer un format pour la profondeur de sa propre racine, et supposer trente-deux
// bits produirait précisément le décalage que ce champ existe pour éviter.
func (s *setup) pixelFormat(formats []byte) error {
	for at := 0; at+8 <= len(formats); at += 8 {
		if formats[at] == s.depth {
			s.bitsPerPixel = formats[at+1]
			return nil
		}
	}
	return fmt.Errorf("the server announces no pixel format for depth %d, which is its own root depth", s.depth)
}
