// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: MIT OR Apache-2.0

// La connexion au serveur : la socket, le découpage du flux qui en vient, et le
// comptage des requêtes qui y vont.
//
// C'est ce que le backend Windows n'a pas : là-bas le système répond à l'appel, ici il
// y a un flux dans chaque sens et personne pour les mettre en face l'un de l'autre.
// Trois choses vivent donc ici et nulle part ailleurs — le numéro de séquence, par
// lequel une erreur se rattache à sa requête ; le tampon de réception, parce qu'un
// message peut arriver en deux morceaux ; et la plage d'identifiants que le serveur
// nous concède.

package backend

import (
	"errors"
	"fmt"
	"math/bits"
	"net"
	"os"
	"syscall"
)

// conn est une connexion ouverte à un serveur d'affichage.
type conn struct {
	sock *net.UnixConn

	// raw sert les lectures qui ne doivent pas attendre, et c'est la seule raison de
	// descendre sous `net.Conn` ici. Un délai de lecture ne convient pas : posé dans
	// le passé il fait échouer la lecture **sans rendre ce que la socket tient
	// déjà**, et posé dans l'avenir il arrête la boucle de jeu le temps qu'il
	// expire. Un `read` qui rend `EAGAIN` dit exactement ce qu'on demande — « rien
	// pour l'instant » — en un appel système et sans attente.
	raw syscall.RawConn

	setup setup

	// out accumule les requêtes jusqu'au prochain envoi. Le protocole est fait pour
	// cela : empiler quatre requêtes et les envoyer ensemble coûte un aller-retour
	// au lieu de quatre.
	out []byte

	// in porte ce qui est arrivé de la socket, have ce qui y est valide, et cut le
	// premier octet non encore rendu.
	//
	// Un curseur plutôt qu'un déplacement à chaque message : un transfert d'image
	// fera des réponses de plusieurs mégaoctets, et les recopier d'un cran à chaque
	// message consommé coûterait plus que de les lire.
	in   []byte
	have int
	cut  int

	// seq est le numéro de la dernière requête envoyée. Il compte à partir de un, le
	// handshake ne comptant pas.
	seq uint16

	// ids compte les identifiants de ressource déjà pris, et shift place ce compte
	// dans le masque que le serveur a concédé.
	ids   uint32
	shift int

	// pending garde les événements rencontrés en attendant une réponse.
	//
	// Ils sont copiés, et c'est le seul endroit du backend qui copie un message :
	// cela n'arrive qu'à l'ouverture, où quatre atomes se demandent pendant que le
	// serveur annonce déjà l'apparition de la fenêtre. Les perdre ferait manquer la
	// première taille réelle.
	pending [][]byte
}

// connect ouvre une connexion au serveur que `DISPLAY` nomme et lit ce qu'il dit de
// lui-même.
//
// Tout ce qui peut échouer échoue ici, et c'est voulu : une fois cette fonction
// passée, le reste du backend n'a plus à se demander si le serveur existe, s'il a
// accepté l'autorisation ou quelle est la profondeur de l'écran.
func connect() (*conn, error) {
	paths, number, screen, err := displaySockets(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, err
	}

	sock, err := dialAny(paths)
	if err != nil {
		return nil, err
	}

	raw, err := sock.SyscallConn()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("taking hold of the display socket: %w", err)
	}

	c := &conn{sock: sock, raw: raw, in: make([]byte, 4096)}

	name, data := authority(number)
	if _, err := sock.Write(authRequest(name, data)); err != nil {
		sock.Close()
		return nil, fmt.Errorf("greeting the display server: %w", err)
	}

	block, err := readSetup(sock)
	if err != nil {
		sock.Close()
		return nil, err
	}
	if c.setup, err = parseSetup(block, screen); err != nil {
		sock.Close()
		return nil, err
	}

	// Le masque des identifiants est une plage de bits contigus que la spécification
	// ne promet pas de placer en bas. Les serveurs courants le font, et c'est
	// justement pourquoi un décalage en dur passerait les contrôles ici pour tomber
	// ailleurs : le compte se place dans le masque, où qu'il soit.
	c.shift = bits.TrailingZeros32(c.setup.idMask)

	return c, nil
}

// dialAny essaie les chemins dans l'ordre et rend la première connexion obtenue.
//
// Les échecs sont rendus ensemble et non seulement le dernier : les deux formes de
// socket échouent pour des raisons différentes — un fichier absent, une socket
// abstraite qui n'écoute pas — et savoir laquelle a manqué est ce qui distingue « il
// n'y a pas de serveur » de « le serveur n'écoute pas là où on l'a cherché ».
func dialAny(paths []string) (*net.UnixConn, error) {
	var failures []error
	for _, path := range paths {
		sock, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
		if err == nil {
			return sock, nil
		}
		failures = append(failures, err)
	}
	return nil, fmt.Errorf("no display server answered: %w", errors.Join(failures...))
}

// Close rend la socket. Les ressources créées sur le serveur partent avec elle, le
// protocole les détruisant à la fermeture d'une connexion.
func (c *conn) Close() {
	if c.sock != nil {
		c.sock.Close()
		c.sock = nil
	}
}

// next compte une requête de plus et rend son numéro de séquence.
//
// C'est ce numéro que portent les erreurs et les réponses, et c'est la seule façon de
// rattacher une erreur à ce qui l'a causée : le serveur ne répond pas aux requêtes qui
// réussissent, donc une erreur arrive plus tard, pendant qu'on fait autre chose. Un
// décalage d'un seul rang ferait accuser la requête voisine.
func (c *conn) next() uint16 {
	c.seq++
	return c.seq
}

// id réserve un identifiant de ressource dans la plage que le serveur a concédée.
func (c *conn) id() uint32 {
	c.ids++
	return c.setup.idBase | (c.ids<<c.shift)&c.setup.idMask
}

// flush envoie ce que les requêtes ont accumulé.
//
// Rien ne part avant : le protocole n'exige aucune réponse pour la plupart des
// requêtes, donc une fenêtre entière — création, titre, bornes, apparition — tient en
// un seul envoi.
func (c *conn) flush() error {
	if len(c.out) == 0 {
		return nil
	}
	if _, err := c.sock.Write(c.out); err != nil {
		return fmt.Errorf("writing to the display server: %w", err)
	}
	c.out = c.out[:0]
	return nil
}

// room garantit place octets libres derrière ce qui est déjà reçu.
//
// Elle compacte d'abord, ce qui suffit presque toujours : les messages se consomment
// au rythme où ils arrivent, donc l'espace repris en tête est celui que la lecture
// suivante demande. L'agrandissement ne sert qu'aux réponses longues.
func (c *conn) room(need int) {
	if len(c.in)-c.have >= need {
		return
	}
	if c.cut > 0 {
		copy(c.in, c.in[c.cut:c.have])
		c.have -= c.cut
		c.cut = 0
	}
	if len(c.in)-c.have < need {
		grown := make([]byte, 2*(c.have+need))
		copy(grown, c.in[:c.have])
		c.in = grown
	}
}

// message rend le prochain message complet déjà reçu, ou nil s'il en manque des
// octets.
//
// La tranche rendue pointe dans le tampon de réception et ne vaut que jusqu'à la
// lecture suivante. C'est ce qui évite une copie par événement, et aucun appelant ne
// la garde.
func (c *conn) message() []byte {
	available := c.have - c.cut
	if available < eventSize {
		return nil
	}

	size := eventSize
	if c.in[c.cut] == msgReply {
		// Seule une réponse peut dépasser les trente-deux octets, et elle dit de
		// combien de mots de quatre octets.
		size += 4 * int(order.Uint32(c.in[c.cut+4:]))
	}
	if available < size {
		c.room(size - available)
		return nil
	}

	m := c.in[c.cut : c.cut+size]
	c.cut += size
	return m
}

// receive lit ce que la socket tient déjà, sans attendre, et dit si quelque chose est
// arrivé.
func (c *conn) receive() (bool, error) {
	c.room(eventSize)

	var n int
	var errno error
	if err := c.raw.Read(func(fd uintptr) bool {
		for {
			n, errno = syscall.Read(int(fd), c.in[c.have:])
			// Un signal interrompt la lecture sans que rien n'ait été lu ni
			// perdu. Elle se reprend, ce que la couche réseau ferait aussi.
			if errno != syscall.EINTR {
				return true
			}
		}
	}); err != nil {
		return false, fmt.Errorf("reading from the display server: %w", err)
	}

	switch {
	case errno == syscall.EAGAIN:
		return false, nil
	case errno != nil:
		return false, fmt.Errorf("reading from the display server: %w", errno)
	case n == 0:
		// Une lecture qui rend zéro sans erreur est la fermeture du pair, et c'est
		// le régime que le contrat de fenêtre a prévu : un serveur d'affichage qui
		// meurt n'est pas un joueur qui ferme la fenêtre.
		return false, errors.New("the display server closed the connection")
	}

	c.have += n
	return true, nil
}

// poll rend le prochain message du serveur, ou nil quand rien n'attend.
//
// C'est ce qu'une image emploie : la boucle ne peut pas se permettre d'attendre un
// événement qui ne viendra pas.
func (c *conn) poll() ([]byte, error) {
	for {
		if m := c.message(); m != nil {
			return m, nil
		}
		got, err := c.receive()
		if err != nil {
			return nil, err
		}
		if !got {
			return nil, nil
		}
	}
}

// await rend le prochain message du serveur, en attendant qu'il arrive.
//
// Réservé à l'ouverture, où l'on sait qu'une réponse vient. L'employer dans une image
// arrêterait le jeu jusqu'au prochain événement, c'est-à-dire jusqu'à ce que le joueur
// touche quelque chose.
func (c *conn) await() ([]byte, error) {
	for {
		if m := c.message(); m != nil {
			return m, nil
		}
		c.room(eventSize)
		n, err := c.sock.Read(c.in[c.have:])
		if err != nil {
			return nil, fmt.Errorf("waiting on the display server: %w", err)
		}
		c.have += n
	}
}

// reply attend la réponse de la requête seq, en mettant de côté les événements qui la
// précèdent.
//
// **Toute erreur reçue est rendue, quelle que soit la requête qu'elle accuse.** Une
// erreur de protocole ne se rattrape pas : elle dit qu'une requête était fausse, donc
// que ce qui la suivait s'est appuyé sur quelque chose qui n'existe pas. La poursuivre
// ferait rapporter l'échec quelques requêtes plus loin, sur un objet sans rapport.
func (c *conn) reply(seq uint16) ([]byte, error) {
	for {
		m, err := c.await()
		if err != nil {
			return nil, err
		}

		switch {
		case m[0] == msgError:
			return nil, protocolError(m)

		case m[0] == msgReply:
			if got := order.Uint16(m[2:]); got != seq {
				return nil, fmt.Errorf("the display server answered request %d while %d was awaited", got, seq)
			}
			return m, nil

		default:
			c.pending = append(c.pending, append([]byte(nil), m...))
		}
	}
}

// protocolError nomme une erreur du protocole, telle que le serveur l'a envoyée.
//
// Le numéro de séquence y est, et c'est lui qui sert : il dit laquelle des requêtes
// envoyées était fausse, là où le code d'erreur ne dit que de quelle sorte.
func protocolError(m []byte) error {
	return fmt.Errorf("the display server refused request %d with a %s error: value %#x, opcode %d.%d",
		order.Uint16(m[2:]), errorName(m[1]), order.Uint32(m[4:]), m[10], order.Uint16(m[8:]))
}
